package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestServiceRejectQuote(t *testing.T) {
	t.Run("test_it_should_return_order_to_diagnosis_with_client_note", func(t *testing.T) {
		f := newFixture(quotedOrder(t))
		order, err := f.service().RejectQuote(context.Background(), "wo-1", "client-1", "too expensive")
		if err != nil {
			t.Fatalf("RejectQuote() error = %v", err)
		}
		if order.Status != domain.Diagnosis || order.Quotes[0].Decision != "rejected" {
			t.Errorf("status = %s decision = %s, want diagnosis/rejected", order.Status, order.Quotes[0].Decision)
		}
		last := f.store.events[len(f.store.events)-1]
		if last.Note != "too expensive" || last.ClientID != "client-1" {
			t.Errorf("event = %+v, want client note", last)
		}
		if f.inventory.reserved != nil {
			t.Error("rejection must not reserve stock")
		}
	})

	t.Run("test_it_should_keep_default_note_when_none_is_given", func(t *testing.T) {
		f := newFixture(quotedOrder(t))
		if _, err := f.service().RejectQuote(context.Background(), "wo-1", "client-1", ""); err != nil {
			t.Fatal(err)
		}
		if got := f.store.events[len(f.store.events)-1].Note; got != "quote rejected" {
			t.Errorf("note = %q, want default", got)
		}
	})

	t.Run("test_it_should_reject_when_no_quote_is_pending", func(t *testing.T) {
		f := newFixture(orderIn(t, domain.Diagnosis))
		if _, err := f.service().RejectQuote(context.Background(), "wo-1", "client-1", "x"); !errors.Is(err, domain.ErrQuoteUnavailable) {
			t.Errorf("RejectQuote() error = %v, want %v", err, domain.ErrQuoteUnavailable)
		}
	})
}
