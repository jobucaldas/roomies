FROM ghcr.io/cirruslabs/flutter:3.27.1 AS builder

WORKDIR /workspace/src/flutter
ARG ROOMIES_API_URL=/api

COPY src/flutter/pubspec.yaml src/flutter/pubspec.lock ./
RUN flutter pub get

COPY src/flutter/ ./
COPY src/app/public/roomies-sw.js ./web/roomies-sw.js

<<<<<<< HEAD
# CanvasKit stays on the app origin so the Caddy CSP can keep script-src and
# connect-src on 'self' (the default build fetches it from www.gstatic.com).
RUN flutter build web --release \
  --no-web-resources-cdn \
=======
# Bundle CanvasKit in the image so CSP can keep script-src/connect-src off www.gstatic.com.
RUN flutter build web --release --no-web-resources-cdn \
>>>>>>> 140b7e0 (fix(web): boot Flutter under CSP with local CanvasKit)
  --dart-define=ROOMIES_API_URL=${ROOMIES_API_URL}

FROM docker.io/alpine:3.20
WORKDIR /app
COPY --from=builder /workspace/src/flutter/build/web/ /app/dist/
CMD ["/bin/true"]
