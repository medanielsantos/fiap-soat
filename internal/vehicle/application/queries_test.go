package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestVehicleServiceList(t *testing.T) {
	store := newMemoryVehicleStore()
	for _, item := range []struct{ id, customerID, plate string }{
		{"vehicle-1", "customer-1", "ABC1234"},
		{"vehicle-2", "customer-2", "DEF5678"},
	} {
		service := NewVehicleService(store, activeCustomers(item.customerID), stubIDGenerator{id: item.id})
		input := validInput(item.customerID)
		input.Plate = item.plate
		if _, err := service.Create(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	service := NewVehicleService(store, activeCustomers(), stubIDGenerator{})

	page, err := service.List(context.Background(), ListFilter{CustomerID: "customer-2"}, pagination.Params{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.TotalItems != 1 || page.Items[0].ID() != "vehicle-2" {
		t.Errorf("page = %+v, want only vehicle-2", page)
	}
}

func TestVehicleServiceGetAndDelete(t *testing.T) {
	store := seededStore(t)
	service := NewVehicleService(store, activeCustomers(), stubIDGenerator{})

	vehicle, err := service.Get(context.Background(), "vehicle-1")
	if err != nil || vehicle.Plate().String() != "ABC1234" {
		t.Fatalf("Get() = %+v, %v", vehicle, err)
	}
	if err := service.Delete(context.Background(), "vehicle-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(context.Background(), "vehicle-1"); !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("Get() after delete error = %v, want %v", err, ErrVehicleNotFound)
	}
	if err := service.Delete(context.Background(), "vehicle-1"); !errors.Is(err, ErrVehicleNotFound) {
		t.Errorf("second Delete() error = %v, want %v", err, ErrVehicleNotFound)
	}
}
