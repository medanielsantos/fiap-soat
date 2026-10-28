package customerhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

type handlerCustomerStore struct {
	customers map[string]domain.Customer
}

func (store *handlerCustomerStore) Save(_ context.Context, customer domain.Customer) error {
	store.customers[customer.ID()] = customer
	return nil
}

func (store *handlerCustomerStore) FindByID(_ context.Context, id string) (domain.Customer, error) {
	customer, ok := store.customers[id]
	if !ok {
		return domain.Customer{}, application.ErrCustomerNotFound
	}
	return customer, nil
}

func (store *handlerCustomerStore) List(context.Context) ([]domain.Customer, error) {
	customers := make([]domain.Customer, 0, len(store.customers))
	for _, customer := range store.customers {
		customers = append(customers, customer)
	}
	return customers, nil
}

func (store *handlerCustomerStore) Update(_ context.Context, customer domain.Customer) error {
	if _, ok := store.customers[customer.ID()]; !ok {
		return application.ErrCustomerNotFound
	}
	store.customers[customer.ID()] = customer
	return nil
}

func (store *handlerCustomerStore) Delete(_ context.Context, id string) error {
	if _, ok := store.customers[id]; !ok {
		return application.ErrCustomerNotFound
	}
	delete(store.customers, id)
	return nil
}

type handlerIDGenerator struct{}

func (handlerIDGenerator) NewID(context.Context) (string, error) { return "customer-1", nil }

func newTestEcho() *echo.Echo {
	store := &handlerCustomerStore{customers: make(map[string]domain.Customer)}
	service := application.NewCustomerService(store, handlerIDGenerator{})
	e := echo.New()
	NewHandler(service).RegisterRoutes(e.Group("/api/v1").Group("/customers"))
	return e
}

func TestCustomerHTTPHandler(t *testing.T) {
	t.Run("test_it_should_return_no_content_when_customer_list_is_empty", func(t *testing.T) {
		e := newTestEcho()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/customers", nil)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusNoContent, response.Body.String())
		}
		if response.Body.Len() != 0 {
			t.Errorf("204 response body = %q, want empty", response.Body.String())
		}
	})

	t.Run("test_it_should_create_customer_and_return_versioned_location_when_payload_is_valid", func(t *testing.T) {
		e := newTestEcho()
		body := `{"document":"529.982.247-25","name":"Ana Silva","email":"ana@example.com","phone":"+5511999999999"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(body))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
		}
		if got, want := response.Header().Get(echo.HeaderLocation), "/api/v1/customers/customer-1"; got != want {
			t.Errorf("Location = %q, want %q", got, want)
		}
		var customer customerResponse
		if err := json.Unmarshal(response.Body.Bytes(), &customer); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if customer.ID != "customer-1" {
			t.Errorf("id = %q, want %q", customer.ID, "customer-1")
		}
		if customer.Document != "***.***.***-25" {
			t.Errorf("document = %q, want masked CPF", customer.Document)
		}
	})

	t.Run("test_it_should_return_invalid_document_detail_when_document_is_invalid", func(t *testing.T) {
		e := newTestEcho()
		body := `{"document":"111.111.111-11","name":"Ana Silva","email":"ana@example.com","phone":"123"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(body))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
		}
		var problem struct {
			Detail string `json:"detail"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
			t.Fatalf("decode problem details: %v", err)
		}
		if problem.Detail != "invalid document" {
			t.Errorf("detail = %q, want %q", problem.Detail, "invalid document")
		}
	})
}
