# Flutter client

Production web and Android client for Roomies.

```bash
flutter pub get
flutter analyze
flutter test
flutter build web --release --no-web-resources-cdn --dart-define=ROOMIES_API_URL=/api
flutter build apk --release --dart-define=ROOMIES_API_URL=
```

Web builds bake `ROOMIES_API_URL` at compile time (Compose/K8s use `/api`). Mobile APKs leave it empty so a self-hosted backend URL can be chosen later in-app.
