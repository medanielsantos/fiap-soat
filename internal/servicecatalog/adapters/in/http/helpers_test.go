package servicecataloghttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

var adminPermissions = []identityapp.Permission{identityapp.PermissionProductsCreate, identityapp.PermissionProductsDelete, identityapp.PermissionProductsPriceUpdate}

// require mimics the identity middleware: it denies the request unless the permission was granted.
func require(permission identityapp.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !hasPermission(c, permission) {
				return write(c, http.StatusForbidden, "Insufficient permissions", "")
			}
			return next(c)
		}
	}
}

// newTestEcho wires the handler behind a fake authentication step granting the given permissions.
func newTestEcho(store *handlerCatalogStore, permissions ...identityapp.Permission) *echo.Echo {
	e := echo.New()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "user-1")
			c.Set("permissions", permissions)
			return next(c)
		}
	})
	service := application.NewCatalogService(store, &handlerIDs{})
	NewHandler(service).RegisterRoutes(api, require(identityapp.PermissionProductsCreate), require(identityapp.PermissionProductsDelete))
	return e
}

func serve(e *echo.Echo, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func expectStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, want, response.Body.String())
	}
}

func decodeBody[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, response.Body.String())
	}
	return out
}

func seedService(t *testing.T, store *handlerCatalogStore, id string, price int64) {
	t.Helper()
	v, err := domain.NewService(id, "S-"+id, "Alignment", "", price, 30, true, "admin")
	if err != nil {
		t.Fatal(err)
	}
	store.services[id] = v
}

func seedPart(t *testing.T, store *handlerCatalogStore, id string, purchase, sale, stock int64) {
	t.Helper()
	v, err := domain.NewPart(id, "SKU-"+id, "P-"+id, "Oil filter", "", "un", purchase, sale, stock, 0, true, "admin")
	if err != nil {
		t.Fatal(err)
	}
	store.parts[id] = v
}
