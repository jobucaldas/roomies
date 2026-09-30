# Testing baseline

## Backend
- SQLite: `cd src/backend && DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go test ./... -v`
- PostgreSQL: `cd src/backend && DATABASE_URL=postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable JWT_SECRET=dev-secret go test ./... -v`

## Frontend
- Production web client (Flutter): `cd src/flutter && flutter analyze && flutter test && flutter build web --release --no-web-resources-cdn --dart-define=ROOMIES_API_URL=/api`
- Rust reference client: `cargo test -p roomies-app --no-default-features --features web`
- Rust web release bundle: `nix develop .#web --command sh -c 'cd src/app && dx build --release --debug-symbols=false'`
- Desktop: `cargo check -p roomies-app --no-default-features --features desktop`
- Android ARM64 check: `nix develop .#android --command cargo check -p roomies-app --no-default-features --features mobile --target aarch64-linux-android`
- Android APK (development signing): `nix develop .#android --command make android`

Generated Rust web output is ignored under `target/`.

## Invitation onboarding browser flow
The Playwright harness uses synthetic accounts and Mailpit; it does not print or render bearer tokens. Start the stack with SMTP pointed at the disposable sink, then run:

```sh
podman compose -f e2e/docker-compose.mailpit.yml up -d
SMTP_HOST=localhost SMTP_PORT=1025 ROOMIES_PUBLIC_BASE_URL=http://localhost \
  podman compose -f docker-compose.yml up --build -d
cd e2e && npm ci && npx playwright install --with-deps chromium && npm test
podman compose -f docker-compose.yml down -v
podman compose -f e2e/docker-compose.mailpit.yml down -v
```

Screenshots and Playwright reports are written under `e2e/artifacts/` (gitignored). The suite covers acceptance after auth redirect, wrong-account/revoked denial, idempotent duplicate acceptance, monitor authorization, and desktop/narrow layouts. Expiry and retryable-transient-failure cases stay covered by backend invitation contract tests.

## CI-gated core and household browser regression
The `household-browser` GitHub Actions job runs the desktop and narrow projects for `e2e/tests/core.spec.ts` and `e2e/tests/household.spec.ts` against one disposable Compose PostgreSQL/backend/worker/frontend/Caddy/Mailpit stack. It builds the production web client through `src/flutter/Dockerfile`. Reproduce locally:

```sh
make test-e2e-household
```

The target generates synthetic secrets, installs Chromium via Playwright, runs both specs, and removes its Compose project and volumes. Core tests disable screenshots and traces; on CI failure, only sanitized evidence JSON is retained. Reports, traces, screenshots, and test-result directories are not uploaded.

## Core household UI
Browser coverage includes expense creation with shared/private visibility, non-recipient denial, payer-or-admin edit/delete, notes create/edit/delete, balance settlement suggestions, and role restrictions. Core evidence records counts and served-asset hashes only.

## Household domains UI
Groceries, Chores, Calendar, and Chat use the online API. Run web unit and lint checks in an isolated Nix target directory:

```sh
CARGO_TARGET_DIR=/tmp/roomies-household-ui-target-$$ \
  nix develop .#web --command cargo test --manifest-path src/app/Cargo.toml --no-default-features --features web
CARGO_TARGET_DIR=/tmp/roomies-household-ui-target-$$ \
  nix develop .#web --command cargo clippy --manifest-path src/app/Cargo.toml --no-default-features --features web -- -D warnings
```

Chore completion requires an exact UTC RFC3339 occurrence. Chat refresh and older-page loading are explicit; chat bodies are not cached or queued by the client.

## Compose / release smoke
- `make check-compose`
- `make build`
- `make smoke`

## Nix shells
- `nix develop .#web --command cargo test -p roomies-app --no-default-features --features web`
- `nix develop .#desktop --command cargo check -p roomies-app --no-default-features --features desktop`
- `nix develop .#android --command make android`
- `nix develop .#container --command podman-compose -f docker-compose.yml config`

## Environment notes
Desktop checks require GTK/WebKit development libraries. Use the `desktop` shell or install those packages locally.

## Notification preferences and schedules
The web notification panel covers self-owned preferences and house-scoped scheduled events. Creators and admins can prefill, edit, cancel, and save event recurrence forms; monitors are view-only. Browser push is requested only after **Enable browser push**; a supported browser needs `WEB_PUSH_PUBLIC_KEY`. The root-scoped `roomies-sw.js` shows generic notification text.
