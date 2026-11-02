package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/containers"
	"go.uber.org/dig"
)

func TestRunServesHealthUntilContextIsCanceled(t *testing.T) {
	baseURL := setAPIEnvironment(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, containers.New) }()

	waitHealthy(t, baseURL)
	cancel()
	if err := waitDone(t, done); err != nil {
		t.Fatalf("run after cancel = %v, want nil", err)
	}
}

func TestRunWrapsContainerFailure(t *testing.T) {
	boom := errors.New("boom")
	err := run(context.Background(), func() (*dig.Container, error) { return nil, boom })
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "initialize dependencies:") {
		t.Fatalf("err = %v, want wrapped container error", err)
	}
}

func TestRunWrapsDependencyResolutionFailure(t *testing.T) {
	setAPIEnvironment(t)
	t.Setenv("REDIS_URL", "http://not-redis")

	err := run(context.Background(), containers.New)
	if err == nil || !strings.HasPrefix(err.Error(), "run API:") || !strings.Contains(err.Error(), "parse REDIS_URL") {
		t.Fatalf("err = %v, want wrapped resolution error", err)
	}
}
