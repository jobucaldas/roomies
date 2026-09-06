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
- `INVITATION_DELIVERY_KEY_ID`, `INVITATION_DELIVERY_KEY`, and (during rotation) `INVITATION_DELIVERY_OLD_KEYS`
- `S3_*`
- `WEB_PUSH_*`
- `FCM_*`

## Notes
- The backend runs migrations on startup.
- `SMTP_HOST` selects SMTP in every environment (use Mailpit for local development); leaving it empty selects the fake non-delivery provider.
- Invitation creation returns `manual_acceptance_url` only in the initial successful response. Idempotent replays retain the invitation result but omit its one-time bearer URL. When SMTP is configured, the same one-time URL is AES-256-GCM encrypted in the outbox with invitation, house, and job context as associated data; the worker decrypts it only immediately before SMTP dispatch.
- `SMTP_HOST` requires the distinct operator-supplied `INVITATION_DELIVERY_KEY_ID` and base64 32-byte `INVITATION_DELIVERY_KEY`; startup fails explicitly without them. Use a generated non-committed test key for Mailpit fixtures. To rotate, make the new key active and keep old `key-id:base64-key` entries in `INVITATION_DELIVERY_OLD_KEYS` until all invitations encrypted with them are delivered or terminal. Terminal invitation states and completed deliveries redact ciphertext from outbox storage.
- SMTP is at-least-once: a crash after SMTP accepts a message can cause a retry. Messages carry a stable invitation-based `Message-ID`; Roomies makes no exactly-once delivery claim.
- The frontend image is a static bundle source; the Kubernetes overlay serves it with Caddy.
- No Apple target or Apple packaging work is included in this baseline.
