package containers

import (
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/httpserver"
	"github.com/medanielsantos/fiap-soat/internal/platform/ratelimit"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

func TestNewResolvesHTTPServerWithRegisteredRoutes(t *testing.T) {
	setValidEnvironment(t)
	container, err := New()
	if err != nil {
		t.Fatal(err)
	}

	err = container.Invoke(func(server *httpserver.Server, engine *echo.Echo, limiter ratelimit.Limiter, db *gorm.DB) {
		if server == nil || limiter == nil {
			t.Fatal("server and limiter must be resolved")
		}
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sqlDB.Close() })
		registered := map[string]bool{}
		for _, route := range engine.Router().Routes() {
			registered[route.Method+" "+route.Path] = true
		}
		for _, route := range []string{"GET /health", "POST /api/v1/auth/login", "POST /api/v1/customers", "POST /api/v1/work-orders"} {
			if !registered[route] {
				t.Errorf("route %q not registered by the resolved server", route)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewSurfacesConstructorErrorsOnInvoke(t *testing.T) {
	cases := []struct{ name, key, value, want string }{
		{"invalid redis url", "REDIS_URL", "http://not-redis", "parse REDIS_URL"},
		{"short jwt secret", "JWT_SECRET", "short", "JWT_SECRET must contain at least 32 bytes"},
		{"missing database", "DATABASE_URL", "", "DATABASE_URL is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tc.key, tc.value)
			container, err := New()
			if err != nil {
				t.Fatal(err)
			}
			err = container.Invoke(func(*httpserver.Server) {})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Invoke error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestProvideStopsAtFirstInvalidConstructor(t *testing.T) {
	container := dig.New()
	err := provide(container, func() int { return 1 }, "not a constructor", func() string { return "never" })
	if err == nil {
		t.Fatal("expected error for a non-function constructor")
	}
	if err := container.Invoke(func(string) {}); err == nil {
		t.Fatal("constructors after the failing one must not be provided")
	}
}
