package httpserver

import (
	"context"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	identityhttp "github.com/medanielsantos/fiap-soat/internal/identity/adapters/in/http"
	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	cataloghttp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/in/http"
	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	vehiclehttp "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/in/http"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	vehicledomain "github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
	workorderhttp "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/in/http"
	workorderapp "github.com/medanielsantos/fiap-soat/internal/workorder/application"
)

type routeTestCustomerStore struct{}

func (routeTestCustomerStore) Save(context.Context, domain.Customer) error { return nil }
func (routeTestCustomerStore) FindByID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, application.ErrCustomerNotFound
}
func (routeTestCustomerStore) List(_ context.Context, params pagination.Params) (pagination.Page[domain.Customer], error) {
	return pagination.NewPage[domain.Customer](nil, params, 0), nil
}
func (routeTestCustomerStore) Update(context.Context, domain.Customer) error { return nil }
func (routeTestCustomerStore) Delete(context.Context, string) error          { return nil }

type routeTestVehicleStore struct{}

func (routeTestVehicleStore) Save(context.Context, vehicledomain.Vehicle) error { return nil }
func (routeTestVehicleStore) FindByID(context.Context, string) (vehicledomain.Vehicle, error) {
	return vehicledomain.Vehicle{}, vehicleapp.ErrVehicleNotFound
}
func (routeTestVehicleStore) List(_ context.Context, _ vehicleapp.ListFilter, params pagination.Params) (pagination.Page[vehicledomain.Vehicle], error) {
	return pagination.NewPage[vehicledomain.Vehicle](nil, params, 0), nil
}
func (routeTestVehicleStore) Update(context.Context, vehicledomain.Vehicle) error { return nil }
func (routeTestVehicleStore) Delete(context.Context, string) error                { return nil }

type routeTestCustomerDirectory struct{}

func (routeTestCustomerDirectory) IsActiveCustomer(context.Context, string) (bool, error) {
	return true, nil
}

type routeTestIDGenerator struct{}

func (routeTestIDGenerator) NewID(context.Context) (string, error) { return "id", nil }

type routeTestCatalogStore struct{ catalogapp.Store }
type routeTestWorkOrderStore struct{ workorderapp.Store }
type routeTestWorkOrderDependencies struct {
	workorderapp.TransactionManager
	workorderapp.Ownership
	workorderapp.Catalog
	workorderapp.Inventory
}

func newTestRouter(t *testing.T, engine *echo.Echo) *Router {
	t.Helper()
	service := application.NewCustomerService(routeTestCustomerStore{}, routeTestIDGenerator{})
	vehicleService := vehicleapp.NewVehicleService(routeTestVehicleStore{}, routeTestCustomerDirectory{}, routeTestIDGenerator{})
	catalogService := catalogapp.NewCatalogService(routeTestCatalogStore{}, routeTestIDGenerator{})
	workOrderService := workorderapp.NewService(routeTestWorkOrderStore{}, routeTestWorkOrderDependencies{}, routeTestIDGenerator{}, routeTestWorkOrderDependencies{}, routeTestWorkOrderDependencies{}, routeTestWorkOrderDependencies{})
	issuer, err := identityjwt.NewIssuer("01234567890123456789012345678901", "test", 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	identityService := identityapp.NewService(nil, nil, issuer, nil, "")
	identityHandler := identityhttp.NewHandler(identityService, issuer)
	return NewRouter(engine, customerhttp.NewHandler(service), vehiclehttp.NewHandler(vehicleService), cataloghttp.NewHandler(catalogService), workorderhttp.NewHandler(workOrderService), identityHandler)
}
