.PHONY: run build test test-unit test-integration lint vet migrate rollback swagger swagger-check up down logs tidy fmt

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./... -cover

test-unit:
	go test ./internal/pkg/... -cover

test-integration:
	go test ./tests/... -tags=integration -cover -count=1

lint:
	golangci-lint run

vet:
	go vet ./...

migrate:
	go run cmd/server/main.go -migrate-only

# Rollback to a specific version: make rollback VERSION=20260811000000
rollback:
	go run cmd/server/main.go -rollback-to $(VERSION)

swagger:
	swag init -g cmd/server/main.go -o docs/swagger

# Fail the build when docs/swagger is stale vs. the current route annotations.
# swag derives the generated package name from the -o directory, so regen into
# a dir named "swagger" to keep docs.go byte-identical.
swagger-check:
	@tmp=$$(mktemp -d); \
	swag init -g cmd/server/main.go -o $$tmp/swagger >/dev/null 2>&1; \
	if diff -rq $$tmp/swagger docs/swagger >/dev/null 2>&1; then \
		echo "Swagger documentation is up to date"; \
	else \
		echo "Swagger documentation is STALE — run 'make swagger' and commit the diff:"; \
		diff -rq $$tmp/swagger docs/swagger | head -30; \
		rm -rf $$tmp; \
		exit 1; \
	fi; \
	rm -rf $$tmp

up:
	docker compose up -d postgres redis nats

# Full stack incl. the TLS reverse proxy (caddy) and metrics (prometheus).
up-all:
	docker compose up -d --build

# Stand up the observability layer only (reverse proxy + metrics).
observability:
	docker compose up -d caddy prometheus

down:
	docker compose down

logs:
	docker compose logs -f

tidy:
	go mod tidy

fmt:
	gofmt -l -w .

# Load test the booking flow with k6 (via Docker; no local install needed).
# --network host reaches the app at BASE_URL (default localhost:9090) on
# Linux/macOS; on Docker Desktop for Windows override BASE_URL with the host
# machine IP.
load-test:
	docker run --rm -i --network host \
		-v "$$(pwd)/scripts/loadtest:/scripts" \
		-e BASE_URL="$${BASE_URL:-http://localhost:9090/api/v1}" \
		grafana/k6 run /scripts/booking-flow.js

# Back up Postgres + Redis into ./backups (see scripts/backup.sh for env knobs).
backup:
	./scripts/backup.sh

# Restore from a backup directory: make restore BACKUP=backups/20260810T030000Z
restore:
	./scripts/restore.sh $(BACKUP)
