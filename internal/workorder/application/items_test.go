package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func activePart() CatalogItem {
	return CatalogItem{ID: "part-1", Code: "PAD-01", Name: "Brake pad", Unit: "pair", SalePriceInCents: 2500, Active: true}
}

func TestServiceDiagnosis(t *testing.T) {
	f := newFixture(orderIn(t, domain.Received))
	order, err := f.service().Diagnosis(context.Background(), "wo-1", "  brake noise ", "user-2")
	if err != nil {
		t.Fatalf("Diagnosis() error = %v", err)
	}
	if order.Status != domain.Diagnosis || order.Diagnosis != "brake noise" || order.UpdatedBy != "user-2" {
		t.Errorf("order = %+v, want diagnosis recorded by user-2", order)
	}
	if got := f.store.orders["wo-1"].Status; got != domain.Diagnosis {
		t.Errorf("persisted status = %s, want diagnosis", got)
	}
	if _, err := f.service().Diagnosis(context.Background(), "wo-1", " ", "user-2"); !errors.Is(err, domain.ErrInvalidDiagnosis) {
		t.Errorf("blank Diagnosis() error = %v, want %v", err, domain.ErrInvalidDiagnosis)
	}
}

func TestServiceAddItem(t *testing.T) {
	t.Run("test_it_should_snapshot_catalog_part_into_order", func(t *testing.T) {
		f := newFixture(orderIn(t, domain.Diagnosis))
		f.catalog.item = activePart()
		order, err := f.service().AddItem(context.Background(), "wo-1", domain.PartItem, "part-1", 2, "user-1")
		if err != nil {
			t.Fatalf("AddItem() error = %v", err)
		}
		if f.catalog.kind != "part" || len(order.Items) != 1 {
			t.Fatalf("catalog lookup = %q, items = %+v", f.catalog.kind, order.Items)
		}
		item := order.Items[0]
		if item.ID != "id-1" || item.Code != "PAD-01" || item.UnitPriceInCents != 2500 || item.Quantity != 2 {
			t.Errorf("item = %+v, want catalog snapshot", item)
		}
	})

	t.Run("test_it_should_use_service_catalog_for_service_items", func(t *testing.T) {
		f := newFixture(orderIn(t, domain.Diagnosis))
		f.catalog.item = CatalogItem{ID: "svc-1", Code: "SVC", Name: "Alignment", SalePriceInCents: 5000, EstimatedDurationMinutes: 45, Active: true}
		order, err := f.service().AddItem(context.Background(), "wo-1", domain.ServiceItem, "svc-1", 1, "user-1")
		if err != nil || f.catalog.kind != "service" || order.Items[0].Unit != "service" {
			t.Fatalf("AddItem(service) = %+v, %v (lookup %q)", order.Items, err, f.catalog.kind)
		}
	})

	inactive := activePart()
	inactive.Active = false
	tests := []struct {
		name     string
		itemType domain.ItemType
		catalog  stubCatalog
		quantity int64
		failID   int
		status   domain.Status
		wantErr  error
	}{
		{name: "test_it_should_reject_unknown_item_type", itemType: "labour", quantity: 1, status: domain.Diagnosis, wantErr: domain.ErrInvalidItem},
		{name: "test_it_should_reject_missing_catalog_item", itemType: domain.PartItem, catalog: stubCatalog{err: errBoom}, quantity: 1, status: domain.Diagnosis, wantErr: ErrCatalogItemUnavailable},
		{name: "test_it_should_reject_inactive_catalog_item", itemType: domain.PartItem, catalog: stubCatalog{item: inactive}, quantity: 1, status: domain.Diagnosis, wantErr: ErrCatalogItemUnavailable},
		{name: "test_it_should_fail_when_item_id_generation_fails", itemType: domain.PartItem, catalog: stubCatalog{item: activePart()}, quantity: 1, failID: 1, status: domain.Diagnosis, wantErr: errBoom},
		{name: "test_it_should_reject_non_positive_quantity", itemType: domain.PartItem, catalog: stubCatalog{item: activePart()}, quantity: 0, status: domain.Diagnosis, wantErr: domain.ErrInvalidItem},
		{name: "test_it_should_reject_item_outside_diagnosis", itemType: domain.PartItem, catalog: stubCatalog{item: activePart()}, quantity: 1, status: domain.Received, wantErr: domain.ErrInvalidTransition},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(orderIn(t, test.status))
			f.catalog, f.ids.failAt = &test.catalog, test.failID
			_, err := f.service().AddItem(context.Background(), "wo-1", test.itemType, "part-1", test.quantity, "user-1")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("AddItem() error = %v, want %v", err, test.wantErr)
			}
			if len(f.store.orders["wo-1"].Items) != 0 {
				t.Error("item persisted despite failure")
			}
		})
	}
}

func TestServiceRemoveItem(t *testing.T) {
	f := newFixture(orderIn(t, domain.Diagnosis, mustItem(t, "item-1", domain.PartItem, "part-1", 1)))
	order, err := f.service().RemoveItem(context.Background(), "wo-1", "item-1", "user-1")
	if err != nil || len(order.Items) != 0 {
		t.Fatalf("RemoveItem() = %+v, %v", order.Items, err)
	}
	if _, err := f.service().RemoveItem(context.Background(), "wo-1", "item-1", "user-1"); !errors.Is(err, domain.ErrInvalidItem) {
		t.Errorf("second RemoveItem() error = %v, want %v", err, domain.ErrInvalidItem)
	}
}
