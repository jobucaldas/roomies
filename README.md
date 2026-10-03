# Roomies

Roomies helps roommates manage shared expenses, notes, balances, and house-scoped household workflows.

## Layout
| Path | Role |
|------|------|
| `src/flutter/` | App client (web + Android) |
| `src/backend/` | Go API and worker |
| `.github/workflows/ci.yml` | CI and GHCR image publishing |
| `docker-compose.yml`, `Caddyfile` | Self-hosted stack |
| `.env.example` | Every config variable, with comments |

## Quick start
1. Copy `.env.example` to `.env` and fill in secrets.
2. Run what you need:

```bash
make check-compose
make test-backend
make test-flutter
make build
```

Local backend without Compose:

```bash
cd src/backend && DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go run ./main.go
```

Flutter web:

```bash
cd src/flutter && flutter pub get && flutter run -d chrome \
  --dart-define=ROOMIES_API_URL=http://localhost:8080/api
```

Android release APK (debug-signed unless signing is configured):

```bash
make android
```

## CI and deploys
CI runs on every push and pull request: pre-commit, `go vet`, backend tests (SQLite and Postgres), Flutter analyze/test/web build, Android APK, and a Compose health smoke.

On pushes to `main`, CI publishes `ghcr.io/jobucaldas/roomies-backend` and `roomies-frontend` as `:nightly`, `:dev`, and an immutable `YYYYMMDDHHMMSS_<shortsha>` tag. The cluster pulls `:nightly` on its own; CI holds no cluster credentials.

### Android signing
On pushes to `main`, CI signs the APK when these repository secrets exist: `ANDROID_KEYSTORE_BASE64` (`base64 -w0 upload-keystore.jks`), `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, `ANDROID_KEY_PASSWORD`. Everything else builds with debug keys. Locally, put the same values in `src/flutter/android/key.properties` (gitignored).

## WorkOS AuthKit
Set `WORKOS_API_KEY` and `WORKOS_CLIENT_ID` to sign in through hosted AuthKit; leave them empty to keep local email/password (used by CI). The redirect URI is `WORKOS_REDIRECT_URI` or `{ROOMIES_PUBLIC_BASE_URL}/callback`. In the WorkOS Dashboard, register that `/callback` URL for every origin you sign in from, and set the login and logout-return URLs to the app's public origin.

## Cleanup
```bash
make clean
```

## Security
Report vulnerabilities privately; see [SECURITY.md](SECURITY.md).

## License
[GPL-3.0](LICENSE).
