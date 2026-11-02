package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestUpdateStatus(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("user", application.UserRoleAttendant))

	if err := repository.UpdateStatus(ctx, "user", application.UserStatusSuspended); err != nil {
		t.Fatal(err)
	}
	user, err := repository.FindByID(ctx, "user")
	if err != nil || user.Status != application.UserStatusSuspended || user.DeletedAt != nil {
		t.Fatalf("suspended user = (%+v, %v)", user, err)
	}

	if err := repository.UpdateStatus(ctx, "user", application.UserStatusDeleted); err != nil {
		t.Fatal(err)
	}
	user, err = repository.FindByID(ctx, "user")
	if err != nil || user.Status != application.UserStatusDeleted || user.DeletedAt == nil {
		t.Fatalf("deleted user = (%+v, %v)", user, err)
	}

	if err := repository.UpdateStatus(ctx, "user", application.UserStatusActive); err != nil {
		t.Fatal(err)
	}
	user, err = repository.FindByID(ctx, "user")
	if err != nil || user.Status != application.UserStatusActive || user.DeletedAt != nil {
		t.Fatalf("restored user = (%+v, %v)", user, err)
	}

	if err := repository.UpdateStatus(ctx, "missing", application.UserStatusActive); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("UpdateStatus(missing) error = %v, want %v", err, application.ErrUserNotFound)
	}
	if err := repository.UpdateStatus(ctx, "user", application.UserStatus("archived")); err == nil {
		t.Fatal("UpdateStatus() accepted a status rejected by the database check")
	}
}

func TestUpdateRoles(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("user", application.UserRoleAttendant))
	roles := []application.UserRole{application.UserRoleManager, application.UserRoleAdmin}
	if err := repository.UpdateRoles(ctx, "user", roles); err != nil {
		t.Fatal(err)
	}
	user, err := repository.FindByID(ctx, "user")
	if err != nil || len(user.Roles) != 2 || user.Roles[0] != application.UserRoleAdmin || user.Roles[1] != application.UserRoleManager {
		t.Fatalf("roles after update = (%v, %v)", user.Roles, err)
	}
	if err := repository.UpdateRoles(ctx, "user", []application.UserRole{"root"}); err == nil {
		t.Fatal("UpdateRoles() accepted an unknown role")
	}
	if err := repository.UpdateRoles(ctx, "missing", roles); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("UpdateRoles(missing) error = %v, want %v", err, application.ErrUserNotFound)
	}
}
