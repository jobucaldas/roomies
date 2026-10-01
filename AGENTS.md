# Roomies Agent Notes

Roomies is a roommate-management app for shared expenses, notes, balances, settlements, and house-scoped household workflows.

## Codebase facts
- Client: Flutter under `src/flutter/` (web + Android).
- Backend: Go under `src/backend/`.
- Release manifests: `deploy/kustomize/`.
- Docs: `docs/` (what/how only).
- Blob store: filesystem locally, S3-compatible (for example R2) in production.
- Email: SMTP (Mailpit in local/test).
- Notifications: Web Push + Android FCM; fakes when credentials are absent.
- Household resources are house-scoped. Recurrence uses RFC 5545 with exceptions.

## Baseline commands
- `make check-compose`
- `make test-backend`
- `make test-flutter`
- `make build`
- `make smoke`
- `make render-manifests`
- `make clean`

## Working rules
- Preserve completed MVP changes and intended deletions.
- Do not add secrets or generated binaries to the repo.
- Keep docs and release artifacts in sync with code.
- Prefer narrow edits and validate with the smallest useful test set (unit/lint/build).
- Product flows are verified by agents/cloud as needed; do not reintroduce heavy browser CI gates as the main quality bar.
- Public docs state what/how for running and self-hosting; do not add private product or design rationale.
- Keep the repo lean: Flutter + Go only — no Rust, Nix, or AI scratch docs in git.
