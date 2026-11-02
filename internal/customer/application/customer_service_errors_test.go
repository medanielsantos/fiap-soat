package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"golang.org/x/crypto/bcrypt"
)

type failingUpdateStore struct{ *memoryCustomerStore }

func (failingUpdateStore) Update(context.Context, domain.Customer) error { return errStoreUnavailable }

func validInput() CustomerInput {
	return CustomerInput{Document: "529.982.247-25", Name: "Ana Silva", Email: "ana@example.com", Phone: "123"}
}

func TestCustomerServiceCreateErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*CustomerInput)
		ids     stubIDGenerator
		saveErr error
		want    error
	}{
		{name: "test_it_should_propagate_id_generator_error", ids: stubIDGenerator{err: errStoreUnavailable}, want: errStoreUnavailable},
		{name: "test_it_should_reject_short_password", mutate: func(i *CustomerInput) { i.Password = "short" }, want: ErrInvalidPassword},
		{name: "test_it_should_reject_invalid_email", mutate: func(i *CustomerInput) { i.Email = "bad" }, want: domain.ErrInvalidEmail},
		{name: "test_it_should_reject_invalid_status", mutate: func(i *CustomerInput) { i.Status = "blocked" }, want: domain.ErrInvalidStatus},
		{name: "test_it_should_propagate_store_save_error", saveErr: errStoreUnavailable, want: errStoreUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryCustomerStore()
			store.err = tc.saveErr
			ids := tc.ids
			if ids.id == "" && ids.err == nil {
				ids.id = "customer-1"
			}
			input := validInput()
			if tc.mutate != nil {
				tc.mutate(&input)
			}
			if _, err := NewCustomerService(store, ids).Create(context.Background(), input); !errors.Is(err, tc.want) {
				t.Errorf("Create() error = %v, want %v", err, tc.want)
			}
		})
	}

	t.Run("test_it_should_store_bcrypt_hash_when_password_is_given", func(t *testing.T) {
		input := validInput()
		input.Password = validPassword
		customer, err := NewCustomerService(newMemoryCustomerStore(), stubIDGenerator{id: "c-1"}).Create(context.Background(), input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if bcrypt.CompareHashAndPassword([]byte(customer.Password()), []byte(validPassword)) != nil {
			t.Error("customer password is not a bcrypt hash of the input")
		}
	})
}

func TestCustomerServiceUpdateErrors(t *testing.T) {
	seed := func(t *testing.T) (*memoryCustomerStore, *CustomerService) {
		store := newMemoryCustomerStore()
		service := NewCustomerService(store, stubIDGenerator{id: "c-1"})
		input := validInput()
		input.Password = validPassword
		if _, err := service.Create(context.Background(), input); err != nil {
			t.Fatal(err)
		}
		return store, service
	}

	cases := []struct {
		name   string
		mutate func(*CustomerInput)
		want   error
	}{
		{"test_it_should_reject_invalid_document", func(i *CustomerInput) { i.Document = "123" }, domain.ErrInvalidDocument},
		{"test_it_should_reject_invalid_password", func(i *CustomerInput) { i.Password = "short" }, ErrInvalidPassword},
		{"test_it_should_reject_missing_status", func(i *CustomerInput) { i.Status = "" }, domain.ErrInvalidStatus},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, service := seed(t)
			input := validInput()
			input.Status = domain.CustomerStatusActive
			tc.mutate(&input)
			if _, err := service.Update(context.Background(), "c-1", input); !errors.Is(err, tc.want) {
				t.Errorf("Update() error = %v, want %v", err, tc.want)
			}
		})
	}

	t.Run("test_it_should_keep_existing_hash_when_password_is_empty_and_rehash_otherwise", func(t *testing.T) {
		store, service := seed(t)
		original := store.customers["c-1"].Password()
		input := validInput()
		input.Status = domain.CustomerStatusActive
		kept, err := service.Update(context.Background(), "c-1", input)
		if err != nil || kept.Password() != original {
			t.Fatalf("Update() without password = (%q, %v), want original hash", kept.Password(), err)
		}
		input.Password = "another-strong-password"
		changed, err := service.Update(context.Background(), "c-1", input)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(changed.Password()), []byte(input.Password)) != nil {
			t.Errorf("Update() with password did not rehash (err = %v)", err)
		}
	})

	t.Run("test_it_should_propagate_store_update_error", func(t *testing.T) {
		store, _ := seed(t)
		service := NewCustomerService(failingUpdateStore{store}, stubIDGenerator{})
		input := validInput()
		input.Status = domain.CustomerStatusActive
		if _, err := service.Update(context.Background(), "c-1", input); !errors.Is(err, errStoreUnavailable) {
			t.Errorf("Update() error = %v, want %v", err, errStoreUnavailable)
		}
	})
}

func TestCustomerServiceList(t *testing.T) {
	store := newMemoryCustomerStore()
	service := NewCustomerService(store, stubIDGenerator{id: "c-1"})
	if _, err := service.Create(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}
	params, _ := pagination.NewParams(1, 10)
	page, err := service.List(context.Background(), params)
	if err != nil || page.TotalItems != 1 || len(page.Items) != 1 {
		t.Errorf("List() = (%d items, total %d, %v), want one customer", len(page.Items), page.TotalItems, err)
	}
}
