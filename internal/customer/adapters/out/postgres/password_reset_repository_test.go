package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

func hash64(c byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func TestCustomerRepositoryPasswordReset(t *testing.T) {
	repository, db := newTestRepository(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	saveCustomer(t, repository, newCustomer(t, "c-1", cpfAna, "ana@example.com", domain.CustomerStatusActive))
	saveCustomer(t, repository, newCustomer(t, "c-2", cnpjCo, "co@example.com", domain.CustomerStatusInactive))

	if err := repository.CreateCustomerReset(ctx, "c-1", hash64('a'), now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateCustomerReset() error = %v", err)
	}
	if err := repository.CreateCustomerReset(ctx, "c-1", hash64('b'), now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateCustomerReset() second error = %v", err)
	}
	var pending int64
	db.Model(&customerPasswordResetRecord{}).Where("customer_id = 'c-1'").Count(&pending)
	if pending != 1 {
		t.Fatalf("pending reset tokens = %d, want previous token replaced", pending)
	}

	t.Run("test_it_should_reject_replaced_token", func(t *testing.T) {
		if err := repository.ResetCustomerPassword(ctx, hash64('a'), "new", now); !errors.Is(err, application.ErrInvalidResetToken) {
			t.Errorf("error = %v, want %v", err, application.ErrInvalidResetToken)
		}
	})

	t.Run("test_it_should_reject_expired_token", func(t *testing.T) {
		if err := repository.ResetCustomerPassword(ctx, hash64('b'), "new", now.Add(2*time.Hour)); !errors.Is(err, application.ErrInvalidResetToken) {
			t.Errorf("error = %v, want %v", err, application.ErrInvalidResetToken)
		}
	})

	t.Run("test_it_should_update_password_and_consume_token", func(t *testing.T) {
		if err := repository.ResetCustomerPassword(ctx, hash64('b'), "new-hash", now); err != nil {
			t.Fatalf("ResetCustomerPassword() error = %v", err)
		}
		found, _ := repository.FindByID(ctx, "c-1")
		if found.Password() != "new-hash" {
			t.Errorf("password = %q, want new-hash", found.Password())
		}
		if err := repository.ResetCustomerPassword(ctx, hash64('b'), "again", now); !errors.Is(err, application.ErrInvalidResetToken) {
			t.Errorf("reuse error = %v, want %v", err, application.ErrInvalidResetToken)
		}
	})

	t.Run("test_it_should_reject_token_of_inactive_customer", func(t *testing.T) {
		if err := repository.CreateCustomerReset(ctx, "c-2", hash64('c'), now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := repository.ResetCustomerPassword(ctx, hash64('c'), "new", now); !errors.Is(err, application.ErrInvalidResetToken) {
			t.Errorf("error = %v, want %v", err, application.ErrInvalidResetToken)
		}
	})

	t.Run("test_it_should_fail_for_unknown_customer", func(t *testing.T) {
		if err := repository.CreateCustomerReset(ctx, "missing", hash64('d'), now.Add(time.Hour)); err == nil {
			t.Error("CreateCustomerReset(missing customer) error = nil, want foreign key error")
		}
	})

	t.Run("test_it_should_return_error_when_context_is_cancelled", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if err := repository.ResetCustomerPassword(cancelled, hash64('b'), "x", now); err == nil || errors.Is(err, application.ErrInvalidResetToken) {
			t.Errorf("error = %v, want raw database error", err)
		}
		if err := repository.CreateCustomerReset(cancelled, "c-1", hash64('e'), now); err == nil {
			t.Error("CreateCustomerReset(cancelled) error = nil, want error")
		}
	})
}
