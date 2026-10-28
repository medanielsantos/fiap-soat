.PHONY: run migrate test lint build

run:
	go run ./cmd/api

migrate:
	go run ./cmd/migrate up

test:
	go test ./...

lint:
	golangci-lint run ./...

build:
	go build -o /tmp/fiap-api ./cmd/api
