package customer

import (
	"context"
	"errors"
	"testing"

	customerapp "github.com/medanielsantos/fiap-soat/internal/customer/application"
	customerdomain "github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

type stubCustomerFinder struct {
	customer customerdomain.Customer
	err      error
}

func (finder stubCustomerFinder) Get(context.Context, string) (customerdomain.Customer, error) {
	return finder.customer, finder.err
}

func TestDirectoryIsActiveCustomer(t *testing.T) {
	document, err := customerdomain.NewDocument("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	active, err := customerdomain.NewCustomer("customer-1", document, "Ana", "ana@example.com", "123")
	if err != nil {
		t.Fatal(err)
	}
	inactive := active
	inactive.Deactivate()
	lookupErr := errors.New("database down")

	tests := []struct {
		name    string
		finder  stubCustomerFinder
		want    bool
		wantErr error
	}{
		{name: "test_it_should_return_true_when_customer_is_active", finder: stubCustomerFinder{customer: active}, want: true},
		{name: "test_it_should_return_false_when_customer_is_inactive", finder: stubCustomerFinder{customer: inactive}},
		{name: "test_it_should_return_false_when_customer_does_not_exist", finder: stubCustomerFinder{err: customerapp.ErrCustomerNotFound}},
		{name: "test_it_should_propagate_unexpected_error", finder: stubCustomerFinder{err: lookupErr}, wantErr: lookupErr},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := &Directory{customers: test.finder}
			got, err := directory.IsActiveCustomer(context.Background(), "customer-1")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("IsActiveCustomer() error = %v, want %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("IsActiveCustomer() = %v, want %v", got, test.want)
			}
		})
	}
}
