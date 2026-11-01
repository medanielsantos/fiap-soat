package catalog

import (
	"context"
	"errors"

	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	catalogdomain "github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
)

type Directory struct{ catalog *catalogapp.CatalogService }

func NewDirectory(c *catalogapp.CatalogService) *Directory { return &Directory{catalog: c} }
func (d *Directory) Service(ctx context.Context, id string) (application.CatalogItem, error) {
	v, e := d.catalog.GetService(ctx, id)
	if e != nil {
		return application.CatalogItem{}, e
	}
	return application.CatalogItem{ID: v.ID(), Code: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: "service", SalePriceInCents: v.SalePriceInCents(), EstimatedDurationMinutes: v.EstimatedDurationMinutes(), Active: v.Active()}, nil
}
func (d *Directory) Part(ctx context.Context, id string) (application.CatalogItem, error) {
	v, e := d.catalog.GetPart(ctx, id)
	if e != nil {
		return application.CatalogItem{}, e
	}
	return application.CatalogItem{ID: v.ID(), Code: v.SKU(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), SalePriceInCents: v.SalePriceInCents(), Active: v.Active()}, nil
}

type Inventory struct{ catalog *catalogapp.CatalogService }

func NewInventory(c *catalogapp.CatalogService) *Inventory { return &Inventory{catalog: c} }
func (i *Inventory) Reserve(ctx context.Context, orderID, clientID string, requirements []application.StockRequirement) error {
	err := i.catalog.ReserveStock(ctx, orderID, clientID, convert(requirements))
	if errors.Is(err, catalogdomain.ErrInsufficientStock) {
		return application.ErrInsufficientStock
	}
	if errors.Is(err, catalogapp.ErrNotFound) {
		return application.ErrCatalogItemUnavailable
	}
	return err
}
func (i *Inventory) Consume(ctx context.Context, orderID, userID string, requirements []application.StockRequirement) error {
	err := i.catalog.ConsumeReservedStock(ctx, orderID, userID, convert(requirements))
	if errors.Is(err, catalogdomain.ErrInsufficientStock) {
		return application.ErrInsufficientStock
	}
	if errors.Is(err, catalogapp.ErrNotFound) {
		return application.ErrCatalogItemUnavailable
	}
	return err
}
func convert(in []application.StockRequirement) []catalogapp.StockRequirement {
	out := make([]catalogapp.StockRequirement, 0, len(in))
	for _, r := range in {
		out = append(out, catalogapp.StockRequirement{PartID: r.PartID, MovementID: r.MovementID, Quantity: r.Quantity})
	}
	return out
}
