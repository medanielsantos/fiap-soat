package customerhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

const workOrdersPath = "/api/v1/customers/client/work-orders"

func TestClientWorkOrders(t *testing.T) {
	t.Run("test_it_should_list_only_orders_of_token_vehicle", func(t *testing.T) {
		env := newClientEnv(t)
		env.workOrders.orders["wo-1"] = pendingQuoteOrder("wo-1", "c-1", "v-1")
		env.workOrders.orders["wo-2"] = pendingQuoteOrder("wo-2", "c-1", "v-2")
		response := env.do(http.MethodGet, workOrdersPath, "", env.token(t, "c-1", "v-1"))
		assertStatus(t, response, http.StatusOK)
		page := decodeJSON[pagination.Response[clientWorkOrder]](t, response)
		if len(page.Data) != 1 || page.Data[0].ID != "wo-1" {
			t.Fatalf("data = %+v, want only wo-1", page.Data)
		}
		order := page.Data[0]
		if len(order.Items) != 1 || order.Items[0].Description != "Oil filter" || len(order.Quotes) != 1 || order.Quotes[0].Items[0].SubtotalInCents != 3000 {
			t.Errorf("order mapping = %+v, want items and quote items", order)
		}
	})

	t.Run("test_it_should_return_no_content_when_customer_has_no_orders", func(t *testing.T) {
		env := newClientEnv(t)
		assertStatus(t, env.do(http.MethodGet, workOrdersPath, "", env.token(t, "c-1", "")), http.StatusNoContent)
	})

	t.Run("test_it_should_return_bad_request_when_pagination_is_invalid", func(t *testing.T) {
		env := newClientEnv(t)
		assertProblem(t, env.do(http.MethodGet, workOrdersPath+"?per_page=x", "", env.token(t, "c-1", "")), http.StatusBadRequest, "Invalid pagination")
	})

	t.Run("test_it_should_return_internal_error_when_listing_fails", func(t *testing.T) {
		env := newClientEnv(t)
		env.workOrders.listErr = errBackend
		assertProblem(t, env.do(http.MethodGet, workOrdersPath, "", env.token(t, "c-1", "")), http.StatusInternalServerError, "Internal server error")
	})
}

func TestClientWorkOrder(t *testing.T) {
	env := newClientEnv(t)
	env.workOrders.orders["wo-1"] = pendingQuoteOrder("wo-1", "c-1", "v-1")
	token := env.token(t, "c-1", "")

	response := env.do(http.MethodGet, workOrdersPath+"/wo-1", "", token)
	assertStatus(t, response, http.StatusOK)
	if order := decodeJSON[clientWorkOrder](t, response); order.ID != "wo-1" || order.Diagnosis != "Worn filter" {
		t.Errorf("order = %+v, want wo-1", order)
	}

	t.Run("test_it_should_hide_orders_of_other_vehicles", func(t *testing.T) {
		assertProblem(t, env.do(http.MethodGet, workOrdersPath+"/wo-1", "", env.token(t, "c-1", "v-other")), http.StatusNotFound, "Work order not found")
	})

	t.Run("test_it_should_return_not_found_for_unknown_id", func(t *testing.T) {
		assertProblem(t, env.do(http.MethodGet, workOrdersPath+"/missing", "", token), http.StatusNotFound, "Work order not found")
	})
}

func TestClientQuoteDecision(t *testing.T) {
	cases := []struct {
		name, action, body string
		inventoryErr       error
		status             int
		title, wantStatus  string
	}{
		{name: "test_it_should_approve_pending_quote", action: "approve", status: http.StatusOK, wantStatus: "in_progress"},
		{name: "test_it_should_conflict_when_stock_is_insufficient", action: "approve", inventoryErr: errBackend, status: http.StatusConflict, title: "Quote cannot be approved"},
		{name: "test_it_should_reject_quote_without_body", action: "reject", status: http.StatusOK, wantStatus: "diagnosis"},
		{name: "test_it_should_reject_quote_with_note", action: "reject", body: `{"note":"too expensive"}`, status: http.StatusOK, wantStatus: "diagnosis"},
		{name: "test_it_should_return_bad_request_when_reject_body_is_invalid", action: "reject", body: `{"reason":"x"}`, status: http.StatusBadRequest, title: "Invalid request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newClientEnv(t)
			env.workOrders.orders["wo-1"] = pendingQuoteOrder("wo-1", "c-1", "v-1")
			env.inventory.err = tc.inventoryErr
			response := env.do(http.MethodPost, workOrdersPath+"/wo-1/quotes/"+tc.action, tc.body, env.token(t, "c-1", ""))
			if tc.title != "" {
				assertProblem(t, response, tc.status, tc.title)
				return
			}
			assertStatus(t, response, tc.status)
			if order := decodeJSON[clientWorkOrder](t, response); string(order.Status) != tc.wantStatus || order.Quotes[0].DecidedAt == nil {
				t.Errorf("status = %q decided_at = %v, want %q and decided", order.Status, order.Quotes[0].DecidedAt, tc.wantStatus)
			}
		})
	}

	t.Run("test_it_should_conflict_when_quote_was_already_decided", func(t *testing.T) {
		env := newClientEnv(t)
		env.workOrders.orders["wo-1"] = pendingQuoteOrder("wo-1", "c-1", "v-1")
		token := env.token(t, "c-1", "")
		assertStatus(t, env.do(http.MethodPost, workOrdersPath+"/wo-1/quotes/reject", "", token), http.StatusOK)
		assertProblem(t, env.do(http.MethodPost, workOrdersPath+"/wo-1/quotes/reject", "", token), http.StatusConflict, "Quote cannot be rejected")
	})

	t.Run("test_it_should_return_not_found_for_orders_of_other_customers", func(t *testing.T) {
		env := newClientEnv(t)
		env.workOrders.orders["wo-1"] = pendingQuoteOrder("wo-1", "c-other", "v-1")
		for _, action := range []string{"approve", "reject"} {
			assertProblem(t, env.do(http.MethodPost, workOrdersPath+"/wo-1/quotes/"+action, "", env.token(t, "c-1", "")), http.StatusNotFound, "Work order not found")
		}
	})
}
