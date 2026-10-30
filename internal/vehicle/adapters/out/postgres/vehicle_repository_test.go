package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestVehicleRepositoryPlateUniqueness(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL vehicle integration test")
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
	if err := tx.Exec(`CREATE TEMP TABLE vehicles (
		id TEXT PRIMARY KEY,
		customer_id TEXT NOT NULL,
		plate VARCHAR(7) NOT NULL,
		brand TEXT NOT NULL,
		model TEXT NOT NULL,
		year INTEGER NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		deleted_at TIMESTAMPTZ
	)`).Error; err != nil {
		t.Fatalf("create temporary vehicles table: %v", err)
	}
	if err := tx.Exec("CREATE UNIQUE INDEX ux_vehicles_plate ON vehicles (plate) WHERE deleted_at IS NULL").Error; err != nil {
		t.Fatalf("create temporary plate index: %v", err)
	}

	plate, err := domain.NewPlate("ABC1D23")
	if err != nil {
		t.Fatal(err)
	}
	newVehicle := func(id, customerID string) domain.Vehicle {
		vehicle, err := domain.NewVehicle(id, customerID, plate, "Fiat", "Uno", 2020)
		if err != nil {
			t.Fatal(err)
		}
		return vehicle
	}
	repository := NewVehicleRepository(tx)
	ctx := context.Background()

	if err := repository.Save(ctx, newVehicle("vehicle-1", "customer-1")); err != nil {
		t.Fatalf("save vehicle: %v", err)
	}
	if err := tx.Exec("SAVEPOINT duplicate_plate").Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(ctx, newVehicle("vehicle-2", "customer-2")); !errors.Is(err, application.ErrDuplicatePlate) {
		t.Errorf("Save() duplicate plate error = %v, want %v", err, application.ErrDuplicatePlate)
	}
	if err := tx.Exec("ROLLBACK TO SAVEPOINT duplicate_plate").Error; err != nil {
		t.Fatal(err)
	}

	params, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	page, err := repository.List(ctx, application.ListFilter{CustomerID: "customer-1"}, params)
	if err != nil {
		t.Fatalf("list vehicles by customer: %v", err)
	}
	if page.TotalItems != 1 {
		t.Errorf("List() total = %d, want 1", page.TotalItems)
	}

	if err := repository.Delete(ctx, "vehicle-1"); err != nil {
		t.Fatalf("soft delete vehicle: %v", err)
	}
	if _, err := repository.FindByID(ctx, "vehicle-1"); !errors.Is(err, application.ErrVehicleNotFound) {
		t.Errorf("FindByID() after delete error = %v, want %v", err, application.ErrVehicleNotFound)
	}
	if err := repository.Save(ctx, newVehicle("vehicle-3", "customer-2")); err != nil {
		t.Errorf("save vehicle with previously deleted plate: %v", err)
	}
}
