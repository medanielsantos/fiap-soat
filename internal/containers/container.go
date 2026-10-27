package containers

import (
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	"go.uber.org/dig"
)

func New() (*dig.Container, error) {
	container := dig.New()
	if err := container.Provide(config.Load); err != nil {
		return nil, err
	}
	if err := container.Provide(httpserver.New); err != nil {
		return nil, err
	}
	return container, nil
}
