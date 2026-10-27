package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/medanielsantos/fiap-soat/internal/containers"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	container, err := containers.New()
	if err != nil {
		log.Fatalf("initialize dependencies: %v", err)
	}
	if err := container.Invoke(func(server *httpserver.Server) error {
		return server.Run(ctx)
	}); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run API: %v", err)
	}
}
