.PHONY: help tidy fmt test build run run-mcp clean metrics compose-config compose-up compose-down compose-logs compose-ps

SHELL := /bin/sh

APP_CONFIG ?= $(CURDIR)/configs/config.example.yaml
COMPOSE_FILE ?= deploy/docker-compose.yml

help: ## Show available make targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "%-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

tidy: ## Tidy Go modules
	go mod tidy

fmt: ## Format Go code
	gofmt -w $$(go list -f '{{.Dir}}' ./... )

test: ## Run all tests
	go test ./...

build: ## Build API binary
	go build -o bin/pokemon-ai ./cmd/api

run: ## Run API with APP_CONFIG file
	APP_CONFIG_FILE="$(APP_CONFIG)" go run ./cmd/api

run-mcp: ## Run API with MCP enabled override
	ENABLE_MCP=true APP_CONFIG_FILE="$(APP_CONFIG)" go run ./cmd/api

clean: ## Remove build artifacts
	rm -rf bin

metrics: ## Show exported Prometheus metric names
	curl -fsS http://localhost:8080/metrics | awk -F'{' '/^pokemon_ai_/ {print $$1}' | sort -u

compose-config: ## Render and validate docker compose config
	docker compose -f "$(COMPOSE_FILE)" config

compose-up: ## Start full local stack in background
	docker compose -f "$(COMPOSE_FILE)" up -d

compose-down: ## Stop and remove stack
	docker compose -f "$(COMPOSE_FILE)" down

compose-logs: ## Tail compose logs
	docker compose -f "$(COMPOSE_FILE)" logs -f --tail=200

compose-ps: ## Show compose service status
	docker compose -f "$(COMPOSE_FILE)" ps
