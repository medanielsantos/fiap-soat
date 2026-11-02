package customer

import (
	"context"
	"testing"

	customerapp "github.com/medanielsantos/fiap-soat/internal/customer/application"
	customerdomain "github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

type notFoundStore struct{ customerapp.CustomerStore }

func (notFoundStore) FindByID(context.Context, string) (customerdomain.Customer, error) {
	return customerdomain.Customer{}, customerapp.ErrCustomerNotFound
}

func TestNewDirectoryUsesCustomerService(t *testing.T) {
	directory := NewDirectory(customerapp.NewCustomerService(notFoundStore{}, nil))
	active, err := directory.IsActiveCustomer(context.Background(), "customer-1")
	if err != nil || active {
		t.Errorf("IsActiveCustomer() = %v, %v; want false, nil for unknown customer", active, err)
	}
}
