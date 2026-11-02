package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

func newTestServer(t *testing.T, address string) *Server {
	t.Helper()
	return New(config.Config{HTTPAddress: address}, newTestRouter(t, NewEngine()))
}

func serve(server *Server, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	server.echo.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}

func TestNewRegistersRoutesAndKeepsAddress(t *testing.T) {
	server := newTestServer(t, ":9999")
	if server.address != ":9999" {
		t.Fatalf("address = %q, want :9999", server.address)
	}
	response := serve(server, http.MethodGet, "/health", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("GET /health = %d %s", response.Code, response.Body.String())
	}
}

func TestNewRecoversFromHandlerPanics(t *testing.T) {
	server := newTestServer(t, ":0")
	server.echo.GET("/panic", func(*echo.Context) error { panic("boom") })

	if response := serve(server, http.MethodGet, "/panic", ""); response.Code != http.StatusInternalServerError {
		t.Fatalf("panic route status = %d, want 500", response.Code)
	}
}

func TestNewLimitsRequestBodySize(t *testing.T) {
	server := newTestServer(t, ":0")
	oversized := strings.Repeat("x", 1<<20+1)

	if response := serve(server, http.MethodPost, "/api/v1/auth/login", oversized); response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d, want 413", response.Code)
	}
}
