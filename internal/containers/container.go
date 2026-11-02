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
	if err := provide(container, constructors()...); err != nil {
		return nil, err
	}
	return container, nil
}

func provide(container *dig.Container, constructors ...any) error {
	for _, constructor := range constructors {
		if err := container.Provide(constructor); err != nil {
			return err
		}
	}
	return nil
}

func constructors() []any {
	return []any{
		config.Load,
		func(cfg config.Config) (ratelimit.Limiter, error) {
			return redisadapter.NewRateLimiter(cfg.RedisURL)
		},
		func(cfg config.Config) (*identityjwt.Issuer, error) {
			return identityjwt.NewIssuer(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiry)
		},
		email.NewSMTPMailer,
		identitypostgres.NewRepository,
		func(repository *identitypostgres.Repository) identityapp.Store { return repository },
		func() identityapp.IDGenerator { return platformid.UUIDGenerator{} },
		func(store identityapp.Store, ids identityapp.IDGenerator, issuer *identityjwt.Issuer, mailer *email.SMTPMailer, cfg config.Config) *identityapp.Service {
			return identityapp.NewService(store, ids, issuer, mailer, cfg.PasswordResetURL)
		},
		identityhttp.NewHandler,
		database.OpenPostgres,
		customerpostgres.NewCustomerRepository,
		func(repository *customerpostgres.CustomerRepository) application.CustomerStore {
			return repository
		},
		func() application.IDGenerator { return platformid.UUIDGenerator{} },
		func(store application.CustomerStore, ids application.IDGenerator, mailer *email.SMTPMailer, cfg config.Config) *application.CustomerService {
			return application.NewCustomerService(store, ids, application.CustomerResetOptions{Mailer: mailer, URL: cfg.CustomerPasswordResetURL})
		},
		func(service *application.CustomerService, vehicles *vehicleapp.VehicleService, issuer *identityjwt.Issuer, workOrders *workorderapp.Service, limiter ratelimit.Limiter) *customerhttp.Handler {
			return customerhttp.NewHandler(service, customerhttp.ClientDependencies{Vehicles: vehicles, Issuer: issuer, WorkOrders: workOrders, RateLimiter: limiter})
		},
		vehiclepostgres.NewVehicleRepository,
		func(repository *vehiclepostgres.VehicleRepository) vehicleapp.VehicleStore {
			return repository
		},
		func(service *application.CustomerService) vehicleapp.CustomerDirectory {
			return vehiclecustomer.NewDirectory(service)
		},
		func() vehicleapp.IDGenerator { return platformid.UUIDGenerator{} },
		vehicleapp.NewVehicleService,
		vehiclehttp.NewHandler,
		catalogpostgres.NewRepository,
		func(repository *catalogpostgres.Repository) catalogapp.Store { return repository },
		func() catalogapp.IDGenerator { return platformid.UUIDGenerator{} },
		catalogapp.NewCatalogService,
		cataloghttp.NewHandler,
		database.NewTransactionManager,
		workorderpostgres.NewRepository,
		func(repository *workorderpostgres.Repository) workorderapp.Store { return repository },
		func(manager *database.TransactionManager) workorderapp.TransactionManager { return manager },
		func() workorderapp.IDGenerator { return platformid.UUIDGenerator{} },
		workorderownership.NewDirectory,
		func(directory *workorderownership.Directory) workorderapp.Ownership { return directory },
		workordercatalog.NewDirectory,
		func(directory *workordercatalog.Directory) workorderapp.Catalog { return directory },
		workordercatalog.NewInventory,
		func(inventory *workordercatalog.Inventory) workorderapp.Inventory { return inventory },
		workorderapp.NewService,
		workorderhttp.NewHandler,
		httpserver.NewEngine,
		httpserver.NewRouter,
		httpserver.New,
	}
}
