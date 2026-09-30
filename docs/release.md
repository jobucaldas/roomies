# Release baseline

## Build
- Backend image: `podman build -f src/backend/Dockerfile -t <registry>/roomies-backend:<tag> src/backend`
- Frontend bundle image: `podman build -f src/flutter/Dockerfile -t <registry>/roomies-frontend:<tag> .`
- Rust web release bundle: `nix develop .#web --command sh -c 'cd src/app && dx build --release --debug-symbols=false'`
- Android APK with an optimized Rust payload: `nix develop .#android --command make android`

The Rust web command uses the pinned toolchain in the Nix web shell and writes the bundle under `target/dx/roomies-app/release/web/public/`.

Without a signing configuration, the Android build produces a debug-signed APK under `target/dx/roomies-app/release/android/app/app/build/outputs/apk/debug/app-debug.apk`, suitable for development and runtime checks, not store publication.

## Render manifests
- `kustomize build deploy/kustomize/overlays/production`

## Publication and Kubernetes boundary
The checked-in production overlay is a non-deployable placeholder. Do not apply it
directly. Publish images under your registry and create a publication overlay that
replaces **every** image (including Caddy) with a verified `name@sha256:<64-hex>`
reference. Render that overlay to `published.yaml`, verify all image references
are digest-pinned (no tags such as `latest`), and complete cluster/secrets/TLS
checks before any apply:

```bash
kustomize build "$PUBLICATION_OVERLAY" > published.yaml
awk '/^[[:space:]]*image:/ { count++; if ($2 !~ /@sha256:[0-9a-f]{64}$/) bad=1 } END { exit (bad || count == 0) }' published.yaml
```

For credential-free local rehearsal, run:
`nix develop .#container -c env RELEASE_SHA="$(git rev-parse HEAD)" RELEASE_VERSION=local-review make release-dry-run`.
That output is local-only, not a publication overlay.

## Required secrets
Provide the values from `docs/credentials.example.env` via a Kubernetes Secret or your secret manager:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*`
- `INVITATION_DELIVERY_KEY_ID`, `INVITATION_DELIVERY_KEY`, and (during rotation) `INVITATION_DELIVERY_OLD_KEYS`
- `S3_*`
- `NOTIFICATION_DELIVERY_KEY_ID`, `NOTIFICATION_DELIVERY_KEY`, and rotation-only `NOTIFICATION_DELIVERY_OLD_KEYS` (dedicated AES-256 key; do not reuse JWT/invitation keys)
- `WEB_PUSH_PUBLIC_KEY`, `WEB_PUSH_PRIVATE_KEY`, and `WEB_PUSH_SUBJECT`
- `FCM_PROJECT_ID`; use workload identity or a Secret-mounted ADC file selected by `GOOGLE_APPLICATION_CREDENTIALS`

## Notes
- The backend builder and CI pin Go with `GOTOOLCHAIN=local` to match module requirements of `github.com/marknefedov/go-webpush/v2`. Recurrence uses `github.com/teambition/rrule-go`; expansion is bounded and does not call `All`.
- The backend runs migrations on startup.
- `SMTP_HOST` selects SMTP; leaving it empty selects the fake non-delivery provider. Mailpit is the usual local sink.
- Invitation creation returns `manual_acceptance_url` only in the initial successful response. Idempotent replays omit the one-time URL. When SMTP is configured, the URL is encrypted in the outbox and decrypted only immediately before dispatch.
- `SMTP_HOST` requires `INVITATION_DELIVERY_KEY_ID` and a base64 32-byte `INVITATION_DELIVERY_KEY`. To rotate, make the new key active and keep old `key-id:base64-key` entries in `INVITATION_DELIVERY_OLD_KEYS` until pending jobs are terminal.
- SMTP is at-least-once. Messages carry a stable invitation-based `Message-ID`.
- The frontend image is a static bundle; the Kubernetes overlay serves it with Caddy.
