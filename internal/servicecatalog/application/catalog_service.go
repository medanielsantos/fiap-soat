package application

import (
	"context"
	"errors"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
	"time"
)

var (
	ErrNotFound  = errors.New("catalog item not found")
	ErrDuplicate = errors.New("catalog item already exists")
)

type IDGenerator interface {
	NewID(context.Context) (string, error)
}
type StockRequirement struct {
	PartID, MovementID string
	Quantity           int64
}
type StockMovement struct {
	ID, PartID, Type, Description, PerformedBy string
	WorkOrderID                                *string
	Quantity                                   int64
	CreatedAt                                  time.Time
}
type Store interface {
	SaveService(context.Context, domain.Service) error
	FindService(context.Context, string) (domain.Service, error)
	ListServices(context.Context, pagination.Params) (pagination.Page[domain.Service], error)
	UpdateService(context.Context, domain.Service) error
	DeleteService(context.Context, string) error
	SavePart(context.Context, domain.Part, *domain.StockMovement) error
	FindPart(context.Context, string) (domain.Part, error)
	ListParts(context.Context, pagination.Params) (pagination.Page[domain.Part], error)
	UpdatePart(context.Context, domain.Part) error
	DeletePart(context.Context, string) error
	MoveStock(context.Context, domain.StockMovement) (domain.Part, error)
	ListStockMovements(context.Context, string, pagination.Params) (pagination.Page[StockMovement], error)
	ReserveStock(context.Context, string, string, []StockRequirement) error
	ConsumeReservedStock(context.Context, string, string, []StockRequirement) error
}
type CatalogService struct {
	store Store
	ids   IDGenerator
}

func NewCatalogService(store Store, ids IDGenerator) *CatalogService {
	return &CatalogService{store: store, ids: ids}
}

type ServiceInput struct {
	InternalCode, Name, Description string
	PerformedBy                     string
	SalePriceInCents                int64
	EstimatedDurationMinutes        int
	Active                          bool
}
type PartInput struct {
	SKU, InternalCode, Name, Description, Unit string
	PerformedBy                                string
	PurchasePriceInCents, SalePriceInCents     int64
	InitialStock                               int64
	Active                                     bool
}

func (s *CatalogService) CreateService(ctx context.Context, in ServiceInput) (domain.Service, error) {
	id, e := s.ids.NewID(ctx)
	if e != nil {
		return domain.Service{}, e
	}
	v, e := domain.NewService(id, in.InternalCode, in.Name, in.Description, in.SalePriceInCents, in.EstimatedDurationMinutes, in.Active, in.PerformedBy)
	if e != nil {
		return domain.Service{}, e
	}
	e = s.store.SaveService(ctx, v)
	return v, e
}
func (s *CatalogService) GetService(ctx context.Context, id string) (domain.Service, error) {
	return s.store.FindService(ctx, id)
}
func (s *CatalogService) ListServices(ctx context.Context, p pagination.Params) (pagination.Page[domain.Service], error) {
	return s.store.ListServices(ctx, p)
}
func (s *CatalogService) UpdateService(ctx context.Context, id string, in ServiceInput) (domain.Service, error) {
	if _, e := s.store.FindService(ctx, id); e != nil {
		return domain.Service{}, e
	}
	v, e := domain.NewService(id, in.InternalCode, in.Name, in.Description, in.SalePriceInCents, in.EstimatedDurationMinutes, in.Active, in.PerformedBy)
	if e != nil {
		return domain.Service{}, e
	}
	e = s.store.UpdateService(ctx, v)
	return v, e
}
func (s *CatalogService) DeleteService(ctx context.Context, id string) error {
	return s.store.DeleteService(ctx, id)
}
func (s *CatalogService) CreatePart(ctx context.Context, in PartInput) (domain.Part, error) {
	id, e := s.ids.NewID(ctx)
	if e != nil {
		return domain.Part{}, e
	}
	v, e := domain.NewPart(id, in.SKU, in.InternalCode, in.Name, in.Description, in.Unit, in.PurchasePriceInCents, in.SalePriceInCents, in.InitialStock, 0, in.Active, in.PerformedBy)
	if e != nil {
		return domain.Part{}, e
	}
	var openingMovement *domain.StockMovement
	if v.Stock() > 0 {
		movementID, err := s.ids.NewID(ctx)
		if err != nil {
			return domain.Part{}, err
		}
		movement, err := domain.NewStockMovement(movementID, v.ID(), domain.MovementEntry, v.Stock(), "initial stock", in.PerformedBy)
		if err != nil {
			return domain.Part{}, err
		}
		openingMovement = &movement
	}
	e = s.store.SavePart(ctx, v, openingMovement)
	return v, e
}
func (s *CatalogService) GetPart(ctx context.Context, id string) (domain.Part, error) {
	return s.store.FindPart(ctx, id)
}
func (s *CatalogService) ListParts(ctx context.Context, p pagination.Params) (pagination.Page[domain.Part], error) {
	return s.store.ListParts(ctx, p)
}
func (s *CatalogService) UpdatePart(ctx context.Context, id string, in PartInput) (domain.Part, error) {
	current, e := s.store.FindPart(ctx, id)
	if e != nil {
		return domain.Part{}, e
	}
	v, e := domain.NewPart(id, in.SKU, in.InternalCode, in.Name, in.Description, in.Unit, in.PurchasePriceInCents, in.SalePriceInCents, current.Stock(), current.ReservedStock(), in.Active, in.PerformedBy)
	if e != nil {
		return domain.Part{}, e
	}
	e = s.store.UpdatePart(ctx, v)
	return v, e
}
func (s *CatalogService) DeletePart(ctx context.Context, id string) error {
	return s.store.DeletePart(ctx, id)
}
func (s *CatalogService) MoveStock(ctx context.Context, partID string, kind domain.MovementType, quantity int64, description, performedBy string) (domain.Part, error) {
	id, e := s.ids.NewID(ctx)
	if e != nil {
		return domain.Part{}, e
	}
	m, e := domain.NewStockMovement(id, partID, kind, quantity, description, performedBy)
	if e != nil {
		return domain.Part{}, e
	}
	return s.store.MoveStock(ctx, m)
}
func (s *CatalogService) ListStockMovements(ctx context.Context, partID string, p pagination.Params) (pagination.Page[StockMovement], error) {
	if _, err := s.store.FindPart(ctx, partID); err != nil {
		return pagination.Page[StockMovement]{}, err
	}
	return s.store.ListStockMovements(ctx, partID, p)
}
func (s *CatalogService) ReserveStock(ctx context.Context, orderID, actor string, requirements []StockRequirement) error {
	return s.store.ReserveStock(ctx, orderID, actor, requirements)
}
func (s *CatalogService) ConsumeReservedStock(ctx context.Context, orderID, actor string, requirements []StockRequirement) error {
	return s.store.ConsumeReservedStock(ctx, orderID, actor, requirements)
}
