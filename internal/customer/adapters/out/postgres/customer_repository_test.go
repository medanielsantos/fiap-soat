package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCustomerRepositorySoftDelete(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL soft-delete integration test")
	}
	db, err := gorm.Open(postgresdriver.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	if err := tx.Exec(`CREATE TEMP TABLE customers (
		id TEXT PRIMARY KEY,
		document VARCHAR(14) NOT NULL,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone TEXT NOT NULL,
		status VARCHAR(8) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		deleted_at TIMESTAMPTZ
	)`).Error; err != nil {
		t.Fatalf("create temporary customers table: %v", err)
	}
	if err := tx.Exec("CREATE UNIQUE INDEX ux_customers_document ON customers (document) WHERE deleted_at IS NULL").Error; err != nil {
		t.Fatalf("create temporary document index: %v", err)
	}

	document, err := domain.NewDocument("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	customer, err := domain.NewCustomer("soft-delete-test", document, "Test Customer", "test@example.com", "123")
	if err != nil {
		t.Fatal(err)
	}
	repository := NewCustomerRepository(tx)
	if err := repository.Save(context.Background(), customer); err != nil {
		t.Fatalf("save customer: %v", err)
	}
	if err := repository.Delete(context.Background(), customer.ID()); err != nil {
		t.Fatalf("soft delete customer: %v", err)
	}
	if _, err := repository.FindByID(context.Background(), customer.ID()); err != application.ErrCustomerNotFound {
		t.Errorf("FindByID() after delete error = %v, want %v", err, application.ErrCustomerNotFound)
	}
	customers, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list customers after delete: %v", err)
	}
	if len(customers) != 0 {
		t.Errorf("List() returned %d customers after delete, want 0", len(customers))
	}
	var deleted customerRecord
	if err := tx.Unscoped().First(&deleted, "id = ?", customer.ID()).Error; err != nil {
		t.Fatalf("load soft-deleted customer: %v", err)
	}
	if !deleted.DeletedAt.Valid {
		t.Error("deleted_at was not set")
	}

	replacement, err := domain.NewCustomer("soft-delete-replacement", document, "Replacement Customer", "replacement@example.com", "456")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), replacement); err != nil {
		t.Errorf("save replacement with previously deleted document: %v", err)
	}
}
