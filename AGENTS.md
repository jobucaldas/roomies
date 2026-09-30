# Roomies Agent Notes

Roomies is a roommate-management app for shared expenses, notes, balances, settlements, and house-scoped household workflows.

## Codebase facts
- Client targets in this repo: web, Linux desktop, and Android.
- Backend: `src/backend/`
- Production web client: `src/flutter/`
- Rust reference client: `src/app/`
- Release manifests: `deploy/kustomize/`
- Docs: `docs/`
- Production blob store configuration uses S3-compatible settings (for example Cloudflare R2); local filesystem storage is fine for dev and tests.
- Email uses generic SMTP; Mailpit is the common local sink.
- Notifications use Web Push plus Android FCM; fake adapters are used when credentials are absent.
- Household resources are house-scoped. Recurrence uses RFC 5545 with exceptions.

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
- Public docs state what/how for running and self-hosting; do not add private product or design rationale.
