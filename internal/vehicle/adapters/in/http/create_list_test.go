package vehiclehttp

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestVehicleHTTPHandler(t *testing.T) {
	const validBody = `{"customer_id":"customer-1","plate":"abc-1d23","brand":"Fiat","model":"Uno","year":2020}`

	t.Run("test_it_should_return_no_content_when_vehicle_list_is_empty", func(t *testing.T) {
		response := serve(newTestEcho(), http.MethodGet, "/api/v1/vehicles", "")
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusNoContent, response.Body.String())
		}
	})

	t.Run("test_it_should_create_vehicle_and_return_versioned_location_when_payload_is_valid", func(t *testing.T) {
		response := serve(newTestEcho(), http.MethodPost, "/api/v1/vehicles", validBody)
		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
		}
		if got, want := response.Header().Get(echo.HeaderLocation), "/api/v1/vehicles/vehicle-1"; got != want {
			t.Errorf("Location = %q, want %q", got, want)
		}
		var vehicle vehicleResponse
		if err := json.Unmarshal(response.Body.Bytes(), &vehicle); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if vehicle.Plate != "ABC1D23" || vehicle.CustomerID != "customer-1" || vehicle.Year != 2020 {
			t.Errorf("vehicle = %+v, want normalized plate for customer-1", vehicle)
		}
	})

	t.Run("test_it_should_return_conflict_when_plate_already_exists", func(t *testing.T) {
		e := newTestEcho()
		if response := serve(e, http.MethodPost, "/api/v1/vehicles", validBody); response.Code != http.StatusCreated {
			t.Fatalf("first create status = %d", response.Code)
		}
		response := serve(e, http.MethodPost, "/api/v1/vehicles", validBody)
		if response.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusConflict, response.Body.String())
		}
	})

	t.Run("test_it_should_return_unprocessable_entity_when_customer_is_unavailable", func(t *testing.T) {
		body := `{"customer_id":"customer-9","plate":"ABC1234","brand":"Fiat","model":"Uno","year":2020}`
		response := serve(newTestEcho(), http.MethodPost, "/api/v1/vehicles", body)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
		}
		if got := response.Header().Get(echo.HeaderContentType); got != "application/problem+json" {
			t.Errorf("Content-Type = %q, want problem+json", got)
		}
	})

	t.Run("test_it_should_return_invalid_plate_detail_when_plate_is_invalid", func(t *testing.T) {
		body := `{"customer_id":"customer-1","plate":"12345","brand":"Fiat","model":"Uno","year":2020}`
		response := serve(newTestEcho(), http.MethodPost, "/api/v1/vehicles", body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
		}
		var problem struct {
			Detail string `json:"detail"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
			t.Fatalf("decode problem details: %v", err)
		}
		if problem.Detail != "invalid plate" {
			t.Errorf("detail = %q, want %q", problem.Detail, "invalid plate")
		}
	})

	t.Run("test_it_should_filter_vehicle_list_by_customer", func(t *testing.T) {
		e := newTestEcho()
		if response := serve(e, http.MethodPost, "/api/v1/vehicles", validBody); response.Code != http.StatusCreated {
			t.Fatalf("create status = %d", response.Code)
		}
		if response := serve(e, http.MethodGet, "/api/v1/vehicles?customer_id=customer-2", ""); response.Code != http.StatusNoContent {
			t.Errorf("other customer status = %d, want %d", response.Code, http.StatusNoContent)
		}
		response := serve(e, http.MethodGet, "/api/v1/vehicles?customer_id=customer-1", "")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
		}
		var page pagination.Response[vehicleResponse]
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(page.Data) != 1 || page.Data[0].ID != "vehicle-1" {
			t.Errorf("data = %+v, want vehicle-1", page.Data)
		}
	})
}
