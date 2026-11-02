package application

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

var errBoom = errors.New("boom")

type memoryCatalogStore struct {
	services  map[string]domain.Service
	parts     map[string]domain.Part
	opening   *domain.StockMovement
	moved     []domain.StockMovement
	reserved  []StockRequirement
	consumed  []StockRequirement
	errs      map[string]error
	movements []StockMovement
}

func newMemoryCatalogStore() *memoryCatalogStore {
	return &memoryCatalogStore{services: map[string]domain.Service{}, parts: map[string]domain.Part{}, errs: map[string]error{}}
}

func (s *memoryCatalogStore) SaveService(_ context.Context, v domain.Service) error {
	if err := s.errs["SaveService"]; err != nil {
		return err
	}
	s.services[v.ID()] = v
	return nil
}
func (s *memoryCatalogStore) FindService(_ context.Context, id string) (domain.Service, error) {
	v, ok := s.services[id]
	if !ok {
		return domain.Service{}, ErrNotFound
	}
	return v, nil
}
func (s *memoryCatalogStore) ListServices(_ context.Context, p pagination.Params) (pagination.Page[domain.Service], error) {
	out := make([]domain.Service, 0, len(s.services))
	for _, v := range s.services {
		out = append(out, v)
	}
	return pagination.NewPage(out, p, int64(len(out))), s.errs["ListServices"]
}
func (s *memoryCatalogStore) UpdateService(_ context.Context, v domain.Service) error {
	s.services[v.ID()] = v
	return s.errs["UpdateService"]
}
func (s *memoryCatalogStore) DeleteService(_ context.Context, id string) error {
	if _, ok := s.services[id]; !ok {
		return ErrNotFound
	}
	delete(s.services, id)
	return nil
}
func (s *memoryCatalogStore) SavePart(_ context.Context, v domain.Part, opening *domain.StockMovement) error {
	if err := s.errs["SavePart"]; err != nil {
		return err
	}
	s.parts[v.ID()], s.opening = v, opening
	return nil
}
func (s *memoryCatalogStore) FindPart(_ context.Context, id string) (domain.Part, error) {
	v, ok := s.parts[id]
	if !ok {
		return domain.Part{}, ErrNotFound
	}
	return v, nil
}
func (s *memoryCatalogStore) ListParts(_ context.Context, p pagination.Params) (pagination.Page[domain.Part], error) {
	out := make([]domain.Part, 0, len(s.parts))
	for _, v := range s.parts {
		out = append(out, v)
	}
	return pagination.NewPage(out, p, int64(len(out))), s.errs["ListParts"]
}
func (s *memoryCatalogStore) UpdatePart(_ context.Context, v domain.Part) error {
	s.parts[v.ID()] = v
	return s.errs["UpdatePart"]
}
func (s *memoryCatalogStore) DeletePart(_ context.Context, id string) error {
	if _, ok := s.parts[id]; !ok {
		return ErrNotFound
	}
	delete(s.parts, id)
	return nil
}
func (s *memoryCatalogStore) MoveStock(_ context.Context, m domain.StockMovement) (domain.Part, error) {
	s.moved = append(s.moved, m)
	return s.parts[m.PartID()], s.errs["MoveStock"]
}
func (s *memoryCatalogStore) ListStockMovements(_ context.Context, _ string, p pagination.Params) (pagination.Page[StockMovement], error) {
	return pagination.NewPage(s.movements, p, int64(len(s.movements))), nil
}
func (s *memoryCatalogStore) ReserveStock(_ context.Context, _, _ string, reqs []StockRequirement) error {
	s.reserved = append(s.reserved, reqs...)
	return s.errs["ReserveStock"]
}
func (s *memoryCatalogStore) ConsumeReservedStock(_ context.Context, _, _ string, reqs []StockRequirement) error {
	s.consumed = append(s.consumed, reqs...)
	return s.errs["ConsumeReservedStock"]
}

type sequenceIDs struct{ calls, failAt int }

func (g *sequenceIDs) NewID(context.Context) (string, error) {
	g.calls++
	if g.calls == g.failAt {
		return "", errBoom
	}
	return fmt.Sprintf("id-%d", g.calls), nil
}

func newTestService(failIDAt int) (*CatalogService, *memoryCatalogStore) {
	store := newMemoryCatalogStore()
	return NewCatalogService(store, &sequenceIDs{failAt: failIDAt}), store
}

func mustPart(t *testing.T, id string, stock, reserved int64) domain.Part {
	t.Helper()
	p, err := domain.NewPart(id, "SKU-"+id, "P-"+id, "Filter", "", "un", 500, 1000, stock, reserved, true, "u")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func defaultParams(t *testing.T) pagination.Params {
	t.Helper()
	p, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
