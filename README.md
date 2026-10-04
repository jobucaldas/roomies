# Roomies

Roomies helps roommates manage shared expenses, notes, balances, and house-scoped household workflows.

- **Apps:** web, Android, Windows, Linux (Flutter)
- **Server:** Go API + worker, PostgreSQL, behind Caddy
- **Sign-in:** local email/password, or WorkOS AuthKit

You only need Docker (Podman works too); no clone of this repo.

## Self-hosting

**1. Save this as `compose.yaml`:**

```yaml
name: roomies

x-backend: &backend
  image: ghcr.io/jobucaldas/roomies-backend:nightly
  environment:
    DATABASE_URL: postgres://roomies:${POSTGRES_PASSWORD}@db:5432/roomies?sslmode=verify-full&sslrootcert=/tls/ca.crt
    APP_ENV: production
    JWT_SECRET: ${JWT_SECRET:?set JWT_SECRET in .env}
    ROOMIES_PUBLIC_BASE_URL: ${ROOMIES_PUBLIC_BASE_URL:?set ROOMIES_PUBLIC_BASE_URL in .env}
    CORS_ALLOWED_ORIGINS: ${ROOMIES_PUBLIC_BASE_URL}
    TLS_CERT_FILE: /tls/backend.crt
    TLS_KEY_FILE: /tls/backend.key
    WORKOS_CLIENT_ID: ${WORKOS_CLIENT_ID:-}
    WORKOS_API_KEY: ${WORKOS_API_KEY:-}
  volumes:
    - tls:/tls:ro
  depends_on:
    db:
      condition: service_healthy
  restart: unless-stopped

services:
  # One-shot: issues the private CA and certificates that encrypt traffic
  # between the containers below. Runs as root only to set key ownership.
  tls:
    image: ghcr.io/jobucaldas/roomies-backend:nightly
    command: ["tls-init"]
    user: "0"
    volumes:
      - tls:/tls

  db:
    image: postgres:16-alpine
    command: ["postgres", "-c", "ssl=on", "-c", "ssl_cert_file=/tls/db.crt", "-c", "ssl_key_file=/tls/db.key",
      "-c", "ssl_min_protocol_version=TLSv1.2", "-c", "hba_file=/tls/pg_hba.conf"]
    environment:
      POSTGRES_USER: roomies
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD in .env}
      POSTGRES_DB: roomies
    volumes:
      - pgdata:/var/lib/postgresql/data
      - tls:/tls:ro
    depends_on:
      tls:
        condition: service_completed_successfully
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U roomies -d roomies"]
      interval: 5s
      timeout: 5s
      retries: 10
    restart: unless-stopped

  backend:
    <<: *backend
    command: ["serve"]

  worker:
    <<: *backend
    command: ["worker"]

  web:
    image: ghcr.io/jobucaldas/roomies-frontend:nightly
    environment:
      SITE_ADDRESS: ${SITE_ADDRESS:-:8080}
    ports:
      - "${HTTP_PORT:-8080}:8080"
      # Automatic HTTPS on your own domain: set SITE_ADDRESS=roomies.example.com
      # in .env and publish these two instead of the line above.
      # - "80:80"
      # - "443:443"
    volumes:
      - caddy:/data
      - tls:/tls:ro
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  pgdata:
  caddy:
  tls:
```

**2. Save this as `.env` next to it** (keep it private):

```sh
POSTGRES_PASSWORD=          # openssl rand -hex 24
JWT_SECRET=                 # openssl rand -hex 48
ROOMIES_PUBLIC_BASE_URL=https://roomies.example.com   # or http://localhost:8080 to try it
# SITE_ADDRESS=roomies.example.com                    # see the comment in compose.yaml
# WORKOS_CLIENT_ID=client_...                         # optional, see below
# WORKOS_API_KEY=sk_...
```

**3. Start it:**

```sh
docker compose up -d
curl -i http://localhost:8080/api/auth/me   # 401 means it is up
```

Open `ROOMIES_PUBLIC_BASE_URL`. With `APP_ENV: production` the server refuses to start unless that URL is `https://` (or `localhost`), so put it behind `SITE_ADDRESS` or your own TLS proxy. Update with `docker compose pull && docker compose up -d`; pin `:nightly` to a `YYYYMMDDHHMMSS_<shortsha>` tag to stay on a fixed version. Data lives in the `pgdata` volume.

Optional backend variables (email invitations, Web Push, FCM) are listed with comments in [`.env.example`](.env.example); add the ones you set to the `x-backend` environment.

**Encryption in transit.** Browsers and apps reach Caddy over HTTPS when `SITE_ADDRESS` is a domain (HSTS is sent). Between containers, `tls-init` issues a private CA: Caddy reaches the API, and the API and worker reach PostgreSQL, over TLS with certificate checks; PostgreSQL rejects plaintext TCP connections. Invitation email requires STARTTLS or TLS (`SMTP_TLS`). Using an external PostgreSQL? Point `DATABASE_URL` at it with `sslmode=verify-full`; production refuses `disable`, `allow`, and `prefer`.

**WorkOS AuthKit (optional).** Without it, people sign up with email and password. To use hosted sign-in, create a [WorkOS](https://workos.com/docs/authkit) project, set `WORKOS_CLIENT_ID` and `WORKOS_API_KEY`, and register `{ROOMIES_PUBLIC_BASE_URL}/callback` as a redirect URI.

## Apps

CI builds the Android APK, Windows bundle and Linux AppImage on every push as workflow artifacts. The native apps ask for your server's address on first launch. The web app is served by your server.

Android release signing on `main` uses these repository secrets when present: `ANDROID_KEYSTORE_BASE64` (`base64 -w0 upload-keystore.jks`), `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, `ANDROID_KEY_PASSWORD`. Otherwise APKs use debug keys. Locally, put the same values in `src/frontend/android/key.properties` (gitignored).

## Development

Everything runs through Docker Compose; no Go or Flutter install needed. The root `docker-compose.yml` builds **your checkout**, unlike the compose file above, which runs the published images.

```sh
docker compose up --build    # optional: cp .env.example .env first
```

Open <http://localhost:8080/> and sign up with any email and password. This starts Postgres, the Go API and worker (`go run` on the mounted `src/backend/`), and `web`, which builds the Flutter web app from `src/frontend/` and serves it with Caddy next to the API. Change the port with `HTTP_PORT` and a matching `ROOMIES_PUBLIC_BASE_URL`. Stop with `docker compose down` (`-v` also wipes the database).

| Changed | Do |
|---|---|
| `src/backend/` | `docker compose restart backend worker` |
| `src/frontend/` | `docker compose up --build -d web` |

Checks and tools:

```sh
docker compose run --rm test-backend    # gofmt, vet, tests (SQLite + Postgres)
docker compose run --rm test-frontend   # flutter analyze + test
docker compose run --rm apk             # debug APK -> src/frontend/build/app/outputs/flutter-apk/
```

Windows and Linux builds need their own OS: `flutter build windows|linux --release` inside `src/frontend/`.

| Path | Role |
|---|---|
| `src/backend/` | Go API and worker. `Dockerfile`: `dev` stage (toolchain used by compose) and the production image |
| `src/frontend/` | Flutter app. `Dockerfile`: `local` builds the web bundle from source, `prebuilt` (default, what CI publishes) copies `build/web`; both serve it with Caddy and proxy `/api/*` (`SITE_ADDRESS`, `BACKEND_UPSTREAM`) |
| `docker-compose.yml` | Development stack and tooling |
| `.env.example` | Every config variable, with comments |
| `.github/workflows/ci.yml` | Checks on every push; on `main`, publishes `ghcr.io/jobucaldas/roomies-backend` and `roomies-frontend` as `:nightly` and `YYYYMMDDHHMMSS_<shortsha>` |

## Security
Report vulnerabilities privately; see [SECURITY.md](SECURITY.md).

## License
[GPL-3.0](LICENSE).
