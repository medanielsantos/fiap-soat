package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

type memoryVehicleStore struct {
	vehicles map[string]domain.Vehicle
}

func newMemoryVehicleStore() *memoryVehicleStore {
	return &memoryVehicleStore{vehicles: make(map[string]domain.Vehicle)}
}

func (store *memoryVehicleStore) Save(_ context.Context, vehicle domain.Vehicle) error {
	store.vehicles[vehicle.ID()] = vehicle
	return nil
}

func (store *memoryVehicleStore) FindByID(_ context.Context, id string) (domain.Vehicle, error) {
	vehicle, ok := store.vehicles[id]
	if !ok {
		return domain.Vehicle{}, ErrVehicleNotFound
	}
	return vehicle, nil
}

func (store *memoryVehicleStore) List(_ context.Context, filter ListFilter, params pagination.Params) (pagination.Page[domain.Vehicle], error) {
	vehicles := make([]domain.Vehicle, 0, len(store.vehicles))
	for _, vehicle := range store.vehicles {
		if filter.CustomerID != "" && vehicle.CustomerID() != filter.CustomerID {
			continue
		}
		vehicles = append(vehicles, vehicle)
	}
	return pagination.NewPage(vehicles, params, int64(len(vehicles))), nil
}

func (store *memoryVehicleStore) Update(_ context.Context, vehicle domain.Vehicle) error {
	if _, ok := store.vehicles[vehicle.ID()]; !ok {
		return ErrVehicleNotFound
	}
	store.vehicles[vehicle.ID()] = vehicle
	return nil
}

func (store *memoryVehicleStore) Delete(_ context.Context, id string) error {
	if _, ok := store.vehicles[id]; !ok {
		return ErrVehicleNotFound
	}
	delete(store.vehicles, id)
	return nil
}

type stubCustomerDirectory struct {
	active map[string]bool
	err    error
}

func (directory stubCustomerDirectory) IsActiveCustomer(_ context.Context, id string) (bool, error) {
	return directory.active[id], directory.err
}

type stubIDGenerator struct {
	id  string
	err error
}

func (generator stubIDGenerator) NewID(context.Context) (string, error) {
	return generator.id, generator.err
}

func activeCustomers(ids ...string) stubCustomerDirectory {
	active := make(map[string]bool, len(ids))
	for _, id := range ids {
		active[id] = true
	}
	return stubCustomerDirectory{active: active}
}

func TestVehicleService(t *testing.T) {
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

	t.Run("test_it_should_not_create_vehicle_when_plate_is_invalid", func(t *testing.T) {
		service := NewVehicleService(newMemoryVehicleStore(), activeCustomers("customer-1"), stubIDGenerator{id: "vehicle-1"})
		_, err := service.Create(context.Background(), VehicleInput{
			CustomerID: "customer-1", Plate: "invalid", Brand: "Fiat", Model: "Uno", Year: 2020,
		})
		if !errors.Is(err, domain.ErrInvalidPlate) {
			t.Fatalf("Create() error = %v, want %v", err, domain.ErrInvalidPlate)
		}
	})

	t.Run("test_it_should_not_create_vehicle_when_customer_is_unavailable", func(t *testing.T) {
		store := newMemoryVehicleStore()
		service := NewVehicleService(store, activeCustomers(), stubIDGenerator{id: "vehicle-1"})
		_, err := service.Create(context.Background(), VehicleInput{
			CustomerID: "customer-1", Plate: "ABC1234", Brand: "Fiat", Model: "Uno", Year: 2020,
		})
		if !errors.Is(err, ErrCustomerUnavailable) {
			t.Fatalf("Create() error = %v, want %v", err, ErrCustomerUnavailable)
		}
		if len(store.vehicles) != 0 {
			t.Error("vehicle was saved for unavailable customer")
		}
	})

	t.Run("test_it_should_propagate_customer_directory_error", func(t *testing.T) {
		directoryErr := errors.New("directory down")
		service := NewVehicleService(newMemoryVehicleStore(), stubCustomerDirectory{err: directoryErr}, stubIDGenerator{id: "vehicle-1"})
		_, err := service.Create(context.Background(), VehicleInput{
			CustomerID: "customer-1", Plate: "ABC1234", Brand: "Fiat", Model: "Uno", Year: 2020,
		})
		if !errors.Is(err, directoryErr) {
			t.Fatalf("Create() error = %v, want %v", err, directoryErr)
		}
	})

	t.Run("test_it_should_update_vehicle_when_vehicle_exists", func(t *testing.T) {
		store := newMemoryVehicleStore()
		service := NewVehicleService(store, activeCustomers("customer-1", "customer-2"), stubIDGenerator{id: "vehicle-1"})
		if _, err := service.Create(context.Background(), VehicleInput{
			CustomerID: "customer-1", Plate: "ABC1234", Brand: "Fiat", Model: "Uno", Year: 2020,
		}); err != nil {
			t.Fatal(err)
		}

		updated, err := service.Update(context.Background(), "vehicle-1", VehicleInput{
			CustomerID: "customer-2", Plate: "ABC1234", Brand: "Fiat", Model: "Mobi", Year: 2021,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.CustomerID() != "customer-2" || updated.Model() != "Mobi" || updated.Year() != 2021 {
			t.Errorf("updated vehicle = %+v", updated)
		}
	})

	t.Run("test_it_should_return_not_found_when_updating_missing_vehicle", func(t *testing.T) {
		service := NewVehicleService(newMemoryVehicleStore(), activeCustomers("customer-1"), stubIDGenerator{id: "unused"})
		_, err := service.Update(context.Background(), "missing", VehicleInput{
			CustomerID: "customer-1", Plate: "ABC1234", Brand: "Fiat", Model: "Uno", Year: 2020,
		})
		if !errors.Is(err, ErrVehicleNotFound) {
			t.Fatalf("Update() error = %v, want %v", err, ErrVehicleNotFound)
		}
	})

	t.Run("test_it_should_list_vehicles_filtered_by_customer", func(t *testing.T) {
		store := newMemoryVehicleStore()
		for _, item := range []struct{ id, customerID, plate string }{
			{"vehicle-1", "customer-1", "ABC1234"},
			{"vehicle-2", "customer-2", "DEF5678"},
		} {
			service := NewVehicleService(store, activeCustomers(item.customerID), stubIDGenerator{id: item.id})
			if _, err := service.Create(context.Background(), VehicleInput{
				CustomerID: item.customerID, Plate: item.plate, Brand: "Fiat", Model: "Uno", Year: 2020,
			}); err != nil {
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
	})

	t.Run("test_it_should_delete_vehicle_when_vehicle_exists", func(t *testing.T) {
		store := newMemoryVehicleStore()
		service := NewVehicleService(store, activeCustomers("customer-1"), stubIDGenerator{id: "vehicle-1"})
		if _, err := service.Create(context.Background(), VehicleInput{
			CustomerID: "customer-1", Plate: "ABC1234", Brand: "Fiat", Model: "Uno", Year: 2020,
		}); err != nil {
			t.Fatal(err)
		}
		if err := service.Delete(context.Background(), "vehicle-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, err := service.Get(context.Background(), "vehicle-1"); !errors.Is(err, ErrVehicleNotFound) {
			t.Fatalf("Get() error = %v, want %v", err, ErrVehicleNotFound)
		}
	})
}
