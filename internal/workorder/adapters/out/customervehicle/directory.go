package customervehicle

import (
	"context"

	customerapp "github.com/medanielsantos/fiap-soat/internal/customer/application"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
)

type Directory struct {
	customers *customerapp.CustomerService
	vehicles  *vehicleapp.VehicleService
}

func NewDirectory(customers *customerapp.CustomerService, vehicles *vehicleapp.VehicleService) *Directory {
	return &Directory{customers: customers, vehicles: vehicles}
}
func (d *Directory) Validate(ctx context.Context, customerID, vehicleID string) error {
	c, e := d.customers.Get(ctx, customerID)
	if e != nil || !c.IsActive() {
		return application.ErrCustomerOrVehicleUnavailable
	}
	v, e := d.vehicles.Get(ctx, vehicleID)
	if e != nil || v.CustomerID() != customerID {
		return application.ErrCustomerOrVehicleUnavailable
	}
	return nil
}
