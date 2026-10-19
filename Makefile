# =============================================================================
# DispatchAI Makefile
#
# Run from the repository root.  Usage:  make help
#
# Recipes use ">" instead of a TAB (see .RECIPEPREFIX), so editors that convert
# tabs to spaces cannot break this file.  Requires GNU Make >= 3.82.
# =============================================================================

.RECIPEPREFIX := >
.DEFAULT_GOAL := help
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# ---- Project ----------------------------------------------------------------
APP        := dispatchai
MAIN_PKG   := ./cmd/assistant
EXAMPLE_PKG:= ./examples/ecommerce/cmd
BIN_DIR    := bin
BIN        := $(BIN_DIR)/$(APP)
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)

# ---- Local infrastructure (matches deployments/docker/docker-compose.yml) ---
COMPOSE_FILE := deployments/docker/docker-compose.yml
COMPOSE      := docker compose -f $(COMPOSE_FILE)
DATABASE_URL ?= postgres://dispatchai:dispatchai@localhost:5438/dispatch?sslmode=disable
MIGRATIONS   := migrations

# ---- Runtime ----------------------------------------------------------------
# ENV selects .env.dev (default), .env.staging or .env.prod.
ENV          ?= dev
BASE_URL     ?= http://localhost:8989

# ---- Onboarding helpers (override on the command line) ----------------------
# Example: make admin-create ADMIN_TOKEN=... DISPLAY_NAME="Acme"
ADMIN_TOKEN      ?=
DISPLAY_NAME     ?= Acme Logistics
CLIENT_ID        ?=
ENROLLMENT_TOKEN ?=
KEY_ID           ?= dev-key-1
KEY_DIR          ?= .
PRIVATE_KEY      ?= $(KEY_DIR)/private.pem
PUBLIC_KEY_FILE  ?= $(KEY_DIR)/public.b64
PERMISSIONS      ?= "assistant:chat","tools:register","tools:read","tools:update","tools:enable","tools:delete"

# ---- Tooling ----------------------------------------------------------------
GO            ?= go
GOLANGCI_LINT ?= golangci-lint
MIGRATE       ?= migrate
COVER_OUT     := coverage.out

.PHONY: help
help: ## Show this help
> @awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n"} \
>   /^[a-zA-Z0-9_.-]+:.*##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } \
>   /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
> @echo

##@ Build and run

.PHONY: deps
deps: ## Download Go module dependencies
> $(GO) mod download

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
> $(GO) mod tidy

.PHONY: build
build: ## Build the server binary into ./bin
> @mkdir -p $(BIN_DIR)
> CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN) $(MAIN_PKG)
> @echo "built $(BIN) ($(VERSION))"

.PHONY: run
run: ## Run the server (loads .env.$(ENV); migrations apply on startup; listens on :8989)
> ENV=$(ENV) $(GO) run $(MAIN_PKG)

.PHONY: run-bin
run-bin: build ## Build, then run the compiled binary
> ENV=$(ENV) ./$(BIN)

.PHONY: run-example
run-example: ## Run the reference e-commerce host (tool gateway on :8087)
> $(GO) run $(EXAMPLE_PKG)

.PHONY: clean
clean: ## Remove build and coverage artifacts
> rm -rf $(BIN_DIR) $(COVER_OUT) coverage.html

##@ Quality

.PHONY: fmt
fmt: ## Format all Go code
> $(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet
> $(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint (skipped with a notice if not installed)
> @if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
>   $(GOLANGCI_LINT) run ./...; \
> else \
>   echo "golangci-lint not installed; skipping (https://golangci-lint.run)"; \
> fi

.PHONY: test
test: ## Run all tests
> $(GO) test ./... -count=1

.PHONY: test-race
test-race: ## Run all tests with the race detector
> $(GO) test ./... -count=1 -race

.PHONY: test-short
test-short: ## Run tests in -short mode
> $(GO) test ./... -count=1 -short

.PHONY: cover
cover: ## Run tests with coverage and write coverage.html
> $(GO) test ./... -count=1 -coverprofile=$(COVER_OUT)
> $(GO) tool cover -func=$(COVER_OUT) | tail -n 1
> $(GO) tool cover -html=$(COVER_OUT) -o coverage.html
> @echo "wrote coverage.html"

.PHONY: whitespace
whitespace: ## Fail on whitespace errors in the working tree diff
> git diff --check

.PHONY: check
check: fmt vet test whitespace ## Format, vet, test, whitespace check

.PHONY: ci
ci: deps vet test-race whitespace ## What a CI job should run

##@ Local database (Docker Compose)

.PHONY: db-up
db-up: ## Start local PostgreSQL (localhost:5438) and wait until healthy
> $(COMPOSE) up -d
> @echo "waiting for PostgreSQL to become healthy..."
> @for i in $$(seq 1 30); do \
>   status=$$(docker inspect -f '{{.State.Health.Status}}' dispatchai-postgres 2>/dev/null || echo starting); \
>   if [ "$$status" = "healthy" ]; then echo "PostgreSQL is healthy"; exit 0; fi; \
>   sleep 2; \
> done; echo "PostgreSQL did not become healthy in time" >&2; exit 1

.PHONY: db-down
db-down: ## Stop local PostgreSQL (data volume is kept)
> $(COMPOSE) down

.PHONY: db-reset
db-reset: ## DESTROY the local database volume and start fresh
> @read -r -p "This deletes ALL local DispatchAI data. Type 'yes' to continue: " a; \
>   [ "$$a" = "yes" ] || { echo "aborted"; exit 1; }
> $(COMPOSE) down -v
> $(MAKE) db-up

.PHONY: db-logs
db-logs: ## Tail PostgreSQL logs
> $(COMPOSE) logs -f postgres

.PHONY: db-psql
db-psql: ## Open psql against the local database
> psql "$(DATABASE_URL)"

##@ Migrations (the server also applies migrations automatically at startup)

.PHONY: migrate-up
migrate-up: ## Apply all pending migrations with the migrate CLI
> $(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

.PHONY: migrate-down-1
migrate-down-1: ## Roll back ONE migration (013 has no .down.sql)
> $(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

.PHONY: migrate-version
migrate-version: ## Show the current migration version
> $(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" version

.PHONY: migrate-new
migrate-new: ## Create a migration pair: make migrate-new NAME=add_something
> @test -n "$(NAME)" || { echo "usage: make migrate-new NAME=add_something" >&2; exit 1; }
> $(MIGRATE) create -ext sql -dir $(MIGRATIONS) -seq $(NAME)

##@ Integration onboarding helpers

.PHONY: keygen
keygen: ## Generate a P-256 key pair: private.pem (PKCS#8) + public.b64 (enrollment value)
> @test ! -e "$(PRIVATE_KEY)" || { echo "$(PRIVATE_KEY) already exists; refusing to overwrite" >&2; exit 1; }
> openssl ecparam -name prime256v1 -genkey -noout | openssl pkcs8 -topk8 -nocrypt -out "$(PRIVATE_KEY)"
> chmod 600 "$(PRIVATE_KEY)"
> openssl ec -in "$(PRIVATE_KEY)" -pubout -outform DER 2>/dev/null | tail -c 65 | base64 | tr -d '\n' > "$(PUBLIC_KEY_FILE)"
> @echo "private key : $(PRIVATE_KEY)   (keep secret, never commit)"
> @echo "public key  : $(PUBLIC_KEY_FILE)"
> @cat "$(PUBLIC_KEY_FILE)"; echo

.PHONY: admin-create
admin-create: ## 1) Admin: create a pending integration (needs ADMIN_TOKEN)
> @test -n "$(ADMIN_TOKEN)" || { echo "set ADMIN_TOKEN" >&2; exit 1; }
> curl -sS -X POST "$(BASE_URL)/v1/admin/integrations" \
>   -H "Authorization: Bearer $(ADMIN_TOKEN)" -H "Content-Type: application/json" \
>   -d '{"display_name":"$(DISPLAY_NAME)","metadata":{"env":"$(ENV)"}}'; echo

.PHONY: enroll
enroll: ## 2) Host: enroll public key (needs ENROLLMENT_TOKEN, KEY_ID, public.b64)
> @test -n "$(ENROLLMENT_TOKEN)" || { echo "set ENROLLMENT_TOKEN" >&2; exit 1; }
> @test -f "$(PUBLIC_KEY_FILE)" || { echo "run 'make keygen' first" >&2; exit 1; }
> curl -sS -X POST "$(BASE_URL)/v1/integrations/enroll" \
>   -H "Content-Type: application/json" \
>   -d "{\"enrollment_token\":\"$(ENROLLMENT_TOKEN)\",\"key_id\":\"$(KEY_ID)\",\"public_key\":\"$$(cat $(PUBLIC_KEY_FILE))\"}"; echo

.PHONY: admin-activate
admin-activate: ## 3) Admin: activate the integration (needs ADMIN_TOKEN, CLIENT_ID)
> @test -n "$(ADMIN_TOKEN)" -a -n "$(CLIENT_ID)" || { echo "set ADMIN_TOKEN and CLIENT_ID" >&2; exit 1; }
> curl -sS -X POST "$(BASE_URL)/v1/admin/integrations/$(CLIENT_ID)/toggle" \
>   -H "Authorization: Bearer $(ADMIN_TOKEN)" -H "Content-Type: application/json" \
>   -d '{"status":"active"}'; echo

.PHONY: admin-suspend
admin-suspend: ## Admin: suspend the integration (needs ADMIN_TOKEN, CLIENT_ID)
> @test -n "$(ADMIN_TOKEN)" -a -n "$(CLIENT_ID)" || { echo "set ADMIN_TOKEN and CLIENT_ID" >&2; exit 1; }
> curl -sS -X POST "$(BASE_URL)/v1/admin/integrations/$(CLIENT_ID)/toggle" \
>   -H "Authorization: Bearer $(ADMIN_TOKEN)" -H "Content-Type: application/json" \
>   -d '{"status":"suspended"}'; echo

.PHONY: admin-permissions
admin-permissions: ## Admin: list the permission catalog (needs ADMIN_TOKEN)
> @test -n "$(ADMIN_TOKEN)" || { echo "set ADMIN_TOKEN" >&2; exit 1; }
> curl -sS "$(BASE_URL)/v1/admin/permissions" -H "Authorization: Bearer $(ADMIN_TOKEN)"; echo

.PHONY: admin-grant
admin-grant: ## 4) Admin: grant PERMISSIONS to the integration (needs ADMIN_TOKEN, CLIENT_ID)
> @test -n "$(ADMIN_TOKEN)" -a -n "$(CLIENT_ID)" || { echo "set ADMIN_TOKEN and CLIENT_ID" >&2; exit 1; }
> curl -sS -X POST "$(BASE_URL)/v1/admin/integrations/$(CLIENT_ID)/permissions" \
>   -H "Authorization: Bearer $(ADMIN_TOKEN)" -H "Content-Type: application/json" \
>   -d '{"permissions":[$(PERMISSIONS)]}'; echo

##@ Safety

.PHONY: check-secrets
check-secrets: ## Scan tracked files for private keys / API-key patterns; verify key files are ignored
> @if git grep -nEI 'BEGIN (EC |RSA )?PRIVATE KEY|AIza[0-9A-Za-z_-]{30,}|sk-[A-Za-z0-9]{20,}' -- . ':!Makefile' ':!README.md'; then \
>   echo "possible secrets found above" >&2; exit 1; \
> else echo "no obvious secrets in tracked files"; fi
> @for f in .env.dev .env.staging .env.prod private.pem; do \
>   if [ -e "$$f" ] && ! git check-ignore -q "$$f"; then echo "WARNING: $$f exists but is not git-ignored" >&2; fi; \
> done

.PHONY: docker-build
docker-build: ## Build the container image (requires a non-empty Dockerfile)
> @test -s deployments/docker/Dockerfile || { echo "deployments/docker/Dockerfile is empty; add one first" >&2; exit 1; }
> docker build -f deployments/docker/Dockerfile -t $(APP):$(VERSION) .