package postgres

import (
	"context"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
	"gorm.io/gorm"
)

const (
	cpfAna = "52998224725"
	cnpjCo = "11222333000181"
)

func newTestRepository(t *testing.T) (*CustomerRepository, *gorm.DB) {
	t.Helper()
	db := testdb.Open(t)
	return NewCustomerRepository(db), db
}

func newCustomer(t *testing.T, id, document, email string, status domain.CustomerStatus) domain.Customer {
	t.Helper()
	doc, err := domain.NewDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	customer, err := domain.RestoreCustomerWithPassword(id, doc, "Customer "+id, email, "123", status, "hash-"+id)
	if err != nil {
		t.Fatal(err)
	}
	return customer
}

func saveCustomer(t *testing.T, repository *CustomerRepository, customer domain.Customer) {
	t.Helper()
	if err := repository.Save(context.Background(), customer); err != nil {
		t.Fatalf("save customer %s: %v", customer.ID(), err)
	}
}

func insertVehicle(t *testing.T, db *gorm.DB, id, customerID, plate string) {
	t.Helper()
	if err := db.Exec("INSERT INTO vehicles (id, customer_id, plate, brand, model, year) VALUES (?, ?, ?, 'Fiat', 'Uno', 2010)", id, customerID, plate).Error; err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}
}
