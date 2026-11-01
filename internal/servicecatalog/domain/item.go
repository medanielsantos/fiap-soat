package domain

import (
	"errors"
	"strings"
)

var (
	ErrInvalidID            = errors.New("item ID is required")
	ErrInvalidName          = errors.New("item name is required")
	ErrInvalidInternalCode  = errors.New("internal code is required")
	ErrInvalidSKU           = errors.New("SKU is required")
	ErrInvalidPrice         = errors.New("sale price must be greater than zero")
	ErrInvalidPurchasePrice = errors.New("purchase price cannot be negative")
	ErrInvalidDuration      = errors.New("estimated duration must be positive")
	ErrInvalidUnit          = errors.New("part unit is required")
	ErrInvalidStockChange   = errors.New("stock movement quantity must be positive")
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrInvalidMovement      = errors.New("movement type must be entry or exit")
	ErrPartHasStock         = errors.New("part cannot be deleted while stock is on hand or reserved")
)

type Service struct {
	id, internalCode, name, description, performedBy string
	salePriceInCents                                 int64
	estimatedDurationMinutes                         int
	active                                           bool
}

func NewService(id, internalCode, name, description string, salePriceInCents int64, estimatedDurationMinutes int, active bool, performedBy string) (Service, error) {
	id, internalCode, name = strings.TrimSpace(id), strings.TrimSpace(internalCode), strings.TrimSpace(name)
	if id == "" {
		return Service{}, ErrInvalidID
	}
	if internalCode == "" {
		return Service{}, ErrInvalidInternalCode
	}
	if name == "" {
		return Service{}, ErrInvalidName
	}
	if salePriceInCents <= 0 {
		return Service{}, ErrInvalidPrice
	}
	if estimatedDurationMinutes <= 0 {
		return Service{}, ErrInvalidDuration
	}
	return Service{id: id, internalCode: internalCode, name: name, description: strings.TrimSpace(description), salePriceInCents: salePriceInCents, estimatedDurationMinutes: estimatedDurationMinutes, active: active, performedBy: strings.TrimSpace(performedBy)}, nil
}
func (s Service) ID() string                    { return s.id }
func (s Service) InternalCode() string          { return s.internalCode }
func (s Service) Name() string                  { return s.name }
func (s Service) Description() string           { return s.description }
func (s Service) SalePriceInCents() int64       { return s.salePriceInCents }
func (s Service) EstimatedDurationMinutes() int { return s.estimatedDurationMinutes }
func (s Service) Active() bool                  { return s.active }
func (s Service) PerformedBy() string           { return s.performedBy }

type Part struct {
	id, sku, internalCode, name, description, unit, performedBy string
	purchasePriceInCents, salePriceInCents                      int64
	stock, reservedStock                                        int64
	active                                                      bool
}

func NewPart(id, sku, internalCode, name, description, unit string, purchasePriceInCents, salePriceInCents, stock, reservedStock int64, active bool, performedBy string) (Part, error) {
	id, sku, internalCode, name, unit = strings.TrimSpace(id), strings.TrimSpace(sku), strings.TrimSpace(internalCode), strings.TrimSpace(name), strings.TrimSpace(unit)
	if id == "" {
		return Part{}, ErrInvalidID
	}
	if sku == "" {
		return Part{}, ErrInvalidSKU
	}
	if internalCode == "" {
		return Part{}, ErrInvalidInternalCode
	}
	if name == "" {
		return Part{}, ErrInvalidName
	}
	if purchasePriceInCents < 0 {
		return Part{}, ErrInvalidPurchasePrice
	}
	if salePriceInCents <= 0 {
		return Part{}, ErrInvalidPrice
	}
	if unit == "" {
		return Part{}, ErrInvalidUnit
	}
	if stock < 0 {
		return Part{}, ErrInsufficientStock
	}
	if reservedStock < 0 || reservedStock > stock {
		return Part{}, ErrInsufficientStock
	}
	return Part{id: id, sku: sku, internalCode: internalCode, name: name, description: strings.TrimSpace(description), unit: unit, purchasePriceInCents: purchasePriceInCents, salePriceInCents: salePriceInCents, stock: stock, reservedStock: reservedStock, active: active, performedBy: strings.TrimSpace(performedBy)}, nil
}
func (p Part) ID() string                  { return p.id }
func (p Part) SKU() string                 { return p.sku }
func (p Part) InternalCode() string        { return p.internalCode }
func (p Part) Name() string                { return p.name }
func (p Part) Description() string         { return p.description }
func (p Part) Unit() string                { return p.unit }
func (p Part) PurchasePriceInCents() int64 { return p.purchasePriceInCents }
func (p Part) SalePriceInCents() int64     { return p.salePriceInCents }
func (p Part) Stock() int64                { return p.stock }
func (p Part) ReservedStock() int64        { return p.reservedStock }
func (p Part) AvailableStock() int64       { return p.stock - p.reservedStock }
func (p Part) Active() bool                { return p.active }
func (p Part) PerformedBy() string         { return p.performedBy }

type MovementType string

const (
	MovementEntry MovementType = "entry"
	MovementExit  MovementType = "exit"
)

type StockMovement struct {
	id, partID, description, performedBy string
	movementType                         MovementType
	quantity                             int64
}

func NewStockMovement(id, partID string, movementType MovementType, quantity int64, description, performedBy string) (StockMovement, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(partID) == "" {
		return StockMovement{}, ErrInvalidID
	}
	if quantity <= 0 {
		return StockMovement{}, ErrInvalidStockChange
	}
	if movementType != MovementEntry && movementType != MovementExit {
		return StockMovement{}, ErrInvalidMovement
	}
	return StockMovement{id: strings.TrimSpace(id), partID: strings.TrimSpace(partID), movementType: movementType, quantity: quantity, description: strings.TrimSpace(description), performedBy: strings.TrimSpace(performedBy)}, nil
}
func (m StockMovement) ID() string          { return m.id }
func (m StockMovement) PartID() string      { return m.partID }
func (m StockMovement) Type() MovementType  { return m.movementType }
func (m StockMovement) Quantity() int64     { return m.quantity }
func (m StockMovement) Description() string { return m.description }
func (m StockMovement) PerformedBy() string { return m.performedBy }
