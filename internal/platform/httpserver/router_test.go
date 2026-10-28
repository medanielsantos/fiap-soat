package httpserver

import (
	"context"
	"testing"

	"github.com/labstack/echo/v5"
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
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

type routeTestIDGenerator struct{}

func (routeTestIDGenerator) NewID(context.Context) (string, error) { return "id", nil }

func TestRouterRegistersVersionedRoutes(t *testing.T) {
	e := echo.New()
	service := application.NewCustomerService(routeTestCustomerStore{}, routeTestIDGenerator{})
	router := NewRouter(e, customerhttp.NewHandler(service))
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
	} {
		if !registered[route] {
			t.Errorf("route %q was not registered", route)
		}
	}
}
