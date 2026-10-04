# `dev` stage: Go toolchain used by docker-compose.yml (`go run` on the mounted source)
# and `docker compose run --rm test-backend`.
# Default (last) stage: the production image CI publishes.
FROM docker.io/golang:1.26.8-alpine AS dev
RUN apk add --no-cache git ca-certificates gcc musl-dev
ENV GOTOOLCHAIN=local CGO_ENABLED=1
WORKDIR /app

FROM dev AS build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /roomies-backend .

FROM docker.io/alpine:3.20
# Fixed uid: `tls-init` makes backend.key readable by this user only.
RUN addgroup -S -g 10001 roomies && adduser -S -u 10001 -G roomies roomies
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /roomies-backend /roomies-backend
USER roomies
EXPOSE 8080 8081
ENTRYPOINT ["/roomies-backend"]
CMD ["serve"]
