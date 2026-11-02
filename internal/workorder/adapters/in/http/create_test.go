package workorderhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

const createBody = `{"customer_id":"customer-1","vehicle_id":"vehicle-1"}`

func TestCreateWorkOrder(t *testing.T) {
	t.Run("test_it_should_create_order_with_location_and_authenticated_user", func(t *testing.T) {
		s := newTestServer()
		response := s.do(http.MethodPost, "/api/v1/work-orders", createBody)
		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body: %s", response.Code, response.Body.String())
		}
		if got := response.Header().Get(echo.HeaderLocation); got != "/api/v1/work-orders/id-1" {
			t.Errorf("Location = %q", got)
		}
		var order domain.WorkOrder
		if err := json.Unmarshal(response.Body.Bytes(), &order); err != nil {
			t.Fatal(err)
		}
		if order.Status != domain.Received || order.CreatedBy != "user-1" {
			t.Errorf("order = %+v, want received created by user-1", order)
		}
	})

	t.Run("test_it_should_return_unprocessable_entity_when_vehicle_is_not_owned", func(t *testing.T) {
		response := newTestServer().do(http.MethodPost, "/api/v1/work-orders", `{"customer_id":"customer-2","vehicle_id":"vehicle-1"}`)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", response.Code)
		}
		if p := decodeProblem(t, response); p.Title != "Related resource unavailable" || p.Instance != "/api/v1/work-orders" {
			t.Errorf("problem = %+v", p)
		}
	})

	for name, body := range map[string]string{
		"test_it_should_reject_malformed_json":     `{"customer_id":`,
		"test_it_should_reject_unknown_fields":     `{"customer_id":"customer-1","extra":1}`,
		"test_it_should_reject_multiple_documents": createBody + createBody,
	} {
		t.Run(name, func(t *testing.T) {
			response := newTestServer().do(http.MethodPost, "/api/v1/work-orders", body)
			if response.Code != http.StatusBadRequest || response.Header().Get(echo.HeaderContentType) != "application/problem+json" {
				t.Fatalf("status = %d content-type = %q, want 400 problem", response.Code, response.Header().Get(echo.HeaderContentType))
			}
		})
	}
}

func TestGetAndListWorkOrders(t *testing.T) {
	t.Run("test_it_should_get_existing_order_and_404_missing", func(t *testing.T) {
		s := newTestServer()
		s.mustDo(t, http.MethodPost, "/api/v1/work-orders", createBody, http.StatusCreated)
		if order := s.mustDo(t, http.MethodGet, "/api/v1/work-orders/id-1", "", http.StatusOK); order.ID != "id-1" {
			t.Errorf("order id = %q", order.ID)
		}
		response := s.do(http.MethodGet, "/api/v1/work-orders/missing", "")
		if p := decodeProblem(t, response); response.Code != http.StatusNotFound || p.Title != "Work order not found" {
			t.Errorf("status = %d problem = %+v", response.Code, p)
		}
	})

	t.Run("test_it_should_list_by_status_and_return_no_content_when_empty", func(t *testing.T) {
		s := newTestServer()
		s.mustDo(t, http.MethodGet, "/api/v1/work-orders", "", http.StatusNoContent)
		s.mustDo(t, http.MethodPost, "/api/v1/work-orders", createBody, http.StatusCreated)
		s.mustDo(t, http.MethodGet, "/api/v1/work-orders?status=delivered", "", http.StatusNoContent)

		response := s.do(http.MethodGet, "/api/v1/work-orders?status=received", "")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", response.Code)
		}
		var page pagination.Response[domain.WorkOrder]
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Data) != 1 || page.Data[0].ID != "id-1" {
			t.Errorf("data = %+v, want id-1", page.Data)
		}
	})

	t.Run("test_it_should_reject_invalid_pagination", func(t *testing.T) {
		newTestServer().mustDo(t, http.MethodGet, "/api/v1/work-orders?page=0", "", http.StatusBadRequest)
	})

	t.Run("test_it_should_hide_store_failure_behind_internal_error", func(t *testing.T) {
		s := newTestServer()
		s.store.err = errors.New("connection reset")
		response := s.do(http.MethodGet, "/api/v1/work-orders", "")
		if p := decodeProblem(t, response); response.Code != http.StatusInternalServerError || p.Detail != "The request could not be completed." {
			t.Errorf("status = %d problem = %+v", response.Code, p)
		}
	})
}
