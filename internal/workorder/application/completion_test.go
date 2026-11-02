package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestServiceCompleteAndDeliver(t *testing.T) {
	t.Run("test_it_should_complete_and_deliver_after_full_consumption", func(t *testing.T) {
		f := newFixture(inProgressOrder(t))
		s := f.service()
		if _, err := s.Consume(context.Background(), "wo-1", "part-1", 2, "user-1"); err != nil {
			t.Fatal(err)
		}
		order, err := s.Complete(context.Background(), "wo-1", "user-1")
		if err != nil || order.Status != domain.Completed {
			t.Fatalf("Complete() = %s, %v", order.Status, err)
		}
		order, err = s.Deliver(context.Background(), "wo-1", "user-2")
		if err != nil || order.Status != domain.Delivered || order.UpdatedBy != "user-2" {
			t.Fatalf("Deliver() = %s by %s, %v", order.Status, order.UpdatedBy, err)
		}
	})

	t.Run("test_it_should_not_complete_with_unconsumed_parts", func(t *testing.T) {
		f := newFixture(inProgressOrder(t))
		if _, err := f.service().Complete(context.Background(), "wo-1", "user-1"); !errors.Is(err, domain.ErrItemsNotConsumed) {
			t.Errorf("Complete() error = %v, want %v", err, domain.ErrItemsNotConsumed)
		}
	})

	t.Run("test_it_should_not_deliver_before_completion", func(t *testing.T) {
		f := newFixture(inProgressOrder(t))
		if _, err := f.service().Deliver(context.Background(), "wo-1", "user-1"); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Errorf("Deliver() error = %v, want %v", err, domain.ErrInvalidTransition)
		}
	})
}

func TestServiceMutateFailures(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		setup   func(*fixture)
		wantErr error
	}{
		{name: "test_it_should_fail_when_event_id_generation_fails", userID: "user-1", setup: func(f *fixture) { f.ids.failAt = 1 }, wantErr: errBoom},
		{name: "test_it_should_propagate_save_error", userID: "user-1", setup: func(f *fixture) { f.store.saveErr = errStorage }, wantErr: errStorage},
		{name: "test_it_should_reject_event_without_principal", userID: "", setup: func(*fixture) {}, wantErr: domain.ErrInvalidPrincipal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(orderIn(t, domain.Received))
			test.setup(f)
			order, err := f.service().Diagnosis(context.Background(), "wo-1", "noise", test.userID)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Diagnosis() error = %v, want %v", err, test.wantErr)
			}
			if order.ID != "" || f.store.orders["wo-1"].Status != domain.Received {
				t.Errorf("failed mutation leaked: order=%q persisted status=%s", order.ID, f.store.orders["wo-1"].Status)
			}
		})
	}
}
