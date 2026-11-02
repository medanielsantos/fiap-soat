package workorderhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func quotedServer(t *testing.T) *testServer {
	t.Helper()
	s := diagnosedWithPart(t)
	s.mustDo(t, http.MethodPost, base+"/quotes", `{}`, http.StatusOK)
	return s
}

func TestSubmitQuote(t *testing.T) {
	s := newTestServer()
	s.mustDo(t, http.MethodPost, "/api/v1/work-orders", createBody, http.StatusCreated)
	s.mustDo(t, http.MethodPost, base+"/diagnosis", `{"diagnosis":"noise"}`, http.StatusOK)
	response := s.do(http.MethodPost, base+"/quotes", `{}`)
	if p := decodeProblem(t, response); response.Code != http.StatusConflict || p.Detail != domain.ErrQuoteEmpty.Error() {
		t.Errorf("empty quote status = %d problem = %+v", response.Code, p)
	}

	quotedServer(t).mustDo(t, http.MethodPost, base+"/quotes", `{}`, http.StatusConflict)
}

func TestApproveQuote(t *testing.T) {
	t.Run("test_it_should_approve_quote_and_record_client_decision", func(t *testing.T) {
		order := quotedServer(t).mustDo(t, http.MethodPost, base+"/quotes/approve", `{"client_id":"client-1"}`, http.StatusOK)
		quote := order.Quotes[len(order.Quotes)-1]
		if quote.Decision != "approved" || quote.DecidedByClientID != "client-1" || quote.TotalInCents != 5000 {
			t.Errorf("quote = %+v, want approved by client-1 totaling 5000", quote)
		}
	})

	t.Run("test_it_should_return_conflict_and_keep_quote_pending_when_stock_is_insufficient", func(t *testing.T) {
		s := quotedServer(t)
		s.inventory.err = application.ErrInsufficientStock
		response := s.do(http.MethodPost, base+"/quotes/approve", `{"client_id":"client-1"}`)
		if p := decodeProblem(t, response); response.Code != http.StatusConflict || p.Title != "Insufficient stock" {
			t.Fatalf("status = %d problem = %+v", response.Code, p)
		}
		if got := s.store.orders["id-1"].Status; got != domain.AwaitingApproval {
			t.Errorf("status = %s, want awaiting_approval after failed reservation", got)
		}
	})

	t.Run("test_it_should_return_not_found_for_missing_order", func(t *testing.T) {
		quotedServer(t).mustDo(t, http.MethodPost, "/api/v1/work-orders/missing/quotes/approve", `{"client_id":"client-1"}`, http.StatusNotFound)
	})
}

func TestRejectQuote(t *testing.T) {
	s := quotedServer(t)
	order := s.mustDo(t, http.MethodPost, base+"/quotes/reject", `{"client_id":"client-1","note":"too expensive"}`, http.StatusOK)
	if order.Status != domain.Diagnosis || order.Quotes[0].Decision != "rejected" {
		t.Fatalf("status = %s decision = %s, want diagnosis/rejected", order.Status, order.Quotes[0].Decision)
	}
	s.mustDo(t, http.MethodPost, base+"/quotes/reject", `{"client_id":"client-1"}`, http.StatusConflict)
	resubmitted := s.mustDo(t, http.MethodPost, base+"/quotes", `{}`, http.StatusOK)
	if resubmitted.QuoteVersion != 2 {
		t.Errorf("quote_version = %d, want 2 after resubmission", resubmitted.QuoteVersion)
	}
}
