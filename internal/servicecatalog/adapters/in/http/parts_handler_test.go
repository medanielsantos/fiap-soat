package servicecataloghttp

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

const validPartBody = `{"sku":"SKU-1","internal_code":"P-1","name":"Oil filter","unit":"un","purchase_price_in_cents":500,"sale_price_in_cents":1200,"initial_stock":4}`

func TestPartHandlers(t *testing.T) {
	t.Run("test_it_should_create_part_with_initial_stock", func(t *testing.T) {
		response := serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/parts", validPartBody)
		expectStatus(t, response, http.StatusCreated)
		if got := response.Header().Get(echo.HeaderLocation); got != "/api/v1/parts/id-1" {
			t.Errorf("Location = %q", got)
		}
		dto := decodeBody[partDTO](t, response)
		if dto.Stock != 4 || dto.AvailableStock != 4 || dto.ReservedStock != 0 || dto.PerformedBy != "user-1" || !dto.Active {
			t.Errorf("part = %+v", dto)
		}
	})

	t.Run("test_it_should_reject_invalid_part", func(t *testing.T) {
		body := `{"sku":"","internal_code":"P-1","name":"x","unit":"un","sale_price_in_cents":1}`
		expectStatus(t, serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/parts", body), http.StatusBadRequest)
	})

	t.Run("test_it_should_reject_malformed_body", func(t *testing.T) {
		expectStatus(t, serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/parts", `[]`), http.StatusBadRequest)
	})

	t.Run("test_it_should_list_parts_and_return_no_content_when_empty", func(t *testing.T) {
		store := newHandlerCatalogStore()
		e := newTestEcho(store)
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/parts", ""), http.StatusNoContent)
		seedPart(t, store, "p1", 500, 1200, 3)
		response := serve(e, http.MethodGet, "/api/v1/parts", "")
		expectStatus(t, response, http.StatusOK)
		page := decodeBody[pagination.Response[partDTO]](t, response)
		if len(page.Data) != 1 || page.Data[0].SKU != "SKU-p1" || page.Data[0].Stock != 3 {
			t.Errorf("page = %+v", page)
		}
	})

	t.Run("test_it_should_map_list_errors", func(t *testing.T) {
		store := newHandlerCatalogStore()
		e := newTestEcho(store)
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/parts?per_page=500", ""), http.StatusBadRequest)
		store.err = errors.New("db down")
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/parts", ""), http.StatusInternalServerError)
	})

	t.Run("test_it_should_get_part_or_return_not_found", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 0)
		e := newTestEcho(store)
		response := serve(e, http.MethodGet, "/api/v1/parts/p1", "")
		expectStatus(t, response, http.StatusOK)
		if dto := decodeBody[partDTO](t, response); dto.ID != "p1" || dto.Unit != "un" {
			t.Errorf("part = %+v", dto)
		}
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/parts/missing", ""), http.StatusNotFound)
	})

	t.Run("test_it_should_update_part_keeping_stock", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 6)
		body := `{"sku":"SKU-1","internal_code":"P-1","name":"Air filter","unit":"un","purchase_price_in_cents":600,"sale_price_in_cents":1500,"active":false}`
		response := serve(newTestEcho(store, adminPermissions...), http.MethodPut, "/api/v1/parts/p1", body)
		expectStatus(t, response, http.StatusOK)
		dto := decodeBody[partDTO](t, response)
		if dto.Name != "Air filter" || dto.Stock != 6 || dto.SalePriceInCents != 1500 || dto.Active {
			t.Errorf("part = %+v", dto)
		}
	})

	t.Run("test_it_should_map_update_part_errors", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 0)
		e := newTestEcho(store, adminPermissions...)
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/parts/missing", validPartBody), http.StatusNotFound)
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/parts/p1", `{"bogus":1}`), http.StatusBadRequest)
		body := `{"sku":"SKU-1","internal_code":"P-1","name":"x","unit":"un","purchase_price_in_cents":-1,"sale_price_in_cents":1200}`
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/parts/p1", body), http.StatusBadRequest)
	})

	t.Run("test_it_should_refuse_deleting_part_with_stock", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedPart(t, store, "p1", 500, 1200, 2)
		seedPart(t, store, "p2", 500, 1200, 0)
		e := newTestEcho(store, adminPermissions...)
		response := serve(e, http.MethodDelete, "/api/v1/parts/p1", "")
		expectStatus(t, response, http.StatusConflict)
		if p := decodeBody[problem](t, response); p.Title != "Part still has stock" {
			t.Errorf("problem = %+v", p)
		}
		expectStatus(t, serve(e, http.MethodDelete, "/api/v1/parts/p2", ""), http.StatusNoContent)
		expectStatus(t, serve(e, http.MethodDelete, "/api/v1/parts/p2", ""), http.StatusNotFound)
	})
}
