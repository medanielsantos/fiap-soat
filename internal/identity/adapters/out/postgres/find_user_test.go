package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestFindByEmailLoadsRolesAndPermissions(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("attendant", application.UserRoleAttendant))
	if err := repository.GrantPermissionToUser(ctx, "attendant", application.PermissionProductsCreate); err != nil {
		t.Fatal(err)
	}
	user, err := repository.FindByEmail(ctx, "attendant@example.test")
	if err != nil {
		t.Fatalf("FindByEmail(): %v", err)
	}
	if user.ID != "attendant" || user.Password != "hash-attendant" || len(user.Roles) != 1 || user.Roles[0] != application.UserRoleAttendant {
		t.Fatalf("FindByEmail() user = %+v", user)
	}
	if len(user.Permissions) != 1 || user.Permissions[0] != application.PermissionProductsCreate {
		t.Fatalf("FindByEmail() permissions = %v", user.Permissions)
	}
	if _, err := repository.FindByEmail(ctx, "missing@example.test"); err == nil {
		t.Fatal("FindByEmail() found unknown email")
	}
}

func TestFindByEmailIgnoresDeletedUsers(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("gone", application.UserRoleManager))
	if err := repository.Delete(ctx, "gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindByEmail(ctx, "gone@example.test"); err == nil {
		t.Fatal("FindByEmail() returned a deleted user")
	}
	replacement := testUser("replacement", application.UserRoleManager)
	replacement.Email = "gone@example.test"
	if err := repository.Create(ctx, replacement); err != nil {
		t.Fatalf("Create() reusing a deleted user's email: %v", err)
	}
}

func TestFindByIDCombinesRoleAndDirectPermissions(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("manager", application.UserRoleManager))
	user, err := repository.FindByID(ctx, "manager")
	if err != nil {
		t.Fatal(err)
	}
	if !hasPermission(user.Permissions, application.PermissionUsersRead) || hasPermission(user.Permissions, application.PermissionUsersDelete) {
		t.Fatalf("manager permissions = %v", user.Permissions)
	}
	if _, err := repository.FindByID(ctx, "missing"); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("FindByID(missing) error = %v, want %v", err, application.ErrUserNotFound)
	}
}

func TestListExcludesDeletedUsers(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		mustCreate(t, repository, testUser(id, application.UserRoleAttendant))
	}
	if err := repository.Delete(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	params, err := pagination.NewParams(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	page, err := repository.List(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalItems != 2 || len(page.Items) != 2 {
		t.Fatalf("List() total = %d, items = %d, want 2/2", page.TotalItems, len(page.Items))
	}
	for _, user := range page.Items {
		if user.ID == "b" || len(user.Roles) != 1 {
			t.Fatalf("List() returned %+v", user)
		}
	}
}
