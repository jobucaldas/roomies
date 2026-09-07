module github.com/roomies/backend

go 1.26

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/marknefedov/go-webpush/v2 v2.0.0
	github.com/mattn/go-sqlite3 v1.14.24
	github.com/oklog/ulid/v2 v2.1.1
	github.com/teambition/rrule-go v1.8.2
	golang.org/x/crypto v0.32.0
	golang.org/x/oauth2 v0.27.0
)

require cloud.google.com/go/compute/metadata v0.3.0 // indirect
