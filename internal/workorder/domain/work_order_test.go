package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestWorkOrderLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	o, created, err := NewWorkOrder("wo-1", "customer-1", "vehicle-1", "user-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if created.To != Received {
		t.Fatalf("created status = %q", created.To)
	}
	if _, err = o.SetDiagnosis("Replace brake pads", "user-1", now); err != nil {
		t.Fatal(err)
	}
	service, err := NewItem("item-1", ServiceItem, "svc-1", "BRAKE-SERVICE", "Brake service", "service", 1, 5000, 60)
	if err != nil {
		t.Fatal(err)
	}
	part, err := NewItem("item-2", PartItem, "part-1", "PAD-01", "Brake pad", "pair", 2, 2500, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.AddItem(service, "user-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err = o.AddItem(part, "user-1", now); err != nil {
		t.Fatal(err)
	}
	quote, _, err := o.SubmitQuote("quote-1", "user-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if quote.TotalInCents != 10000 {
		t.Fatalf("quote total = %d, want 10000", quote.TotalInCents)
	}
	if quote.Version != 1 {
		t.Fatalf("quote version = %d, want 1", quote.Version)
	}
	if _, err = o.DecideQuote(false, "customer-1", now); err != nil {
		t.Fatal(err)
	}
	if o.Status != Diagnosis || o.Quotes[0].Decision != "rejected" {
		t.Fatalf("rejection did not return order to diagnosis: status=%s decision=%s", o.Status, o.Quotes[0].Decision)
	}
	if _, _, err = o.SubmitQuote("quote-2", "user-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err = o.DecideQuote(true, "customer-1", now); err != nil {
		t.Fatal(err)
	}
	if o.Status != InProgress {
		t.Fatalf("approved status = %s, want %s", o.Status, InProgress)
	}
	if _, err = o.Consume("part-1", 2, "mechanic-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err = o.Complete("user-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err = o.Deliver("user-1", now); err != nil {
		t.Fatal(err)
	}
	if o.Status != Delivered {
		t.Fatalf("final status = %s, want %s", o.Status, Delivered)
	}
}

func TestWorkOrderRejectsInvalidTransitionsAndOverConsumption(t *testing.T) {
	now := time.Now()
	o, _, err := NewWorkOrder("wo-1", "customer-1", "vehicle-1", "user-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.Complete("user-1", now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("complete error = %v, want invalid transition", err)
	}
	if _, err = o.SetDiagnosis("diagnosis", "user-1", now); err != nil {
		t.Fatal(err)
	}
	part, _ := NewItem("item-1", PartItem, "part-1", "SKU-1", "Part", "unit", 1, 100, 0)
	if _, err = o.AddItem(part, "user-1", now); err != nil {
		t.Fatal(err)
	}
	if _, _, err = o.SubmitQuote("quote-1", "user-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err = o.DecideQuote(true, "customer-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err = o.Consume("part-1", 2, "mechanic-1", now); !errors.Is(err, ErrInsufficientConsumption) {
		t.Fatalf("consume error = %v, want insufficient consumption", err)
	}
}

func TestWorkOrderValidationMutationAndOverflowBranches(t *testing.T) {
	now := time.Now()
	if _, _, err := NewWorkOrder("", "customer", "vehicle", "actor", now); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("invalid order error = %v", err)
	}
	o, _, err := NewWorkOrder("wo", "customer", "vehicle", "actor", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.SetDiagnosis("  ", "actor", now); !errors.Is(err, ErrInvalidDiagnosis) {
		t.Fatalf("empty diagnosis error = %v", err)
	}
	if _, _, err := o.SubmitQuote("quote", "actor", now); !errors.Is(err, ErrQuoteEmpty) {
		t.Fatalf("empty quote error = %v", err)
	}
	if _, err := o.RemoveItem("missing", "actor", now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("remove item before diagnosis error = %v", err)
	}
	if _, err := o.AddItem(Item{ID: "invalid"}, "actor", now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("add item before diagnosis error = %v", err)
	}
	if _, err := o.SetDiagnosis("check brakes", "actor", now); err != nil {
		t.Fatal(err)
	}
	if _, err := o.RemoveItem("missing", "actor", now); !errors.Is(err, ErrInvalidItem) {
		t.Fatalf("missing item error = %v", err)
	}
	part, _ := NewItem("part-item", PartItem, "part", "SKU", "Part", "piece", 1, 100, 0)
	if _, err := o.AddItem(part, "actor", now); err != nil {
		t.Fatal(err)
	}
	if _, err := o.AddItem(part, "actor", now); !errors.Is(err, ErrInvalidItem) {
		t.Fatalf("duplicate item error = %v", err)
	}
	if _, err := o.RemoveItem("part-item", "actor", now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.SubmitQuote("quote", "actor", now); !errors.Is(err, ErrQuoteEmpty) {
		t.Fatalf("quote after removing all items error = %v", err)
	}
	if _, err := o.Consume("part", 1, "actor", now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("consume before approval error = %v", err)
	}
	item, _ := NewItem("service-item", ServiceItem, "service", "SVC", "Service", "ignored", 1, 100, 1)
	if item.Unit != "service" {
		t.Fatalf("service unit = %q, want service", item.Unit)
	}
	if _, err := NewItem("", ServiceItem, "service", "SVC", "Service", "", 1, 100, 1); !errors.Is(err, ErrInvalidItem) {
		t.Fatalf("invalid item error = %v", err)
	}
	if _, err := NewItem("part", PartItem, "part", "SKU", "Part", "", 1, 100, 0); !errors.Is(err, ErrInvalidItem) {
		t.Fatalf("part without unit error = %v", err)
	}
	if _, err := NewItem("item", ServiceItem, "service", "SVC", "Service", "", math.MaxInt64, 2, 1); err != nil {
		t.Fatal(err)
	} else {
		o.Items = []Item{{ID: "large", Type: ServiceItem, CatalogID: "service", Quantity: math.MaxInt64, UnitPriceInCents: 2}}
		if _, _, err := o.SubmitQuote("overflow", "actor", now); !errors.Is(err, ErrAmountOverflow) {
			t.Fatalf("overflow quote error = %v", err)
		}
	}
}

func TestWorkOrderQuoteAndCompletionGuards(t *testing.T) {
	now := time.Now()
	o, _, _ := NewWorkOrder("wo", "customer", "vehicle", "actor", now)
	if _, err := o.DecideQuote(true, "actor", now); !errors.Is(err, ErrQuoteUnavailable) {
		t.Fatalf("decision without quote error = %v", err)
	}
	if _, err := o.Deliver("actor", now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("deliver before completion error = %v", err)
	}
	_, _ = o.SetDiagnosis("repair", "actor", now)
	part, _ := NewItem("item", PartItem, "part", "SKU", "Part", "piece", 1, 100, 0)
	_, _ = o.AddItem(part, "actor", now)
	_, _, _ = o.SubmitQuote("quote", "actor", now)
	_, _ = o.DecideQuote(true, "actor", now)
	if _, err := o.Complete("actor", now); !errors.Is(err, ErrItemsNotConsumed) {
		t.Fatalf("complete before consumption error = %v", err)
	}
	if _, err := o.Consume("part", 0, "actor", now); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("zero consumption error = %v", err)
	}
	_, _ = o.Consume("part", 1, "actor", now)
	_, _ = o.DecideQuote(true, "actor", now)
	if _, err := o.DecideQuote(false, "actor", now); !errors.Is(err, ErrInvalidTransition) && !errors.Is(err, ErrQuoteUnavailable) {
		t.Fatalf("repeated decision error = %v", err)
	}
	o.RestoreTransition(Completed, "actor", "restored", now)
	if o.Status != Completed || o.UpdatedBy != "actor" || o.UpdatedAt != now {
		t.Fatalf("restore transition did not update order: %#v", o)
	}
}
