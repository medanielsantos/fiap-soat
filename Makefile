.PHONY: run migrate test test-integration test-coverage lint build

GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || printf '%s/bin/golangci-lint' "$$(go env GOPATH)")

run:
	go run ./cmd/api

migrate:
	go run ./cmd/migrate up

test:
	go test ./...

test-integration:
	go test ./tests/integration

test-coverage:
	go test -cover ./internal/workorder/domain ./internal/servicecatalog/domain

lint:
	$(GOLANGCI_LINT) run ./...

build:
	go build -o /tmp/fiap-api ./cmd/api
