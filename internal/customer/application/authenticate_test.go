package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

const validPassword = "correct-horse-battery"

func TestCustomerServiceAuthenticate(t *testing.T) {
	store := newCredentialCustomerStore()
	store.byEmail["ana@example.com"] = newTestCustomer(t, "c-1", "529.982.247-25", "ana@example.com", domain.CustomerStatusActive, validPassword)
	store.byEmail["inactive@example.com"] = newTestCustomer(t, "c-2", "529.982.247-25", "inactive@example.com", domain.CustomerStatusInactive, validPassword)
	store.byEmail["nopass@example.com"] = newTestCustomer(t, "c-3", "529.982.247-25", "nopass@example.com", domain.CustomerStatusActive, "")
	service := NewCustomerService(store, stubIDGenerator{})

	t.Run("test_it_should_authenticate_customer_when_email_is_normalized_and_password_matches", func(t *testing.T) {
		customer, err := service.Authenticate(context.Background(), "  ANA@Example.com ", validPassword)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}
		if customer.ID() != "c-1" {
			t.Errorf("customer ID = %q, want c-1", customer.ID())
		}
	})

	cases := []struct{ name, email, password string }{
		{"test_it_should_reject_when_password_is_wrong", "ana@example.com", "wrong-password-123"},
		{"test_it_should_reject_when_customer_is_unknown", "missing@example.com", validPassword},
		{"test_it_should_reject_when_customer_is_inactive", "inactive@example.com", validPassword},
		{"test_it_should_reject_when_customer_has_no_password", "nopass@example.com", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.Authenticate(context.Background(), tc.email, tc.password); !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Authenticate() error = %v, want %v", err, ErrInvalidCredentials)
			}
		})
	}

	t.Run("test_it_should_reject_when_store_has_no_credential_port", func(t *testing.T) {
		plain := NewCustomerService(newMemoryCustomerStore(), stubIDGenerator{})
		if _, err := plain.Authenticate(context.Background(), "ana@example.com", validPassword); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("Authenticate() error = %v, want %v", err, ErrInvalidCredentials)
		}
	})
}

func TestCustomerServiceAuthenticateVehicle(t *testing.T) {
	store := newCredentialCustomerStore()
	active := newTestCustomer(t, "c-1", "529.982.247-25", "ana@example.com", domain.CustomerStatusActive, "")
	store.byPlate["ABC1D23|52998"] = plateMatch{customer: active, vehicleID: "v-1"}
	store.byPlate["INA1D23|52998"] = plateMatch{customer: newTestCustomer(t, "c-2", "529.982.247-25", "x@example.com", domain.CustomerStatusInactive, ""), vehicleID: "v-2"}
	store.byPlate["MIS1D23|11222"] = plateMatch{customer: active, vehicleID: "v-3"}
	service := NewCustomerService(store, stubIDGenerator{})

	t.Run("test_it_should_return_customer_and_vehicle_when_plate_and_prefix_match", func(t *testing.T) {
		customer, vehicleID, err := service.AuthenticateVehicle(context.Background(), " abc-1d23 ", "529.98")
		if err != nil {
			t.Fatalf("AuthenticateVehicle() error = %v", err)
		}
		if customer.ID() != "c-1" || vehicleID != "v-1" {
			t.Errorf("got (%q, %q), want (c-1, v-1)", customer.ID(), vehicleID)
		}
	})

	cases := []struct{ name, plate, prefix string }{
		{"test_it_should_reject_prefix_with_letters", "ABC1D23", "52a98"},
		{"test_it_should_reject_prefix_with_wrong_length", "ABC1D23", "5299"},
		{"test_it_should_reject_unknown_plate", "ZZZ9Z99", "52998"},
		{"test_it_should_reject_inactive_customer", "INA1D23", "52998"},
		{"test_it_should_reject_document_not_matching_prefix", "MIS1D23", "11.222"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := service.AuthenticateVehicle(context.Background(), tc.plate, tc.prefix); err == nil {
				t.Error("AuthenticateVehicle() error = nil, want error")
			}
		})
	}

	t.Run("test_it_should_reject_when_store_has_no_credential_port", func(t *testing.T) {
		plain := NewCustomerService(newMemoryCustomerStore(), stubIDGenerator{})
		if _, _, err := plain.AuthenticateVehicle(context.Background(), "ABC1D23", "52998"); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("AuthenticateVehicle() error = %v, want %v", err, ErrInvalidCredentials)
		}
	})
}
