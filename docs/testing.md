# Testing baseline

## Backend
- SQLite: `cd src/backend && DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go test ./... -v`
- PostgreSQL: `cd src/backend && DATABASE_URL=postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable JWT_SECRET=dev-secret go test ./... -v`

## Frontend
- Web: `cargo test -p roomies-app --no-default-features --features web`
- Desktop: `cargo check -p roomies-app --no-default-features --features desktop`
- Android: `cd src/app && dx build --release --android`

## Invitation onboarding browser flow
The reusable Playwright harness uses synthetic accounts and Mailpit; it never prints or renders bearer tokens. Start the existing stack with SMTP pointed at the disposable sink, then run:

```sh
podman compose -f e2e/docker-compose.mailpit.yml up -d
SMTP_HOST=localhost SMTP_PORT=1025 ROOMIES_PUBLIC_BASE_URL=http://localhost \
  podman compose -f docker-compose.yml up --build -d
cd e2e && npm ci && npx playwright install --with-deps chromium && npm test
podman compose -f docker-compose.yml down -v
podman compose -f e2e/docker-compose.mailpit.yml down -v
```

Screenshots and Playwright reports are written under `e2e/artifacts/` (ignored). The suite covers intended-user acceptance after auth redirect, wrong-account/revoked denial, idempotent duplicate acceptance, monitor authorization, and desktop/narrow layouts. Expiry and retryable-transient-failure invariants remain covered by the backend invitation contract tests.

## Compose / release smoke
- `make check-compose`
- `make build`
- `make smoke`

## Nix shells
- `nix develop .#web --command cargo test -p roomies-app --no-default-features --features web`
- `nix develop .#desktop --command cargo check -p roomies-app --no-default-features --features desktop`
- `nix develop .#container --command podman-compose -f docker-compose.yml config`

## Current environment gap
Desktop checks require GTK/WebKit development libraries. Use the `desktop` shell or install those packages locally before running the desktop check.
