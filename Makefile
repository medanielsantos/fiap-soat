.PHONY: run test lint build

run:
	go run ./cmd/api

test:
	go test ./...

lint:
	golangci-lint run ./...

build:
	go build -o /tmp/fiap-api ./cmd/api
