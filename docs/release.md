# Release baseline

## Build
- Backend image: `podman build -f src/backend/Dockerfile -t <registry>/roomies-backend:<tag> src/backend`
- Frontend bundle image: `podman build -f src/app/Dockerfile -t <registry>/roomies-frontend:<tag> .`
- Android APK with an optimized Rust payload: `nix develop .#android --command make android`

The Android command uses the repository-pinned Rust 1.89.0, Dioxus CLI 0.7.10, Android platform 34/build-tools 34.0.0, and NDK 27.0.12077973. Android CI explicitly sets both NDK discovery variables to that revision. The Nix Dioxus wrapper includes wasm-bindgen CLI 0.2.127 to match Cargo.lock; the separate Nix web-shell limitation (missing wasm32-unknown-unknown target) remains. Without a signing configuration, Dioxus intentionally runs Gradle's `assembleDebug` and produces `target/dx/roomies-app/release/android/app/app/build/outputs/apk/debug/app-debug.apk`; this needs no external credentials and is suitable for development/runtime acceptance, not store publication. Device/emulator Keystore behavior must still be accepted separately.

## Render manifests
- `kustomize build deploy/kustomize/overlays/production`

## Owner publication and Kubernetes boundary
The checked-in production overlay is a non-deployable placeholder. Never apply it
directly. First publish images under the owner-selected registry and create a
publication overlay replacing **every** image (including Caddy) with a verified,
published `name@sha256:<64-hex>` reference. Local rehearsal digests are not proof
of registry availability. Render that owner overlay to `published.yaml`, verify
all image references are digest-pinned (no tags such as `latest`), and complete
owner cluster/secrets/TLS checks before any apply. No deployment command against
the mutable template is provided here. Validate the owner-supplied render before
requesting deployment approval (this does not prove registry pullability):

```bash
kustomize build "$PUBLICATION_OVERLAY" > published.yaml
awk '/^[[:space:]]*image:/ { count++; if ($2 !~ /@sha256:[0-9a-f]{64}$/) bad=1 } END { exit (bad || count == 0) }' published.yaml
```

The local rehearsal output intentionally does not pass this publication gate.

For credential-free rehearsal, run:
`nix develop .#container -c env RELEASE_SHA="$(git rev-parse HEAD)" RELEASE_VERSION=local-review make release-dry-run`.
This output is local-only, not the owner publication overlay.

## Required secrets
Provide the values from `docs/credentials.example.env` via a Kubernetes Secret or your secret manager:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*`
- `INVITATION_DELIVERY_KEY_ID`, `INVITATION_DELIVERY_KEY`, and (during rotation) `INVITATION_DELIVERY_OLD_KEYS`
- `S3_*`
- `NOTIFICATION_DELIVERY_KEY_ID`, `NOTIFICATION_DELIVERY_KEY`, and rotation-only `NOTIFICATION_DELIVERY_OLD_KEYS` (dedicated AES-256 key; never reuse JWT/invitation keys)
- `WEB_PUSH_PUBLIC_KEY`, `WEB_PUSH_PRIVATE_KEY`, and `WEB_PUSH_SUBJECT`
- `FCM_PROJECT_ID`; use workload identity or a Secret-mounted ADC file selected by `GOOGLE_APPLICATION_CREDENTIALS`

## Notes

The backend builder and CI use Go 1.26.8 with GOTOOLCHAIN=local because the pinned `github.com/marknefedov/go-webpush/v2` v2.0.0 module declares Go 1.26 as its minimum. Recurrence is pinned to `github.com/teambition/rrule-go` v1.8.2; expansion is bounded and never calls `All`.
- The backend runs migrations on startup.
- `SMTP_HOST` selects SMTP in every environment (use Mailpit for local development); leaving it empty selects the fake non-delivery provider.
- Invitation creation returns `manual_acceptance_url` only in the initial successful response. Idempotent replays retain the invitation result but omit its one-time bearer URL. When SMTP is configured, the same one-time URL is AES-256-GCM encrypted in the outbox with invitation, house, and job context as associated data; the worker decrypts it only immediately before SMTP dispatch.
- `SMTP_HOST` requires the distinct operator-supplied `INVITATION_DELIVERY_KEY_ID` and base64 32-byte `INVITATION_DELIVERY_KEY`; startup fails explicitly without them. Use a generated non-committed test key for Mailpit fixtures. To rotate, make the new key active and keep old `key-id:base64-key` entries in `INVITATION_DELIVERY_OLD_KEYS` until all invitations encrypted with them are delivered or terminal. Terminal invitation states and completed deliveries redact ciphertext from outbox storage.
- SMTP is at-least-once: a crash after SMTP accepts a message can cause a retry. Messages carry a stable invitation-based `Message-ID`; Roomies makes no exactly-once delivery claim.
- The frontend image is a static bundle source; the Kubernetes overlay serves it with Caddy.
- No Apple target or Apple packaging work is included in this baseline.
