package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func quotedOrder(t *testing.T, repo *Repository) domain.WorkOrder {
	t.Helper()
	o := createOrder(t, repo, orderSpec{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"})
	apply(t, repo, &o, "diagnosis", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.SetDiagnosis("noise", "user-1", baseTime)
	})
	apply(t, repo, &o, "item", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.AddItem(newItem(t, "item-1", domain.PartItem, "part-1", 1), "user-1", baseTime)
	})
	apply(t, repo, &o, "quote", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		_, e, err := o.SubmitQuote("quote-1", "user-1", baseTime)
		return e, err
	})
	return o
}

func find(repo *Repository, _ domain.WorkOrder) error {
	_, err := repo.Find(context.Background(), "wo-1")
	return err
}

func list(repo *Repository, _ domain.WorkOrder) error {
	p, err := pagination.NewParams(1, 10)
	if err != nil {
		return err
	}
	_, err = repo.List(context.Background(), "", p)
	return err
}

func save(repo *Repository, o domain.WorkOrder) error {
	return repo.Save(context.Background(), o, domain.HistoryEntry{ID: "evt-x", To: o.Status, UserID: "user-1", CreatedAt: baseTime})
}

func TestRepositorySurfacesDatabaseErrors(t *testing.T) {
	tests := []struct {
		name string
		ddl  string
		op   func(*Repository, domain.WorkOrder) error
	}{
		{name: "find_order_query", ddl: "DROP TABLE work_orders CASCADE", op: find},
		{name: "find_items_query", ddl: "DROP TABLE work_order_items", op: find},
		{name: "find_quotes_query", ddl: "DROP TABLE work_order_quotes CASCADE", op: find},
		{name: "find_quote_items_query", ddl: "DROP TABLE work_order_quote_items", op: find},
		{name: "find_history_query", ddl: "DROP TABLE work_order_status_history", op: find},
		{name: "list_count_query", ddl: "DROP TABLE work_orders CASCADE", op: list},
		{name: "list_rows_query", ddl: "ALTER TABLE work_orders RENAME COLUMN created_at TO opened_at", op: list},
		{name: "list_load_query", ddl: "DROP TABLE work_order_items", op: list},
		{name: "save_update_query", ddl: "DROP TABLE work_orders CASCADE", op: save},
		{name: "save_delete_items_query", ddl: "DROP TABLE work_order_items", op: save},
		{name: "save_quote_lookup_query", ddl: "DROP TABLE work_order_quotes CASCADE", op: save},
		{name: "save_quote_update_query", ddl: "ALTER TABLE work_order_quotes RENAME COLUMN decided_at TO decided_on", op: save},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, db := openRepository(t)
			o := quotedOrder(t, repo)
			if err := db.Exec(test.ddl).Error; err != nil {
				t.Fatalf("break schema: %v", err)
			}
			err := test.op(repo, o)
			if err == nil || errors.Is(err, application.ErrNotFound) {
				t.Errorf("error = %v, want underlying database error", err)
			}
		})
	}
}
