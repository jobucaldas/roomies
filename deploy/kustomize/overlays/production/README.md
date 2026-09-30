# Production overlay

Kustomize overlay for deploying Roomies to Kubernetes.

Provide the `roomies-secrets` Secret from `docs/credentials.example.env` before applying:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*` (SMTP_HOST/PORT/USERNAME/PASSWORD/FROM enable invite delivery)
- `INVITATION_DELIVERY_KEY_ID` and `INVITATION_DELIVERY_KEY` (base64 32-byte AES-256 key); retain old `key-id:base64-key` values in `INVITATION_DELIVERY_OLD_KEYS` until pending invitation jobs are terminal.
- `S3_*`
- `NOTIFICATION_DELIVERY_KEY_ID` and `NOTIFICATION_DELIVERY_KEY` (dedicated base64 32-byte AES-256 key; do not reuse JWT/invitation keys); retain rotation overlap in `NOTIFICATION_DELIVERY_OLD_KEYS`.
- `WEB_PUSH_PUBLIC_KEY` / `WEB_PUSH_PRIVATE_KEY` for Web Push. If omitted, that channel does not deliver.
- `FCM_PROJECT_ID` plus workload identity for FCM HTTP v1. If a service-account file is required, mount it as a Secret volume and set `GOOGLE_APPLICATION_CREDENTIALS`; do not put JSON directly in an environment variable.

Workers use SMTP when `SMTP_HOST` is set; startup fails without the invitation delivery key. Without SMTP, the worker uses a fake non-delivery provider. Without push credentials, the worker uses a fake notification adapter. Real push channels require notification delivery encryption at startup.

Render with:
```bash
kustomize build deploy/kustomize/overlays/production
```

Do not apply this template unchanged. Replace every image (including Caddy) with digest-pinned published references in a publication overlay before deploy. See `docs/release.md`.

## Credential-free local release rehearsal

Run `nix develop .#container -c make release-dry-run` from a clean commit.
Output lands under ignored `artifacts/release/<full-commit-sha>/` and includes rendered YAML, local OCI archives, provenance labels, SBOMs, and `SHA256SUMS`. Existing bundles are never overwritten. `RELEASE_SHA` defaults to HEAD and must match the clean checked-out HEAD. `RELEASE_VERSION` defaults to the full SHA; explicit values must be 1–128-character OCI tags excluding `latest`.

This is local rehearsal only, not proof that registry digests are pullable.

## Workload security boundary

Base and production manifests apply the same hardening to backend, worker, frontend bundle init, and Caddy:
`runAsNonRoot`, `allowPrivilegeEscalation: false`, `RuntimeDefault` seccomp,
`capabilities.drop: [ALL]`, and a read-only root filesystem. No `hostPath` volumes.

- Backend image account: UID 100 / GID 101; `/tmp` is an `emptyDir`. Production must supply PostgreSQL `DATABASE_URL`.
- Frontend bundle init: UID/GID 65532:65532.
- Caddy: UID 65534 with GID/fsGroup 65532; site volume read-only; `/data`, `/config`, and `/tmp` are writable `emptyDir` volumes. Caddy listens on port 8080; the frontend Service remains port 80.
