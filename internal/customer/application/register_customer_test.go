package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

type stubCustomerRepository struct {
	saved domain.Customer
	err   error
}

func (repository *stubCustomerRepository) Save(_ context.Context, customer domain.Customer) error {
	repository.saved = customer
	return repository.err
}

type stubIDGenerator struct {
	id  string
	err error
}

func (generator stubIDGenerator) NewID(context.Context) (string, error) {
	return generator.id, generator.err
}

func TestRegisterCustomer(t *testing.T) {
	t.Run("test_it_should_register_customer_with_normalized_document_when_input_is_valid", func(t *testing.T) {
		repository := &stubCustomerRepository{}
		useCase := NewRegisterCustomer(repository, stubIDGenerator{id: "customer-123"})

		customer, err := useCase.Execute(context.Background(), RegisterCustomerCommand{
			Document: "529.982.247-25",
			Name:     "  Ana Silva  ",
			Email:    "ana@example.com",
			Phone:    "+55 11 99999-0000",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if customer.ID() != "customer-123" {
			t.Errorf("customer ID = %q, want %q", customer.ID(), "customer-123")
		}
		if customer.Document().String() != "52998224725" {
			t.Errorf("document = %q, want digits-only CPF", customer.Document().String())
		}
		if customer.Name() != "Ana Silva" {
			t.Errorf("name = %q, want trimmed name", customer.Name())
		}
		if repository.saved.ID() != customer.ID() {
			t.Errorf("saved customer ID = %q, want %q", repository.saved.ID(), customer.ID())
		}
	})

	t.Run("test_it_should_not_save_customer_when_document_is_invalid", func(t *testing.T) {
		repository := &stubCustomerRepository{}
		useCase := NewRegisterCustomer(repository, stubIDGenerator{id: "customer-123"})

		_, err := useCase.Execute(context.Background(), RegisterCustomerCommand{
			Document: "111.111.111-11",
			Name:     "Ana Silva",
			Email:    "ana@example.com",
			Phone:    "+55 11 99999-0000",
		})
		if !errors.Is(err, domain.ErrInvalidDocument) {
			t.Errorf("Execute() error = %v, want %v", err, domain.ErrInvalidDocument)
		}
		if repository.saved.ID() != "" {
			t.Error("repository.Save() was called for an invalid document")
		}
	})

	t.Run("test_it_should_return_error_when_repository_save_fails", func(t *testing.T) {
		wantErr := errors.New("database unavailable")
		repository := &stubCustomerRepository{err: wantErr}
		useCase := NewRegisterCustomer(repository, stubIDGenerator{id: "customer-123"})

		_, err := useCase.Execute(context.Background(), RegisterCustomerCommand{
			Document: "529.982.247-25",
			Name:     "Ana Silva",
			Email:    "ana@example.com",
			Phone:    "+55 11 99999-0000",
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("Execute() error = %v, want wrapped %v", err, wantErr)
		}
	})
}
