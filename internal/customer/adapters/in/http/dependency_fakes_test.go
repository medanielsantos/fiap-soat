package customerhttp

import (
	"context"
	"fmt"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	vehicledomain "github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
	workorderapp "github.com/medanielsantos/fiap-soat/internal/workorder/application"
	workorderdomain "github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

type fakeVehicleStore struct {
	vehicles   []vehicledomain.Vehicle
	err        error
	lastFilter vehicleapp.ListFilter
}

func (store *fakeVehicleStore) Save(context.Context, vehicledomain.Vehicle) error { return nil }
func (store *fakeVehicleStore) FindByID(context.Context, string) (vehicledomain.Vehicle, error) {
	return vehicledomain.Vehicle{}, vehicleapp.ErrVehicleNotFound
}
func (store *fakeVehicleStore) Update(context.Context, vehicledomain.Vehicle) error { return nil }
func (store *fakeVehicleStore) Delete(context.Context, string) error                { return nil }
func (store *fakeVehicleStore) List(_ context.Context, filter vehicleapp.ListFilter, params pagination.Params) (pagination.Page[vehicledomain.Vehicle], error) {
	store.lastFilter = filter
	if store.err != nil {
		return pagination.Page[vehicledomain.Vehicle]{}, store.err
	}
	return pagination.NewPage(store.vehicles, params, int64(len(store.vehicles))), nil
}

type fakeWorkOrderStore struct {
	orders  map[string]workorderdomain.WorkOrder
	listErr error
}

func (store *fakeWorkOrderStore) Create(context.Context, workorderdomain.WorkOrder, workorderdomain.HistoryEntry) error {
	return nil
}
func (store *fakeWorkOrderStore) Find(_ context.Context, id string) (workorderdomain.WorkOrder, error) {
	order, ok := store.orders[id]
	if !ok {
		return workorderdomain.WorkOrder{}, workorderapp.ErrNotFound
	}
	return order, nil
}
func (store *fakeWorkOrderStore) List(context.Context, string, pagination.Params) (pagination.Page[workorderdomain.WorkOrder], error) {
	return pagination.Page[workorderdomain.WorkOrder]{}, nil
}
func (store *fakeWorkOrderStore) Save(_ context.Context, order workorderdomain.WorkOrder, _ workorderdomain.HistoryEntry) error {
	store.orders[order.ID] = order
	return nil
}
func (store *fakeWorkOrderStore) ListForClient(_ context.Context, customerID, vehicleID string, params pagination.Params) (pagination.Page[workorderdomain.WorkOrder], error) {
	if store.listErr != nil {
		return pagination.Page[workorderdomain.WorkOrder]{}, store.listErr
	}
	var out []workorderdomain.WorkOrder
	for _, order := range store.orders {
		if order.CustomerID == customerID && (vehicleID == "" || order.VehicleID == vehicleID) {
			out = append(out, order)
		}
	}
	return pagination.NewPage(out, params, int64(len(out))), nil
}

type inlineTx struct{}

func (inlineTx) Within(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type sequenceIDs struct{ n int }

func (ids *sequenceIDs) NewID(context.Context) (string, error) {
	ids.n++
	return fmt.Sprintf("id-%d", ids.n), nil
}

type fakeInventory struct{ err error }

func (inventory fakeInventory) Reserve(context.Context, string, string, []workorderapp.StockRequirement) error {
	return inventory.err
}
func (inventory fakeInventory) Consume(context.Context, string, string, []workorderapp.StockRequirement) error {
	return inventory.err
}

// pendingQuoteOrder returns a work order awaiting approval of quote version 1.
func pendingQuoteOrder(id, customerID, vehicleID string) workorderdomain.WorkOrder {
	part := workorderdomain.Item{ID: "item-1", Type: workorderdomain.PartItem, CatalogID: "part-1", Code: "P1", Description: "Oil filter", Unit: "un", Quantity: 2, UnitPriceInCents: 1500}
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	return workorderdomain.WorkOrder{
		ID: id, CustomerID: customerID, VehicleID: vehicleID, Status: workorderdomain.AwaitingApproval, Diagnosis: "Worn filter", QuoteVersion: 1,
		Items:     []workorderdomain.Item{part},
		Quotes:    []workorderdomain.Quote{{ID: "q-1", Version: 1, TotalInCents: 3000, Decision: "pending", Items: []workorderdomain.QuoteItem{{Item: part, SubtotalInCents: 3000}}}},
		CreatedAt: now, UpdatedAt: now,
	}
}
