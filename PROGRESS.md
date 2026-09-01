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
