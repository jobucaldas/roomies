# Production overlay

This overlay is intended for the user's Kubernetes cluster.

Provide the `roomies-secrets` Secret from `docs/credentials.example.env` before applying:
- `DATABASE_URL`
- `JWT_SECRET`
- `SMTP_*` (SMTP_HOST/PORT/USERNAME/PASSWORD/FROM enable real invite delivery)
- `INVITATION_DELIVERY_KEY_ID` and `INVITATION_DELIVERY_KEY` (a distinct base64 32-byte AES-256 key); retain old `key-id:base64-key` values in `INVITATION_DELIVERY_OLD_KEYS` until pending invitation jobs are terminal.

Workers use SMTP in every environment when SMTP_HOST is configured. SMTP startup fails without the invitation delivery key; Mailpit uses a generated, non-committed fixture key. If SMTP is absent, the worker runs an explicitly fake-only provider and records no claim of recipient delivery; configure SMTP before production use.
- `S3_*`
- `WEB_PUSH_*`
- `FCM_*`

Render with:
```bash
kustomize build deploy/kustomize/overlays/production
```
