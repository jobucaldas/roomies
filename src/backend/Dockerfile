FROM docker.io/golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /roomies-backend ./main.go

FROM docker.io/alpine:3.20

RUN addgroup -S roomies && adduser -S -G roomies roomies
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /roomies-backend /roomies-backend

USER roomies
EXPOSE 8080 8081
ENTRYPOINT ["/roomies-backend"]
