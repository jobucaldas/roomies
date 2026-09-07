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
