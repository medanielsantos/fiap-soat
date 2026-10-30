package httpserver

import (
	"context"
	"testing"

	"github.com/labstack/echo/v5"
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	cataloghttp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/in/http"
	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	vehiclehttp "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/in/http"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	vehicledomain "github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
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

func TestRouterRegistersVersionedRoutes(t *testing.T) {
	e := echo.New()
	service := application.NewCustomerService(routeTestCustomerStore{}, routeTestIDGenerator{})
	vehicleService := vehicleapp.NewVehicleService(routeTestVehicleStore{}, routeTestCustomerDirectory{}, routeTestIDGenerator{})
	catalogService := catalogapp.NewCatalogService(routeTestCatalogStore{}, routeTestIDGenerator{})
	router := NewRouter(e, customerhttp.NewHandler(service), vehiclehttp.NewHandler(vehicleService), cataloghttp.NewHandler(catalogService))
	router.Register()

	registered := make(map[string]bool)
	for _, route := range e.Router().Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"GET /health",
		"POST /api/v1/customers",
		"GET /api/v1/customers",
		"GET /api/v1/customers/:id",
		"PUT /api/v1/customers/:id",
		"DELETE /api/v1/customers/:id",
		"POST /api/v1/vehicles",
		"GET /api/v1/vehicles",
		"GET /api/v1/vehicles/:id",
		"PUT /api/v1/vehicles/:id",
		"DELETE /api/v1/vehicles/:id",
		"POST /api/v1/services",
		"GET /api/v1/services",
		"POST /api/v1/parts",
		"POST /api/v1/parts/:id/stock-movements",
	} {
		if !registered[route] {
			t.Errorf("route %q was not registered", route)
		}
	}
}
