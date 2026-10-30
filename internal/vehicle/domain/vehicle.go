package domain

import (
	"errors"
	"strings"
	"time"
)

const MinVehicleYear = 1900

var (
	ErrInvalidVehicleID  = errors.New("vehicle ID is required")
	ErrInvalidCustomerID = errors.New("vehicle customer ID is required")
	ErrInvalidBrand      = errors.New("vehicle brand is required")
	ErrInvalidModel      = errors.New("vehicle model is required")
	ErrInvalidYear       = errors.New("invalid vehicle year")
)

type Vehicle struct {
	id         string
	customerID string
	plate      Plate
	brand      string
	model      string
	year       int
}

func NewVehicle(id, customerID string, plate Plate, brand, model string, year int) (Vehicle, error) {
	id = strings.TrimSpace(id)
	customerID = strings.TrimSpace(customerID)
	brand = strings.TrimSpace(brand)
	model = strings.TrimSpace(model)

	switch {
	case id == "":
		return Vehicle{}, ErrInvalidVehicleID
	case customerID == "":
		return Vehicle{}, ErrInvalidCustomerID
	case !isValidPlate(plate):
		return Vehicle{}, ErrInvalidPlate
	case brand == "":
		return Vehicle{}, ErrInvalidBrand
	case model == "":
		return Vehicle{}, ErrInvalidModel
	case year < MinVehicleYear || year > time.Now().Year()+1:
		return Vehicle{}, ErrInvalidYear
	}

	return Vehicle{
		id:         id,
		customerID: customerID,
		plate:      plate,
		brand:      brand,
		model:      model,
		year:       year,
	}, nil
}

func isValidPlate(plate Plate) bool {
	_, err := NewPlate(plate.String())
	return err == nil
}

func (v Vehicle) ID() string         { return v.id }
func (v Vehicle) CustomerID() string { return v.customerID }
func (v Vehicle) Plate() Plate       { return v.plate }
func (v Vehicle) Brand() string      { return v.brand }
func (v Vehicle) Model() string      { return v.model }
func (v Vehicle) Year() int          { return v.year }
