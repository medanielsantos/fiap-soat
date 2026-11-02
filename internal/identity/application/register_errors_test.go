package application

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegisterRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name, userName, email, password string
		roles                           []UserRole
	}{
		{"no roles", "Admin", "admin@example.test", "correct horse battery", nil},
		{"unknown role", "Admin", "admin@example.test", "correct horse battery", []UserRole{"root"}},
		{"blank name", "  ", "admin@example.test", "correct horse battery", []UserRole{UserRoleAdmin}},
		{"long name", strings.Repeat("a", 121), "admin@example.test", "correct horse battery", []UserRole{UserRoleAdmin}},
		{"malformed email", "Admin", "not-an-email", "correct horse battery", []UserRole{UserRoleAdmin}},
		{"display-name email", "Admin", "Admin <admin@example.test>", "correct horse battery", []UserRole{UserRoleAdmin}},
		{"short password", "Admin", "admin@example.test", "short", []UserRole{UserRoleAdmin}},
		{"long password", "Admin", "admin@example.test", strings.Repeat("p", 73), []UserRole{UserRoleAdmin}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMemoryStore()
			service := NewService(store, fixedIDs{"user-1"}, fixedTokens{}, nil, "")
			if _, err := service.Register(context.Background(), tt.userName, tt.email, tt.password, tt.roles); err == nil {
				t.Fatal("Register() accepted invalid input")
			}
			if len(store.users) != 0 {
				t.Fatalf("users stored after rejected registration = %d", len(store.users))
			}
		})
	}
}

func TestRegisterPropagatesDependencyFailures(t *testing.T) {
	roles := []UserRole{UserRoleAttendant}
	withoutIDs := NewService(newMemoryStore(), failingIDs{}, fixedTokens{}, nil, "")
	if _, err := withoutIDs.Register(context.Background(), "User", "user@example.test", "correct horse battery", roles); err == nil {
		t.Error("Register() ignored id generator failure")
	}
	failing := NewService(newFailingStore("Create"), fixedIDs{"user-1"}, fixedTokens{}, nil, "")
	if _, err := failing.Register(context.Background(), "User", "user@example.test", "correct horse battery", roles); !errors.Is(err, errStoreFailure) {
		t.Errorf("Register() error = %v, want %v", err, errStoreFailure)
	}
	store := newMemoryStore()
	store.users["existing"] = User{ID: "existing", Email: "user@example.test"}
	taken := NewService(store, fixedIDs{"user-1"}, fixedTokens{}, nil, "")
	if _, err := taken.Register(context.Background(), "User", "user@example.test", "correct horse battery", roles); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("Register() duplicate email error = %v, want %v", err, ErrEmailTaken)
	}
}

func TestSeedAdminPropagatesFailures(t *testing.T) {
	invalid := NewService(newMemoryStore(), fixedIDs{"seed"}, fixedTokens{}, nil, "")
	if created, err := invalid.SeedAdmin(context.Background(), "Admin", "admin@example.test", "short"); err == nil || created {
		t.Errorf("SeedAdmin() with short password = (%v, %v)", created, err)
	}
	failing := NewService(newFailingStore("CreateFirst"), fixedIDs{"seed"}, fixedTokens{}, nil, "")
	if created, err := failing.SeedAdmin(context.Background(), "Admin", "admin@example.test", "correct horse battery"); !errors.Is(err, errStoreFailure) || created {
		t.Errorf("SeedAdmin() store failure = (%v, %v), want %v", created, err, errStoreFailure)
	}
}
