SHELL := bash

APPLICATION_NAME := dungeon-campaign-engine
BINARY_OUTPUT_PATH := ./build/$(APPLICATION_NAME)
TEMP_BINARY_PATH := ./tmp/server
MIGRATIONS_DIRECTORY := ./db/migrations
TOOLS_DIRECTORY := ./.bin

GO := go
GO_PACKAGES := ./...
GO_TEST_TIMEOUT := 60s
CGO_ENABLED ?= 0
APP_PORT ?= 8080

AIR_MODULE := github.com/air-verse/air
AIR_VERSION ?= latest
TEMPL_MODULE := github.com/a-h/templ/cmd/templ
TEMPL_VERSION ?= latest
GOTESTSUM_MODULE := gotest.tools/gotestsum
GOTESTSUM_VERSION ?= latest
GOLANGCI_LINT_MODULE := github.com/golangci/golangci-lint/cmd/golangci-lint
GOLANGCI_LINT_VERSION ?= latest
GOOSE_MODULE := github.com/pressly/goose/v3/cmd/goose
GOOSE_VERSION ?= latest

export GOBIN := $(abspath $(TOOLS_DIRECTORY))
export CGO_ENABLED

# Helper: run a command with .env exported
define with_dotenv
bash -lc 'set -a; [ -f .env ] && source ./.env; set +a; $$1'
endef

.PHONY: all tools dev build run test test-race cover lint fmt tidy clean \
        db-up db-up-all db-down db-destroy db-logs db-psql \
        db-migrate-new db-migrate-up db-migrate-down db-backup db-restore

all: build

tools:
	@mkdir -p $(TOOLS_DIRECTORY)
	@$(GO) install $(AIR_MODULE)@$(AIR_VERSION)
	@$(GO) install $(TEMPL_MODULE)@$(TEMPL_VERSION)
	@$(GO) install $(GOTESTSUM_MODULE)@$(GOTESTSUM_VERSION)
	@$(GO) install $(GOLANGCI_LINT_MODULE)@$(GOLANGCI_LINT_VERSION)
	@$(GO) install $(GOOSE_MODULE)@$(GOOSE_VERSION)

dev:
	@echo "==> Starting development mode (tailwind:watch + Templ --watch + Air)..."
	npm run tailwind:build && \
	npm run tailwind:watch & \
	PID_TW=$$!; \
	$(TOOLS_DIRECTORY)/templ generate --watch --proxy="http://localhost:$(APP_PORT)" --open-browser=false -path=./internal/web/views & \
	PID_TEMPL=$$!; \
	trap "kill $$PID_TW $$PID_TEMPL 2>/dev/null || true" EXIT; \
	$(TOOLS_DIRECTORY)/air -c .air.toml


build:
	@$(TOOLS_DIRECTORY)/templ generate -path=./internal/web/views
	@$(GO) build -trimpath -ldflags="-s -w" -o $(BINARY_OUTPUT_PATH) ./cmd/server

run:
	@$(TOOLS_DIRECTORY)/templ generate -path=./internal/web/views
	@$(GO) run ./cmd/server

test:
	@$(TOOLS_DIRECTORY)/gotestsum --format testname -- -timeout $(GO_TEST_TIMEOUT) $(GO_PACKAGES)

test-race:
	@$(TOOLS_DIRECTORY)/gotestsum --format testname -- -race -timeout $(GO_TEST_TIMEOUT) $(GO_PACKAGES)

cover:
	@$(TOOLS_DIRECTORY)/gotestsum --format testname -- -coverprofile=coverage.out -covermode=atomic $(GO_PACKAGES)
	@$(GO) tool cover -func=coverage.out | tail -n 1

lint:
	@$(TOOLS_DIRECTORY)/golangci-lint run

fmt:
	@$(GO) fmt $(GO_PACKAGES)
	@$(GO) vet $(GO_PACKAGES)

tidy:
	@$(GO) mod tidy

clean:
	@rm -rf tmp build coverage.out

# --- Docker / Postgres ---
db-up:
	@$(call with_dotenv, docker compose --env-file .env up -d postgres)

db-up-all:
	@$(call with_dotenv, docker compose --env-file .env up -d)

db-down:
	@docker compose --env-file .env down

db-destroy:
	@docker compose --env-file .env down -v

db-logs:
	@docker compose --env-file .env logs -f postgres

db-psql:
	@$(call with_dotenv, docker exec -it hq_postgres psql -U $$POSTGRES_USER -d $$POSTGRES_DB)

db-backup:
	@$(call with_dotenv, mkdir -p db/backups && docker exec -t hq_postgres pg_dump -U $$POSTGRES_USER -d $$POSTGRES_DB > db/backups/backup-$$(date +%Y%m%d-%H%M%S).sql)

db-restore:
	@read -p "backup file path: " file; \
	$(call with_dotenv, cat $$file | docker exec -i hq_postgres psql -U $$POSTGRES_USER -d $$POSTGRES_DB)

# --- Goose migrations ---
db-migrate-new:
	@read -p "name: " name; $(TOOLS_DIRECTORY)/goose -s create $$name sql

db-migrate-up:
	$(TOOLS_DIRECTORY)/goose up

db-migrate-down:
	$(TOOLS_DIRECTORY)/goose down

# --- Tailwind commands ---
tailwind-build:
	@npm run tailwind:build

tailwind-watch:
	@npm run tailwind:watch

build: tailwind-build
	@$(TOOLS_DIRECTORY)/templ generate -path=./internal/web/views
	@$(GO) build -trimpath -ldflags="-s -w" -o $(BINARY_OUTPUT_PATH) ./cmd/server