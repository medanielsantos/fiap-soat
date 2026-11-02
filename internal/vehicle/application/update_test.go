package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

func seededStore(t *testing.T) *memoryVehicleStore {
	t.Helper()
	store := newMemoryVehicleStore()
	service := NewVehicleService(store, activeCustomers("customer-1"), stubIDGenerator{id: "vehicle-1"})
	if _, err := service.Create(context.Background(), validInput("customer-1")); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestVehicleServiceUpdate(t *testing.T) {
	t.Run("test_it_should_update_vehicle_when_vehicle_exists", func(t *testing.T) {
		store := seededStore(t)
		service := NewVehicleService(store, activeCustomers("customer-1", "customer-2"), stubIDGenerator{})
		updated, err := service.Update(context.Background(), "vehicle-1", VehicleInput{
			CustomerID: "customer-2", Plate: "ABC1234", Brand: "Fiat", Model: "Mobi", Year: 2021,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.CustomerID() != "customer-2" || updated.Model() != "Mobi" || updated.Year() != 2021 {
			t.Errorf("updated vehicle = %+v", updated)
		}
		if store.vehicles["vehicle-1"].Model() != "Mobi" {
			t.Error("update was not persisted")
		}
	})

	invalidPlate := validInput("customer-1")
	invalidPlate.Plate = "12"
	blankBrand := validInput("customer-1")
	blankBrand.Brand = " "
	directoryErr := errors.New("directory down")
	storeErr := errors.New("write failed")

	tests := []struct {
		name      string
		id        string
		input     VehicleInput
		customers stubCustomerDirectory
		storeErr  error
		wantErr   error
	}{
		{name: "test_it_should_return_not_found_when_updating_missing_vehicle", id: "missing", input: validInput("customer-1"), customers: activeCustomers("customer-1"), wantErr: ErrVehicleNotFound},
		{name: "test_it_should_reject_invalid_plate", id: "vehicle-1", input: invalidPlate, customers: activeCustomers("customer-1"), wantErr: domain.ErrInvalidPlate},
		{name: "test_it_should_reject_blank_brand", id: "vehicle-1", input: blankBrand, customers: activeCustomers("customer-1"), wantErr: domain.ErrInvalidBrand},
		{name: "test_it_should_reject_transfer_to_inactive_customer", id: "vehicle-1", input: validInput("customer-9"), customers: activeCustomers("customer-1"), wantErr: ErrCustomerUnavailable},
		{name: "test_it_should_propagate_customer_directory_error", id: "vehicle-1", input: validInput("customer-1"), customers: stubCustomerDirectory{err: directoryErr}, wantErr: directoryErr},
		{name: "test_it_should_propagate_store_error", id: "vehicle-1", input: validInput("customer-1"), customers: activeCustomers("customer-1"), storeErr: storeErr, wantErr: storeErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory := seededStore(t)
			var store VehicleStore = memory
			if test.storeErr != nil {
				store = failingStore{memory, test.storeErr}
			}
			_, err := NewVehicleService(store, test.customers, stubIDGenerator{}).Update(context.Background(), test.id, test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, test.wantErr)
			}
			if got := memory.vehicles["vehicle-1"]; got.Brand() != "Fiat" || got.CustomerID() != "customer-1" {
				t.Errorf("stored vehicle changed on failed update: %+v", got)
			}
		})
	}
}
