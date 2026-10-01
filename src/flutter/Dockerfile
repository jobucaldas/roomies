FROM ghcr.io/cirruslabs/flutter:3.27.1 AS builder

WORKDIR /workspace/src/flutter
ARG ROOMIES_API_URL=/api

COPY src/flutter/pubspec.yaml src/flutter/pubspec.lock ./
RUN flutter pub get

COPY src/flutter/ ./

# CanvasKit stays on the app origin so the Caddy CSP can keep script-src and
# connect-src off www.gstatic.com (the default build fetches CanvasKit from CDN).
RUN flutter build web --release \
  --no-web-resources-cdn \
  --dart-define=ROOMIES_API_URL=${ROOMIES_API_URL}

FROM docker.io/alpine:3.20
WORKDIR /app
COPY --from=builder /workspace/src/flutter/build/web/ /app/dist/
CMD ["/bin/true"]
