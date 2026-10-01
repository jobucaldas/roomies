# Release baseline

## Build
- Backend image: `podman build -f src/backend/Dockerfile -t <registry>/roomies-backend:<tag> src/backend`
- Frontend image: `podman build -f src/flutter/Dockerfile -t <registry>/roomies-frontend:<tag> .`
- Android APK: `cd src/flutter && flutter build apk --release --dart-define=ROOMIES_API_URL=` (debug-signed unless you configure signing)

## Render manifests
```bash
kustomize build deploy/kustomize/overlays/production
```

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

For credential-free local rehearsal (needs podman, skopeo, kustomize, syft, jq):

```bash
RELEASE_SHA="$(git rev-parse HEAD)" RELEASE_VERSION=local-review make release-dry-run
```

Output is local-only under `artifacts/release/<sha>/`, not a publication overlay.

## Required secrets
Provide the values from `docs/credentials.example.env` via a Kubernetes Secret or your secret manager:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*`
- `INVITATION_DELIVERY_KEY_ID`, `INVITATION_DELIVERY_KEY`, and (during rotation) `INVITATION_DELIVERY_OLD_KEYS`
- `S3_*`
- `NOTIFICATION_DELIVERY_KEY_ID`, `NOTIFICATION_DELIVERY_KEY`, and rotation-only `NOTIFICATION_DELIVERY_OLD_KEYS`
- `WEB_PUSH_PUBLIC_KEY`, `WEB_PUSH_PRIVATE_KEY`, and `WEB_PUSH_SUBJECT`
- `FCM_PROJECT_ID`; use workload identity or a Secret-mounted ADC file selected by `GOOGLE_APPLICATION_CREDENTIALS`

## Notes
- The backend pins Go with `GOTOOLCHAIN=local` where required by module constraints.
- The backend runs migrations on startup.
- Empty `SMTP_HOST` selects the fake non-delivery provider; Mailpit is the usual local sink.
- Invitation creation returns `manual_acceptance_url` only on the first successful response.
- The frontend image is a static Flutter web bundle; the Kubernetes overlay serves it with Caddy.
