package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func TestServiceGet(t *testing.T) {
	f := newFixture(orderIn(t, domain.Received))
	order, err := f.service().Get(context.Background(), "wo-1")
	if err != nil || order.ID != "wo-1" {
		t.Fatalf("Get() = %+v, %v", order, err)
	}
	if _, err := f.service().Get(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want %v", err, ErrNotFound)
	}
}

func TestServiceGetForClient(t *testing.T) {
	tests := []struct {
		name, id, customerID, vehicleID string
		wantErr                         error
	}{
		{name: "test_it_should_return_order_owned_by_customer", id: "wo-1", customerID: "customer-1"},
		{name: "test_it_should_return_order_matching_vehicle_scope", id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"},
		{name: "test_it_should_hide_order_of_other_customer", id: "wo-1", customerID: "customer-2", wantErr: ErrNotFound},
		{name: "test_it_should_hide_order_of_other_vehicle", id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-2", wantErr: ErrNotFound},
		{name: "test_it_should_return_not_found_for_missing_order", id: "missing", customerID: "customer-1", wantErr: ErrNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(orderIn(t, domain.Received))
			order, err := f.service().GetForClient(context.Background(), test.id, test.customerID, test.vehicleID)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("GetForClient() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr != nil && order.ID != "" {
				t.Errorf("GetForClient() leaked order %q on error", order.ID)
			}
		})
	}
}

func TestServiceList(t *testing.T) {
	received := orderIn(t, domain.Received)
	diagnosis := orderIn(t, domain.Diagnosis)
	diagnosis.ID = "wo-2"
	f := newFixture(received, diagnosis)

	page, err := f.service().List(context.Background(), string(domain.Diagnosis), pagination.Params{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.TotalItems != 1 || page.Items[0].ID != "wo-2" {
		t.Errorf("List(diagnosis) = %+v, want only wo-2", page.Items)
	}
	f.store.listErr = errStorage
	if _, err := f.service().List(context.Background(), "", pagination.Params{}); !errors.Is(err, errStorage) {
		t.Errorf("List() error = %v, want %v", err, errStorage)
	}
}

func TestServiceListForClient(t *testing.T) {
	t.Run("test_it_should_filter_by_customer_and_vehicle", func(t *testing.T) {
		other := orderIn(t, domain.Received)
		other.ID, other.VehicleID = "wo-2", "vehicle-2"
		f := newFixture(orderIn(t, domain.Received), other)

		page, err := f.service().ListForClient(context.Background(), "customer-1", "vehicle-2", pagination.Params{})
		if err != nil {
			t.Fatalf("ListForClient() error = %v", err)
		}
		if page.TotalItems != 1 || page.Items[0].ID != "wo-2" {
			t.Errorf("ListForClient() = %+v, want only wo-2", page.Items)
		}
	})

	t.Run("test_it_should_fail_when_store_cannot_list_for_clients", func(t *testing.T) {
		f := newFixture()
		s := NewService(plainStore{f.store}, f.tx, f.ids, f.ownership, f.catalog, f.inventory)
		if _, err := s.ListForClient(context.Background(), "customer-1", "", pagination.Params{}); err == nil {
			t.Fatal("ListForClient() error = nil, want unavailable listing error")
		}
	})
}
