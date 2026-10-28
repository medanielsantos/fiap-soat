package domain

import (
	"errors"
	"testing"
)

func TestRestoreCustomer(t *testing.T) {
	document, err := NewDocument("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		id     string
		nameIn string
		email  string
		phone  string
		status CustomerStatus
		want   error
	}{
		{name: "test_it_should_restore_active_customer_when_data_is_valid", id: "id-1", nameIn: " Ana Silva ", email: "ana@example.com", phone: "+5511999999999", status: CustomerStatusActive},
		{name: "test_it_should_reject_customer_without_id", nameIn: "Ana Silva", email: "ana@example.com", phone: "123", status: CustomerStatusActive, want: ErrInvalidCustomerID},
		{name: "test_it_should_reject_customer_without_name", id: "id-1", email: "ana@example.com", phone: "123", status: CustomerStatusActive, want: ErrInvalidName},
		{name: "test_it_should_reject_customer_with_invalid_email", id: "id-1", nameIn: "Ana Silva", email: "not-an-email", phone: "123", status: CustomerStatusActive, want: ErrInvalidEmail},
		{name: "test_it_should_reject_customer_without_phone", id: "id-1", nameIn: "Ana Silva", email: "ana@example.com", status: CustomerStatusActive, want: ErrInvalidPhone},
		{name: "test_it_should_reject_customer_with_invalid_status", id: "id-1", nameIn: "Ana Silva", email: "ana@example.com", phone: "123", status: "deleted", want: ErrInvalidStatus},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			customer, err := RestoreCustomer(test.id, document, test.nameIn, test.email, test.phone, test.status)
			if !errors.Is(err, test.want) {
				t.Fatalf("RestoreCustomer() error = %v, want %v", err, test.want)
			}
			if err == nil && customer.Name() != "Ana Silva" {
				t.Errorf("customer name = %q, want trimmed name", customer.Name())
			}
		})
	}
}

func TestCustomerStatus(t *testing.T) {
	document, err := NewDocument("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	customer, err := NewCustomer("id-1", document, "Ana", "ana@example.com", "123")
	if err != nil {
		t.Fatal(err)
	}
	if !customer.IsActive() {
		t.Fatal("new customer should be active")
	}
	customer.Deactivate()
	if customer.IsActive() || customer.Status() != CustomerStatusInactive {
		t.Fatal("deactivated customer should be inactive")
	}
	customer.Activate()
	if !customer.IsActive() {
		t.Fatal("activated customer should be active")
	}
}
