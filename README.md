# Roomies

Roomies helps roommates manage shared expenses, notes, balances, and house-scoped household workflows.

## Supported platforms
- Web
- Linux desktop
- Android

No Apple targets are planned for this repo baseline.

## Production baseline
- Production runs in the user’s Kubernetes cluster.
- OCI images are published for backend and frontend.
- Portable Kustomize manifests live in `deploy/kustomize/`.
- Cloudflare R2 is the production blob store.
- SMTP is the mail adapter; Mailpit is used in dev/test.
- Web Push and Android FCM are notification adapters; fakes are used when credentials are absent.

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
- Web: `cd src/app && dx serve`
- Linux desktop: `cargo check -p roomies-app --no-default-features --features desktop`
- Android APK (optimized Rust payload, development signing): `nix develop .#android --command make android`

## Nix shells
Available shells: `default`, `web`, `desktop`, `android`, `e2e`, `ocr`, `export`, and `container`.

Example:
```bash
nix develop .#web
```

## Release assets
- `deploy/kustomize/overlays/production`
- `docs/release.md`
- `docs/testing.md`
- `docs/cleanup.md`
- `docs/credentials.example.env`

## Cleanup
```bash
make clean
```
