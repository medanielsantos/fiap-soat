package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewVehicle(t *testing.T) {
	plate, err := NewPlate("ABC1D23")
	if err != nil {
		t.Fatal(err)
	}
	nextYear := time.Now().Year() + 1
	tests := []struct {
		name       string
		id         string
		customerID string
		plate      Plate
		brand      string
		model      string
		year       int
		want       error
	}{
		{name: "test_it_should_create_vehicle_when_data_is_valid", id: "id-1", customerID: "customer-1", plate: plate, brand: " Fiat ", model: "Uno", year: 2020},
		{name: "test_it_should_accept_next_year_model", id: "id-1", customerID: "customer-1", plate: plate, brand: "Fiat", model: "Uno", year: nextYear},
		{name: "test_it_should_reject_vehicle_without_id", customerID: "customer-1", plate: plate, brand: "Fiat", model: "Uno", year: 2020, want: ErrInvalidVehicleID},
		{name: "test_it_should_reject_vehicle_without_customer", id: "id-1", plate: plate, brand: "Fiat", model: "Uno", year: 2020, want: ErrInvalidCustomerID},
		{name: "test_it_should_reject_vehicle_with_zero_plate", id: "id-1", customerID: "customer-1", brand: "Fiat", model: "Uno", year: 2020, want: ErrInvalidPlate},
		{name: "test_it_should_reject_vehicle_without_brand", id: "id-1", customerID: "customer-1", plate: plate, model: "Uno", year: 2020, want: ErrInvalidBrand},
		{name: "test_it_should_reject_vehicle_without_model", id: "id-1", customerID: "customer-1", plate: plate, brand: "Fiat", year: 2020, want: ErrInvalidModel},
		{name: "test_it_should_reject_vehicle_before_min_year", id: "id-1", customerID: "customer-1", plate: plate, brand: "Fiat", model: "Uno", year: MinVehicleYear - 1, want: ErrInvalidYear},
		{name: "test_it_should_reject_vehicle_after_next_year", id: "id-1", customerID: "customer-1", plate: plate, brand: "Fiat", model: "Uno", year: nextYear + 1, want: ErrInvalidYear},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vehicle, err := NewVehicle(test.id, test.customerID, test.plate, test.brand, test.model, test.year)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewVehicle() error = %v, want %v", err, test.want)
			}
			if err == nil && vehicle.Brand() != "Fiat" {
				t.Errorf("vehicle brand = %q, want trimmed brand", vehicle.Brand())
			}
		})
	}
}
