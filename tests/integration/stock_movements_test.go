package integration_test

import (
	"net/http"
	"testing"
)

func TestStockMovementHistoryAndDeleteGuard(t *testing.T) {
	server := newTestServer(t)
	partID := createPart(t, server, "HIST-1", 5)
	call(t, server, http.MethodPost, "/api/v1/parts/"+partID+"/stock-movements", map[string]any{"type": "entry", "quantity": 3, "created_by": testUserID}, http.StatusOK)
	call(t, server, http.MethodPost, "/api/v1/parts/"+partID+"/stock-movements", map[string]any{"type": "exit", "quantity": 2, "created_by": testUserID}, http.StatusOK)
	assertStock(t, server, partID, 6, 0)
	orderID := prepareOrder(t, server, partID, 2)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/quotes/approve", map[string]any{"client_id": testClientID}, http.StatusOK)
	call(t, server, http.MethodDelete, "/api/v1/parts/"+partID, nil, http.StatusConflict)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/consumptions", map[string]any{"part_id": partID, "quantity": 1}, http.StatusOK)
	history := call(t, server, http.MethodGet, "/api/v1/parts/"+partID+"/stock-movements?page=1&per_page=20", nil, http.StatusOK)
	data := history["data"].([]any)
	if len(data) != 5 {
		t.Fatalf("movement history length = %d, want 5", len(data))
	}
	types := map[string]bool{}
	for _, raw := range data {
		types[raw.(map[string]any)["type"].(string)] = true
	}
	for _, kind := range []string{"entry", "exit", "reservation", "consumption"} {
		if !types[kind] {
			t.Errorf("movement history is missing type %q", kind)
		}
	}
}
