package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestGrantAndRevokeDirectPermission(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("attendant", application.UserRoleAttendant))
	for range 2 {
		if err := repository.GrantPermissionToUser(ctx, "attendant", application.PermissionUsersRead); err != nil {
			t.Fatalf("GrantPermissionToUser(): %v", err)
		}
	}
	user, err := repository.FindByID(ctx, "attendant")
	if err != nil || len(user.Permissions) != 1 || user.Permissions[0] != application.PermissionUsersRead {
		t.Fatalf("permissions after grant = (%v, %v)", user.Permissions, err)
	}
	if err := repository.RevokePermissionFromUser(ctx, "attendant", application.PermissionUsersRead); err != nil {
		t.Fatalf("RevokePermissionFromUser(): %v", err)
	}
	user, err = repository.FindByID(ctx, "attendant")
	if err != nil || len(user.Permissions) != 0 {
		t.Fatalf("permissions after revoke = (%v, %v)", user.Permissions, err)
	}
}

func TestPermissionChangesRequireExistingUser(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("deleted", application.UserRoleAttendant))
	if err := repository.Delete(ctx, "deleted"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"missing", "deleted"} {
		if err := repository.GrantPermissionToUser(ctx, id, application.PermissionUsersRead); !errors.Is(err, application.ErrUserNotFound) {
			t.Errorf("GrantPermissionToUser(%s) error = %v, want %v", id, err, application.ErrUserNotFound)
		}
		if err := repository.RevokePermissionFromUser(ctx, id, application.PermissionUsersRead); !errors.Is(err, application.ErrUserNotFound) {
			t.Errorf("RevokePermissionFromUser(%s) error = %v, want %v", id, err, application.ErrUserNotFound)
		}
	}
}
