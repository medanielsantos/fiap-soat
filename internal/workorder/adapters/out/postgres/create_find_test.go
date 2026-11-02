package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestRepositoryCreateAndFind(t *testing.T) {
	repo, _ := openRepository(t)
	created := createOrder(t, repo, orderSpec{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"})

	found, err := repo.Find(context.Background(), "wo-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found.CustomerID != "customer-1" || found.VehicleID != "vehicle-1" || found.Status != domain.Received || found.CreatedBy != "user-1" {
		t.Errorf("found = %+v", found)
	}
	if !found.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("created_at = %v, want %v", found.CreatedAt, created.CreatedAt)
	}
	if found.Items == nil || found.Quotes == nil || len(found.History) != 1 {
		t.Fatalf("collections = items:%v quotes:%v history:%+v", found.Items, found.Quotes, found.History)
	}
	h := found.History[0]
	if h.ID != "wo-1-created" || h.From != nil || h.To != domain.Received || h.UserID != "user-1" || h.ClientID != "" {
		t.Errorf("history = %+v, want creation event by user-1 without from_status", h)
	}
}

func TestRepositoryFindMissingReturnsNotFound(t *testing.T) {
	repo, _ := openRepository(t)
	if _, err := repo.Find(context.Background(), "missing"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("Find() error = %v, want %v", err, application.ErrNotFound)
	}
}

func TestRepositoryCreatePersistsInitialItems(t *testing.T) {
	repo, _ := openRepository(t)
	o, event, err := domain.NewWorkOrder("wo-1", "customer-1", "vehicle-1", "user-1", baseTime)
	if err != nil {
		t.Fatal(err)
	}
	event.ID = "evt-1"
	o.Items = []domain.Item{newItem(t, "item-1", domain.PartItem, "part-1", 2)}
	if err := repo.Create(context.Background(), o, event); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	found, err := repo.Find(context.Background(), "wo-1")
	if err != nil || len(found.Items) != 1 || found.Items[0].Unit != "pair" || found.Items[0].Quantity != 2 {
		t.Fatalf("Find() items = %+v, %v", found.Items, err)
	}
}

func TestRepositoryCreateFailures(t *testing.T) {
	validEvent := func(id string) domain.HistoryEntry {
		return domain.HistoryEntry{ID: id, To: domain.Received, UserID: "user-1", CreatedAt: baseTime}
	}
	tests := []struct {
		name   string
		mutate func(*domain.WorkOrder, *domain.HistoryEntry)
	}{
		{name: "test_it_should_fail_for_unknown_customer", mutate: func(o *domain.WorkOrder, _ *domain.HistoryEntry) { o.CustomerID = "ghost" }},
		{name: "test_it_should_fail_for_invalid_item", mutate: func(o *domain.WorkOrder, _ *domain.HistoryEntry) {
			o.Items = []domain.Item{{ID: "item-1", Type: domain.PartItem, CatalogID: "p", Code: "c", Description: "d", Unit: "u"}}
		}},
		{name: "test_it_should_fail_for_event_without_principal", mutate: func(_ *domain.WorkOrder, e *domain.HistoryEntry) { e.UserID = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, _ := openRepository(t)
			o, _, err := domain.NewWorkOrder("wo-1", "customer-1", "vehicle-1", "user-1", baseTime)
			if err != nil {
				t.Fatal(err)
			}
			event := validEvent("evt-1")
			test.mutate(&o, &event)
			if err := repo.Create(context.Background(), o, event); err == nil {
				t.Fatal("Create() error = nil, want database constraint error")
			}
		})
	}
}
