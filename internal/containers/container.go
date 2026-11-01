package containers

import (
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	customerpostgres "github.com/medanielsantos/fiap-soat/internal/customer/adapters/out/postgres"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	identityhttp "github.com/medanielsantos/fiap-soat/internal/identity/adapters/in/http"
	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	identitypostgres "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/postgres"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/platform/email"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	platformid "github.com/medanielsantos/fiap-soat/internal/platform/id"
	"github.com/medanielsantos/fiap-soat/internal/platform/ratelimit"
	redisadapter "github.com/medanielsantos/fiap-soat/internal/platform/redis"
	cataloghttp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/in/http"
	catalogpostgres "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/out/postgres"
	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	vehiclehttp "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/in/http"
	vehiclecustomer "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/out/customer"
	vehiclepostgres "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/out/postgres"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	workorderhttp "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/in/http"
	workordercatalog "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/out/catalog"
	workorderownership "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/out/customervehicle"
	workorderpostgres "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/out/postgres"
	workorderapp "github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"go.uber.org/dig"
)

func New() (*dig.Container, error) {
	container := dig.New()
	if err := container.Provide(config.Load); err != nil {
		return nil, err
	}
	if err := container.Provide(func(cfg config.Config) (ratelimit.Limiter, error) {
		return redisadapter.NewRateLimiter(cfg.RedisURL)
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(func(cfg config.Config) (*identityjwt.Issuer, error) {
		return identityjwt.NewIssuer(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiry)
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(email.NewSMTPMailer); err != nil {
		return nil, err
	}
	if err := container.Provide(identitypostgres.NewRepository); err != nil {
		return nil, err
	}
	if err := container.Provide(func(repository *identitypostgres.Repository) identityapp.Store { return repository }); err != nil {
		return nil, err
	}
	if err := container.Provide(func() identityapp.IDGenerator { return platformid.UUIDGenerator{} }); err != nil {
		return nil, err
	}
	if err := container.Provide(func(store identityapp.Store, ids identityapp.IDGenerator, issuer *identityjwt.Issuer, mailer *email.SMTPMailer, cfg config.Config) *identityapp.Service {
		return identityapp.NewService(store, ids, issuer, mailer, cfg.PasswordResetURL)
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(identityhttp.NewHandler); err != nil {
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
	if err := container.Provide(func(store application.CustomerStore, ids application.IDGenerator, mailer *email.SMTPMailer, cfg config.Config) *application.CustomerService {
		return application.NewCustomerService(store, ids, application.CustomerResetOptions{Mailer: mailer, URL: cfg.CustomerPasswordResetURL})
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(func(service *application.CustomerService, vehicles *vehicleapp.VehicleService, issuer *identityjwt.Issuer, workOrders *workorderapp.Service, limiter ratelimit.Limiter) *customerhttp.Handler {
		return customerhttp.NewHandler(service, customerhttp.ClientDependencies{Vehicles: vehicles, Issuer: issuer, WorkOrders: workOrders, RateLimiter: limiter})
	}); err != nil {
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
	if err := container.Provide(database.NewTransactionManager); err != nil {
		return nil, err
	}
	if err := container.Provide(workorderpostgres.NewRepository); err != nil {
		return nil, err
	}
	if err := container.Provide(func(repository *workorderpostgres.Repository) workorderapp.Store { return repository }); err != nil {
		return nil, err
	}
	if err := container.Provide(func(manager *database.TransactionManager) workorderapp.TransactionManager { return manager }); err != nil {
		return nil, err
	}
	if err := container.Provide(func() workorderapp.IDGenerator { return platformid.UUIDGenerator{} }); err != nil {
		return nil, err
	}
	if err := container.Provide(workorderownership.NewDirectory); err != nil {
		return nil, err
	}
	if err := container.Provide(func(directory *workorderownership.Directory) workorderapp.Ownership { return directory }); err != nil {
		return nil, err
	}
	if err := container.Provide(workordercatalog.NewDirectory); err != nil {
		return nil, err
	}
	if err := container.Provide(func(directory *workordercatalog.Directory) workorderapp.Catalog { return directory }); err != nil {
		return nil, err
	}
	if err := container.Provide(workordercatalog.NewInventory); err != nil {
		return nil, err
	}
	if err := container.Provide(func(inventory *workordercatalog.Inventory) workorderapp.Inventory { return inventory }); err != nil {
		return nil, err
	}
	if err := container.Provide(workorderapp.NewService); err != nil {
		return nil, err
	}
	if err := container.Provide(workorderhttp.NewHandler); err != nil {
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
