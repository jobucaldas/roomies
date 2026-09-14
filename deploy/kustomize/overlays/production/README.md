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
never overwritten. `RELEASE_SHA` defaults to HEAD and must resolve to the clean
checked-out HEAD; alternate commits and any non-ignored dirty/untracked files
are rejected before creating a bundle. `RELEASE_VERSION` defaults to the full
resolved SHA; explicit values must be 1–128-character OCI tags, excluding
`latest`. It is recorded in release metadata and Roomies OCI version labels;
image tags and bundle directory remain SHA-addressed. Source, lockfile and
Kustomize inputs all come from `git archive` of that commit.

Builds run in a temporary sibling directory, removed on failure; only fully
validated, checksummed output is atomically renamed to the final SHA directory.
Failed runs can be retried. A completed SHA bundle is never overwritten, even
when a different version is requested (move it aside explicitly first).

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

## Workload security boundary

The base and production manifests now apply the same bounded hardening to all four
Roomies workload containers (backend, worker, frontend bundle init, and Caddy):
`runAsNonRoot`, `allowPrivilegeEscalation: false`, `RuntimeDefault` seccomp,
`capabilities.drop: [ALL]`, and a read-only root filesystem. No `hostPath` volumes
are used.

The backend image's Alpine `roomies` account is UID 100/GID 101. Both backend
containers enforce those IDs, use fsGroup 101, and mount only an `/tmp` `emptyDir`;
production must still provide the required PostgreSQL `DATABASE_URL` rather than
relying on the image's writable SQLite default. The frontend bundle init runs as
UID/GID 65532:65532 and copies into the `site` volume without preserving root
ownership. The Caddy 2.8 Alpine image's `nobody` UID 65534 runs with GID/fsGroup
65532. The site volume is read-only to Caddy; `/data` and `/config` are separate
writable `emptyDir` volumes for Caddy runtime state, and `/tmp` is a writable
`emptyDir` used to copy the image's setcap-marked binary to a capability-free
path before execution. Caddy listens on unprivileged port 8080, while the
frontend Service remains port 80 and targets the named `http` port. The Caddyfile
is read-only.

These are manifest/render and local container checks only. Registry
namespace/publication, multi-architecture target policy, production
secrets/providers, ingress/TLS, cluster admission/runtime, signing and
attestations remain owner/environment gates. No cluster acceptance is claimed.
The rehearsal also does not accept native runtime, Linux Secret Service, or
Android Keystore behavior.
