package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

var (
	testNow    = time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	errBoom    = errors.New("boom")
	errStorage = errors.New("storage down")
)

// memoryStore keeps deep copies so a failed mutation never leaks into persisted state.
type memoryStore struct {
	orders    map[string]domain.WorkOrder
	events    []domain.HistoryEntry
	createErr error
	saveErr   error
	listErr   error
}

func newMemoryStore(orders ...domain.WorkOrder) *memoryStore {
	s := &memoryStore{orders: map[string]domain.WorkOrder{}}
	for _, o := range orders {
		s.orders[o.ID] = clone(o)
	}
	return s
}

func clone(o domain.WorkOrder) domain.WorkOrder {
	o.Items = append([]domain.Item{}, o.Items...)
	o.Quotes = append([]domain.Quote{}, o.Quotes...)
	o.History = append([]domain.HistoryEntry{}, o.History...)
	return o
}

func (s *memoryStore) Create(_ context.Context, o domain.WorkOrder, e domain.HistoryEntry) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.orders[o.ID] = clone(o)
	s.events = append(s.events, e)
	return nil
}

func (s *memoryStore) Find(_ context.Context, id string) (domain.WorkOrder, error) {
	o, ok := s.orders[id]
	if !ok {
		return domain.WorkOrder{}, ErrNotFound
	}
	return clone(o), nil
}

func (s *memoryStore) List(_ context.Context, status string, p pagination.Params) (pagination.Page[domain.WorkOrder], error) {
	var out []domain.WorkOrder
	for _, o := range s.orders {
		if status == "" || string(o.Status) == status {
			out = append(out, o)
		}
	}
	return pagination.NewPage(out, p, int64(len(out))), s.listErr
}

func (s *memoryStore) ListForClient(_ context.Context, customerID, vehicleID string, p pagination.Params) (pagination.Page[domain.WorkOrder], error) {
	var out []domain.WorkOrder
	for _, o := range s.orders {
		if o.CustomerID == customerID && (vehicleID == "" || o.VehicleID == vehicleID) {
			out = append(out, o)
		}
	}
	return pagination.NewPage(out, p, int64(len(out))), s.listErr
}

func (s *memoryStore) Save(_ context.Context, o domain.WorkOrder, e domain.HistoryEntry) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.orders[o.ID] = clone(o)
	s.events = append(s.events, e)
	return nil
}

// plainStore implements Store without ClientOrderStore.
type plainStore struct{ Store }

type passthroughTx struct{ calls int }

func (tx *passthroughTx) Within(ctx context.Context, fn func(context.Context) error) error {
	tx.calls++
	return fn(ctx)
}

// sequenceIDs returns id-1, id-2, ... and fails on the call number failAt (1-based).
type sequenceIDs struct {
	n, failAt int
}

func (g *sequenceIDs) NewID(context.Context) (string, error) {
	g.n++
	if g.n == g.failAt {
		return "", errBoom
	}
	return fmt.Sprintf("id-%d", g.n), nil
}

type stubOwnership struct{ err error }

func (o stubOwnership) Validate(context.Context, string, string) error { return o.err }

type stubCatalog struct {
	item CatalogItem
	err  error
	kind string
}

func (c *stubCatalog) Service(_ context.Context, id string) (CatalogItem, error) {
	c.kind = "service"
	return c.item, c.err
}

func (c *stubCatalog) Part(_ context.Context, id string) (CatalogItem, error) {
	c.kind = "part"
	return c.item, c.err
}

type recordingInventory struct {
	reserved, consumed []StockRequirement
	principal          string
	err                error
}

func (i *recordingInventory) Reserve(_ context.Context, _, clientID string, r []StockRequirement) error {
	i.reserved, i.principal = r, clientID
	return i.err
}

func (i *recordingInventory) Consume(_ context.Context, _, userID string, r []StockRequirement) error {
	i.consumed, i.principal = r, userID
	return i.err
}

type fixture struct {
	store     *memoryStore
	tx        *passthroughTx
	ids       *sequenceIDs
	ownership stubOwnership
	catalog   *stubCatalog
	inventory *recordingInventory
}

func newFixture(orders ...domain.WorkOrder) *fixture {
	return &fixture{store: newMemoryStore(orders...), tx: &passthroughTx{}, ids: &sequenceIDs{}, catalog: &stubCatalog{}, inventory: &recordingInventory{}}
}

func (f *fixture) service() *Service {
	s := NewService(f.store, f.tx, f.ids, f.ownership, f.catalog, f.inventory)
	s.now = func() time.Time { return testNow }
	return s
}
