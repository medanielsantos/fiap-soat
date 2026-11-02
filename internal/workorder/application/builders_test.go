package application

import (
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func mustItem(t *testing.T, id string, kind domain.ItemType, catalogID string, qty int64) domain.Item {
	t.Helper()
	item, err := domain.NewItem(id, kind, catalogID, "CODE-"+catalogID, "Item "+catalogID, "unit", qty, 1000, 30)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func orderIn(t *testing.T, status domain.Status, items ...domain.Item) domain.WorkOrder {
	t.Helper()
	o, _, err := domain.NewWorkOrder("wo-1", "customer-1", "vehicle-1", "user-1", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if status == domain.Received {
		return o
	}
	if _, err := o.SetDiagnosis("noise", "user-1", testNow); err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if _, err := o.AddItem(item, "user-1", testNow); err != nil {
			t.Fatal(err)
		}
	}
	if status == domain.Diagnosis {
		return o
	}
	if _, _, err := o.SubmitQuote("quote-1", "user-1", testNow); err != nil {
		t.Fatal(err)
	}
	if status == domain.AwaitingApproval {
		return o
	}
	if _, err := o.DecideQuote(true, "client-1", testNow); err != nil {
		t.Fatal(err)
	}
	return o
}
