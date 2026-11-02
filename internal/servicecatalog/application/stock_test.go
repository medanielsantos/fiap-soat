package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

func TestCatalogServiceMoveStock(t *testing.T) {
	t.Run("test_it_should_build_movement_and_delegate_to_store", func(t *testing.T) {
		svc, store := newTestService(0)
		store.parts["p1"] = mustPart(t, "p1", 5, 0)
		got, err := svc.MoveStock(context.Background(), "p1", domain.MovementExit, 2, " sale ", "clerk")
		if err != nil {
			t.Fatalf("MoveStock() error = %v", err)
		}
		if got.ID() != "p1" || len(store.moved) != 1 {
			t.Fatalf("part = %s, moved = %d, want p1 and 1 movement", got.ID(), len(store.moved))
		}
		m := store.moved[0]
		if m.ID() != "id-1" || m.Type() != domain.MovementExit || m.Quantity() != 2 || m.Description() != "sale" || m.PerformedBy() != "clerk" {
			t.Errorf("movement = %+v", m)
		}
	})

	tests := []struct {
		name     string
		failID   int
		kind     domain.MovementType
		quantity int64
		storeErr error
		want     error
	}{
		{"id_generation_fails", 1, domain.MovementEntry, 1, nil, errBoom},
		{"quantity_is_zero", 0, domain.MovementEntry, 0, nil, domain.ErrInvalidStockChange},
		{"type_is_unknown", 0, "transfer", 1, nil, domain.ErrInvalidMovement},
		{"stock_is_insufficient", 0, domain.MovementExit, 9, domain.ErrInsufficientStock, domain.ErrInsufficientStock},
	}
	for _, tt := range tests {
		t.Run("test_it_should_fail_when_"+tt.name, func(t *testing.T) {
			svc, store := newTestService(tt.failID)
			store.errs["MoveStock"] = tt.storeErr
			if _, err := svc.MoveStock(context.Background(), "p1", tt.kind, tt.quantity, "", "u"); !errors.Is(err, tt.want) {
				t.Errorf("MoveStock() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCatalogServiceListStockMovements(t *testing.T) {
	t.Run("test_it_should_return_not_found_when_part_is_missing", func(t *testing.T) {
		svc, _ := newTestService(0)
		if _, err := svc.ListStockMovements(context.Background(), "missing", defaultParams(t)); !errors.Is(err, ErrNotFound) {
			t.Errorf("ListStockMovements() error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("test_it_should_list_movements_of_existing_part", func(t *testing.T) {
		svc, store := newTestService(0)
		store.parts["p1"] = mustPart(t, "p1", 1, 0)
		store.movements = []StockMovement{{ID: "m1", PartID: "p1", Type: "entry", Quantity: 1}}
		page, err := svc.ListStockMovements(context.Background(), "p1", defaultParams(t))
		if err != nil || page.TotalItems != 1 || page.Items[0].ID != "m1" {
			t.Errorf("ListStockMovements() = %+v, %v", page, err)
		}
	})
}

func TestCatalogServiceReservations(t *testing.T) {
	reqs := []StockRequirement{{PartID: "p1", MovementID: "m1", Quantity: 2}}

	t.Run("test_it_should_forward_reservation_and_consumption", func(t *testing.T) {
		svc, store := newTestService(0)
		if err := svc.ReserveStock(context.Background(), "wo-1", "c1", reqs); err != nil {
			t.Fatalf("ReserveStock() error = %v", err)
		}
		if err := svc.ConsumeReservedStock(context.Background(), "wo-1", "u1", reqs); err != nil {
			t.Fatalf("ConsumeReservedStock() error = %v", err)
		}
		if len(store.reserved) != 1 || len(store.consumed) != 1 {
			t.Errorf("reserved = %d, consumed = %d, want 1 each", len(store.reserved), len(store.consumed))
		}
	})

	t.Run("test_it_should_propagate_store_errors", func(t *testing.T) {
		svc, store := newTestService(0)
		store.errs["ReserveStock"] = domain.ErrInsufficientStock
		store.errs["ConsumeReservedStock"] = ErrNotFound
		if err := svc.ReserveStock(context.Background(), "wo-1", "c1", reqs); !errors.Is(err, domain.ErrInsufficientStock) {
			t.Errorf("ReserveStock() error = %v", err)
		}
		if err := svc.ConsumeReservedStock(context.Background(), "wo-1", "u1", reqs); !errors.Is(err, ErrNotFound) {
			t.Errorf("ConsumeReservedStock() error = %v", err)
		}
	})
}
