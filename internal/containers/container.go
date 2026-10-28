package containers

import (
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	customerpostgres "github.com/medanielsantos/fiap-soat/internal/customer/adapters/out/postgres"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	platformid "github.com/medanielsantos/fiap-soat/internal/platform/id"
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
	if err := container.Provide(func(repository *customerpostgres.CustomerRepository) application.CustomerStore {
		return repository
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(func() application.IDGenerator { return platformid.UUIDGenerator{} }); err != nil {
		return nil, err
	}
	if err := container.Provide(application.NewCustomerService); err != nil {
		return nil, err
	}
	if err := container.Provide(customerhttp.NewHandler); err != nil {
		return nil, err
	}
	if err := container.Provide(httpserver.NewEngine); err != nil {
		return nil, err
	}
	if err := container.Provide(httpserver.NewRouter); err != nil {
		return nil, err
	}
	if err := container.Provide(httpserver.New); err != nil {
		return nil, err
	}
	return container, nil
}
