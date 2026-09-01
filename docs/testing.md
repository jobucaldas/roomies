# Testing baseline

## Backend
- SQLite: `cd src/backend && DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go test ./... -v`
- PostgreSQL: `cd src/backend && DATABASE_URL=postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable JWT_SECRET=dev-secret go test ./... -v`

## Frontend
- Web: `cargo test -p roomies-app --no-default-features --features web`
- Desktop: `cargo check -p roomies-app --no-default-features --features desktop`
- Android: `cd src/app && dx build --release --android`

## Compose / release smoke
- `make check-compose`
- `make build`
- `make smoke`

## Nix shells
- `nix develop .#web --command cargo test -p roomies-app --no-default-features --features web`
- `nix develop .#desktop --command cargo check -p roomies-app --no-default-features --features desktop`
- `nix develop .#container --command podman-compose -f docker-compose.yml config`

## Current environment gap
Desktop checks require GTK/WebKit development libraries. Use the `desktop` shell or install those packages locally before running the desktop check.
