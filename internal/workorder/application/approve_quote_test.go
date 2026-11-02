package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func quotedOrder(t *testing.T) domain.WorkOrder {
	return orderIn(t, domain.AwaitingApproval,
		mustItem(t, "item-1", domain.PartItem, "part-b", 2),
		mustItem(t, "item-2", domain.ServiceItem, "svc-1", 1),
		mustItem(t, "item-3", domain.PartItem, "part-a", 1),
		mustItem(t, "item-4", domain.PartItem, "part-b", 3),
	)
}

func TestServiceSubmitQuote(t *testing.T) {
	f := newFixture(orderIn(t, domain.Diagnosis, mustItem(t, "item-1", domain.PartItem, "part-1", 2)))
	order, err := f.service().SubmitQuote(context.Background(), "wo-1", "user-1")
	if err != nil {
		t.Fatalf("SubmitQuote() error = %v", err)
	}
	if order.Status != domain.AwaitingApproval || len(order.Quotes) != 1 || order.Quotes[0].ID != "id-1" || order.Quotes[0].TotalInCents != 2000 {
		t.Errorf("order = %+v, want awaiting approval with quote id-1 of 2000", order)
	}

	empty := newFixture(orderIn(t, domain.Diagnosis))
	if _, err := empty.service().SubmitQuote(context.Background(), "wo-1", "user-1"); !errors.Is(err, domain.ErrQuoteEmpty) {
		t.Errorf("empty SubmitQuote() error = %v, want %v", err, domain.ErrQuoteEmpty)
	}
	failing := newFixture(orderIn(t, domain.Diagnosis))
	failing.ids.failAt = 1
	if _, err := failing.service().SubmitQuote(context.Background(), "wo-1", "user-1"); !errors.Is(err, errBoom) {
		t.Errorf("SubmitQuote() id error = %v, want %v", err, errBoom)
	}
}

func TestServiceApproveQuote(t *testing.T) {
	t.Run("test_it_should_reserve_aggregated_parts_and_move_to_in_progress", func(t *testing.T) {
		f := newFixture(quotedOrder(t))
		order, err := f.service().ApproveQuote(context.Background(), "wo-1", "client-1")
		if err != nil {
			t.Fatalf("ApproveQuote() error = %v", err)
		}
		if order.Status != domain.InProgress || order.Quotes[0].Decision != "approved" {
			t.Errorf("order status = %s decision = %s", order.Status, order.Quotes[0].Decision)
		}
		want := []StockRequirement{{PartID: "part-a", MovementID: "id-2", Quantity: 1}, {PartID: "part-b", MovementID: "id-3", Quantity: 5}}
		if !reflect.DeepEqual(f.inventory.reserved, want) || f.inventory.principal != "client-1" {
			t.Errorf("reserved = %+v by %q, want %+v by client-1", f.inventory.reserved, f.inventory.principal, want)
		}
		last := f.store.events[len(f.store.events)-1]
		if last.ID != "id-1" || last.ClientID != "client-1" || last.UserID != "" {
			t.Errorf("history event = %+v, want client event id-1", last)
		}
	})

	tests := []struct {
		name    string
		id      string
		client  string
		setup   func(*fixture)
		wantErr error
	}{
		{name: "test_it_should_return_not_found_for_missing_order", id: "missing", client: "client-1", setup: func(*fixture) {}, wantErr: ErrNotFound},
		{name: "test_it_should_require_client_principal", id: "wo-1", client: " ", setup: func(*fixture) {}, wantErr: domain.ErrInvalidPrincipal},
		{name: "test_it_should_fail_when_event_id_generation_fails", id: "wo-1", client: "client-1", setup: func(f *fixture) { f.ids.failAt = 1 }, wantErr: errBoom},
		{name: "test_it_should_fail_when_movement_id_generation_fails", id: "wo-1", client: "client-1", setup: func(f *fixture) { f.ids.failAt = 3 }, wantErr: errBoom},
		{name: "test_it_should_not_approve_when_stock_is_insufficient", id: "wo-1", client: "client-1", setup: func(f *fixture) { f.inventory.err = ErrInsufficientStock }, wantErr: ErrInsufficientStock},
		{name: "test_it_should_propagate_save_error_after_reservation", id: "wo-1", client: "client-1", setup: func(f *fixture) { f.store.saveErr = errStorage }, wantErr: errStorage},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(quotedOrder(t))
			test.setup(f)
			order, err := f.service().ApproveQuote(context.Background(), test.id, test.client)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ApproveQuote() error = %v, want %v", err, test.wantErr)
			}
			if order.ID != "" {
				t.Errorf("ApproveQuote() returned order %q on error", order.ID)
			}
			persisted := f.store.orders["wo-1"]
			if persisted.Status != domain.AwaitingApproval || persisted.Quotes[0].Decision != "pending" {
				t.Errorf("persisted order changed: status=%s decision=%s", persisted.Status, persisted.Quotes[0].Decision)
			}
		})
	}

	t.Run("test_it_should_not_approve_twice", func(t *testing.T) {
		f := newFixture(quotedOrder(t))
		if _, err := f.service().ApproveQuote(context.Background(), "wo-1", "client-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := f.service().ApproveQuote(context.Background(), "wo-1", "client-1"); !errors.Is(err, domain.ErrQuoteUnavailable) {
			t.Errorf("second ApproveQuote() error = %v, want %v", err, domain.ErrQuoteUnavailable)
		}
	})
}
