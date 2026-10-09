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
# Settings read from .env when set there (a second checkout, such as the online worktree,
# uses its own container and ports), with the defaults of the main checkout.
dotenv = $(shell sed -n 's/^$(1)=//p' .env 2>/dev/null)
APP_PORT ?= $(or $(call dotenv,APP_PORT),8080)
TEMPL_PROXY_PORT ?= $(or $(call dotenv,TEMPL_PROXY_PORT),7331)
DB_CONTAINER ?= $(or $(call dotenv,DB_CONTAINER),hq_postgres)

AIR_MODULE := github.com/air-verse/air
AIR_VERSION ?= v1.67.4
GOTESTSUM_MODULE := gotest.tools/gotestsum
GOTESTSUM_VERSION ?= v1.13.0
GOLANGCI_LINT_MODULE := github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.14.0
GOOSE_MODULE := github.com/pressly/goose/v3/cmd/goose
GOOSE_VERSION ?= v3.28.0

export GOBIN := $(abspath $(TOOLS_DIRECTORY))
export CGO_ENABLED

# Helper: run a command with .env exported
define with_dotenv
bash -lc 'set -a; [ -f .env ] && source ./.env; set +a; $$1'
endef

.PHONY: all tools dev build run test test-race cover lint fmt tidy clean image push deploy import-audio hosted-backup hosted-restore-check hosted-restore \
        test-db test-js test-all \
        db-up db-up-all db-down db-destroy db-logs db-psql \
        db-migrate-new db-migrate-up db-migrate-down db-backup db-restore import-content import-classes \
        load-script print-script fill-campaign narration-text

all: build

tools:
	@mkdir -p $(TOOLS_DIRECTORY)
	@$(GO) install $(AIR_MODULE)@$(AIR_VERSION)
	@$(GO) install $(GOTESTSUM_MODULE)@$(GOTESTSUM_VERSION)
	@$(GO) install $(GOLANGCI_LINT_MODULE)@$(GOLANGCI_LINT_VERSION)
	@$(GO) install $(GOOSE_MODULE)@$(GOOSE_VERSION)

dev:
	@echo "==> Starting development mode (tailwind + TS watch + templ proxy + Air)..."
	bun run tailwind:build && \
	bun run tailwind:watch & \
	PID_TW=$$!; \
	bun run watch:web & \
	PID_WEB=$$!; \
	$(GO) tool templ generate --watch --proxy="http://localhost:$(APP_PORT)" --proxyport=$(TEMPL_PROXY_PORT) --open-browser=false -path=./internal/web/views & \
	PID_TEMPL=$$!; \
	trap "kill $$PID_TW $$PID_WEB $$PID_TEMPL 2>/dev/null || true" EXIT; \
	$(TOOLS_DIRECTORY)/air -c .air.toml

run:
	@$(GO) tool templ generate -path=./internal/web/views
	@$(GO) run ./cmd/server

test:
	@$(TOOLS_DIRECTORY)/gotestsum --format testname -- -timeout $(GO_TEST_TIMEOUT) $(GO_PACKAGES)

test-race:
	@$(TOOLS_DIRECTORY)/gotestsum --format testname -- -race -timeout $(GO_TEST_TIMEOUT) $(GO_PACKAGES)

cover:
	@$(TOOLS_DIRECTORY)/gotestsum --format testname -- -coverprofile=coverage.out -covermode=atomic $(GO_PACKAGES)
	@$(GO) tool cover -func=coverage.out | tail -n 1

# Runs the internal packages including database tests. Each test uses its own
# throwaway schema in the dev database (make db-up), so dev data is untouched.
test-db:
	@set -a; [ -f .env ] && . ./.env; set +a; \
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:-$$DATABASE_URL}" \
	$(TOOLS_DIRECTORY)/gotestsum --format testname -- -count=1 -timeout $(GO_TEST_TIMEOUT) $(GO_PACKAGES)

test-js:
	@echo "==> Running JavaScript tests..."
	@bun run test

test-all: test test-js
	@echo "==> All tests completed"

lint:
	@$(TOOLS_DIRECTORY)/golangci-lint run
	@bun run lint
	@bun run typecheck
	@bun run spelling

fmt:
	@$(GO) fmt $(GO_PACKAGES)
	@$(GO) vet $(GO_PACKAGES)

tidy:
	@$(GO) mod tidy

clean:
	@rm -rf tmp build coverage.out

# --- Docker / Postgres ---
db-up:
	@docker compose --env-file .env up -d postgres

db-up-all:
	@docker compose --env-file .env up -d

db-down:
	@docker compose --env-file .env down

db-destroy:
	@docker compose --env-file .env down -v

db-logs:
	@docker compose --env-file .env logs -f postgres

db-psql:
	@docker exec -it $(DB_CONTAINER) psql -U $$POSTGRES_USER -d $$POSTGRES_DB

db-backup:
	@mkdir -p db/backups && docker exec -t $(DB_CONTAINER) pg_dump -U $$POSTGRES_USER -d $$POSTGRES_DB > db/backups/backup-$$(date +%Y%m%d-%H%M%S).sql

db-restore:
	@read -p "backup file path: " file; \
	cat $$file | docker exec -i $(DB_CONTAINER) psql -U $$POSTGRES_USER -d $$POSTGRES_DB

# --- Goose migrations ---
db-migrate-new:
	@read -p "name: " name; $(TOOLS_DIRECTORY)/goose -s create $$name sql

db-migrate-up:
	$(TOOLS_DIRECTORY)/goose up

db-migrate-down:
	$(TOOLS_DIRECTORY)/goose down

# --- Content import ---
# Loads content/board.json + a quest into the database (idempotent).
# Override the quest with: make import-content QUEST=base/quests/quest-02.json
QUEST ?= base/quests/quest-01.json
import-content:
	@set -a; [ -f .env ] && . ./.env; set +a; $(GO) run ./cmd/import-content -quest $(QUEST) $(if $(DB),-db "$(DB)")

# The base game's hero classes only (content/heroes into custom_hero_class), into
# DATABASE_URL, the database in DB=..., or the hosted one with HOSTED=1 (its URL is
# read from .env and never printed): make import-classes HOSTED=1
import-classes:
	@set -a; [ -f .env ] && . ./.env; set +a; $(GO) run ./cmd/import-content -classes-only \
		$(if $(HOSTED),-db "$$HOSTED_DATABASE_URL",$(if $(DB),-db "$(DB)"))

# Read-aloud script: join a script folder's numbered files, check them and save them as a
# campaign's script. make load-script CAMPAIGN="Three Plagues" [SCRIPT_DIR=...]
SCRIPT_DIR ?= docs/campaigns/three-plagues/script
load-script:
	@set -a; [ -f .env ] && . ./.env; set +a; $(GO) run ./cmd/load-script -dir $(SCRIPT_DIR) -campaign "$(CAMPAIGN)"

# Load the agreed combat numbers (classes, monster stats, starting kits) into a
# campaign. A dry run unless APPLY=1:
# Copy read-aloud clips from a folder of <campaign id>/<passage id>.<ext> files (main's
# AUDIO_DIR layout) into DATABASE_URL, or the database in DB=...; unchanged clips are skipped.
AUDIO_FOLDER ?= audio
import-audio:
	@set -a; [ -f .env ] && . ./.env; set +a; $(GO) run ./cmd/import-audio -dir $(AUDIO_FOLDER) $(if $(DB),-db "$(DB)")

# make fill-campaign CAMPAIGN="Three Plagues" [APPLY=1] [COMBAT_FILE=...]
COMBAT_FILE ?= docs/campaigns/three-plagues/combat.json
fill-campaign:
	@set -a; [ -f .env ] && . ./.env; set +a; $(GO) run ./cmd/fill-campaign -file $(COMBAT_FILE) -campaign "$(CAMPAIGN)" $(if $(APPLY),-apply)

# Write the joined script to the terminal (for pasting into the campaign page).
print-script:
	@$(GO) run ./cmd/load-script -dir $(SCRIPT_DIR) -print

# Write paste-ready text for voicing the script (one <id>.txt per passage plus a
# README.md with speaker turns and pronunciation) into ./narration (gitignored).
# make narration-text [ONLY=P0,Q1]
narration-text:
	@$(GO) run ./cmd/narration-text -dir $(SCRIPT_DIR) $(if $(ONLY),-only "$(ONLY)")

# --- Tailwind commands ---
tailwind-build:
	@bun run tailwind:build

tailwind-watch:
	@bun run tailwind:watch

build: tailwind-build
	@bun run build:web
	@$(GO) tool templ generate -path=./internal/web/views
	@$(GO) build -trimpath -ldflags="-s -w" -o $(BINARY_OUTPUT_PATH) ./cmd/server
# --- Hosting (docs/ONLINE_AND_RULES_PLAN.md, Phase 6) ---
# The image carries content/ and assets/ (HeroQuest material, passed in as named build
# contexts). Build it here and push it only to the private registry in DOCKER_IMAGE
# (.env, e.g. youruser/dce), never to GitHub. Render pulls it; RENDER_DEPLOY_HOOK (.env,
# a secret) starts the deploy.
DOCKER_IMAGE ?= $(call dotenv,DOCKER_IMAGE)
IMAGE_TAG ?= $(shell git describe --always --dirty --abbrev=7)
IMAGE_PLATFORM ?= linux/amd64
CONTENT_SRC ?= $(realpath content)
ASSETS_SRC ?= $(realpath assets)

image:
	@test -n "$(DOCKER_IMAGE)" || { echo "Set DOCKER_IMAGE in .env (e.g. youruser/dce)."; exit 1; }
	@test -d "$(CONTENT_SRC)" && test -d "$(ASSETS_SRC)" || { echo "content/ and assets/ are needed (see CLAUDE.md)."; exit 1; }
	docker buildx build --platform $(IMAGE_PLATFORM) \
		--build-context content=$(CONTENT_SRC) --build-context assets=$(ASSETS_SRC) \
		-t $(DOCKER_IMAGE):$(IMAGE_TAG) --load .

# Docker Hub answers 200 to anyone for a public repository (404 if private or missing).
hub_public = test "$$(curl -s -o /dev/null -w '%{http_code}' https://hub.docker.com/v2/namespaces/$(firstword $(subst /, ,$(DOCKER_IMAGE)))/repositories/$(lastword $(subst /, ,$(DOCKER_IMAGE))))" = 200

push: image
	@case "$(IMAGE_TAG)" in *-dirty) echo "Commit first: $(IMAGE_TAG) has uncommitted changes."; exit 1;; esac
	@if $(hub_public); then echo "$(DOCKER_IMAGE) is PUBLIC on Docker Hub: make it private before pushing."; exit 1; fi
	docker push $(DOCKER_IMAGE):$(IMAGE_TAG)
	@if $(hub_public); then echo "WARNING: $(DOCKER_IMAGE) is PUBLIC on Docker Hub. Make it private now (Settings > Visibility)."; exit 1; fi

# Backups of the hosted database (scripts/hosted-db.sh; HOSTED_DATABASE_URL in .env).
# hosted-backup downloads a dump to db/backups/; hosted-restore-check restores one into a
# throwaway database and checks its row counts; hosted-restore replaces the hosted database.
hosted-backup:
	@set -a; . ./.env; set +a; scripts/hosted-db.sh backup

hosted-restore-check:
	@scripts/hosted-db.sh check "$(or $(FILE),$$(ls -t db/backups/hosted-*.dump | head -1))"

hosted-restore:
	@set -a; . ./.env; set +a; scripts/hosted-db.sh restore "$(FILE)"

# Deploys the pushed tag. The hook URL holds a key, so it is never printed.
deploy: push
	@hook="$(call dotenv,RENDER_DEPLOY_HOOK)"; test -n "$$hook" || { echo "Set RENDER_DEPLOY_HOOK in .env."; exit 1; }; \
		curl -fsS -o /dev/null -G --data-urlencode "imgURL=docker.io/$(DOCKER_IMAGE):$(IMAGE_TAG)" "$$hook" \
		&& echo "Render is deploying $(DOCKER_IMAGE):$(IMAGE_TAG)."
