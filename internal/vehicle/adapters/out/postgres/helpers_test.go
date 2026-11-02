package postgres

import (
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
	"gorm.io/gorm"
)

// openRepository returns a repository over an isolated, migrated schema seeded with
// the active customers customer-1 and customer-2.
func openRepository(t *testing.T) (*VehicleRepository, *gorm.DB) {
	t.Helper()
	db := testdb.Open(t)
	if err := db.Exec(`INSERT INTO customers (id, document, name, email, phone) VALUES
		('customer-1', '52998224725', 'Ana', 'ana@example.com', '1'),
		('customer-2', '11144477735', 'Bia', 'bia@example.com', '2')`).Error; err != nil {
		t.Fatalf("seed customers: %v", err)
	}
	return NewVehicleRepository(db), db
}

func newVehicle(t *testing.T, id, customerID, plateValue string) domain.Vehicle {
	t.Helper()
	plate, err := domain.NewPlate(plateValue)
	if err != nil {
		t.Fatal(err)
	}
	vehicle, err := domain.NewVehicle(id, customerID, plate, "Fiat", "Uno", 2020)
	if err != nil {
		t.Fatal(err)
	}
	return vehicle
}

func firstPage(t *testing.T, perPage int) pagination.Params {
	t.Helper()
	params, err := pagination.NewParams(1, perPage)
	if err != nil {
		t.Fatal(err)
	}
	return params
}
