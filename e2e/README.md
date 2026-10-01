# Optional browser flows

Playwright specs under `tests/` for local debugging against a running Compose stack. Not part of required CI.

```sh
# start stack (example ports match docker-compose defaults / CI mapping)
docker compose -f docker-compose.yml -f e2e/docker-compose.mailpit.yml up -d --build
cd e2e && npm ci && npx playwright install chromium
ROOMIES_WEB_URL=http://localhost:58000 ROOMIES_API_URL=http://localhost:58000/api \
  npx playwright test
```

Artifacts land under `e2e/artifacts/` (gitignored). Prefer Flutter unit tests and backend `go test` for day-to-day checks.
