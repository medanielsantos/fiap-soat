package postgres

import (
	"context"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
	"gorm.io/gorm"
)

func newTestRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	db := testdb.Open(t)
	return NewRepository(db), db
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func firstPage(t *testing.T) pagination.Params {
	t.Helper()
	p, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newService(t *testing.T, id, code, name string) domain.Service {
	t.Helper()
	v, err := domain.NewService(id, code, name, "desc", 5000, 30, true, "admin")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func newPart(t *testing.T, id, sku string, stock int64) domain.Part {
	t.Helper()
	v, err := domain.NewPart(id, sku, "P-"+id, "Part "+id, "desc", "un", 500, 1200, stock, 0, true, "admin")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func savePart(t *testing.T, repo *Repository, id string, stock int64) domain.Part {
	t.Helper()
	p := newPart(t, id, "SKU-"+id, stock)
	if err := repo.SavePart(context.Background(), p, nil); err != nil {
		t.Fatalf("SavePart(%s) error = %v", id, err)
	}
	return p
}

func seedWorkOrder(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	statements := []string{
		`INSERT INTO customers (id, document, name, email, phone) VALUES ('cust-1', '52998224725', 'Customer', 'c@example.com', '1')`,
		`INSERT INTO vehicles (id, customer_id, plate, brand, model, year) VALUES ('veh-1', 'cust-1', 'ABC1D23', 'Fiat', 'Uno', 2020)`,
		`INSERT INTO work_orders (id, customer_id, vehicle_id, status) VALUES ('` + id + `', 'cust-1', 'veh-1', 'received')`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("seed work order: %v", err)
		}
	}
}

func stockOf(t *testing.T, repo *Repository, id string) (stock, reserved int64) {
	t.Helper()
	p, err := repo.FindPart(context.Background(), id)
	if err != nil {
		t.Fatalf("FindPart(%s) error = %v", id, err)
	}
	return p.Stock(), p.ReservedStock()
}
