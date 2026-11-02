package application

import (
	"context"
	"errors"
	"testing"
)

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name, id, requester string
		wantErr             error
	}{
		{"deletes another user", "target", "admin", nil},
		{"rejects self deletion", "admin", "admin", ErrCannotDeleteSelf},
		{"reports missing user", "missing", "admin", ErrUserNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMemoryStore()
			store.users["admin"] = activeUser("admin", UserRoleAdmin)
			store.users["target"] = activeUser("target", UserRoleManager)
			service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
			err := service.DeleteUser(context.Background(), tt.id, tt.requester)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DeleteUser() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == ErrCannotDeleteSelf && store.users["admin"].DeletedAt != nil {
				t.Fatal("requesting user was deleted")
			}
			if tt.wantErr == nil {
				deleted := store.users[tt.id]
				if deleted.Status != UserStatusDeleted || deleted.DeletedAt == nil {
					t.Fatalf("deleted user = %+v", deleted)
				}
				if _, _, active := service.ActivePermissions(context.Background(), tt.id); active {
					t.Fatal("deleted user still has active permissions")
				}
			}
		})
	}
}
