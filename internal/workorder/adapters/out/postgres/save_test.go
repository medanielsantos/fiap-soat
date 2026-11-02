package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestRepositorySavePersistsLifecycle(t *testing.T) {
	repo, _ := openRepository(t)
	o := createOrder(t, repo, orderSpec{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"})
	at := baseTime
	apply(t, repo, &o, "diagnosis", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.SetDiagnosis("brake noise", "user-1", at)
	})
	apply(t, repo, &o, "part", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.AddItem(newItem(t, "item-1", domain.PartItem, "part-1", 2), "user-1", at)
	})
	apply(t, repo, &o, "service", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.AddItem(newItem(t, "item-2", domain.ServiceItem, "svc-1", 1), "user-1", at)
	})
	apply(t, repo, &o, "quote", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		_, e, err := o.SubmitQuote("quote-1", "user-1", at)
		return e, err
	})
	apply(t, repo, &o, "approve", func(o *domain.WorkOrder) (domain.HistoryEntry, error) { return o.DecideQuote(true, "client-1", at) })
	apply(t, repo, &o, "consume", func(o *domain.WorkOrder) (domain.HistoryEntry, error) { return o.Consume("part-1", 2, "user-2", at) })

	found, err := repo.Find(context.Background(), "wo-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found.Status != domain.InProgress || found.Diagnosis != "brake noise" || found.QuoteVersion != 1 || found.UpdatedBy != "user-2" {
		t.Errorf("order = %+v", found)
	}
	if len(found.Items) != 2 || found.Items[0].ID != "item-1" || found.Items[0].ConsumedQuantity != 2 || found.Items[1].Unit != "service" {
		t.Errorf("items = %+v, want consumed part then service", found.Items)
	}
	if len(found.Quotes) != 1 {
		t.Fatalf("quotes = %+v, want one", found.Quotes)
	}
	q := found.Quotes[0]
	if q.Decision != "approved" || q.DecidedByClientID != "client-1" || q.DecidedAt == nil || q.TotalInCents != 4500 || len(q.Items) != 2 {
		t.Errorf("quote = %+v, want approved by client-1 with two items totaling 4500", q)
	}
	if len(found.History) != 7 {
		t.Fatalf("history length = %d, want 7", len(found.History))
	}
	byID := map[string]domain.HistoryEntry{}
	for _, h := range found.History {
		byID[h.ID] = h
	}
	approve := byID["wo-1-approve"]
	if approve.ClientID != "client-1" || approve.UserID != "" || approve.From == nil || *approve.From != domain.AwaitingApproval {
		t.Errorf("approve event = %+v, want client event from awaiting_approval", approve)
	}
}

func TestRepositorySaveRejectedQuoteThenResubmit(t *testing.T) {
	repo, _ := openRepository(t)
	o := createOrder(t, repo, orderSpec{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"})
	apply(t, repo, &o, "diagnosis", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.SetDiagnosis("noise", "user-1", baseTime)
	})
	apply(t, repo, &o, "item", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.AddItem(newItem(t, "item-1", domain.ServiceItem, "svc-1", 1), "user-1", baseTime)
	})
	submit := func(id string) func(*domain.WorkOrder) (domain.HistoryEntry, error) {
		return func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
			_, e, err := o.SubmitQuote(id, "user-1", baseTime)
			return e, err
		}
	}
	apply(t, repo, &o, "quote1", submit("quote-1"))
	apply(t, repo, &o, "reject", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.DecideQuote(false, "client-1", baseTime)
	})
	apply(t, repo, &o, "quote2", submit("quote-2"))

	found, err := repo.Find(context.Background(), "wo-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(found.Quotes) != 2 || found.Quotes[0].Decision != "rejected" || found.Quotes[1].Decision != "pending" || found.Quotes[1].Version != 2 {
		t.Errorf("quotes = %+v, want rejected v1 then pending v2", found.Quotes)
	}
}

func TestRepositorySaveFailures(t *testing.T) {
	t.Run("test_it_should_return_not_found_for_missing_order", func(t *testing.T) {
		repo, _ := openRepository(t)
		o, event, _ := domain.NewWorkOrder("ghost", "customer-1", "vehicle-1", "user-1", baseTime)
		event.ID = "evt"
		if err := repo.Save(context.Background(), o, event); !errors.Is(err, application.ErrNotFound) {
			t.Errorf("Save() error = %v, want %v", err, application.ErrNotFound)
		}
	})

	t.Run("test_it_should_reject_invalid_status", func(t *testing.T) {
		repo, _ := openRepository(t)
		o := createOrder(t, repo, orderSpec{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"})
		o.Status = "archived"
		if err := repo.Save(context.Background(), o, domain.HistoryEntry{ID: "evt", To: o.Status, UserID: "u"}); err == nil {
			t.Error("Save() error = nil, want status check violation")
		}
	})

	cases := map[string]func(*domain.WorkOrder, *domain.HistoryEntry){
		"test_it_should_fail_for_invalid_item": func(o *domain.WorkOrder, _ *domain.HistoryEntry) {
			o.Items = []domain.Item{{ID: "bad", Type: domain.PartItem, CatalogID: "p", Code: "c", Description: "d", Unit: "u"}}
		},
		"test_it_should_fail_for_invalid_quote": func(o *domain.WorkOrder, _ *domain.HistoryEntry) {
			o.Quotes = []domain.Quote{{ID: "q", Version: 0, Decision: "pending"}}
		},
		"test_it_should_fail_for_invalid_quote_item": func(o *domain.WorkOrder, _ *domain.HistoryEntry) {
			o.Quotes = []domain.Quote{{ID: "q", Version: 1, Decision: "pending", Items: []domain.QuoteItem{{Item: domain.Item{Type: "bogus"}}}}}
		},
		"test_it_should_fail_for_event_with_two_principals": func(_ *domain.WorkOrder, e *domain.HistoryEntry) { e.ClientID = "client-1" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			repo, _ := openRepository(t)
			o := createOrder(t, repo, orderSpec{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"})
			event := domain.HistoryEntry{ID: "evt", To: o.Status, UserID: "user-1", CreatedAt: baseTime}
			mutate(&o, &event)
			if err := repo.Save(context.Background(), o, event); err == nil {
				t.Error("Save() error = nil, want constraint violation")
			}
		})
	}
}
