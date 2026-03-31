.PHONY: dev build test test-integration lint swagger \
       migrate-up migrate-down migrate-create \
       docker-up docker-down seed clean

# Variables
APP_NAME := spatial-memory
BINARY := ./bin/$(APP_NAME)
DSN := postgres://spatial:spatial@localhost:5432/spatial_memory?sslmode=disable

# ─── Development ──────────────────────────────────────────────────────────────

dev:
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	air -c .air.toml

build:
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/server
	@echo "Built $(BINARY)"

run: build
	$(BINARY)

clean:
	rm -rf bin/ tmp/

# ─── Testing ──────────────────────────────────────────────────────────────────

test:
	go test -v -race -count=1 ./internal/...

test-integration:
	go test -v -race -count=1 -tags=integration ./tests/integration/...

test-coverage:
	go test -race -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ─── Linting ──────────────────────────────────────────────────────────────────

lint:
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

# ─── Database Migrations ─────────────────────────────────────────────────────

migrate-up:
	@command -v migrate >/dev/null 2>&1 || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path migrations -database "$(DSN)" up

migrate-down:
	@command -v migrate >/dev/null 2>&1 || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path migrations -database "$(DSN)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# ─── Docker ───────────────────────────────────────────────────────────────────

docker-up:
	docker compose up -d
	@echo "Waiting for services..."
	@sleep 3
	@docker compose ps

docker-down:
	docker compose down

docker-reset: docker-down
	docker compose down -v
	docker compose up -d

# ─── API Documentation ───────────────────────────────────────────────────────

swagger:
	@command -v swag >/dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/server/main.go -o docs/swagger

# ─── Seed Data ────────────────────────────────────────────────────────────────

seed:
	@echo "TODO: implement seed command"
