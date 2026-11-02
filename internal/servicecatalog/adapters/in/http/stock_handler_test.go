package servicecataloghttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
)

func TestStockHandlers(t *testing.T) {
	t.Run("test_it_should_apply_entry_and_exit_movements", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 2)
		e := newTestEcho(store)
		response := serve(e, http.MethodPost, "/api/v1/parts/p1/stock-movements", `{"type":"entry","quantity":5,"description":"purchase"}`)
		expectStatus(t, response, http.StatusOK)
		if dto := decodeBody[partDTO](t, response); dto.Stock != 7 || dto.PerformedBy != "user-1" {
			t.Errorf("after entry part = %+v", dto)
		}
		response = serve(e, http.MethodPost, "/api/v1/parts/p1/stock-movements", `{"type":"exit","quantity":3}`)
		expectStatus(t, response, http.StatusOK)
		if dto := decodeBody[partDTO](t, response); dto.Stock != 4 || dto.AvailableStock != 4 {
			t.Errorf("after exit part = %+v", dto)
		}
	})

	tests := []struct {
		name  string
		part  string
		body  string
		want  int
		title string
	}{
		{"insufficient_stock", "p1", `{"type":"exit","quantity":99}`, http.StatusConflict, "Insufficient stock"},
		{"zero_quantity", "p1", `{"type":"entry","quantity":0}`, http.StatusBadRequest, "Invalid catalog item"},
		{"unknown_type", "p1", `{"type":"transfer","quantity":1}`, http.StatusBadRequest, "Invalid catalog item"},
		{"missing_part", "missing", `{"type":"entry","quantity":1}`, http.StatusNotFound, "Catalog item not found"},
		{"trailing_payload", "p1", `{"type":"entry","quantity":1}{}`, http.StatusBadRequest, "Invalid request"},
	}
	for _, tt := range tests {
		t.Run("test_it_should_map_"+tt.name, func(t *testing.T) {
			store := newHandlerCatalogStore()
			seedPart(t, store, "p1", 500, 1200, 2)
			response := serve(newTestEcho(store), http.MethodPost, "/api/v1/parts/"+tt.part+"/stock-movements", tt.body)
			expectStatus(t, response, tt.want)
			if p := decodeBody[problem](t, response); p.Title != tt.title || p.Status != tt.want {
				t.Errorf("problem = %+v, want title %q", p, tt.title)
			}
		})
	}
}

func TestListStockMovementsHandler(t *testing.T) {
	t.Run("test_it_should_return_no_content_when_part_has_no_movements", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 0)
		expectStatus(t, serve(newTestEcho(store), http.MethodGet, "/api/v1/parts/p1/stock-movements", ""), http.StatusNoContent)
	})

	t.Run("test_it_should_list_movements_with_work_order_reference", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 1)
		order := "wo-1"
		store.movements = []application.StockMovement{
			{ID: "m2", PartID: "p1", Type: "reservation", Quantity: 1, PerformedBy: "c1", WorkOrderID: &order, CreatedAt: time.Now()},
			{ID: "m1", PartID: "p1", Type: "entry", Quantity: 1, Description: "initial stock", PerformedBy: "admin"},
		}
		response := serve(newTestEcho(store), http.MethodGet, "/api/v1/parts/p1/stock-movements", "")
		expectStatus(t, response, http.StatusOK)
		page := decodeBody[pagination.Response[movementDTO]](t, response)
		if len(page.Data) != 2 || page.Data[0].WorkOrderID == nil || *page.Data[0].WorkOrderID != "wo-1" || page.Data[1].WorkOrderID != nil {
			t.Errorf("movements = %+v", page.Data)
		}
		if page.Data[1].CreatedBy != "admin" || page.Data[1].Type != "entry" {
			t.Errorf("movement = %+v", page.Data[1])
		}
	})

	t.Run("test_it_should_map_errors", func(t *testing.T) {
		e := newTestEcho(newHandlerCatalogStore())
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/parts/p1/stock-movements?page=x", ""), http.StatusBadRequest)
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/parts/missing/stock-movements", ""), http.StatusNotFound)
	})
}
