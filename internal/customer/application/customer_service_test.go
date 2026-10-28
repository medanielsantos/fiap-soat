package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

type memoryCustomerStore struct {
	customers map[string]domain.Customer
	err       error
}

func newMemoryCustomerStore() *memoryCustomerStore {
	return &memoryCustomerStore{customers: make(map[string]domain.Customer)}
}

func (store *memoryCustomerStore) Save(_ context.Context, customer domain.Customer) error {
	if store.err != nil {
		return store.err
	}
	store.customers[customer.ID()] = customer
	return nil
}

func (store *memoryCustomerStore) FindByID(_ context.Context, id string) (domain.Customer, error) {
	customer, ok := store.customers[id]
	if !ok {
		return domain.Customer{}, ErrCustomerNotFound
	}
	return customer, nil
}

func (store *memoryCustomerStore) List(_ context.Context, params pagination.Params) (pagination.Page[domain.Customer], error) {
	customers := make([]domain.Customer, 0, len(store.customers))
	for _, customer := range store.customers {
		customers = append(customers, customer)
	}
	return pagination.NewPage(customers, params, int64(len(customers))), nil
}

func (store *memoryCustomerStore) Update(_ context.Context, customer domain.Customer) error {
	if _, ok := store.customers[customer.ID()]; !ok {
		return ErrCustomerNotFound
	}
	store.customers[customer.ID()] = customer
	return nil
}

func (store *memoryCustomerStore) Delete(_ context.Context, id string) error {
	if _, ok := store.customers[id]; !ok {
		return ErrCustomerNotFound
	}
	delete(store.customers, id)
	return nil
}

func TestCustomerService(t *testing.T) {
	t.Run("test_it_should_create_active_customer_and_normalize_document_when_input_is_valid", func(t *testing.T) {
		store := newMemoryCustomerStore()
		service := NewCustomerService(store, stubIDGenerator{id: "customer-1"})

		customer, err := service.Create(context.Background(), CustomerInput{
			Document: "529.982.247-25", Name: " Ana Silva ", Email: "ana@example.com", Phone: "+5511999999999",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if customer.Document().String() != "52998224725" || customer.Status() != domain.CustomerStatusActive {
			t.Fatalf("created customer has unexpected document or status: %q, %q", customer.Document().String(), customer.Status())
		}
		if _, err := service.Get(context.Background(), customer.ID()); err != nil {
			t.Errorf("Get() error = %v", err)
		}
	})

	t.Run("test_it_should_not_create_customer_when_document_is_invalid", func(t *testing.T) {
		service := NewCustomerService(newMemoryCustomerStore(), stubIDGenerator{id: "customer-1"})
		_, err := service.Create(context.Background(), CustomerInput{
			Document: "111.111.111-11", Name: "Ana Silva", Email: "ana@example.com", Phone: "123",
		})
		if !errors.Is(err, domain.ErrInvalidDocument) {
			t.Errorf("Create() error = %v, want %v", err, domain.ErrInvalidDocument)
		}
	})

	t.Run("test_it_should_update_customer_when_customer_exists", func(t *testing.T) {
		store := newMemoryCustomerStore()
		service := NewCustomerService(store, stubIDGenerator{id: "customer-1"})
		_, err := service.Create(context.Background(), CustomerInput{
			Document: "529.982.247-25", Name: "Ana Silva", Email: "ana@example.com", Phone: "123",
		})
		if err != nil {
			t.Fatal(err)
		}
		updated, err := service.Update(context.Background(), "customer-1", CustomerInput{
			Document: "11.222.333/0001-81", Name: "Ana Silva Santos", Email: "ana.santos@example.com", Phone: "456", Status: domain.CustomerStatusInactive,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Status() != domain.CustomerStatusInactive || updated.Name() != "Ana Silva Santos" {
			t.Errorf("updated customer = %#v", updated)
		}
	})

	t.Run("test_it_should_return_not_found_when_updating_missing_customer", func(t *testing.T) {
		service := NewCustomerService(newMemoryCustomerStore(), stubIDGenerator{id: "unused"})
		_, err := service.Update(context.Background(), "missing", CustomerInput{})
		if !errors.Is(err, ErrCustomerNotFound) {
			t.Errorf("Update() error = %v, want %v", err, ErrCustomerNotFound)
		}
	})

	t.Run("test_it_should_delete_customer_when_customer_exists", func(t *testing.T) {
		store := newMemoryCustomerStore()
		service := NewCustomerService(store, stubIDGenerator{id: "customer-1"})
		_, err := service.Create(context.Background(), CustomerInput{
			Document: "529.982.247-25", Name: "Ana Silva", Email: "ana@example.com", Phone: "123",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Delete(context.Background(), "customer-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, err := service.Get(context.Background(), "customer-1"); !errors.Is(err, ErrCustomerNotFound) {
			t.Errorf("Get() after Delete() error = %v, want %v", err, ErrCustomerNotFound)
		}
	})
}
