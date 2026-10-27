package containers

import (
	customerpostgres "github.com/medanielsantos/fiap-soat/internal/customer/adapters/out/postgres"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	"go.uber.org/dig"
)

func New() (*dig.Container, error) {
	container := dig.New()
	if err := container.Provide(config.Load); err != nil {
		return nil, err
	}
	if err := container.Provide(database.OpenPostgres); err != nil {
		return nil, err
	}
	if err := container.Provide(customerpostgres.NewCustomerRepository); err != nil {
		return nil, err
	}
	if err := container.Provide(httpserver.New); err != nil {
		return nil, err
	}
	return container, nil
}
