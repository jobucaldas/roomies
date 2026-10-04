# Roomies Agent Notes

Roomies is a roommate-management app for shared expenses, notes, balances, settlements, and house-scoped household workflows.

## Codebase facts
- Client: Flutter under `src/frontend/` (web, Android, Windows, Linux).
- Backend: Go under `src/backend/`.
- CI and image publishing: `.github/workflows/ci.yml` (GHCR images on `main`).
- Docker Compose is the only entry point: `docker-compose.yml` is the dev stack (built from the checkout); end users run the compose file embedded in `README.md` against GHCR images. No Makefile.
- The web image (`src/frontend/Dockerfile`) embeds its Caddyfile; there is no standalone Caddyfile.
- Docs: `README.md` only.
- Blob store: filesystem locally, S3-compatible (for example R2) in production.
- Email: SMTP (Mailpit in local/test).
- Notifications: Web Push + Android FCM; fakes when credentials are absent.
- Household resources are house-scoped. Recurrence uses RFC 5545 with exceptions.

## Baseline commands
- `docker compose up --build` (app at http://localhost:8080)
- `docker compose run --rm test-backend`
- `docker compose run --rm test-frontend`
- `docker compose run --rm apk`
- `docker compose down -v`

## Working rules
- Preserve completed MVP changes and intended deletions.
- Do not add secrets or generated binaries to the repo.
- Keep `README.md`, `.env.example`, and CI in sync with code.
- Prefer narrow edits and validate with the smallest useful test set (unit/lint/build).
- Product flows are verified by agents/cloud as needed; do not reintroduce heavy browser CI gates as the main quality bar.
- The README states what/how for running and self-hosting; do not add private product or design rationale or extra doc files.
- Keep the repo lean: Flutter + Go only — no Rust, Nix, helper-script folders, browser e2e suites, or AI scratch docs in git. CI logic lives in the workflow YAML.
