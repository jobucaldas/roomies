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
- The frontend image is a static bundle source; the Kubernetes overlay serves it with Caddy.
- No Apple target or Apple packaging work is included in this baseline.
