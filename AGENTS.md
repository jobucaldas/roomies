# Roomies Agent Notes

Roomies is a roommate-management app for shared expenses, notes, balances, settlements, and house-scoped household workflows.

## Baseline decisions
- Supported client targets: web, Linux desktop, Android only.
- Production runs in the user’s Kubernetes cluster.
- Publish OCI images and portable Kustomize manifests.
- Cloudflare R2 is the production blob store.
- Local filesystem / SeaweedFS-compatible fake storage is fine for dev and tests.
- Email uses generic SMTP; Mailpit is the dev/test sink.
- Notifications use Web Push plus Android FCM; fake adapters are used when credentials are absent.
- No external payment integration: users record settlements/payments and notify active house members.
- Privacy-first policy: no admin override for private expense/receipt/OCR data, deleted files purge after 30 days, chat retention is one year with tombstones, exports are audited and expire after 24 hours, former members retain financial identity only, platform key stores are required for native session secrets, and clients are online-first with read caching plus idempotent retry only.
- All new household resources are house-scoped.
- Use configurable time zones/cadence and RFC 5545 recurrence with exceptions.
- No Apple work.

## Layout
- Backend: `src/backend/`
- Frontend: `src/app/`
- Release manifests: `deploy/kustomize/`
- Docs: `docs/`

## Baseline commands
- `make check-compose`
- `make test-backend`
- `make test-frontend`
- `make build`
- `make smoke`
- `make render-manifests`
- `make clean`

## Working rules
- Preserve completed MVP changes and intended deletions.
- Do not add secrets or generated binaries to the repo.
- Keep docs, templates, and release artifacts in sync with code.
- Prefer narrow edits and validate with the smallest useful test set.
