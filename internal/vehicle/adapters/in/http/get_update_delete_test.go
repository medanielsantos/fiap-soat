package vehiclehttp

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
)

const createVehicleBody = `{"customer_id":"customer-1","plate":"ABC1D23","brand":"Fiat","model":"Uno","year":2020}`

func seededEcho(t *testing.T) *echo.Echo {
	t.Helper()
	e := newTestEcho()
	if response := serve(e, http.MethodPost, "/api/v1/vehicles", createVehicleBody); response.Code != http.StatusCreated {
		t.Fatalf("seed status = %d; body: %s", response.Code, response.Body.String())
	}
	return e
}

func TestGetVehicle(t *testing.T) {
	e := seededEcho(t)
	response := serve(e, http.MethodGet, "/api/v1/vehicles/vehicle-1", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var vehicle vehicleResponse
	if err := json.Unmarshal(response.Body.Bytes(), &vehicle); err != nil {
		t.Fatal(err)
	}
	if vehicle.ID != "vehicle-1" || vehicle.Brand != "Fiat" || vehicle.Model != "Uno" {
		t.Errorf("vehicle = %+v", vehicle)
	}
	if response := serve(e, http.MethodGet, "/api/v1/vehicles/missing", ""); response.Code != http.StatusNotFound {
		t.Errorf("missing status = %d, want 404", response.Code)
	}
}

func TestUpdateVehicle(t *testing.T) {
	tests := []struct {
		name, target, body string
		want               int
	}{
		{name: "test_it_should_update_vehicle", target: "/api/v1/vehicles/vehicle-1", body: `{"customer_id":"customer-1","plate":"ABC1D23","brand":"Fiat","model":"Mobi","year":2022}`, want: http.StatusOK},
		{name: "test_it_should_return_not_found_for_missing_vehicle", target: "/api/v1/vehicles/missing", body: createVehicleBody, want: http.StatusNotFound},
		{name: "test_it_should_reject_unknown_fields", target: "/api/v1/vehicles/vehicle-1", body: `{"color":"red"}`, want: http.StatusBadRequest},
		{name: "test_it_should_reject_trailing_documents", target: "/api/v1/vehicles/vehicle-1", body: createVehicleBody + `{}`, want: http.StatusBadRequest},
		{name: "test_it_should_reject_invalid_year", target: "/api/v1/vehicles/vehicle-1", body: `{"customer_id":"customer-1","plate":"ABC1D23","brand":"Fiat","model":"Uno","year":1800}`, want: http.StatusBadRequest},
		{name: "test_it_should_reject_inactive_customer", target: "/api/v1/vehicles/vehicle-1", body: `{"customer_id":"customer-9","plate":"ABC1D23","brand":"Fiat","model":"Uno","year":2020}`, want: http.StatusUnprocessableEntity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serve(seededEcho(t), http.MethodPut, test.target, test.body)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.want, response.Body.String())
			}
			if test.want != http.StatusOK {
				return
			}
			var vehicle vehicleResponse
			if err := json.Unmarshal(response.Body.Bytes(), &vehicle); err != nil {
				t.Fatal(err)
			}
			if vehicle.Model != "Mobi" || vehicle.Year != 2022 {
				t.Errorf("vehicle = %+v, want updated model/year", vehicle)
			}
		})
	}
}

func TestDeleteVehicle(t *testing.T) {
	e := seededEcho(t)
	if response := serve(e, http.MethodDelete, "/api/v1/vehicles/vehicle-1", ""); response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if response := serve(e, http.MethodGet, "/api/v1/vehicles/vehicle-1", ""); response.Code != http.StatusNotFound {
		t.Errorf("get after delete status = %d, want 404", response.Code)
	}
	if response := serve(e, http.MethodDelete, "/api/v1/vehicles/vehicle-1", ""); response.Code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404", response.Code)
	}
}
