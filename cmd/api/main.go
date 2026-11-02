package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/medanielsantos/fiap-soat/internal/containers"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	"go.uber.org/dig"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, containers.New); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, newContainer func() (*dig.Container, error)) error {
	container, err := newContainer()
	if err != nil {
		return fmt.Errorf("initialize dependencies: %w", err)
	}
	if err := container.Invoke(func(server *httpserver.Server) error {
		return server.Run(ctx)
	}); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("run API: %w", err)
	}
	return nil
}
