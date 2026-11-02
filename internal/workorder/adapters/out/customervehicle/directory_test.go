package customervehicle

import (
	"context"
	"errors"
	"testing"

	customerapp "github.com/medanielsantos/fiap-soat/internal/customer/application"
	customerdomain "github.com/medanielsantos/fiap-soat/internal/customer/domain"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	vehicledomain "github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
)

// Only FindByID is exercised; other store methods panic through the nil embedded interface.
type customerStore struct {
	customerapp.CustomerStore
	customer customerdomain.Customer
	err      error
}

func (s customerStore) FindByID(context.Context, string) (customerdomain.Customer, error) {
	return s.customer, s.err
}

type vehicleStore struct {
	vehicleapp.VehicleStore
	vehicle vehicledomain.Vehicle
	err     error
}

func (s vehicleStore) FindByID(context.Context, string) (vehicledomain.Vehicle, error) {
	return s.vehicle, s.err
}

func TestDirectoryValidate(t *testing.T) {
	document, err := customerdomain.NewDocument("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	active, err := customerdomain.NewCustomer("customer-1", document, "Ana", "ana@example.com", "123")
	if err != nil {
		t.Fatal(err)
	}
	inactive := active
	inactive.Deactivate()
	plate, err := vehicledomain.NewPlate("ABC1D23")
	if err != nil {
		t.Fatal(err)
	}
	owned, err := vehicledomain.NewVehicle("vehicle-1", "customer-1", plate, "Fiat", "Uno", 2020)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := vehicledomain.NewVehicle("vehicle-1", "customer-2", plate, "Fiat", "Uno", 2020)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		customers customerStore
		vehicles  vehicleStore
		wantErr   error
	}{
		{name: "test_it_should_accept_active_customer_owning_vehicle", customers: customerStore{customer: active}, vehicles: vehicleStore{vehicle: owned}},
		{name: "test_it_should_reject_missing_customer", customers: customerStore{err: customerapp.ErrCustomerNotFound}, vehicles: vehicleStore{vehicle: owned}, wantErr: application.ErrCustomerOrVehicleUnavailable},
		{name: "test_it_should_reject_inactive_customer", customers: customerStore{customer: inactive}, vehicles: vehicleStore{vehicle: owned}, wantErr: application.ErrCustomerOrVehicleUnavailable},
		{name: "test_it_should_reject_missing_vehicle", customers: customerStore{customer: active}, vehicles: vehicleStore{err: vehicleapp.ErrVehicleNotFound}, wantErr: application.ErrCustomerOrVehicleUnavailable},
		{name: "test_it_should_reject_vehicle_of_other_customer", customers: customerStore{customer: active}, vehicles: vehicleStore{vehicle: foreign}, wantErr: application.ErrCustomerOrVehicleUnavailable},
		{name: "test_it_should_mask_unexpected_lookup_errors", customers: customerStore{err: errors.New("db down")}, wantErr: application.ErrCustomerOrVehicleUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := NewDirectory(
				customerapp.NewCustomerService(test.customers, nil),
				vehicleapp.NewVehicleService(test.vehicles, nil, nil),
			)
			err := directory.Validate(context.Background(), "customer-1", "vehicle-1")
			if !errors.Is(err, test.wantErr) || (test.wantErr == nil && err != nil) {
				t.Fatalf("Validate() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
