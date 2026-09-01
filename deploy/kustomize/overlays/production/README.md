# Production overlay

This overlay is intended for the user's Kubernetes cluster.

Provide the `roomies-secrets` Secret from `docs/credentials.example.env` before applying:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*`
- `S3_*`
- `WEB_PUSH_*`
- `FCM_*`

Render with:
```bash
kustomize build deploy/kustomize/overlays/production
```
