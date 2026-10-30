package customer

import (
	"context"
	"errors"

	customerapp "github.com/medanielsantos/fiap-soat/internal/customer/application"
	customerdomain "github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

type customerFinder interface {
	Get(ctx context.Context, id string) (customerdomain.Customer, error)
}

// Directory adapts the customer module to the vehicle CustomerDirectory port.
type Directory struct {
	customers customerFinder
}

func NewDirectory(customers *customerapp.CustomerService) *Directory {
	return &Directory{customers: customers}
}

func (directory *Directory) IsActiveCustomer(ctx context.Context, id string) (bool, error) {
	customer, err := directory.customers.Get(ctx, id)
	if errors.Is(err, customerapp.ErrCustomerNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return customer.IsActive(), nil
}
