# Testing

CI runs these checks on Depot CI; see [ci.md](ci.md).

## Backend
```bash
cd src/backend
DATABASE_URL=sqlite://roomies.db JWT_SECRET=dev-secret go test ./...
DATABASE_URL=postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable \
  JWT_SECRET=dev-secret go test ./...
```

## Flutter
```bash
cd src/flutter
flutter pub get
flutter analyze
flutter test
flutter build web --release --no-web-resources-cdn --dart-define=ROOMIES_API_URL=/api
flutter build apk --release --dart-define=ROOMIES_API_URL=
```

## Compose smoke
```bash
make check-compose
make build
# with stack up:
make smoke
```

## Optional browser flows
`e2e/` holds Playwright specs for local debugging. They are not part of the required CI gate; product checks are usually done by agents against a running stack.
