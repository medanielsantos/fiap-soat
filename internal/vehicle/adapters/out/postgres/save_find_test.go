package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

func TestVehicleRepositorySaveAndFind(t *testing.T) {
	repository, _ := openRepository(t)
	ctx := context.Background()
	if err := repository.Save(ctx, newVehicle(t, "vehicle-1", "customer-1", "ABC1D23")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	found, err := repository.FindByID(ctx, "vehicle-1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.CustomerID() != "customer-1" || found.Plate().String() != "ABC1D23" || found.Brand() != "Fiat" || found.Year() != 2020 {
		t.Errorf("found = %+v", found)
	}
	if _, err := repository.FindByID(ctx, "missing"); !errors.Is(err, application.ErrVehicleNotFound) {
		t.Errorf("FindByID(missing) error = %v, want %v", err, application.ErrVehicleNotFound)
	}
}

func TestVehicleRepositorySaveTranslatesConstraintErrors(t *testing.T) {
	repository, _ := openRepository(t)
	ctx := context.Background()
	if err := repository.Save(ctx, newVehicle(t, "vehicle-1", "customer-1", "ABC1D23")); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(ctx, newVehicle(t, "vehicle-2", "customer-2", "ABC1D23")); !errors.Is(err, application.ErrDuplicatePlate) {
		t.Errorf("duplicate plate error = %v, want %v", err, application.ErrDuplicatePlate)
	}
	if err := repository.Save(ctx, newVehicle(t, "vehicle-3", "ghost", "XYZ9876")); !errors.Is(err, application.ErrCustomerUnavailable) {
		t.Errorf("unknown customer error = %v, want %v", err, application.ErrCustomerUnavailable)
	}
}

func TestVehicleRepositorySoftDeleteReleasesPlate(t *testing.T) {
	repository, db := openRepository(t)
	ctx := context.Background()
	if err := repository.Save(ctx, newVehicle(t, "vehicle-1", "customer-1", "ABC1D23")); err != nil {
		t.Fatal(err)
	}
	if err := repository.Delete(ctx, "vehicle-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.FindByID(ctx, "vehicle-1"); !errors.Is(err, application.ErrVehicleNotFound) {
		t.Errorf("FindByID() after delete error = %v, want %v", err, application.ErrVehicleNotFound)
	}
	var deleted vehicleRecord
	if err := db.Unscoped().First(&deleted, "id = ?", "vehicle-1").Error; err != nil || !deleted.DeletedAt.Valid {
		t.Errorf("soft-deleted row = %+v, %v; want deleted_at set", deleted, err)
	}
	if err := repository.Delete(ctx, "vehicle-1"); !errors.Is(err, application.ErrVehicleNotFound) {
		t.Errorf("second Delete() error = %v, want %v", err, application.ErrVehicleNotFound)
	}
	if err := repository.Save(ctx, newVehicle(t, "vehicle-2", "customer-2", "ABC1D23")); err != nil {
		t.Errorf("save vehicle with previously deleted plate: %v", err)
	}
}

func TestVehicleRepositoryRejectsCorruptStoredPlate(t *testing.T) {
	repository, db := openRepository(t)
	if err := db.Exec(`INSERT INTO vehicles (id, customer_id, plate, brand, model, year) VALUES ('vehicle-1', 'customer-1', '1234567', 'Fiat', 'Uno', 2020)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindByID(context.Background(), "vehicle-1"); !errors.Is(err, domain.ErrInvalidPlate) {
		t.Errorf("FindByID() error = %v, want %v", err, domain.ErrInvalidPlate)
	}
	if _, err := repository.List(context.Background(), application.ListFilter{}, firstPage(t, 10)); !errors.Is(err, domain.ErrInvalidPlate) {
		t.Errorf("List() error = %v, want %v", err, domain.ErrInvalidPlate)
	}
}
