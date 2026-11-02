package postgres

import (
	"context"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
)

func TestCustomerRepositorySoftDelete(t *testing.T) {
	tx := testdb.Open(t)

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
	params, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	customers, err := repository.List(context.Background(), params)
	if err != nil {
		t.Fatalf("list customers after delete: %v", err)
	}
	if len(customers.Items) != 0 {
		t.Errorf("List() returned %d customers after delete, want 0", len(customers.Items))
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
