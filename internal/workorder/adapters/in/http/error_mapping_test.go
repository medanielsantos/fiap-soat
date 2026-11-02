package workorderhttp

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestFailMapsErrorsToProblemStatus(t *testing.T) {
	tests := []struct {
		err   error
		want  int
		title string
	}{
		{application.ErrNotFound, http.StatusNotFound, "Work order not found"},
		{application.ErrCustomerOrVehicleUnavailable, http.StatusUnprocessableEntity, "Related resource unavailable"},
		{application.ErrCatalogItemUnavailable, http.StatusUnprocessableEntity, "Related resource unavailable"},
		{domain.ErrInvalidTransition, http.StatusConflict, "Work order state conflict"},
		{domain.ErrQuoteUnavailable, http.StatusConflict, "Work order state conflict"},
		{domain.ErrInsufficientConsumption, http.StatusConflict, "Work order state conflict"},
		{domain.ErrItemsNotConsumed, http.StatusConflict, "Work order state conflict"},
		{domain.ErrQuoteEmpty, http.StatusConflict, "Work order state conflict"},
		{domain.ErrInvalidItem, http.StatusConflict, "Work order state conflict"},
		{domain.ErrInvalidDiagnosis, http.StatusBadRequest, "Invalid work order"},
		{domain.ErrInvalidPrincipal, http.StatusBadRequest, "Invalid work order"},
		{domain.ErrInvalidQuantity, http.StatusBadRequest, "Invalid work order"},
		{domain.ErrInvalidOrder, http.StatusBadRequest, "Invalid work order"},
		{domain.ErrAmountOverflow, http.StatusBadRequest, "Invalid work order"},
		{application.ErrInsufficientStock, http.StatusConflict, "Insufficient stock"},
		{fmt.Errorf("wrapped: %w", application.ErrNotFound), http.StatusNotFound, "Work order not found"},
		{errors.New("unexpected"), http.StatusInternalServerError, "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.err.Error(), func(t *testing.T) {
			response := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/work-orders/x", nil), response)
			if err := fail(c, test.err); err != nil {
				t.Fatal(err)
			}
			p := decodeProblem(t, response)
			if response.Code != test.want || p.Status != test.want || p.Title != test.title || p.Type != "about:blank" {
				t.Errorf("status = %d problem = %+v, want %d %q", response.Code, p, test.want, test.title)
			}
			if test.want == http.StatusInternalServerError && p.Detail == test.err.Error() {
				t.Error("internal error detail must not leak")
			}
		})
	}
}
