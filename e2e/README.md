# Browser evidence

The CI-gated core and household regression builds the pinned production frontend, starts one disposable PostgreSQL/backend/worker/frontend/Caddy/Mailpit Compose project, and runs `tests/core.spec.ts` plus `tests/household.spec.ts` for the desktop and narrow projects. On a Linux host, run the identical local gate from the repository root:

```sh
make test-e2e-household
```

It generates throwaway database and JWT secrets, uses `GIT_COMMIT="$(git rev-parse HEAD)"`, waits for the stack, and tears down the owned project and volumes even when the test fails. Playwright's supported Chromium dependency installation may require sudo. Core tests disable screenshots and traces. CI uploads only sanitized evidence JSON on failure (for seven days); it does not upload traces, reports, screenshots, cookies, request headers, query-bearing URLs, response bodies, or chat bodies.

The harness installs console-error, page-error, and failed-request listeners before every navigation. Any such event fails the test; there is no request allowlist until an intentionally aborted request is documented in code. Existing household tests write a project-qualified screenshot and a sanitized JSON evidence record under `artifacts/evidence/` (ignored by git); core tests write only the sanitized JSON record to avoid retaining private expense or note content. Records contain only the commit, command, test/project, counts, and SHA-256 hashes of served HTML/JavaScript/WASM assets; they do not contain credentials, tokens, URLs with query values, headers, cookies, traces, or response bodies.
