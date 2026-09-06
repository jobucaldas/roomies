# Release baseline

## Build
- Backend image: `podman build -f src/backend/Dockerfile -t <registry>/roomies-backend:<tag> src/backend`
- Frontend bundle image: `podman build -f src/app/Dockerfile -t <registry>/roomies-frontend:<tag> .`

## Render manifests
- `kustomize build deploy/kustomize/overlays/production`

## Apply to Kubernetes
- `kubectl apply -k deploy/kustomize/overlays/production`

## Required secrets
Provide the values from `docs/credentials.example.env` via a Kubernetes Secret or your secret manager:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*`
- `S3_*`
- `WEB_PUSH_*`
- `FCM_*`

## Notes
- The backend runs migrations on startup.
- `SMTP_HOST` selects SMTP in every environment (use Mailpit for local development); leaving it empty selects the fake non-delivery provider.
- Invitation creation returns `manual_acceptance_url` only in the initial successful response. Idempotent replays retain the invitation result but omit its one-time bearer URL.
- The frontend image is a static bundle source; the Kubernetes overlay serves it with Caddy.
- No Apple target or Apple packaging work is included in this baseline.
