DOCKER := docker
COMPOSE := docker compose
DB_URL ?= postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable

.PHONY: dev-backend build build-backend build-frontend test test-backend test-flutter check-compose smoke android windows linux lint-backend lint-flutter clean clean-generated clean-containers

dev-backend:
	cd src/backend && DATABASE_URL="$(DB_URL)" go run ./main.go

build:
	$(COMPOSE) build

build-frontend:
	cd src/flutter && flutter build web --release --no-web-resources-cdn --dart-define=ROOMIES_API_URL=/api

build-backend:
	cd src/backend && go build -o bin/roomies-backend ./main.go

test:
	$(MAKE) test-backend
	$(MAKE) test-flutter

test-backend:
	cd src/backend && go test ./... -v

test-flutter:
	cd src/flutter && flutter test

lint-flutter:
	cd src/flutter && flutter analyze

lint-backend:
	cd src/backend && go vet ./...

check-compose:
	@env \
		POSTGRES_PASSWORD="$${POSTGRES_PASSWORD:-roomies}" \
		DATABASE_URL="$${DATABASE_URL:-postgres://roomies:roomies@db:5432/roomies?sslmode=disable}" \
		JWT_SECRET="$${JWT_SECRET:-roomies-check-secret}" \
		$(COMPOSE) -f docker-compose.yml config >/dev/null

smoke:
	curl --fail --silent --show-error http://localhost:8080/healthz

android:
	cd src/flutter && flutter build apk --release --dart-define=ROOMIES_API_URL=

windows:
	cd src/flutter && flutter build windows --release --dart-define=ROOMIES_API_URL=

linux:
	cd src/flutter && flutter build linux --release --dart-define=ROOMIES_API_URL=

clean-generated:
	rm -rf src/backend/bin src/flutter/build

clean-containers:
	-$(COMPOSE) -f docker-compose.yml down -v
	-$(DOCKER) ps -aq --filter name=roomies | xargs -r $(DOCKER) rm -f

clean: clean-containers clean-generated
