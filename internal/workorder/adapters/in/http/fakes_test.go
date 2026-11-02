package workorderhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

type memoryStore struct {
	orders map[string]domain.WorkOrder
	err    error
}

func (s *memoryStore) Create(_ context.Context, o domain.WorkOrder, _ domain.HistoryEntry) error {
	s.orders[o.ID] = o
	return s.err
}

func (s *memoryStore) Find(_ context.Context, id string) (domain.WorkOrder, error) {
	if s.err != nil {
		return domain.WorkOrder{}, s.err
	}
	o, ok := s.orders[id]
	if !ok {
		return domain.WorkOrder{}, application.ErrNotFound
	}
	o.Items = append([]domain.Item{}, o.Items...)
	o.Quotes = append([]domain.Quote{}, o.Quotes...)
	return o, nil
}

func (s *memoryStore) List(_ context.Context, status string, p pagination.Params) (pagination.Page[domain.WorkOrder], error) {
	var out []domain.WorkOrder
	for _, o := range s.orders {
		if status == "" || string(o.Status) == status {
			out = append(out, o)
		}
	}
	return pagination.NewPage(out, p, int64(len(out))), s.err
}

func (s *memoryStore) Save(_ context.Context, o domain.WorkOrder, _ domain.HistoryEntry) error {
	s.orders[o.ID] = o
	return nil
}

type directTx struct{}

func (directTx) Within(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type counterIDs struct{ n int }

func (g *counterIDs) NewID(context.Context) (string, error) {
	g.n++
	return fmt.Sprintf("id-%d", g.n), nil
}

type ownership struct{}

func (ownership) Validate(_ context.Context, customerID, _ string) error {
	if customerID != "customer-1" {
		return application.ErrCustomerOrVehicleUnavailable
	}
	return nil
}

type catalog struct{}

func (catalog) Service(_ context.Context, id string) (application.CatalogItem, error) {
	return application.CatalogItem{ID: id, Code: "SVC", Name: "Alignment", SalePriceInCents: 5000, EstimatedDurationMinutes: 30, Active: true}, nil
}

func (catalog) Part(_ context.Context, id string) (application.CatalogItem, error) {
	if id != "part-1" {
		return application.CatalogItem{}, application.ErrCatalogItemUnavailable
	}
	return application.CatalogItem{ID: id, Code: "PAD", Name: "Brake pad", Unit: "pair", SalePriceInCents: 2500, Active: true}, nil
}

type inventory struct{ err error }

func (i *inventory) Reserve(context.Context, string, string, []application.StockRequirement) error {
	return i.err
}

func (i *inventory) Consume(context.Context, string, string, []application.StockRequirement) error {
	return i.err
}

type testServer struct {
	echo      *echo.Echo
	store     *memoryStore
	inventory *inventory
}

func newTestServer() *testServer {
	store := &memoryStore{orders: map[string]domain.WorkOrder{}}
	inv := &inventory{}
	service := application.NewService(store, directTx{}, &counterIDs{}, ownership{}, catalog{}, inv)
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "user-1")
			return next(c)
		}
	})
	NewHandler(service).RegisterRoutes(e.Group("/api/v1/work-orders"))
	return &testServer{echo: e, store: store, inventory: inv}
}

func (s *testServer) do(method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()
	s.echo.ServeHTTP(response, request)
	return response
}

// mustDo runs a request and fails the test when the status differs from want.
func (s *testServer) mustDo(t *testing.T, method, target, body string, want int) domain.WorkOrder {
	t.Helper()
	response := s.do(method, target, body)
	if response.Code != want {
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, target, response.Code, want, response.Body.String())
	}
	var order domain.WorkOrder
	_ = json.Unmarshal(response.Body.Bytes(), &order)
	return order
}

func decodeProblem(t *testing.T, response *httptest.ResponseRecorder) problem {
	t.Helper()
	var p problem
	if err := json.Unmarshal(response.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v; body: %s", err, response.Body.String())
	}
	return p
}
