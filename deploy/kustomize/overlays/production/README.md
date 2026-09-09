# Production overlay

This overlay is intended for the user's Kubernetes cluster.

Provide the `roomies-secrets` Secret from `docs/credentials.example.env` before applying:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*` (SMTP_HOST/PORT/USERNAME/PASSWORD/FROM enable real invite delivery)
- `INVITATION_DELIVERY_KEY_ID` and `INVITATION_DELIVERY_KEY` (a distinct base64 32-byte AES-256 key); retain old `key-id:base64-key` values in `INVITATION_DELIVERY_OLD_KEYS` until pending invitation jobs are terminal.

Workers use SMTP in every environment when SMTP_HOST is configured. SMTP startup fails without the invitation delivery key; Mailpit uses a generated, non-committed fixture key. If SMTP is absent, the worker runs an explicitly fake-only provider and records no claim of recipient delivery; configure SMTP before production use.
- `S3_*`
- `NOTIFICATION_DELIVERY_KEY_ID` and `NOTIFICATION_DELIVERY_KEY` (a dedicated base64 32-byte AES-256 key, never the JWT/invitation key); retain rotation overlap in `NOTIFICATION_DELIVERY_OLD_KEYS`.
- `WEB_PUSH_PUBLIC_KEY` / `WEB_PUSH_PRIVATE_KEY` for Web Push. If omitted, that channel is non-delivering.
- `FCM_PROJECT_ID` plus GKE workload identity for FCM HTTP v1. If a service-account file is unavoidable, mount it as a Secret volume and set `GOOGLE_APPLICATION_CREDENTIALS`; never put JSON directly in an environment variable.

If all push credentials are absent, the worker uses a clearly logged fake-only notification adapter. Any real push channel requires notification delivery encryption at startup.

Render with:
```bash
kustomize build deploy/kustomize/overlays/production
```

## Credential-free local release rehearsal

Run `nix develop .#container -c make release-dry-run` from a clean commit.
The ignored `artifacts/release/<full-commit-sha>/` contains production YAML,
local OCI archives, raw manifests/configs, OCI provenance labels, image metadata,
source archive, Syft SPDX JSON SBOMs, tool versions, the flake lock, and verified `SHA256SUMS`. Existing bundles are
never overwritten. Dirty tooling runs are marked preliminary and are not release
acceptance. Source builds always use `git archive HEAD`, excluding local secrets
and generated files. The final run must use a clean committed tree.

Roomies images in the bundle use explicit local SHA tags, not `latest` or an
unapproved registry namespace. The template above remains an owner-customized
publication template; do not deploy it unchanged. Archive manifest digests are
actual local OCI content identities, not claims that registry digest references
are pullable. Caddy's version-tagged runtime is also archived and inspected.
This is a reproducible procedure, not a bit-for-bit build guarantee: upstream
base tags and dependency downloads are not all content-pinned. The flake-locked Syft generator scans local OCI archives without registry access
or update checks. SBOMs are inventories, not signed attestations; scanner IDs and
timestamps are not promised byte-reproducible. Archive blobs, manifests, configs,
and index linkage are verified before the bundle receives its checksum file.

R2/S3 remains the production blob-store baseline for future private files. The
current backend has no blob-consuming feature; `S3_*` is reserved configuration,
not evidence of an implemented or validated storage adapter.

Registry namespace/publication, multi-architecture target policy, production
secrets/providers, ingress/TLS, cluster acceptance, signing and attestations are
owner/environment gates. This rehearsal does not accept native runtime, Linux
Secret Service, or Android Keystore behavior. Frontend Kubernetes security-context
hardening remains a separate bounded follow-up.
