package domain

import (
	"errors"
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
