package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

func TestVehicleServiceCreate(t *testing.T) {
	t.Run("test_it_should_create_vehicle_and_normalize_plate_when_input_is_valid", func(t *testing.T) {
		store := newMemoryVehicleStore()
		service := NewVehicleService(store, activeCustomers("customer-1"), stubIDGenerator{id: "vehicle-1"})

		vehicle, err := service.Create(context.Background(), VehicleInput{
			CustomerID: "customer-1", Plate: "abc-1d23", Brand: "Fiat", Model: "Uno", Year: 2020,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if vehicle.ID() != "vehicle-1" || vehicle.Plate().String() != "ABC1D23" {
			t.Errorf("vehicle = %+v, want vehicle-1 with normalized plate", vehicle)
		}
		if _, ok := store.vehicles["vehicle-1"]; !ok {
			t.Error("vehicle was not saved")
		}
	})

	t.Run("test_it_should_not_create_vehicle_when_customer_is_unavailable", func(t *testing.T) {
		store := newMemoryVehicleStore()
		service := NewVehicleService(store, activeCustomers(), stubIDGenerator{id: "vehicle-1"})
		if _, err := service.Create(context.Background(), validInput("customer-1")); !errors.Is(err, ErrCustomerUnavailable) {
			t.Fatalf("Create() error = %v, want %v", err, ErrCustomerUnavailable)
		}
		if len(store.vehicles) != 0 {
			t.Error("vehicle was saved for unavailable customer")
		}
	})

	directoryErr := errors.New("directory down")
	idErr := errors.New("id source down")
	storeErr := errors.New("duplicate")
	invalidPlate := validInput("customer-1")
	invalidPlate.Plate = "invalid"
	invalidYear := validInput("customer-1")
	invalidYear.Year = 1800

	tests := []struct {
		name      string
		input     VehicleInput
		store     VehicleStore
		customers stubCustomerDirectory
		ids       stubIDGenerator
		wantErr   error
	}{
		{name: "test_it_should_not_create_vehicle_when_plate_is_invalid", input: invalidPlate, customers: activeCustomers("customer-1"), ids: stubIDGenerator{id: "vehicle-1"}, wantErr: domain.ErrInvalidPlate},
		{name: "test_it_should_wrap_id_generation_error", input: validInput("customer-1"), customers: activeCustomers("customer-1"), ids: stubIDGenerator{err: idErr}, wantErr: idErr},
		{name: "test_it_should_reject_invalid_year", input: invalidYear, customers: activeCustomers("customer-1"), ids: stubIDGenerator{id: "vehicle-1"}, wantErr: domain.ErrInvalidYear},
		{name: "test_it_should_propagate_customer_directory_error", input: validInput("customer-1"), customers: stubCustomerDirectory{err: directoryErr}, ids: stubIDGenerator{id: "vehicle-1"}, wantErr: directoryErr},
		{name: "test_it_should_propagate_store_error", input: validInput("customer-1"), store: failingStore{newMemoryVehicleStore(), storeErr}, customers: activeCustomers("customer-1"), ids: stubIDGenerator{id: "vehicle-1"}, wantErr: storeErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := test.store
			if store == nil {
				store = newMemoryVehicleStore()
			}
			vehicle, err := NewVehicleService(store, test.customers, test.ids).Create(context.Background(), test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, test.wantErr)
			}
			if vehicle.ID() != "" {
				t.Errorf("Create() returned vehicle %q on error", vehicle.ID())
			}
		})
	}
}
