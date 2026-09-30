# Roomies

Roomies helps roommates manage shared expenses, notes, balances, and house-scoped household workflows.

## Supported platforms
- Web
- Linux desktop
- Android

## Stack overview
- Backend API and worker under `src/backend/`
- Production web client under `src/flutter/`
- Rust reference client under `src/app/`
- OCI images for backend and frontend; Kustomize manifests in `deploy/kustomize/`
- Blob storage via filesystem locally, or S3-compatible (for example R2) in production
- SMTP for mail (Mailpit in dev/test); Web Push and Android FCM for notifications (fakes when credentials are absent)

## Quick start
1. Copy `.env.example` to `.env` and replace every placeholder.
2. Run the checks that match your platform:

```bash
make check-compose
make test-backend
make test-frontend
make build
make smoke
```

## Development
- Backend: `cd src/backend && DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go run ./main.go`
- Web (Flutter): see `docs/testing.md` and `src/flutter/`
- Rust web: `cd src/app && dx serve`
- Linux desktop: `cargo check -p roomies-app --no-default-features --features desktop`
- Android APK (development signing): `nix develop .#android --command make android`

## Nix shells
Available shells: `default`, `web`, `desktop`, `android`, `e2e`, `ocr`, `export`, and `container`.

Example:
```bash
nix develop .#web
```

## Deploy and ops docs
- `deploy/kustomize/overlays/production`
- `docs/release.md`
- `docs/testing.md`
- `docs/cleanup.md`
- `docs/authkit.md`
- `docs/credentials.example.env`

## Cleanup
```bash
make clean
```
