package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func inProgressOrder(t *testing.T) domain.WorkOrder {
	return orderIn(t, domain.InProgress, mustItem(t, "item-1", domain.PartItem, "part-1", 2))
}

func TestServiceConsume(t *testing.T) {
	t.Run("test_it_should_consume_reserved_stock_and_track_quantity", func(t *testing.T) {
		f := newFixture(inProgressOrder(t))
		order, err := f.service().Consume(context.Background(), "wo-1", "part-1", 2, "user-1")
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}
		if order.Items[0].ConsumedQuantity != 2 {
			t.Errorf("consumed = %d, want 2", order.Items[0].ConsumedQuantity)
		}
		want := []StockRequirement{{PartID: "part-1", MovementID: "id-1", Quantity: 2}}
		if !reflect.DeepEqual(f.inventory.consumed, want) || f.inventory.principal != "user-1" {
			t.Errorf("consumed = %+v by %q, want %+v", f.inventory.consumed, f.inventory.principal, want)
		}
		if last := f.store.events[len(f.store.events)-1]; last.ID != "id-2" || last.UserID != "user-1" {
			t.Errorf("event = %+v, want id-2 by user-1", last)
		}
	})

	tests := []struct {
		name     string
		id       string
		quantity int64
		setup    func(*fixture)
		wantErr  error
	}{
		{name: "test_it_should_return_not_found_for_missing_order", id: "missing", quantity: 1, setup: func(*fixture) {}, wantErr: ErrNotFound},
		{name: "test_it_should_reject_over_consumption", id: "wo-1", quantity: 3, setup: func(*fixture) {}, wantErr: domain.ErrInsufficientConsumption},
		{name: "test_it_should_fail_when_movement_id_generation_fails", id: "wo-1", quantity: 1, setup: func(f *fixture) { f.ids.failAt = 1 }, wantErr: errBoom},
		{name: "test_it_should_fail_when_event_id_generation_fails", id: "wo-1", quantity: 1, setup: func(f *fixture) { f.ids.failAt = 2 }, wantErr: errBoom},
		{name: "test_it_should_propagate_inventory_error", id: "wo-1", quantity: 1, setup: func(f *fixture) { f.inventory.err = ErrInsufficientStock }, wantErr: ErrInsufficientStock},
		{name: "test_it_should_propagate_save_error", id: "wo-1", quantity: 1, setup: func(f *fixture) { f.store.saveErr = errStorage }, wantErr: errStorage},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(inProgressOrder(t))
			test.setup(f)
			if _, err := f.service().Consume(context.Background(), test.id, "part-1", test.quantity, "user-1"); !errors.Is(err, test.wantErr) {
				t.Fatalf("Consume() error = %v, want %v", err, test.wantErr)
			}
			if got := f.store.orders["wo-1"].Items[0].ConsumedQuantity; got != 0 {
				t.Errorf("persisted consumed quantity = %d, want 0", got)
			}
		})
	}
}
