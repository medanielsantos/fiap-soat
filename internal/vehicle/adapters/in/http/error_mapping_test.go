package vehiclehttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

func TestHandleErrorMapsToProblemStatus(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{application.ErrVehicleNotFound, http.StatusNotFound},
		{application.ErrDuplicatePlate, http.StatusConflict},
		{application.ErrCustomerUnavailable, http.StatusUnprocessableEntity},
		{domain.ErrInvalidPlate, http.StatusBadRequest},
		{domain.ErrInvalidCustomerID, http.StatusBadRequest},
		{domain.ErrInvalidBrand, http.StatusBadRequest},
		{domain.ErrInvalidModel, http.StatusBadRequest},
		{domain.ErrInvalidYear, http.StatusBadRequest},
		{fmt.Errorf("check vehicle customer: %w", application.ErrCustomerUnavailable), http.StatusUnprocessableEntity},
		{errors.New("connection refused"), http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.err.Error(), func(t *testing.T) {
			response := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/x", nil), response)
			if err := handleError(c, test.err); err != nil {
				t.Fatal(err)
			}
			var p problem
			if err := json.Unmarshal(response.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.want || p.Status != test.want || p.Instance != "/api/v1/vehicles/x" {
				t.Errorf("status = %d problem = %+v, want %d", response.Code, p, test.want)
			}
			if test.want == http.StatusInternalServerError && p.Detail == test.err.Error() {
				t.Error("internal error detail leaked to client")
			}
		})
	}
}

func TestListFailuresReturnProblems(t *testing.T) {
	if response := serve(newTestEcho(), http.MethodGet, "/api/v1/vehicles?per_page=-1", ""); response.Code != http.StatusBadRequest {
		t.Errorf("invalid pagination status = %d, want 400", response.Code)
	}
	store := failingListStore{&handlerVehicleStore{}, errors.New("db down")}
	if response := serve(newTestEchoWithStore(store), http.MethodGet, "/api/v1/vehicles", ""); response.Code != http.StatusInternalServerError {
		t.Errorf("store failure status = %d, want 500", response.Code)
	}
}

func TestCreateRejectsMalformedBody(t *testing.T) {
	response := serve(newTestEcho(), http.MethodPost, "/api/v1/vehicles", `{"plate":`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	var p problem
	if err := json.Unmarshal(response.Body.Bytes(), &p); err != nil || p.Title != "Invalid request" {
		t.Errorf("problem = %+v, %v", p, err)
	}
}
