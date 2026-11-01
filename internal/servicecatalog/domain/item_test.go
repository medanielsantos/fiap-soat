package domain

import (
	"errors"
	"testing"
)

func TestServiceValidationAndAccessors(t *testing.T) {
	if _, err := NewService("", "CODE", "Name", "", 1, 1, true, "user"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("missing ID error = %v", err)
	}
	if _, err := NewService("id", "", "Name", "", 1, 1, true, "user"); !errors.Is(err, ErrInvalidInternalCode) {
		t.Fatalf("missing code error = %v", err)
	}
	if _, err := NewService("id", "CODE", "", "", 1, 1, true, "user"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("missing name error = %v", err)
	}
	if _, err := NewService("id", "CODE", "Name", "", 0, 1, true, "user"); !errors.Is(err, ErrInvalidPrice) {
		t.Fatalf("invalid price error = %v", err)
	}
	if _, err := NewService("id", "CODE", "Name", "", 1, 0, true, "user"); !errors.Is(err, ErrInvalidDuration) {
		t.Fatalf("invalid duration error = %v", err)
	}
	service, err := NewService(" id ", " CODE ", " Name ", " Description ", 250, 30, false, " user ")
	if err != nil {
		t.Fatal(err)
	}
	if service.ID() != "id" || service.InternalCode() != "CODE" || service.Name() != "Name" || service.Description() != "Description" || service.SalePriceInCents() != 250 || service.EstimatedDurationMinutes() != 30 || service.Active() || service.PerformedBy() != "user" {
		t.Fatalf("unexpected service values: %#v", service)
	}
}

func TestPartValidationStockAndAccessors(t *testing.T) {
	cases := []struct {
		name string
		args []any
		want error
	}{
		{"id", []any{"", "SKU", "CODE", "Name", "", "piece", int64(0), int64(1), int64(0), int64(0), true, ""}, ErrInvalidID},
		{"sku", []any{"id", "", "CODE", "Name", "", "piece", int64(0), int64(1), int64(0), int64(0), true, ""}, ErrInvalidSKU},
		{"code", []any{"id", "SKU", "", "Name", "", "piece", int64(0), int64(1), int64(0), int64(0), true, ""}, ErrInvalidInternalCode},
		{"name", []any{"id", "SKU", "CODE", "", "", "piece", int64(0), int64(1), int64(0), int64(0), true, ""}, ErrInvalidName},
		{"purchase_price", []any{"id", "SKU", "CODE", "Name", "", "piece", int64(-1), int64(1), int64(0), int64(0), true, ""}, ErrInvalidPurchasePrice},
		{"sale_price", []any{"id", "SKU", "CODE", "Name", "", "piece", int64(0), int64(0), int64(0), int64(0), true, ""}, ErrInvalidPrice},
		{"unit", []any{"id", "SKU", "CODE", "Name", "", "", int64(0), int64(1), int64(0), int64(0), true, ""}, ErrInvalidUnit},
		{"negative_stock", []any{"id", "SKU", "CODE", "Name", "", "piece", int64(0), int64(1), int64(-1), int64(0), true, ""}, ErrInsufficientStock},
		{"reserved_above_stock", []any{"id", "SKU", "CODE", "Name", "", "piece", int64(0), int64(1), int64(2), int64(3), true, ""}, ErrInsufficientStock},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args
			_, err := NewPart(args[0].(string), args[1].(string), args[2].(string), args[3].(string), args[4].(string), args[5].(string), args[6].(int64), args[7].(int64), args[8].(int64), args[9].(int64), args[10].(bool), args[11].(string))
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
	part, err := NewPart(" id ", " SKU ", " CODE ", " Name ", " Description ", " piece ", 100, 250, 8, 3, false, " user ")
	if err != nil {
		t.Fatal(err)
	}
	if part.ID() != "id" || part.SKU() != "SKU" || part.InternalCode() != "CODE" || part.Name() != "Name" || part.Description() != "Description" || part.Unit() != "piece" || part.PurchasePriceInCents() != 100 || part.SalePriceInCents() != 250 || part.Stock() != 8 || part.ReservedStock() != 3 || part.AvailableStock() != 5 || part.Active() || part.PerformedBy() != "user" {
		t.Fatalf("unexpected part values: %#v", part)
	}
}

func TestStockMovementValidationAndAccessors(t *testing.T) {
	if _, err := NewStockMovement("", "part", MovementEntry, 1, "", ""); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("missing movement ID error = %v", err)
	}
	if _, err := NewStockMovement("id", "part", MovementEntry, 0, "", ""); !errors.Is(err, ErrInvalidStockChange) {
		t.Fatalf("invalid quantity error = %v", err)
	}
	if _, err := NewStockMovement("id", "part", "reservation", 1, "", ""); !errors.Is(err, ErrInvalidMovement) {
		t.Fatalf("invalid movement type error = %v", err)
	}
	movement, err := NewStockMovement(" id ", " part ", MovementExit, 4, " note ", " user ")
	if err != nil {
		t.Fatal(err)
	}
	if movement.ID() != "id" || movement.PartID() != "part" || movement.Type() != MovementExit || movement.Quantity() != 4 || movement.Description() != "note" || movement.PerformedBy() != "user" {
		t.Fatalf("unexpected movement values: %#v", movement)
	}
}

func TestOverflowAndPartStockErrors(t *testing.T) {
	if _, err := NewPart("id", "sku", "code", "name", "", "unit", 0, 1, 1, -1, true, ""); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("negative reservation error = %v", err)
	}
	if _, err := NewPart("id", "sku", "code", "name", "", "unit", 0, 1, 1, 2, true, ""); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("reservation above stock error = %v", err)
	}
}
