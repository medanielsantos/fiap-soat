package customerhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
)

const vehiclesPath = "/api/v1/customers/client/vehicles"

func TestRequireClient(t *testing.T) {
	t.Run("test_it_should_require_bearer_authorization", func(t *testing.T) {
		env := newClientEnv(t)
		for _, header := range []string{"", "Basic abc", "Bearer"} {
			request := httptest.NewRequest(http.MethodGet, vehiclesPath, nil)
			request.Header.Set(echo.HeaderAuthorization, header)
			response := httptest.NewRecorder()
			env.echo.ServeHTTP(response, request)
			assertProblem(t, response, http.StatusUnauthorized, "Authentication required")
		}
	})

	t.Run("test_it_should_reject_invalid_token", func(t *testing.T) {
		assertProblem(t, newClientEnv(t).do(http.MethodGet, vehiclesPath, "", "not-a-jwt"), http.StatusUnauthorized, "Invalid access token")
	})

	t.Run("test_it_should_reject_token_of_inactive_or_missing_customer", func(t *testing.T) {
		env := newClientEnv(t)
		for _, id := range []string{"c-2", "missing"} {
			assertProblem(t, env.do(http.MethodGet, vehiclesPath, "", env.token(t, id, "")), http.StatusUnauthorized, "Invalid access token")
		}
	})

	t.Run("test_it_should_rate_limit_api_calls_per_customer_and_ip", func(t *testing.T) {
		env := newClientEnv(t)
		assertStatus(t, env.do(http.MethodGet, vehiclesPath, "", env.token(t, "c-1", "")), http.StatusOK)
		if len(env.limiter.keys) != 1 || env.limiter.keys[0] != "client:api:c-1:192.0.2.1" {
			t.Errorf("rate limit keys = %v, want client:api:c-1:<ip>", env.limiter.keys)
		}
		env.limiter.deny = true
		response := env.do(http.MethodGet, vehiclesPath, "", env.token(t, "c-1", ""))
		assertProblem(t, response, http.StatusTooManyRequests, "Too many requests")
		assertHandlerStopped(t, response)
	})
}

func TestEnforceRateLimit(t *testing.T) {
	t.Run("test_it_should_return_too_many_requests_with_retry_after_when_denied", func(t *testing.T) {
		env := newClientEnv(t)
		env.limiter.deny = true
		response := env.do(http.MethodPost, loginPath, `{"email":"ana@example.com","password":"x"}`, "")
		assertProblem(t, response, http.StatusTooManyRequests, "Too many requests")
		if got := response.Header().Get(echo.HeaderRetryAfter); got != "60" {
			t.Errorf("Retry-After = %q, want 60", got)
		}
		if env.limiter.windows[0] != time.Minute {
			t.Errorf("window = %v, want 1m", env.limiter.windows[0])
		}
		assertHandlerStopped(t, response)
	})

	t.Run("test_it_should_fail_closed_when_limiter_errors", func(t *testing.T) {
		env := newClientEnv(t)
		env.limiter.err = errBackend
		for _, path := range []string{loginPath, platePath, forgotPath, resetPath} {
			t.Run(path, func(t *testing.T) {
				response := env.do(http.MethodPost, path, `{}`, "")
				assertProblem(t, response, http.StatusServiceUnavailable, "Client access unavailable")
				assertHandlerStopped(t, response)
			})
		}
	})

	t.Run("test_it_should_fail_closed_when_limiter_is_not_configured", func(t *testing.T) {
		service := application.NewCustomerService(newClientCustomerStore(), handlerIDGenerator{})
		e := echo.New()
		NewHandler(service).RegisterClientRoutes(e.Group("/api/v1/customers"))
		request := httptest.NewRequest(http.MethodPost, loginPath, nil)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		assertProblem(t, response, http.StatusServiceUnavailable, "Client access unavailable")
	})
}
