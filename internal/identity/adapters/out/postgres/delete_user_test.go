package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestDeleteSoftDeletesOnce(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("user", application.UserRoleManager))
	if err := repository.Delete(ctx, "user"); err != nil {
		t.Fatalf("Delete(): %v", err)
	}
	user, err := repository.FindByID(ctx, "user")
	if err != nil || user.Status != application.UserStatusDeleted || user.DeletedAt == nil {
		t.Fatalf("deleted user = (%+v, %v)", user, err)
	}
	if err := repository.Delete(ctx, "user"); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, application.ErrUserNotFound)
	}
	if err := repository.Delete(ctx, "missing"); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("Delete(missing) error = %v, want %v", err, application.ErrUserNotFound)
	}
	if err := repository.UpdateRoles(ctx, "user", []application.UserRole{application.UserRoleAdmin}); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("UpdateRoles(deleted) error = %v, want %v", err, application.ErrUserNotFound)
	}
}
