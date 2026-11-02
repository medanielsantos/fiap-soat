package application

import (
	"context"

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

type failingStore struct {
	*memoryVehicleStore
	err error
}

func (store failingStore) Save(context.Context, domain.Vehicle) error   { return store.err }
func (store failingStore) Update(context.Context, domain.Vehicle) error { return store.err }

func validInput(customerID string) VehicleInput {
	return VehicleInput{CustomerID: customerID, Plate: "ABC1234", Brand: "Fiat", Model: "Uno", Year: 2020}
}
