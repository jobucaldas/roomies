DOCKER := docker
COMPOSE := docker compose
DB_URL ?= postgres://roomies:roomies@localhost:5432/roomies?sslmode=disable
KUSTOMIZE ?= kustomize

.PHONY: dev dev-backend dev-frontend build build-backend build-frontend test test-backend test-frontend test-e2e-household check-compose smoke android shell-db lint-backend lint-frontend clean clean-generated clean-containers render-manifests

dev:
	$(COMPOSE) -f .devcontainer/docker-compose.yml up --build

dev-backend:
	cd src/backend && DATABASE_URL="$(DB_URL)" go run ./main.go

dev-frontend:
	cd src/app && dx serve

build:
	$(COMPOSE) build

build-frontend:
	cd src/app && dx build --release

build-backend:
	cd src/backend && go build -o bin/roomies-backend ./main.go

test:
	$(MAKE) test-backend
	$(MAKE) test-frontend

test-backend:
	cd src/backend && go test ./... -v

test-frontend:
	cargo test -p roomies-app --no-default-features --features web

test-e2e-household:
	@set -eu; \
	project="roomies-household-$$(id -u)"; \
	password="$$(openssl rand -hex 32)"; \
	export POSTGRES_PASSWORD="$$password" DATABASE_URL="postgres://roomies:$$password@db:5432/roomies?sslmode=disable" JWT_SECRET="$$(openssl rand -hex 48)" CORS_ALLOWED_ORIGINS=http://localhost:58000 ROOMIES_PUBLIC_BASE_URL=http://localhost:58000 ROOMIES_BACKEND_PORT=58080 ROOMIES_HTTP_PORT=58000 SMTP_HOST=mailpit SMTP_PORT=1025; \
	trap '$(COMPOSE) -p "$$project" -f docker-compose.yml -f e2e/docker-compose.mailpit.yml down --volumes --remove-orphans' EXIT; \
	$(COMPOSE) -p "$$project" -f docker-compose.yml -f e2e/docker-compose.mailpit.yml up -d --build; \
	timeout 180 sh -c 'until curl -fsS http://localhost:58080/healthz; do sleep 2; done'; \
	timeout 60 sh -c 'until curl -fsS http://localhost:58000/ >/dev/null; do sleep 2; done'; \
	cd e2e && npm ci && npx playwright install --with-deps chromium && GIT_COMMIT="$$(git rev-parse HEAD)" ROOMIES_E2E_COMMAND='npx playwright test tests/household.spec.ts' ROOMIES_WEB_URL=http://localhost:58000 ROOMIES_API_URL=http://localhost:58000/api npx playwright test tests/household.spec.ts

check-compose:
	@env \
		POSTGRES_PASSWORD="$${POSTGRES_PASSWORD:-roomies}" \
		DATABASE_URL="$${DATABASE_URL:-postgres://roomies:roomies@db:5432/roomies?sslmode=disable}" \
		JWT_SECRET="$${JWT_SECRET:-roomies-check-secret}" \
		$(COMPOSE) -f docker-compose.yml config >/dev/null

smoke:
	curl --fail --silent --show-error http://localhost:8080/healthz

android:
	cd src/app && dx build --release --platform android --target aarch64-linux-android

shell-db:
	$(COMPOSE) -f .devcontainer/docker-compose.yml exec db psql -U roomies roomies

lint-backend:
	cd src/backend && go vet ./...

lint-frontend:
	cargo clippy -p roomies-app --no-default-features --features web

render-manifests:
	$(KUSTOMIZE) build deploy/kustomize/overlays/production

clean-generated:
	rm -rf target src/backend/bin src/app/dist src/app/target

clean-containers:
	-$(COMPOSE) -f docker-compose.yml down -v
	-$(COMPOSE) -f .devcontainer/docker-compose.yml down -v
	-$(DOCKER) ps -aq --filter name=roomies | xargs -r $(DOCKER) rm -f

clean: clean-containers clean-generated
