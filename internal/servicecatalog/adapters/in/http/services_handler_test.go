package servicecataloghttp

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

const validServiceBody = `{"internal_code":"S-1","name":"Alignment","sale_price_in_cents":5000,"estimated_duration_minutes":30}`

func TestServiceHandlers(t *testing.T) {
	t.Run("test_it_should_create_service_with_location_and_authenticated_author", func(t *testing.T) {
		response := serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/services", validServiceBody)
		expectStatus(t, response, http.StatusCreated)
		if got := response.Header().Get(echo.HeaderLocation); got != "/api/v1/services/id-1" {
			t.Errorf("Location = %q", got)
		}
		dto := decodeBody[serviceDTO](t, response)
		if dto.ID != "id-1" || dto.PerformedBy != "user-1" || !dto.Active || dto.SalePriceInCents != 5000 {
			t.Errorf("service = %+v", dto)
		}
	})

	t.Run("test_it_should_honor_explicit_inactive_flag", func(t *testing.T) {
		body := `{"internal_code":"S-1","name":"A","sale_price_in_cents":1,"estimated_duration_minutes":1,"active":false}`
		response := serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/services", body)
		expectStatus(t, response, http.StatusCreated)
		if decodeBody[serviceDTO](t, response).Active {
			t.Error("active = true, want false")
		}
	})

	t.Run("test_it_should_return_conflict_when_internal_code_exists", func(t *testing.T) {
		e := newTestEcho(newHandlerCatalogStore(), adminPermissions...)
		serve(e, http.MethodPost, "/api/v1/services", validServiceBody)
		expectStatus(t, serve(e, http.MethodPost, "/api/v1/services", validServiceBody), http.StatusConflict)
	})

	t.Run("test_it_should_return_bad_request_when_service_is_invalid", func(t *testing.T) {
		body := `{"internal_code":"S-1","name":"","sale_price_in_cents":1,"estimated_duration_minutes":1}`
		response := serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/services", body)
		expectStatus(t, response, http.StatusBadRequest)
		if p := decodeBody[problem](t, response); p.Title != "Invalid catalog item" || p.Instance != "/api/v1/services" {
			t.Errorf("problem = %+v", p)
		}
		expectStatus(t, serve(newTestEcho(newHandlerCatalogStore(), adminPermissions...), http.MethodPost, "/api/v1/services", `{"unknown":1}`), http.StatusBadRequest)
	})

	t.Run("test_it_should_list_services_and_return_no_content_when_empty", func(t *testing.T) {
		store := newHandlerCatalogStore()
		e := newTestEcho(store)
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/services", ""), http.StatusNoContent)
		seedService(t, store, "s1", 5000)
		response := serve(e, http.MethodGet, "/api/v1/services?page=1&per_page=10", "")
		expectStatus(t, response, http.StatusOK)
		page := decodeBody[pagination.Response[serviceDTO]](t, response)
		if len(page.Data) != 1 || page.Data[0].ID != "s1" || page.Pagination.TotalItems != 1 {
			t.Errorf("page = %+v", page)
		}
	})

	t.Run("test_it_should_reject_invalid_pagination", func(t *testing.T) {
		expectStatus(t, serve(newTestEcho(newHandlerCatalogStore()), http.MethodGet, "/api/v1/services?page=0", ""), http.StatusBadRequest)
	})

	t.Run("test_it_should_return_internal_error_when_listing_fails", func(t *testing.T) {
		store := newHandlerCatalogStore()
		store.err = errors.New("db down")
		expectStatus(t, serve(newTestEcho(store), http.MethodGet, "/api/v1/services", ""), http.StatusInternalServerError)
	})

	t.Run("test_it_should_get_service_or_return_not_found", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedService(t, store, "s1", 5000)
		e := newTestEcho(store)
		response := serve(e, http.MethodGet, "/api/v1/services/s1", "")
		expectStatus(t, response, http.StatusOK)
		if dto := decodeBody[serviceDTO](t, response); dto.Name != "Alignment" {
			t.Errorf("service = %+v", dto)
		}
		expectStatus(t, serve(e, http.MethodGet, "/api/v1/services/missing", ""), http.StatusNotFound)
	})

	t.Run("test_it_should_update_service_when_user_can_change_prices", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedService(t, store, "s1", 5000)
		body := `{"internal_code":"S-1","name":"Balancing","sale_price_in_cents":7000,"estimated_duration_minutes":45}`
		response := serve(newTestEcho(store, adminPermissions...), http.MethodPut, "/api/v1/services/s1", body)
		expectStatus(t, response, http.StatusOK)
		if dto := decodeBody[serviceDTO](t, response); dto.Name != "Balancing" || dto.SalePriceInCents != 7000 || dto.PerformedBy != "user-1" {
			t.Errorf("service = %+v", dto)
		}
	})

	t.Run("test_it_should_map_update_errors", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedService(t, store, "s1", 5000)
		e := newTestEcho(store, adminPermissions...)
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/services/missing", validServiceBody), http.StatusNotFound)
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/services/s1", `{"name":`), http.StatusBadRequest)
	})

	t.Run("test_it_should_delete_service_once", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedService(t, store, "s1", 5000)
		e := newTestEcho(store, adminPermissions...)
		expectStatus(t, serve(e, http.MethodDelete, "/api/v1/services/s1", ""), http.StatusNoContent)
		expectStatus(t, serve(e, http.MethodDelete, "/api/v1/services/s1", ""), http.StatusNotFound)
	})
}
