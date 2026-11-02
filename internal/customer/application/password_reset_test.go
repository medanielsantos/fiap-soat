package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"golang.org/x/crypto/bcrypt"
)

func newResetService(t *testing.T, mailer CustomerMailer, resetURL string) (*CustomerService, *credentialCustomerStore) {
	t.Helper()
	store := newCredentialCustomerStore()
	store.byEmail["ana@example.com"] = newTestCustomer(t, "c-1", "529.982.247-25", "ana@example.com", domain.CustomerStatusActive, "")
	store.byEmail["inactive@example.com"] = newTestCustomer(t, "c-2", "529.982.247-25", "inactive@example.com", domain.CustomerStatusInactive, "")
	service := NewCustomerService(store, stubIDGenerator{}, CustomerResetOptions{Mailer: mailer, URL: resetURL})
	service.now = func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }
	return service, store
}

func TestCustomerServiceRequestPasswordReset(t *testing.T) {
	t.Run("test_it_should_store_token_digest_and_mail_reset_link_when_customer_is_active", func(t *testing.T) {
		mailer := &recordingMailer{}
		service, store := newResetService(t, mailer, "https://app.example.com/reset/")
		if err := service.RequestPasswordReset(context.Background(), " ANA@example.com "); err != nil {
			t.Fatalf("RequestPasswordReset() error = %v", err)
		}
		if len(store.resetCreated) != 1 || store.resetCreated[0].customerID != "c-1" {
			t.Fatalf("reset requests = %#v, want one for c-1", store.resetCreated)
		}
		if want := service.now().Add(30 * time.Minute); !store.resetCreated[0].expires.Equal(want) {
			t.Errorf("expires = %v, want %v", store.resetCreated[0].expires, want)
		}
		link, err := url.Parse(mailer.body)
		if err != nil || link.Host != "app.example.com" || link.Path != "/reset" {
			t.Fatalf("mail body = %q, want reset link without trailing slash", mailer.body)
		}
		digest := sha256.Sum256([]byte(link.Query().Get("token")))
		if hex.EncodeToString(digest[:]) != store.resetCreated[0].tokenHash {
			t.Error("stored hash does not match sha256 of mailed token")
		}
		if mailer.to != "ana@example.com" {
			t.Errorf("mail to = %q, want ana@example.com", mailer.to)
		}
	})

	t.Run("test_it_should_silently_ignore_unknown_or_inactive_customers", func(t *testing.T) {
		mailer := &recordingMailer{}
		service, store := newResetService(t, mailer, "https://app.example.com/reset")
		for _, email := range []string{"missing@example.com", "inactive@example.com"} {
			if err := service.RequestPasswordReset(context.Background(), email); err != nil {
				t.Errorf("RequestPasswordReset(%q) error = %v, want nil", email, err)
			}
		}
		if len(store.resetCreated) != 0 || mailer.to != "" {
			t.Error("reset token or mail was produced for an unknown or inactive customer")
		}
	})

	errorCases := []struct {
		name   string
		mailer CustomerMailer
		url    string
		setup  func(*credentialCustomerStore)
	}{
		{name: "test_it_should_fail_when_reset_url_is_missing", mailer: &recordingMailer{}},
		{name: "test_it_should_fail_when_store_fails_to_persist_token", mailer: &recordingMailer{}, url: "https://x.io",
			setup: func(s *credentialCustomerStore) { s.resetErr = errStoreUnavailable }},
		{name: "test_it_should_fail_when_mailer_is_not_configured", url: "https://x.io"},
		{name: "test_it_should_fail_when_reset_url_is_invalid", mailer: &recordingMailer{}, url: "://bad"},
		{name: "test_it_should_fail_when_mailer_fails", mailer: &recordingMailer{err: errStoreUnavailable}, url: "https://x.io"},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			service, store := newResetService(t, tc.mailer, tc.url)
			if tc.setup != nil {
				tc.setup(store)
			}
			if err := service.RequestPasswordReset(context.Background(), "ana@example.com"); err == nil {
				t.Error("RequestPasswordReset() error = nil, want error")
			}
		})
	}

	t.Run("test_it_should_fail_when_store_lacks_required_ports", func(t *testing.T) {
		options := CustomerResetOptions{Mailer: &recordingMailer{}, URL: "https://x.io"}
		active := newTestCustomer(t, "c-1", "529.982.247-25", "ana@example.com", domain.CustomerStatusActive, "")
		for _, store := range []CustomerStore{newMemoryCustomerStore(), credentialOnlyStore{memoryCustomerStore: newMemoryCustomerStore(), customer: active}} {
			if err := NewCustomerService(store, stubIDGenerator{}, options).RequestPasswordReset(context.Background(), "ana@example.com"); err == nil {
				t.Errorf("RequestPasswordReset() with %T error = nil, want error", store)
			}
		}
	})
}

func TestCustomerServiceResetPassword(t *testing.T) {
	t.Run("test_it_should_store_bcrypt_hash_for_token_digest", func(t *testing.T) {
		service, store := newResetService(t, nil, "")
		if err := service.ResetPassword(context.Background(), "raw-token", validPassword); err != nil {
			t.Fatalf("ResetPassword() error = %v", err)
		}
		digest := sha256.Sum256([]byte("raw-token"))
		if store.resetToken != hex.EncodeToString(digest[:]) || !store.resetAt.Equal(service.now()) {
			t.Errorf("reset called with token %q at %v", store.resetToken, store.resetAt)
		}
		if bcrypt.CompareHashAndPassword([]byte(store.resetHash), []byte(validPassword)) != nil {
			t.Error("stored password is not a bcrypt hash of the new password")
		}
	})

	t.Run("test_it_should_propagate_store_error", func(t *testing.T) {
		service, store := newResetService(t, nil, "")
		store.resetErr = ErrInvalidResetToken
		if err := service.ResetPassword(context.Background(), "t", validPassword); !errors.Is(err, ErrInvalidResetToken) {
			t.Errorf("ResetPassword() error = %v, want %v", err, ErrInvalidResetToken)
		}
	})

	t.Run("test_it_should_reject_passwords_outside_length_bounds", func(t *testing.T) {
		service, _ := newResetService(t, nil, "")
		for _, password := range []string{"short", string(make([]byte, 73))} {
			if err := service.ResetPassword(context.Background(), "t", password); !errors.Is(err, ErrInvalidPassword) {
				t.Errorf("ResetPassword(len=%d) error = %v, want %v", len(password), err, ErrInvalidPassword)
			}
		}
	})

	t.Run("test_it_should_reject_when_store_has_no_reset_port", func(t *testing.T) {
		service := NewCustomerService(newMemoryCustomerStore(), stubIDGenerator{})
		if err := service.ResetPassword(context.Background(), "t", validPassword); !errors.Is(err, ErrInvalidResetToken) {
			t.Errorf("ResetPassword() error = %v, want %v", err, ErrInvalidResetToken)
		}
	})
}
