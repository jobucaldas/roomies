# Testing baseline

## Backend
- SQLite: `cd src/backend && DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go test ./... -v`
- PostgreSQL: `cd src/backend && DATABASE_URL=postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable JWT_SECRET=dev-secret go test ./... -v`

## Frontend
- Web: `cargo test -p roomies-app --no-default-features --features web`
- Desktop: `cargo check -p roomies-app --no-default-features --features desktop`
- Android ARM64 check: `nix develop .#android --command cargo check -p roomies-app --no-default-features --features mobile --target aarch64-linux-android`
- Android APK (optimized Rust payload, development signing): `nix develop .#android --command make android`

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

## CI-gated household browser regression
The `household-browser` GitHub Actions job runs the desktop and narrow projects for `e2e/tests/household.spec.ts` against a disposable Compose PostgreSQL/backend/worker/frontend/Caddy/Mailpit stack. It builds the frontend through `src/app/Dockerfile`'s pinned Dioxus production builder and uses the checked-out SHA for `GIT_COMMIT`. Reproduce it on Linux from the repository root:

```sh
make test-e2e-household
```

The target generates synthetic secrets, installs Chromium with Playwright's supported dependency path, and always removes its Compose project and volumes. On CI failure, only the sanitized evidence JSON is retained for seven days; reports, traces, screenshots, and test-result directories are deliberately excluded because they can contain private browser data. This is a household web-flow gate only; it is not Linux desktop or Android runtime acceptance, real-provider delivery validation, or full-browser-suite acceptance.

## Household domains UI
The house Groceries, Chores, Calendar, and Chat tabs use the normal online API only. Run the web unit and lint checks in an isolated Nix target directory to avoid reusing host Rust artifacts:

```sh
CARGO_TARGET_DIR=/tmp/roomies-household-ui-target-$$ \
  nix develop .#web --command cargo test --manifest-path src/app/Cargo.toml --no-default-features --features web
CARGO_TARGET_DIR=/tmp/roomies-household-ui-target-$$ \
  nix develop .#web --command cargo clippy --manifest-path src/app/Cargo.toml --no-default-features --features web -- -D warnings
```

Chore completion deliberately requires an exact UTC RFC3339 occurrence, while creation and edits retain the server-validated local time, IANA timezone, RRULE, and EXDATE fields. Chat refresh and older-page loading are explicit; chat bodies are not cached or queued by the client.

## Compose / release smoke
- `make check-compose`
- `make build`
- `make smoke`

## Nix shells
- `nix develop .#web --command cargo test -p roomies-app --no-default-features --features web`
- `nix develop .#desktop --command cargo check -p roomies-app --no-default-features --features desktop`
- `nix develop .#android --command make android`
- `nix develop .#container --command podman-compose -f docker-compose.yml config`

## Current environment gap
Desktop checks require GTK/WebKit development libraries. Use the `desktop` shell or install those packages locally before running the desktop check.

## Notification preferences and schedules
The web notification panel exercises self-owned preferences and house-scoped scheduled events through the normal online API. Creators and admins can prefill, edit, cancel, and save event recurrence forms; monitors remain view-only. Browser push is requested only after the **Enable browser push** button is clicked; a supported browser needs a server configured with `WEB_PUSH_PUBLIC_KEY`. The root-scoped `roomies-sw.js` displays generic notification text and does not persist subscription capabilities outside the browser PushSubscription. To reproduce a release build where host tooling lacks the matching wasm-bindgen CLI, use the cached Dioxus 0.7.10 builder image specified in the notification validation record.
