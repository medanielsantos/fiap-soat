package application

import (
	"context"
	"errors"
	"testing"
)

type failingMailer struct{}

func (failingMailer) Send(context.Context, string, string, string) error {
	return errors.New("smtp failure")
}

func TestRequestPasswordResetFailures(t *testing.T) {
	tests := []struct {
		name, resetURL, email string
		store                 Store
		mailer                Mailer
		wantErr               bool
	}{
		{"missing reset url", "", "user@example.test", newMemoryStore(), &captureMailer{}, true},
		{"unknown email is silent", "https://example.test/reset", "missing@example.test", newMemoryStore(), &captureMailer{}, false},
		{"missing mailer", "https://example.test/reset", "user@example.test", newMemoryStore(), nil, true},
		{"invalid reset url", "://invalid", "user@example.test", newMemoryStore(), &captureMailer{}, true},
		{"store failure", "https://example.test/reset", "user@example.test", newFailingStore("CreateReset"), &captureMailer{}, true},
		{"mailer failure", "https://example.test/reset", "user@example.test", newMemoryStore(), failingMailer{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch store := tt.store.(type) {
			case *memoryStore:
				store.users["user"] = activeUser("user", UserRoleAdmin)
			case *failingStore:
				store.users["user"] = activeUser("user", UserRoleAdmin)
			}
			service := NewService(tt.store, fixedIDs{"unused"}, fixedTokens{}, tt.mailer, tt.resetURL)
			err := service.RequestPasswordReset(context.Background(), tt.email)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RequestPasswordReset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if capture, ok := tt.mailer.(*captureMailer); ok && !tt.wantErr && capture.body != "" {
				t.Fatalf("mail sent for unknown email: %q", capture.body)
			}
		})
	}
}

func TestResetPasswordRejectsInvalidPasswordLength(t *testing.T) {
	service := NewService(newMemoryStore(), fixedIDs{"unused"}, fixedTokens{}, nil, "")
	for _, password := range []string{"short", string(make([]byte, 73))} {
		if err := service.ResetPassword(context.Background(), "token", password); err == nil {
			t.Errorf("ResetPassword() accepted password of %d bytes", len(password))
		}
	}
}
