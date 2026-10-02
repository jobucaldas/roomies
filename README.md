# Roomies

Roomies helps roommates manage shared expenses, notes, balances, and house-scoped household workflows.

## Platforms
- Web (Flutter)
- Android (Flutter APK)
- Self-hosted stack via Docker Compose / Kubernetes

## Layout
| Path | Role |
|------|------|
| `src/flutter/` | App client (web + mobile) |
| `src/backend/` | Go API and worker |
| `deploy/kustomize/` | Kubernetes manifests |
| `docs/` | Auth, credentials, testing, release notes |
| `e2e/` | Optional Playwright flows (not required CI) |

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

Android debug/release APK:

```bash
make android
```

## Ops docs
- `docs/authkit.md` — WorkOS AuthKit
- `docs/credentials.example.env` — secret template
- `docs/testing.md` — unit/lint/build checks
- `docs/ci.md` — GitHub Actions jobs and Android release signing
- `docs/release.md` — images and manifests
- `deploy/kustomize/overlays/dev` and `…/production`

## Cleanup
```bash
make clean
```

## Security
Report vulnerabilities privately; see [SECURITY.md](SECURITY.md).

## License
[GPL-3.0](LICENSE).
