package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

func TestVehicleRepositoryList(t *testing.T) {
	repository, _ := openRepository(t)
	ctx := context.Background()
	for _, v := range []domain.Vehicle{
		newVehicle(t, "vehicle-1", "customer-1", "ABC1D23"),
		newVehicle(t, "vehicle-2", "customer-1", "DEF4G56"),
		newVehicle(t, "vehicle-3", "customer-2", "GHI7J89"),
	} {
		if err := repository.Save(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name      string
		filter    application.ListFilter
		perPage   int
		wantTotal int64
		wantItems int
	}{
		{name: "test_it_should_list_all", perPage: 10, wantTotal: 3, wantItems: 3},
		{name: "test_it_should_limit_page_size", perPage: 2, wantTotal: 3, wantItems: 2},
		{name: "test_it_should_filter_by_customer", filter: application.ListFilter{CustomerID: "customer-1"}, perPage: 10, wantTotal: 2, wantItems: 2},
		{name: "test_it_should_filter_by_vehicle", filter: application.ListFilter{CustomerID: "customer-1", VehicleID: "vehicle-2"}, perPage: 10, wantTotal: 1, wantItems: 1},
		{name: "test_it_should_not_leak_vehicle_of_other_customer", filter: application.ListFilter{CustomerID: "customer-2", VehicleID: "vehicle-1"}, perPage: 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, err := repository.List(ctx, test.filter, firstPage(t, test.perPage))
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if page.TotalItems != test.wantTotal || len(page.Items) != test.wantItems {
				t.Errorf("List() total=%d items=%d, want %d/%d", page.TotalItems, len(page.Items), test.wantTotal, test.wantItems)
			}
		})
	}
}

func TestVehicleRepositoryUpdate(t *testing.T) {
	repository, _ := openRepository(t)
	ctx := context.Background()
	for _, v := range []domain.Vehicle{newVehicle(t, "vehicle-1", "customer-1", "ABC1D23"), newVehicle(t, "vehicle-2", "customer-1", "DEF4G56")} {
		if err := repository.Save(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	plate, _ := domain.NewPlate("XYZ9876")
	moved, err := domain.NewVehicle("vehicle-1", "customer-2", plate, "VW", "Gol", 2015)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(ctx, moved); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	found, err := repository.FindByID(ctx, "vehicle-1")
	if err != nil || found.CustomerID() != "customer-2" || found.Plate().String() != "XYZ9876" || found.Model() != "Gol" || found.Year() != 2015 {
		t.Errorf("found = %+v, %v; want updated vehicle", found, err)
	}

	tests := []struct {
		name    string
		vehicle domain.Vehicle
		wantErr error
	}{
		{name: "test_it_should_return_not_found_for_missing_vehicle", vehicle: newVehicle(t, "missing", "customer-1", "JKL1234"), wantErr: application.ErrVehicleNotFound},
		{name: "test_it_should_reject_plate_of_other_vehicle", vehicle: newVehicle(t, "vehicle-2", "customer-1", "XYZ9876"), wantErr: application.ErrDuplicatePlate},
		{name: "test_it_should_reject_unknown_customer", vehicle: newVehicle(t, "vehicle-2", "ghost", "DEF4G56"), wantErr: application.ErrCustomerUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := repository.Update(ctx, test.vehicle); !errors.Is(err, test.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
