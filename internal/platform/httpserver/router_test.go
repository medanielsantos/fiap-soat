package httpserver

import (
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRouterRegistersVersionedRoutes(t *testing.T) {
	e := echo.New()
	router := newTestRouter(t, e)
	router.Register()

	registered := make(map[string]bool)
	for _, route := range e.Router().Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"GET /health",
		"POST /api/v1/auth/login",
		"POST /api/v1/admin/users",
		"GET /api/v1/admin/users",
		"GET /api/v1/admin/users/:id",
		"PATCH /api/v1/admin/users/:id/status",
		"PUT /api/v1/admin/users/:id/roles",
		"POST /api/v1/admin/users/:id/permissions/:permission",
		"DELETE /api/v1/admin/users/:id/permissions/:permission",
		"DELETE /api/v1/admin/users/:id",
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
		"POST /api/v1/work-orders",
		"POST /api/v1/work-orders/:id/quotes/approve",
		"POST /api/v1/work-orders/:id/consumptions",
	} {
		if !registered[route] {
			t.Errorf("route %q was not registered", route)
		}
	}
}
