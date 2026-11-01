package integration_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestInsufficientStockRollsBackQuoteApproval(t *testing.T) {
	server := newTestServer(t)
	partID := createPart(t, server, "LOW-1", 1)
	orderID := prepareOrder(t, server, partID, 2)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+orderID+"/quotes/approve", map[string]any{"actor_id": testActorID}, http.StatusConflict)
	order := call(t, server, http.MethodGet, "/api/v1/work-orders/"+orderID, nil, http.StatusOK)
	if order["status"] != "awaiting_approval" {
		t.Fatalf("status after failed approval = %v, want awaiting_approval", order["status"])
	}
	assertStock(t, server, partID, 1, 0)
	movements := call(t, server, http.MethodGet, "/api/v1/parts/"+partID+"/stock-movements", nil, http.StatusOK)
	if movements["pagination"].(map[string]any)["total_items"] != float64(1) {
		t.Fatalf("movement count after rollback = %v, want opening entry only", movements["pagination"])
	}
}

func TestConcurrentApprovalsCannotOversellStock(t *testing.T) {
	server := newTestServer(t)
	partID := createPart(t, server, "RACE-1", 1)
	orders := []string{prepareOrder(t, server, partID, 1), prepareOrder(t, server, partID, 1)}
	var wg sync.WaitGroup
	statuses := make(chan int, len(orders))
	for _, orderID := range orders {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/work-orders/"+id+"/quotes/approve", bytes.NewBufferString(`{"actor_id":"integration-user"}`))
			server.ServeHTTP(recorder, request)
			statuses <- recorder.Code
		}(orderID)
	}
	wg.Wait()
	close(statuses)
	approved, conflicted := 0, 0
	for status := range statuses {
		if status == http.StatusOK {
			approved++
		} else if status == http.StatusConflict {
			conflicted++
		} else {
			t.Fatalf("unexpected approval status %d", status)
		}
	}
	if approved != 1 || conflicted != 1 {
		t.Fatalf("approval results = %d successful, %d conflicted; want one of each", approved, conflicted)
	}
	assertStock(t, server, partID, 1, 1)
}
