# Browser evidence

Run the complete desktop/narrow matrix from this directory with:

```sh
GIT_COMMIT="$(git rev-parse HEAD)" ROOMIES_E2E_COMMAND='npx playwright test' npx playwright test
```

The harness installs console-error, page-error, and failed-request listeners before every navigation. Any such event fails the test; there is no request allowlist until an intentionally aborted request is documented in code. Each test writes a project-qualified screenshot and a sanitized JSON evidence record under `artifacts/evidence/` (ignored by git). Records contain only the commit, command, test/project, counts, and SHA-256 hashes of served HTML/JavaScript/WASM assets; they do not contain credentials, tokens, URLs with query values, headers, cookies, traces, or response bodies.
