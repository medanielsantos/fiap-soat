package containers

import (
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	customerpostgres "github.com/medanielsantos/fiap-soat/internal/customer/adapters/out/postgres"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	platformid "github.com/medanielsantos/fiap-soat/internal/platform/id"
	cataloghttp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/in/http"
	catalogpostgres "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/out/postgres"
	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	vehiclehttp "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/in/http"
	vehiclecustomer "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/out/customer"
	vehiclepostgres "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/out/postgres"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
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
	if err := container.Provide(vehiclepostgres.NewVehicleRepository); err != nil {
		return nil, err
	}
	if err := container.Provide(func(repository *vehiclepostgres.VehicleRepository) vehicleapp.VehicleStore {
		return repository
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(func(service *application.CustomerService) vehicleapp.CustomerDirectory {
		return vehiclecustomer.NewDirectory(service)
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(func() vehicleapp.IDGenerator { return platformid.UUIDGenerator{} }); err != nil {
		return nil, err
	}
	if err := container.Provide(vehicleapp.NewVehicleService); err != nil {
		return nil, err
	}
	if err := container.Provide(vehiclehttp.NewHandler); err != nil {
		return nil, err
	}
	if err := container.Provide(catalogpostgres.NewRepository); err != nil {
		return nil, err
	}
	if err := container.Provide(func(repository *catalogpostgres.Repository) catalogapp.Store { return repository }); err != nil {
		return nil, err
	}
	if err := container.Provide(func() catalogapp.IDGenerator { return platformid.UUIDGenerator{} }); err != nil {
		return nil, err
	}
	if err := container.Provide(catalogapp.NewCatalogService); err != nil {
		return nil, err
	}
	if err := container.Provide(cataloghttp.NewHandler); err != nil {
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
