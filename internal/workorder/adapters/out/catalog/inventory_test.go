package catalog

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	catalogdomain "github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
)

func TestInventoryTranslatesRequirementsAndErrors(t *testing.T) {
	unexpected := errors.New("deadlock detected")
	errorCases := []struct {
		name     string
		storeErr error
		want     error
	}{
		{name: "success", storeErr: nil, want: nil},
		{name: "insufficient_stock", storeErr: fmt.Errorf("part-1: %w", catalogdomain.ErrInsufficientStock), want: application.ErrInsufficientStock},
		{name: "missing_part", storeErr: catalogapp.ErrNotFound, want: application.ErrCatalogItemUnavailable},
		{name: "unexpected_error", storeErr: unexpected, want: unexpected},
	}
	operations := map[string]func(*Inventory, []application.StockRequirement) error{
		"reserve": func(i *Inventory, r []application.StockRequirement) error {
			return i.Reserve(context.Background(), "wo-1", "principal-1", r)
		},
		"consume": func(i *Inventory, r []application.StockRequirement) error {
			return i.Consume(context.Background(), "wo-1", "principal-1", r)
		},
	}
	requirements := []application.StockRequirement{
		{PartID: "part-1", MovementID: "mv-1", Quantity: 2},
		{PartID: "part-2", MovementID: "mv-2", Quantity: 1},
	}
	want := []catalogapp.StockRequirement{
		{PartID: "part-1", MovementID: "mv-1", Quantity: 2},
		{PartID: "part-2", MovementID: "mv-2", Quantity: 1},
	}

	for opName, operation := range operations {
		for _, test := range errorCases {
			t.Run(opName+"_"+test.name, func(t *testing.T) {
				store := &stubStore{stockErr: test.storeErr}
				err := operation(NewInventory(newCatalog(store)), requirements)
				if !errors.Is(err, test.want) || (test.want == nil && err != nil) {
					t.Fatalf("error = %v, want %v", err, test.want)
				}
				if store.orderID != "wo-1" || store.principal != "principal-1" {
					t.Errorf("forwarded order=%q principal=%q", store.orderID, store.principal)
				}
				if !reflect.DeepEqual(store.requirements, want) {
					t.Errorf("requirements = %+v, want %+v", store.requirements, want)
				}
			})
		}
	}
}

func TestConvertEmptyRequirements(t *testing.T) {
	if got := convert(nil); got == nil || len(got) != 0 {
		t.Errorf("convert(nil) = %#v, want empty non-nil slice", got)
	}
}
