# Production overlay

This overlay is intended for the user's Kubernetes cluster.

Provide the `roomies-secrets` Secret from `docs/credentials.example.env` before applying:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*` (SMTP_HOST/PORT/USERNAME/PASSWORD/FROM enable real invite delivery)

Workers use real SMTP only outside development/test when SMTP_HOST is configured. If SMTP is absent, the worker runs an explicitly fake-only provider and records no claim of recipient delivery; configure SMTP before production use.
- `S3_*`
- `WEB_PUSH_*`
- `FCM_*`

Render with:
```bash
kustomize build deploy/kustomize/overlays/production
```
