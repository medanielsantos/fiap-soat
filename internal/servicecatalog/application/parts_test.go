package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

var validPart = PartInput{SKU: "SKU-1", InternalCode: "P-1", Name: "Oil filter", Unit: "un", PurchasePriceInCents: 500, SalePriceInCents: 1200, Active: true, PerformedBy: "admin"}

func TestCatalogServiceCreatePart(t *testing.T) {
	t.Run("test_it_should_not_record_opening_movement_when_stock_is_zero", func(t *testing.T) {
		svc, store := newTestService(0)
		got, err := svc.CreatePart(context.Background(), validPart)
		if err != nil {
			t.Fatalf("CreatePart() error = %v", err)
		}
		if got.Stock() != 0 || store.opening != nil {
			t.Errorf("stock = %d, opening = %v, want 0 and nil", got.Stock(), store.opening)
		}
	})

	t.Run("test_it_should_record_opening_entry_when_initial_stock_is_positive", func(t *testing.T) {
		svc, store := newTestService(0)
		in := validPart
		in.InitialStock = 7
		got, err := svc.CreatePart(context.Background(), in)
		if err != nil {
			t.Fatalf("CreatePart() error = %v", err)
		}
		m := store.opening
		if m == nil || m.ID() != "id-2" || m.PartID() != got.ID() || m.Quantity() != 7 || m.Type() != domain.MovementEntry {
			t.Fatalf("opening movement = %+v, want entry of 7 for %s", m, got.ID())
		}
	})

	stocked := validPart
	stocked.InitialStock = 3
	negative := validPart
	negative.InitialStock = -1
	tests := []struct {
		name    string
		failID  int
		input   PartInput
		saveErr error
		want    error
	}{
		{"part_id_generation_fails", 1, validPart, nil, errBoom},
		{"movement_id_generation_fails", 2, stocked, nil, errBoom},
		{"initial_stock_is_negative", 0, negative, nil, domain.ErrInsufficientStock},
		{"store_fails", 0, validPart, ErrDuplicate, ErrDuplicate},
	}
	for _, tt := range tests {
		t.Run("test_it_should_fail_when_"+tt.name, func(t *testing.T) {
			svc, store := newTestService(tt.failID)
			store.errs["SavePart"] = tt.saveErr
			if _, err := svc.CreatePart(context.Background(), tt.input); !errors.Is(err, tt.want) {
				t.Errorf("CreatePart() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCatalogServiceUpdatePart(t *testing.T) {
	t.Run("test_it_should_keep_current_stock_and_reservation", func(t *testing.T) {
		svc, store := newTestService(0)
		store.parts["p1"] = mustPart(t, "p1", 10, 4)
		in := validPart
		in.InitialStock = 999
		got, err := svc.UpdatePart(context.Background(), "p1", in)
		if err != nil {
			t.Fatalf("UpdatePart() error = %v", err)
		}
		if got.Stock() != 10 || got.ReservedStock() != 4 || got.Name() != "Oil filter" {
			t.Errorf("part = stock %d reserved %d name %q, want 10/4/Oil filter", got.Stock(), got.ReservedStock(), got.Name())
		}
	})

	t.Run("test_it_should_return_not_found_when_part_is_missing", func(t *testing.T) {
		svc, _ := newTestService(0)
		if _, err := svc.UpdatePart(context.Background(), "missing", validPart); !errors.Is(err, ErrNotFound) {
			t.Errorf("UpdatePart() error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("test_it_should_reject_missing_unit", func(t *testing.T) {
		svc, store := newTestService(0)
		store.parts["p1"] = mustPart(t, "p1", 0, 0)
		in := validPart
		in.Unit = " "
		if _, err := svc.UpdatePart(context.Background(), "p1", in); !errors.Is(err, domain.ErrInvalidUnit) {
			t.Errorf("UpdatePart() error = %v, want %v", err, domain.ErrInvalidUnit)
		}
	})
}

func TestCatalogServiceReadAndDeletePart(t *testing.T) {
	svc, store := newTestService(0)
	store.parts["p1"] = mustPart(t, "p1", 0, 0)
	if got, err := svc.GetPart(context.Background(), "p1"); err != nil || got.ID() != "p1" {
		t.Errorf("GetPart() = %+v, %v", got, err)
	}
	page, err := svc.ListParts(context.Background(), defaultParams(t))
	if err != nil || page.TotalItems != 1 {
		t.Errorf("ListParts() total = %d, err = %v, want 1", page.TotalItems, err)
	}
	if err := svc.DeletePart(context.Background(), "p1"); err != nil {
		t.Fatalf("DeletePart() error = %v", err)
	}
	if _, err := svc.GetPart(context.Background(), "p1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetPart() after delete error = %v, want %v", err, ErrNotFound)
	}
}
