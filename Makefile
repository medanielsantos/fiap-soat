.PHONY: run migrate test test-integration test-coverage lint build

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
	golangci-lint run ./...

build:
	go build -o /tmp/fiap-api ./cmd/api
