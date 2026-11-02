package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestServiceCreate(t *testing.T) {
	t.Run("test_it_should_create_received_order_with_history_event", func(t *testing.T) {
		f := newFixture()
		order, err := f.service().Create(context.Background(), "customer-1", "vehicle-1", "user-1")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if order.ID != "id-1" || order.Status != domain.Received || order.CreatedAt != testNow {
			t.Errorf("order = %+v, want id-1 received at testNow", order)
		}
		if len(order.History) != 1 || order.History[0].ID != "id-2" || order.History[0].UserID != "user-1" {
			t.Errorf("history = %+v, want one event id-2 by user-1", order.History)
		}
		if _, ok := f.store.orders["id-1"]; !ok || f.tx.calls != 1 {
			t.Errorf("order persisted = %v, tx calls = %d", ok, f.tx.calls)
		}
	})

	tests := []struct {
		name    string
		setup   func(*fixture)
		userID  string
		wantErr error
	}{
		{name: "test_it_should_hide_ownership_error_behind_unavailable", setup: func(f *fixture) { f.ownership.err = errBoom }, userID: "user-1", wantErr: ErrCustomerOrVehicleUnavailable},
		{name: "test_it_should_fail_when_order_id_generation_fails", setup: func(f *fixture) { f.ids.failAt = 1 }, userID: "user-1", wantErr: errBoom},
		{name: "test_it_should_fail_when_event_id_generation_fails", setup: func(f *fixture) { f.ids.failAt = 2 }, userID: "user-1", wantErr: errBoom},
		{name: "test_it_should_reject_missing_principal", setup: func(*fixture) {}, userID: " ", wantErr: domain.ErrInvalidPrincipal},
		{name: "test_it_should_propagate_store_error", setup: func(f *fixture) { f.store.createErr = errStorage }, userID: "user-1", wantErr: errStorage},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture()
			test.setup(f)
			_, err := f.service().Create(context.Background(), "customer-1", "vehicle-1", test.userID)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, test.wantErr)
			}
			if len(f.store.orders) != 0 {
				t.Error("order persisted despite failure")
			}
		})
	}
}
