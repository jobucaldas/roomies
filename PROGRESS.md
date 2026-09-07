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
- Direct web test execution is currently blocked by the pinned Nix Rust 1.86 toolchain versus cached ICU packages requiring Rust 1.88; browser flow was not substituted with a fake pass.
- Verified the invitation route against a cached Dioxus 0.7.10 builder: current dirty route/form source hashes matched the builder inputs, and served index/JavaScript hashes matched the exported assets. Playwright passed all invitation scenarios on desktop and narrow Chromium (intended auth redirect, wrong/revoked denial, idempotent duplicate acceptance, and monitor authorization).
