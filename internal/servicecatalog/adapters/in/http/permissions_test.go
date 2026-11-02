package servicecataloghttp

import (
	"errors"
	"net/http"
	"testing"

	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
)

var attendantPermissions = []identityapp.Permission{identityapp.PermissionProductsCreate}

func TestCatalogPermissions(t *testing.T) {
	t.Run("test_it_should_deny_create_and_delete_without_permission", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedService(t, store, "s1", 5000)
		seedPart(t, store, "p1", 500, 1200, 0)
		e := newTestEcho(store)
		expectStatus(t, serve(e, http.MethodPost, "/api/v1/services", validServiceBody), http.StatusForbidden)
		expectStatus(t, serve(e, http.MethodPost, "/api/v1/parts", validPartBody), http.StatusForbidden)
		expectStatus(t, serve(e, http.MethodDelete, "/api/v1/services/s1", ""), http.StatusForbidden)
		expectStatus(t, serve(e, http.MethodDelete, "/api/v1/parts/p1", ""), http.StatusForbidden)
		if len(store.services) != 1 || len(store.parts) != 1 {
			t.Error("denied requests must not change the catalog")
		}
	})

	t.Run("test_it_should_forbid_attendant_changing_service_price", func(t *testing.T) {
		store := newHandlerCatalogStore()
		seedService(t, store, "s1", 5000)
		e := newTestEcho(store, attendantPermissions...)
		body := `{"internal_code":"S-1","name":"Alignment","sale_price_in_cents":9999,"estimated_duration_minutes":30}`
		response := serve(e, http.MethodPut, "/api/v1/services/s1", body)
		expectStatus(t, response, http.StatusForbidden)
		if p := decodeBody[problem](t, response); p.Detail != "Attendants cannot change prices." {
			t.Errorf("problem = %+v", p)
		}
		if store.services["s1"].SalePriceInCents() != 5000 {
			t.Error("price changed despite denial")
		}
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/services/s1", validServiceBody), http.StatusOK)
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/services/missing", validServiceBody), http.StatusNotFound)
	})

	partBody := func(purchase, sale string) string {
		return `{"sku":"SKU-1","internal_code":"P-1","name":"Oil filter","unit":"un","purchase_price_in_cents":` + purchase + `,"sale_price_in_cents":` + sale + `}`
	}
	tests := []struct {
		name           string
		purchase, sale string
		want           int
	}{
		{"same_prices", "500", "1200", http.StatusOK},
		{"purchase_price_changed", "501", "1200", http.StatusForbidden},
		{"sale_price_changed", "500", "1300", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run("test_it_should_guard_attendant_part_update_when_"+tt.name, func(t *testing.T) {
			store := newHandlerCatalogStore()
			seedPart(t, store, "p1", 500, 1200, 0)
			expectStatus(t, serve(newTestEcho(store, attendantPermissions...), http.MethodPut, "/api/v1/parts/p1", partBody(tt.purchase, tt.sale)), tt.want)
		})
	}

	t.Run("test_it_should_map_lookup_failure_during_attendant_part_update", func(t *testing.T) {
		store := newHandlerCatalogStore()
		e := newTestEcho(store, attendantPermissions...)
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/parts/missing", partBody("500", "1200")), http.StatusNotFound)
		store.err = errors.New("db down")
		expectStatus(t, serve(e, http.MethodPut, "/api/v1/parts/p1", partBody("500", "1200")), http.StatusInternalServerError)
	})
}
