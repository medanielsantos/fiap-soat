package application

import (
	"context"
	"errors"
	"testing"
)

func TestSetRoles(t *testing.T) {
	tests := []struct {
		name, id, requester string
		roles               []UserRole
		wantErr             error
	}{
		{"promotes another user", "target", "admin", []UserRole{UserRoleManager, UserRoleAttendant}, nil},
		{"keeps own admin role", "admin", "admin", []UserRole{UserRoleAdmin, UserRoleManager}, nil},
		{"rejects empty roles", "target", "admin", nil, ErrInvalidRole},
		{"rejects dropping own admin role", "admin", "admin", []UserRole{UserRoleManager}, ErrCannotChangeOwnRole},
		{"reports missing user", "missing", "admin", []UserRole{UserRoleManager}, ErrUserNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMemoryStore()
			store.users["admin"] = activeUser("admin", UserRoleAdmin)
			store.users["target"] = activeUser("target", UserRoleAttendant)
			service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
			got, err := service.SetRoles(context.Background(), tt.id, tt.requester, tt.roles)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetRoles() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && len(got.Roles) != len(tt.roles) {
				t.Fatalf("SetRoles() roles = %v, want %v", got.Roles, tt.roles)
			}
		})
	}
}

func TestSetRolesPropagatesReloadFailure(t *testing.T) {
	store := newFailingStore("FindByID")
	store.users["target"] = activeUser("target", UserRoleAttendant)
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	if _, err := service.SetRoles(context.Background(), "target", "admin", []UserRole{UserRoleManager}); !errors.Is(err, errStoreFailure) {
		t.Fatalf("SetRoles() error = %v, want %v", err, errStoreFailure)
	}
}
