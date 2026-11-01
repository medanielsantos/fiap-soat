package integration_test

import (
	"net/http"
	"testing"
)

func TestWorkOrderFlowAgainstPostgres(t *testing.T) {
	server := newTestServer(t)
	partID := createPart(t, server, "BRK-1", 3)
	orderID := createOrder(t, server)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/diagnosis", map[string]any{"diagnosis": "Replace brake pads"}, http.StatusOK)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/items", map[string]any{"type": "part", "catalog_id": partID, "quantity": 2}, http.StatusCreated)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/quotes", map[string]any{}, http.StatusOK)
	rejected := call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/quotes/reject", map[string]any{"client_id": testClientID, "note": "Customer wants a revised quote"}, http.StatusOK)
	if rejected["status"] != "diagnosis" {
		t.Fatalf("rejected quote status = %v, want diagnosis", rejected["status"])
	}
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/quotes", map[string]any{}, http.StatusOK)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/quotes/approve", map[string]any{"client_id": testClientID}, http.StatusOK)
	assertStock(t, server, partID, 3, 2)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/consumptions", map[string]any{"part_id": partID, "quantity": 1}, http.StatusOK)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/consumptions", map[string]any{"part_id": partID, "quantity": 1}, http.StatusOK)
	assertStock(t, server, partID, 1, 0)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/complete", map[string]any{}, http.StatusOK)
	delivered := call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/deliver", map[string]any{}, http.StatusOK)
	if delivered["status"] != "delivered" {
		t.Fatalf("final status = %v, want delivered", delivered["status"])
	}
	historyEntries := delivered["history"].([]any)
	createdEvent := historyEntries[0].(map[string]any)
	if createdEvent["user_id"] != testUserID {
		t.Fatalf("history user_id = %v, want %s", createdEvent["user_id"], testUserID)
	}
	if _, exists := createdEvent["actor_id"]; exists {
		t.Fatal("history response still contains actor_id")
	}
	clientEventFound := false
	for _, raw := range historyEntries {
		if raw.(map[string]any)["client_id"] == testClientID {
			clientEventFound = true
			break
		}
	}
	if !clientEventFound {
		t.Fatal("history is missing a quote decision with client_id")
	}
	history := call(t, server, http.MethodGet, "/api/v1/parts/"+partID+"/stock-movements?page=1&per_page=20", nil, http.StatusOK)
	if history["pagination"].(map[string]any)["total_items"] != float64(4) {
		t.Fatalf("movement history = %v, want 4 records (opening, reservation, two consumptions)", history["pagination"])
	}
}
