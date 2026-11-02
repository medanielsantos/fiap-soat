package application

import (
	"context"
	"errors"
	"testing"
)

func TestSetStatus(t *testing.T) {
	tests := []struct {
		name, id, requester string
		status              UserStatus
		wantErr             error
	}{
		{"suspends another user", "target", "admin", UserStatusSuspended, nil},
		{"marks another user deleted", "target", "admin", UserStatusDeleted, nil},
		{"keeps self active", "admin", "admin", UserStatusActive, nil},
		{"rejects unknown status", "target", "admin", UserStatus("archived"), ErrInvalidStatus},
		{"rejects self deactivation", "admin", "admin", UserStatusInactive, ErrCannotDeactivateSelf},
		{"reports missing user", "missing", "admin", UserStatusBanned, ErrUserNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMemoryStore()
			store.users["admin"] = activeUser("admin", UserRoleAdmin)
			store.users["target"] = activeUser("target", UserRoleAttendant)
			service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
			got, err := service.SetStatus(context.Background(), tt.id, tt.requester, tt.status)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetStatus() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.ID != tt.id || got.Status != tt.status || store.users[tt.id].Status != tt.status {
				t.Fatalf("SetStatus() = %+v, stored = %+v", got, store.users[tt.id])
			}
		})
	}
}

func TestSetStatusPropagatesReloadFailure(t *testing.T) {
	store := newFailingStore("FindByID")
	store.users["target"] = activeUser("target", UserRoleAttendant)
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	if _, err := service.SetStatus(context.Background(), "target", "admin", UserStatusInactive); !errors.Is(err, errStoreFailure) {
		t.Fatalf("SetStatus() error = %v, want %v", err, errStoreFailure)
	}
}
