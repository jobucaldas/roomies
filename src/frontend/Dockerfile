# syntax=docker/dockerfile:1
# Roomies web: the Flutter web bundle served by Caddy, which also proxies
# /api/* to the backend. One image, nothing to mount.
#
# Stages (pick one with --target):
#   local     builds the bundle from this checkout (docker-compose.yml uses it)
#   prebuilt  copies an existing build/web (CI builds it once and publishes this; default)
#
# Runtime env:
#   SITE_ADDRESS      ":8080" (plain HTTP, default) or a domain for automatic HTTPS
#   BACKEND_UPSTREAM  where the API listens (default backend:8080); always reached over TLS
#   BACKEND_CA_FILE   CA that signed the API certificate (default /tls/ca.crt, from `tls-init`)

FROM ghcr.io/cirruslabs/flutter:3.27.1 AS web-build
WORKDIR /app
COPY pubspec.yaml pubspec.lock ./
RUN flutter pub get
COPY . .
ARG BUILD_SHA=unknown
# CanvasKit stays on the app origin so the CSP below can keep script-src and
# connect-src off www.gstatic.com (the default build fetches CanvasKit from a CDN).
RUN flutter build web --release --no-web-resources-cdn --dart-define=ROOMIES_API_URL=/api \
  && printf 'BUILD_SHA=%s\n' "${BUILD_SHA}" > build/web/BUILD_SHA.txt

FROM docker.io/library/caddy:2.8-alpine AS runtime
COPY <<'CADDYFILE' /etc/caddy/Caddyfile
{
	# Accept X-Forwarded-* from a TLS-terminating proxy on a private network.
	servers {
		trusted_proxies static private_ranges
	}
}

{$SITE_ADDRESS::8080} {
	encode gzip

	header {
		# Ignored by browsers over plain HTTP, so it is safe for localhost trials.
		Strict-Transport-Security "max-age=31536000"
		Permissions-Policy "camera=(), microphone=(), geolocation=(), payment=(), usb=()"
		Referrer-Policy "no-referrer"
		X-Content-Type-Options "nosniff"
		X-Frame-Options "DENY"
		# Flutter ships CanvasKit and the brand fonts locally. The engine still
		# fetches Noto fallback glyphs (emoji, other scripts) from fonts.gstatic.com.
		Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self' https://fonts.gstatic.com; worker-src 'self' blob:; manifest-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'"
	}

	handle /api/* {
		reverse_proxy {$BACKEND_UPSTREAM:backend:8080} {
			# The API only speaks TLS; trust the private CA from `tls-init`.
			transport http {
				tls
				tls_trusted_ca_certs {$BACKEND_CA_FILE:/tls/ca.crt}
			}
			header_up X-Forwarded-For {remote_host}
			header_up X-Forwarded-Host {host}
		}
	}

	# SPA fallback so deep links and refreshes serve the Flutter shell.
	handle {
		root * /app/dist
		header Cache-Control "no-cache"
		try_files {path} /index.html
		file_server
	}
}
CADDYFILE
RUN adduser -D -H -u 65532 -g nonroot nonroot \
  && mkdir -p /app/dist /data /config \
  && chown -R 65532:65532 /data /config
USER 65532:65532
EXPOSE 8080

FROM runtime AS local
COPY --from=web-build /app/build/web/ /app/dist/

FROM runtime AS prebuilt
COPY build/web/ /app/dist/
