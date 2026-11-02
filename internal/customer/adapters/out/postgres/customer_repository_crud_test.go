package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestCustomerRepositorySaveAndFind(t *testing.T) {
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	saveCustomer(t, repository, newCustomer(t, "c-1", cpfAna, "ana@example.com", domain.CustomerStatusActive))

	found, err := repository.FindByID(ctx, "c-1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Document().String() != cpfAna || found.Email() != "ana@example.com" || found.Password() != "hash-c-1" || !found.IsActive() {
		t.Errorf("FindByID() = %#v, want persisted fields", found)
	}
	if _, err := repository.FindByID(ctx, "missing"); !errors.Is(err, application.ErrCustomerNotFound) {
		t.Errorf("FindByID(missing) error = %v, want %v", err, application.ErrCustomerNotFound)
	}

	t.Run("test_it_should_translate_unique_violations", func(t *testing.T) {
		dupDocument := newCustomer(t, "c-2", cpfAna, "other@example.com", domain.CustomerStatusActive)
		if err := repository.Save(ctx, dupDocument); !errors.Is(err, application.ErrDuplicateDocument) {
			t.Errorf("Save(duplicate document) error = %v, want %v", err, application.ErrDuplicateDocument)
		}
		dupEmail := newCustomer(t, "c-3", cnpjCo, "ANA@example.com", domain.CustomerStatusActive)
		if err := repository.Save(ctx, dupEmail); !errors.Is(err, application.ErrDuplicateEmail) {
			t.Errorf("Save(duplicate email) error = %v, want %v", err, application.ErrDuplicateEmail)
		}
	})
}

func TestCustomerRepositoryList(t *testing.T) {
	repository, db := newTestRepository(t)
	ctx := context.Background()
	saveCustomer(t, repository, newCustomer(t, "c-1", cpfAna, "ana@example.com", domain.CustomerStatusActive))
	saveCustomer(t, repository, newCustomer(t, "c-2", cnpjCo, "co@example.com", domain.CustomerStatusInactive))
	if err := db.Exec("UPDATE customers SET created_at = created_at - interval '1 hour' WHERE id = 'c-1'").Error; err != nil {
		t.Fatal(err)
	}

	params, _ := pagination.NewParams(1, 1)
	page, err := repository.List(ctx, params)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.TotalItems != 2 || len(page.Items) != 1 || page.Items[0].ID() != "c-2" {
		t.Errorf("List() page 1 = total %d, %d items, want newest c-2 of 2", page.TotalItems, len(page.Items))
	}

	t.Run("test_it_should_fail_when_stored_document_is_corrupt", func(t *testing.T) {
		if err := db.Exec("UPDATE customers SET document = '11111111111' WHERE id = 'c-1'").Error; err != nil {
			t.Fatal(err)
		}
		all, _ := pagination.NewParams(1, 10)
		if _, err := repository.List(ctx, all); !errors.Is(err, domain.ErrInvalidDocument) {
			t.Errorf("List() error = %v, want %v", err, domain.ErrInvalidDocument)
		}
	})

	t.Run("test_it_should_return_error_when_context_is_cancelled", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := repository.List(cancelled, params); err == nil {
			t.Error("List() with cancelled context error = nil, want error")
		}
	})
}

func TestCustomerRepositoryUpdate(t *testing.T) {
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	saveCustomer(t, repository, newCustomer(t, "c-1", cpfAna, "ana@example.com", domain.CustomerStatusActive))
	saveCustomer(t, repository, newCustomer(t, "c-2", cnpjCo, "co@example.com", domain.CustomerStatusActive))

	updated := newCustomer(t, "c-1", cpfAna, "ana.new@example.com", domain.CustomerStatusInactive)
	if err := repository.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	found, _ := repository.FindByID(ctx, "c-1")
	if found.Email() != "ana.new@example.com" || found.IsActive() {
		t.Errorf("after Update() email = %q active = %v", found.Email(), found.IsActive())
	}

	if err := repository.Update(ctx, newCustomer(t, "missing", cpfAna, "x@example.com", domain.CustomerStatusActive)); !errors.Is(err, application.ErrCustomerNotFound) {
		t.Errorf("Update(missing) error = %v, want %v", err, application.ErrCustomerNotFound)
	}
	if err := repository.Update(ctx, newCustomer(t, "c-1", cnpjCo, "ana@example.com", domain.CustomerStatusActive)); !errors.Is(err, application.ErrDuplicateDocument) {
		t.Errorf("Update(duplicate document) error = %v, want %v", err, application.ErrDuplicateDocument)
	}
	if err := repository.Delete(ctx, "missing"); !errors.Is(err, application.ErrCustomerNotFound) {
		t.Errorf("Delete(missing) error = %v, want %v", err, application.ErrCustomerNotFound)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := repository.Delete(cancelled, "c-2"); err == nil || errors.Is(err, application.ErrCustomerNotFound) {
		t.Errorf("Delete(cancelled) error = %v, want raw database error", err)
	}
}
