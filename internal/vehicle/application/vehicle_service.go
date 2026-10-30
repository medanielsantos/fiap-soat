package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

var (
	ErrVehicleNotFound     = errors.New("vehicle not found")
	ErrDuplicatePlate      = errors.New("vehicle plate already exists")
	ErrCustomerUnavailable = errors.New("customer not found or inactive")
)

type VehicleStore interface {
	Save(context.Context, domain.Vehicle) error
	FindByID(context.Context, string) (domain.Vehicle, error)
	List(context.Context, ListFilter, pagination.Params) (pagination.Page[domain.Vehicle], error)
	Update(context.Context, domain.Vehicle) error
	Delete(context.Context, string) error
}

// CustomerDirectory is the port used to check the owner without importing the customer module.
type CustomerDirectory interface {
	IsActiveCustomer(ctx context.Context, id string) (bool, error)
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type ListFilter struct {
	CustomerID string
}

type VehicleService struct {
	store       VehicleStore
	customers   CustomerDirectory
	idGenerator IDGenerator
}

func NewVehicleService(store VehicleStore, customers CustomerDirectory, idGenerator IDGenerator) *VehicleService {
	return &VehicleService{store: store, customers: customers, idGenerator: idGenerator}
}

type VehicleInput struct {
	CustomerID string
	Plate      string
	Brand      string
	Model      string
	Year       int
}

func (service *VehicleService) Create(ctx context.Context, input VehicleInput) (domain.Vehicle, error) {
	plate, err := domain.NewPlate(input.Plate)
	if err != nil {
		return domain.Vehicle{}, err
	}
	id, err := service.idGenerator.NewID(ctx)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("generate vehicle ID: %w", err)
	}
	vehicle, err := domain.NewVehicle(id, input.CustomerID, plate, input.Brand, input.Model, input.Year)
	if err != nil {
		return domain.Vehicle{}, err
	}
	if err := service.ensureActiveCustomer(ctx, vehicle.CustomerID()); err != nil {
		return domain.Vehicle{}, err
	}
	if err := service.store.Save(ctx, vehicle); err != nil {
		return domain.Vehicle{}, err
	}
	return vehicle, nil
}

func (service *VehicleService) Get(ctx context.Context, id string) (domain.Vehicle, error) {
	return service.store.FindByID(ctx, id)
}

func (service *VehicleService) List(ctx context.Context, filter ListFilter, params pagination.Params) (pagination.Page[domain.Vehicle], error) {
	return service.store.List(ctx, filter, params)
}

func (service *VehicleService) Update(ctx context.Context, id string, input VehicleInput) (domain.Vehicle, error) {
	if _, err := service.store.FindByID(ctx, id); err != nil {
		return domain.Vehicle{}, err
	}
	plate, err := domain.NewPlate(input.Plate)
	if err != nil {
		return domain.Vehicle{}, err
	}
	vehicle, err := domain.NewVehicle(id, input.CustomerID, plate, input.Brand, input.Model, input.Year)
	if err != nil {
		return domain.Vehicle{}, err
	}
	if err := service.ensureActiveCustomer(ctx, vehicle.CustomerID()); err != nil {
		return domain.Vehicle{}, err
	}
	if err := service.store.Update(ctx, vehicle); err != nil {
		return domain.Vehicle{}, err
	}
	return vehicle, nil
}

func (service *VehicleService) Delete(ctx context.Context, id string) error {
	return service.store.Delete(ctx, id)
}

func (service *VehicleService) ensureActiveCustomer(ctx context.Context, customerID string) error {
	active, err := service.customers.IsActiveCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("check vehicle customer: %w", err)
	}
	if !active {
		return ErrCustomerUnavailable
	}
	return nil
}
