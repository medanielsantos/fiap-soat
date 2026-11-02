package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
	"gorm.io/gorm"
)

var baseTime = time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)

// openRepository returns a repository over an isolated schema seeded with
// customers customer-1/customer-2 and vehicles vehicle-1, vehicle-2 (customer-1) and vehicle-3 (customer-2).
func openRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	db := testdb.Open(t)
	seeds := []string{
		`INSERT INTO customers (id, document, name, email, phone) VALUES
			('customer-1', '52998224725', 'Ana', 'ana@example.com', '1'),
			('customer-2', '11144477735', 'Bia', 'bia@example.com', '2')`,
		`INSERT INTO vehicles (id, customer_id, plate, brand, model, year) VALUES
			('vehicle-1', 'customer-1', 'ABC1D23', 'Fiat', 'Uno', 2020),
			('vehicle-2', 'customer-1', 'DEF4G56', 'Fiat', 'Mobi', 2021),
			('vehicle-3', 'customer-2', 'GHI7J89', 'VW', 'Gol', 2019)`,
	}
	for _, seed := range seeds {
		if err := db.Exec(seed).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return NewRepository(db), db
}

type orderSpec struct {
	id, customerID, vehicleID string
	createdAt                 time.Time
}

// createOrder persists a received order and returns it with its creation event.
func createOrder(t *testing.T, repo *Repository, spec orderSpec) domain.WorkOrder {
	t.Helper()
	if spec.createdAt.IsZero() {
		spec.createdAt = baseTime
	}
	o, event, err := domain.NewWorkOrder(spec.id, spec.customerID, spec.vehicleID, "user-1", spec.createdAt)
	if err != nil {
		t.Fatal(err)
	}
	event.ID = spec.id + "-created"
	o.History = append(o.History, event)
	if err := repo.Create(context.Background(), o, event); err != nil {
		t.Fatalf("Create(%s) error = %v", spec.id, err)
	}
	return o
}

func newItem(t *testing.T, id string, kind domain.ItemType, catalogID string, qty int64) domain.Item {
	t.Helper()
	item, err := domain.NewItem(id, kind, catalogID, "CODE-"+catalogID, "Item "+catalogID, "pair", qty, 1500, 20)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// apply runs a domain transition and persists it through Save with a unique event ID.
func apply(t *testing.T, repo *Repository, o *domain.WorkOrder, step string, fn func(*domain.WorkOrder) (domain.HistoryEntry, error)) {
	t.Helper()
	event, err := fn(o)
	if err != nil {
		t.Fatalf("%s: %v", step, err)
	}
	event.ID = fmt.Sprintf("%s-%s", o.ID, step)
	o.History = append(o.History, event)
	if err := repo.Save(context.Background(), *o, event); err != nil {
		t.Fatalf("Save(%s) error = %v", step, err)
	}
}
