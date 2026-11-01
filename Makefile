.PHONY: run migrate seed test test-unit test-integration test-coverage lint build docker-up docker-down

GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || printf '%s/bin/golangci-lint' "$$(go env GOPATH)")

run:
	go run ./cmd/api

migrate:
	go run ./cmd/migrate up

seed:
	go run ./cmd/seed

test:
	go test ./...

test-unit:
	go test ./cmd/... ./internal/... ./migrations

test-integration:
	go test ./tests/integration

test-coverage:
	go test -cover ./internal/workorder/domain ./internal/servicecatalog/domain

lint:
	$(GOLANGCI_LINT) run ./...

build:
	go build -o /tmp/fiap-api ./cmd/api

docker-up:
	docker compose --env-file .env -f .docker/local/compose.yaml up --build

docker-down:
	docker compose --env-file .env -f .docker/local/compose.yaml down
