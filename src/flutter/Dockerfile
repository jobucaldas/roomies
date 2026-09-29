FROM ghcr.io/cirruslabs/flutter:3.27.1 AS builder

WORKDIR /workspace/src/flutter
ARG ROOMIES_API_URL=/api

COPY src/flutter/pubspec.yaml src/flutter/pubspec.lock ./
RUN flutter pub get

COPY src/flutter/ ./
COPY src/app/public/roomies-sw.js ./web/roomies-sw.js

RUN flutter build web --release \
  --dart-define=ROOMIES_API_URL=${ROOMIES_API_URL}

FROM docker.io/alpine:3.20
WORKDIR /app
COPY --from=builder /workspace/src/flutter/build/web/ /app/dist/
CMD ["/bin/true"]
