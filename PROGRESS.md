# Progress

## Baseline
- Preserved the Roomies MVP code and the intended Flutter scaffold deletions.
- Unified Rust lockfile ownership at the repository root.
- Added baseline docs for release, testing, cleanup, and credentials.
- Added portable Kubernetes/Kustomize release scaffolding.
- Added Nix flake shells for the supported development modes.

## Validation
- Backend tests pass on SQLite and PostgreSQL.
- Frontend web tests pass.
- Desktop checks need GTK/WebKit development libraries in the environment.

## Open follow-ups
- Verify the Nix desktop shell on a clean machine.
- Wire real production secret delivery into the user's Kubernetes workflow.
- Expand app-specific release automation once the baseline is stable.

## Invitation onboarding UI
- Added Dioxus invitation create/list/revoke controls for house admins, with explicit loading, expiry, error, and success states while retaining raw user-ID membership adding.
- Added an `/accept-invitation` route that captures the one-time URL into browser session storage, scrubs the address bar, survives login/register, clears it after acceptance/rejection, and refreshes by navigating to the returned house.
- Added API/model contract tests and a synthetic Playwright/Mailpit harness under `e2e/` for desktop and narrow layouts.
- Frontend web tests pass in the Nix web shell (19 tests), and the release WASM assets build with Dioxus 0.7.10.
- Exact-head invitation validation passed all six desktop/narrow Playwright scenarios against disposable PostgreSQL and Mailpit services. Served HTML/JavaScript/WASM hashes matched the current build, browser console/page/network error counts were zero, and the narrow layout had no horizontal overflow. Sanitized screenshots and JSON evidence are retained under ignored `e2e/artifacts/evidence/`.

## Notification web UI
- Added a house Notifications / Schedule tab with self preference controls, subscription metadata, browser-push controls, and role-aware scheduled-event creation, editing, deletion, and viewing. Creator/admin edit forms prefill recurrence data and validate before the server-authoritative request.
- Browser push uses a root-scoped service worker and runtime public VAPID-key discovery; native desktop intentionally reports it unsupported. The production Dioxus 0.7.10 builder image built the web artifacts with the service worker. Real browser provider delivery and Android FCM remain unexercised.
