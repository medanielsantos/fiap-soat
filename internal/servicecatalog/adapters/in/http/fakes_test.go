package servicecataloghttp

import (
	"context"
	"fmt"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

type handlerCatalogStore struct {
	services  map[string]domain.Service
	parts     map[string]domain.Part
	movements []application.StockMovement
	err       error
}

func newHandlerCatalogStore() *handlerCatalogStore {
	return &handlerCatalogStore{services: map[string]domain.Service{}, parts: map[string]domain.Part{}}
}

func (s *handlerCatalogStore) SaveService(_ context.Context, v domain.Service) error {
	for _, existing := range s.services {
		if existing.InternalCode() == v.InternalCode() {
			return application.ErrDuplicate
		}
	}
	s.services[v.ID()] = v
	return s.err
}
func (s *handlerCatalogStore) FindService(_ context.Context, id string) (domain.Service, error) {
	if s.err != nil {
		return domain.Service{}, s.err
	}
	v, ok := s.services[id]
	if !ok {
		return domain.Service{}, application.ErrNotFound
	}
	return v, nil
}
func (s *handlerCatalogStore) ListServices(_ context.Context, p pagination.Params) (pagination.Page[domain.Service], error) {
	out := make([]domain.Service, 0, len(s.services))
	for _, v := range s.services {
		out = append(out, v)
	}
	return pagination.NewPage(out, p, int64(len(out))), s.err
}
func (s *handlerCatalogStore) UpdateService(_ context.Context, v domain.Service) error {
	s.services[v.ID()] = v
	return s.err
}
func (s *handlerCatalogStore) DeleteService(_ context.Context, id string) error {
	if _, ok := s.services[id]; !ok {
		return application.ErrNotFound
	}
	delete(s.services, id)
	return nil
}
func (s *handlerCatalogStore) SavePart(_ context.Context, v domain.Part, _ *domain.StockMovement) error {
	s.parts[v.ID()] = v
	return s.err
}
func (s *handlerCatalogStore) FindPart(_ context.Context, id string) (domain.Part, error) {
	if s.err != nil {
		return domain.Part{}, s.err
	}
	v, ok := s.parts[id]
	if !ok {
		return domain.Part{}, application.ErrNotFound
	}
	return v, nil
}
func (s *handlerCatalogStore) ListParts(_ context.Context, p pagination.Params) (pagination.Page[domain.Part], error) {
	out := make([]domain.Part, 0, len(s.parts))
	for _, v := range s.parts {
		out = append(out, v)
	}
	return pagination.NewPage(out, p, int64(len(out))), s.err
}
func (s *handlerCatalogStore) UpdatePart(_ context.Context, v domain.Part) error {
	s.parts[v.ID()] = v
	return s.err
}
func (s *handlerCatalogStore) DeletePart(_ context.Context, id string) error {
	p, ok := s.parts[id]
	if !ok {
		return application.ErrNotFound
	}
	if p.Stock() > 0 {
		return domain.ErrPartHasStock
	}
	delete(s.parts, id)
	return nil
}
func (s *handlerCatalogStore) MoveStock(_ context.Context, m domain.StockMovement) (domain.Part, error) {
	p, ok := s.parts[m.PartID()]
	if !ok {
		return domain.Part{}, application.ErrNotFound
	}
	delta := m.Quantity()
	if m.Type() == domain.MovementExit {
		delta = -delta
	}
	moved, err := domain.NewPart(p.ID(), p.SKU(), p.InternalCode(), p.Name(), p.Description(), p.Unit(), p.PurchasePriceInCents(), p.SalePriceInCents(), p.Stock()+delta, p.ReservedStock(), p.Active(), m.PerformedBy())
	if err != nil {
		return domain.Part{}, err
	}
	s.parts[p.ID()] = moved
	return moved, nil
}
func (s *handlerCatalogStore) ListStockMovements(_ context.Context, _ string, p pagination.Params) (pagination.Page[application.StockMovement], error) {
	return pagination.NewPage(s.movements, p, int64(len(s.movements))), nil
}
func (s *handlerCatalogStore) ReserveStock(context.Context, string, string, []application.StockRequirement) error {
	return nil
}
func (s *handlerCatalogStore) ConsumeReservedStock(context.Context, string, string, []application.StockRequirement) error {
	return nil
}

type handlerIDs struct{ calls int }

func (g *handlerIDs) NewID(context.Context) (string, error) {
	g.calls++
	return fmt.Sprintf("id-%d", g.calls), nil
}
