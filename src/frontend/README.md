# Roomies app

Flutter client for web, Android, Windows and Linux. See the root `README.md` for the Docker Compose workflow (`docker compose run --rm test-frontend`, `apk`).

Web builds bake `ROOMIES_API_URL=/api` (same origin, served by Caddy). Native builds leave it empty so the app asks for a server on first launch.
