package catalog

import (
	"context"

	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	catalogdomain "github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

type stubStore struct {
	catalogapp.Store
	service      catalogdomain.Service
	part         catalogdomain.Part
	findErr      error
	stockErr     error
	requirements []catalogapp.StockRequirement
	orderID      string
	principal    string
}

func (s *stubStore) FindService(context.Context, string) (catalogdomain.Service, error) {
	return s.service, s.findErr
}

func (s *stubStore) FindPart(context.Context, string) (catalogdomain.Part, error) {
	return s.part, s.findErr
}

func (s *stubStore) ReserveStock(_ context.Context, orderID, clientID string, r []catalogapp.StockRequirement) error {
	s.orderID, s.principal, s.requirements = orderID, clientID, r
	return s.stockErr
}

func (s *stubStore) ConsumeReservedStock(_ context.Context, orderID, userID string, r []catalogapp.StockRequirement) error {
	s.orderID, s.principal, s.requirements = orderID, userID, r
	return s.stockErr
}

func newCatalog(store *stubStore) *catalogapp.CatalogService {
	return catalogapp.NewCatalogService(store, nil)
}
