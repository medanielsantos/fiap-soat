package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

func TestCustomerRepositoryFindByEmail(t *testing.T) {
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	saveCustomer(t, repository, newCustomer(t, "c-1", cpfAna, "ana@example.com", domain.CustomerStatusActive))

	found, err := repository.FindByEmail(ctx, "ana@example.com")
	if err != nil || found.ID() != "c-1" {
		t.Fatalf("FindByEmail() = (%q, %v), want c-1", found.ID(), err)
	}
	if _, err := repository.FindByEmail(ctx, "missing@example.com"); !errors.Is(err, application.ErrCustomerNotFound) {
		t.Errorf("FindByEmail(missing) error = %v, want %v", err, application.ErrCustomerNotFound)
	}
}

func TestCustomerRepositoryFindByPlateAndDocumentPrefix(t *testing.T) {
	repository, db := newTestRepository(t)
	ctx := context.Background()
	saveCustomer(t, repository, newCustomer(t, "c-1", cpfAna, "ana@example.com", domain.CustomerStatusActive))
	saveCustomer(t, repository, newCustomer(t, "c-2", cnpjCo, "co@example.com", domain.CustomerStatusInactive))
	insertVehicle(t, db, "v-1", "c-1", "ABC1D23")
	insertVehicle(t, db, "v-2", "c-2", "XYZ9K87")
	insertVehicle(t, db, "v-3", "c-1", "DEL1E23")
	if err := db.Exec("UPDATE vehicles SET deleted_at = NOW() WHERE id = 'v-3'").Error; err != nil {
		t.Fatal(err)
	}

	customer, vehicleID, err := repository.FindByPlateAndDocumentPrefix(ctx, "ABC1D23", "52998")
	if err != nil {
		t.Fatalf("FindByPlateAndDocumentPrefix() error = %v", err)
	}
	if customer.ID() != "c-1" || vehicleID != "v-1" {
		t.Errorf("got (%q, %q), want (c-1, v-1)", customer.ID(), vehicleID)
	}

	misses := []struct{ name, plate, prefix string }{
		{"test_it_should_not_match_wrong_prefix", "ABC1D23", "11222"},
		{"test_it_should_not_match_inactive_customer", "XYZ9K87", "11222"},
		{"test_it_should_not_match_deleted_vehicle", "DEL1E23", "52998"},
		{"test_it_should_not_match_unknown_plate", "NOP0Q00", "52998"},
	}
	for _, tc := range misses {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := repository.FindByPlateAndDocumentPrefix(ctx, tc.plate, tc.prefix); !errors.Is(err, application.ErrCustomerNotFound) {
				t.Errorf("error = %v, want %v", err, application.ErrCustomerNotFound)
			}
		})
	}

	t.Run("test_it_should_not_match_soft_deleted_customer", func(t *testing.T) {
		if err := repository.Delete(ctx, "c-1"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := repository.FindByPlateAndDocumentPrefix(ctx, "ABC1D23", "52998"); !errors.Is(err, application.ErrCustomerNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrCustomerNotFound)
		}
	})
}
