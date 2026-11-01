package application

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

var (
	ErrNotFound                     = errors.New("work order not found")
	ErrCustomerOrVehicleUnavailable = errors.New("customer is inactive or vehicle does not belong to customer")
	ErrCatalogItemUnavailable       = errors.New("catalog item does not exist or is inactive")
	ErrInsufficientStock            = errors.New("insufficient stock")
)

type IDGenerator interface {
	NewID(context.Context) (string, error)
}
type TransactionManager interface {
	Within(context.Context, func(context.Context) error) error
}
type Store interface {
	Create(context.Context, domain.WorkOrder, domain.HistoryEntry) error
	Find(context.Context, string) (domain.WorkOrder, error)
	List(context.Context, string, pagination.Params) (pagination.Page[domain.WorkOrder], error)
	Save(context.Context, domain.WorkOrder, domain.HistoryEntry) error
}
type Ownership interface {
	Validate(context.Context, string, string) error
}
type CatalogItem struct {
	ID, Code, Name, Description, Unit string
	SalePriceInCents                  int64
	EstimatedDurationMinutes          int
	Active                            bool
}
type Catalog interface {
	Service(context.Context, string) (CatalogItem, error)
	Part(context.Context, string) (CatalogItem, error)
}
type StockRequirement struct {
	PartID, MovementID string
	Quantity           int64
}
type Inventory interface {
	Reserve(context.Context, string, string, []StockRequirement) error
	Consume(context.Context, string, string, []StockRequirement) error
}
type Service struct {
	store     Store
	tx        TransactionManager
	ids       IDGenerator
	ownership Ownership
	catalog   Catalog
	inventory Inventory
	now       func() time.Time
}

func NewService(store Store, tx TransactionManager, ids IDGenerator, ownership Ownership, catalog Catalog, inventory Inventory) *Service {
	return &Service{store: store, tx: tx, ids: ids, ownership: ownership, catalog: catalog, inventory: inventory, now: time.Now}
}

func (s *Service) Create(ctx context.Context, customerID, vehicleID, actor string) (domain.WorkOrder, error) {
	if err := s.ownership.Validate(ctx, customerID, vehicleID); err != nil {
		return domain.WorkOrder{}, ErrCustomerOrVehicleUnavailable
	}
	id, err := s.ids.NewID(ctx)
	if err != nil {
		return domain.WorkOrder{}, err
	}
	eventID, err := s.ids.NewID(ctx)
	if err != nil {
		return domain.WorkOrder{}, err
	}
	order, event, err := domain.NewWorkOrder(id, customerID, vehicleID, actor, s.now())
	if err != nil {
		return domain.WorkOrder{}, err
	}
	event.ID = eventID
	order.History = append(order.History, event)
	err = s.tx.Within(ctx, func(txctx context.Context) error { return s.store.Create(txctx, order, event) })
	return order, err
}
func (s *Service) Get(ctx context.Context, id string) (domain.WorkOrder, error) {
	return s.store.Find(ctx, id)
}
func (s *Service) List(ctx context.Context, status string, p pagination.Params) (pagination.Page[domain.WorkOrder], error) {
	return s.store.List(ctx, status, p)
}

func (s *Service) Diagnosis(ctx context.Context, id, text, actor string) (domain.WorkOrder, error) {
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		e, err := o.SetDiagnosis(text, actor, s.now())
		return []domain.HistoryEntry{e}, err
	})
}
func (s *Service) AddItem(ctx context.Context, id string, itemType domain.ItemType, catalogID string, quantity int64, actor string) (domain.WorkOrder, error) {
	var snapshot CatalogItem
	var err error
	switch itemType {
	case domain.ServiceItem:
		snapshot, err = s.catalog.Service(ctx, catalogID)
	case domain.PartItem:
		snapshot, err = s.catalog.Part(ctx, catalogID)
	default:
		return domain.WorkOrder{}, domain.ErrInvalidItem
	}
	if err != nil || !snapshot.Active {
		return domain.WorkOrder{}, ErrCatalogItemUnavailable
	}
	itemID, err := s.ids.NewID(ctx)
	if err != nil {
		return domain.WorkOrder{}, err
	}
	item, err := domain.NewItem(itemID, itemType, snapshot.ID, snapshot.Code, snapshot.Name, snapshot.Unit, quantity, snapshot.SalePriceInCents, snapshot.EstimatedDurationMinutes)
	if err != nil {
		return domain.WorkOrder{}, err
	}
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		e, err := o.AddItem(item, actor, s.now())
		return []domain.HistoryEntry{e}, err
	})
}
func (s *Service) RemoveItem(ctx context.Context, id, itemID, actor string) (domain.WorkOrder, error) {
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		e, err := o.RemoveItem(itemID, actor, s.now())
		return []domain.HistoryEntry{e}, err
	})
}
func (s *Service) SubmitQuote(ctx context.Context, id, actor string) (domain.WorkOrder, error) {
	quoteID, err := s.ids.NewID(ctx)
	if err != nil {
		return domain.WorkOrder{}, err
	}
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		_, e, err := o.SubmitQuote(quoteID, actor, s.now())
		return []domain.HistoryEntry{e}, err
	})
}
func (s *Service) ApproveQuote(ctx context.Context, id, actor string) (order domain.WorkOrder, err error) {
	err = s.tx.Within(ctx, func(txctx context.Context) error {
		o, e := s.store.Find(txctx, id)
		if e != nil {
			return e
		}
		entry, e := o.DecideQuote(true, actor, s.now())
		if e != nil {
			return e
		}
		entry.ID, e = s.ids.NewID(txctx)
		if e != nil {
			return e
		}
		requirements, e := s.stockRequirements(txctx, o)
		if e != nil {
			return e
		}
		if e = s.inventory.Reserve(txctx, o.ID, actor, requirements); e != nil {
			return e
		}
		if e = s.save(txctx, &o, entry); e != nil {
			return e
		}
		order = o
		return nil
	})
	return
}
func (s *Service) RejectQuote(ctx context.Context, id, actor, note string) (domain.WorkOrder, error) {
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		e, err := o.DecideQuote(false, actor, s.now())
		if err == nil && note != "" {
			e.Note = note
		}
		return []domain.HistoryEntry{e}, err
	})
}
func (s *Service) Consume(ctx context.Context, id, partID string, quantity int64, actor string) (order domain.WorkOrder, err error) {
	err = s.tx.Within(ctx, func(txctx context.Context) error {
		o, e := s.store.Find(txctx, id)
		if e != nil {
			return e
		}
		entry, e := o.Consume(partID, quantity, actor, s.now())
		if e != nil {
			return e
		}
		movementID, e := s.ids.NewID(txctx)
		if e != nil {
			return e
		}
		entry.ID, e = s.ids.NewID(txctx)
		if e != nil {
			return e
		}
		if e = s.inventory.Consume(txctx, o.ID, actor, []StockRequirement{{PartID: partID, MovementID: movementID, Quantity: quantity}}); e != nil {
			return e
		}
		if e = s.save(txctx, &o, entry); e != nil {
			return e
		}
		order = o
		return nil
	})
	return
}
func (s *Service) Complete(ctx context.Context, id, actor string) (domain.WorkOrder, error) {
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		e, err := o.Complete(actor, s.now())
		return []domain.HistoryEntry{e}, err
	})
}
func (s *Service) Deliver(ctx context.Context, id, actor string) (domain.WorkOrder, error) {
	return s.mutate(ctx, id, actor, func(o *domain.WorkOrder) ([]domain.HistoryEntry, error) {
		e, err := o.Deliver(actor, s.now())
		return []domain.HistoryEntry{e}, err
	})
}

func (s *Service) mutate(ctx context.Context, id, actor string, fn func(*domain.WorkOrder) ([]domain.HistoryEntry, error)) (domain.WorkOrder, error) {
	var order domain.WorkOrder
	err := s.tx.Within(ctx, func(txctx context.Context) error {
		o, e := s.store.Find(txctx, id)
		if e != nil {
			return e
		}
		events, e := fn(&o)
		if e != nil {
			return e
		}
		for i := range events {
			events[i].ID, e = s.ids.NewID(txctx)
			if e != nil {
				return e
			}
		}
		for _, event := range events {
			if e = s.save(txctx, &o, event); e != nil {
				return e
			}
		}
		order = o
		return nil
	})
	return order, err
}
func (s *Service) save(ctx context.Context, o *domain.WorkOrder, event domain.HistoryEntry) error {
	o.History = append(o.History, event)
	return s.store.Save(ctx, *o, event)
}
func (s *Service) stockRequirements(ctx context.Context, o domain.WorkOrder) ([]StockRequirement, error) {
	byPart := map[string]int64{}
	for _, item := range o.Items {
		if item.Type == domain.PartItem {
			byPart[item.CatalogID] += item.Quantity
		}
	}
	out := make([]StockRequirement, 0, len(byPart))
	partIDs := make([]string, 0, len(byPart))
	for partID := range byPart {
		partIDs = append(partIDs, partID)
	}
	sort.Strings(partIDs)
	for _, partID := range partIDs {
		qty := byPart[partID]
		id, err := s.ids.NewID(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, StockRequirement{PartID: partID, MovementID: id, Quantity: qty})
	}
	return out, nil
}
