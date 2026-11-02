package application

import (
	"context"
	"errors"
	"testing"
)

func TestDirectPermissionsAreAppliedAndRevoked(t *testing.T) {
	store := newMemoryStore()
	store.users["attendant"] = User{ID: "attendant", Status: UserStatusActive, Roles: []UserRole{UserRoleAttendant}}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	if err := service.GrantPermission(context.Background(), "attendant", PermissionProductsCreate); err != nil {
		t.Fatal(err)
	}
	roles, permissions, active := service.ActivePermissions(context.Background(), "attendant")
	if !active || len(roles) != 1 || len(permissions) != 1 || permissions[0] != PermissionProductsCreate {
		t.Fatalf("authorization = (%v, %v, %v)", roles, permissions, active)
	}
	if err := service.RevokePermission(context.Background(), "attendant", PermissionProductsCreate); err != nil {
		t.Fatal(err)
	}
	_, permissions, _ = service.ActivePermissions(context.Background(), "attendant")
	if len(permissions) != 0 {
		t.Fatalf("permissions after revoke = %v", permissions)
	}
}

func TestRoleAndPermissionValidation(t *testing.T) {
	if validRoles([]UserRole{UserRoleAdmin, UserRoleAdmin}) {
		t.Fatal("duplicate roles were accepted")
	}
	if validRoles([]UserRole{"unknown"}) {
		t.Fatal("unknown role was accepted")
	}
	if validPermission(Permission("system.root")) {
		t.Fatal("unknown permission was accepted")
	}
	store := newMemoryStore()
	store.users["attendant"] = User{ID: "attendant", Status: UserStatusActive, Roles: []UserRole{UserRoleAttendant}}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	if err := service.GrantPermission(context.Background(), "attendant", PermissionProductsDelete); !errors.Is(err, ErrInvalidPermission) {
		t.Fatalf("attendant product-delete grant error = %v", err)
	}
}

func TestGrantAndRevokePermissionErrors(t *testing.T) {
	store := newMemoryStore()
	store.users["manager"] = activeUser("manager", UserRoleManager)
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	ctx := context.Background()
	if err := service.GrantPermission(ctx, "manager", PermissionUsersDelete); !errors.Is(err, ErrInvalidPermission) {
		t.Errorf("manager users.delete grant error = %v, want %v", err, ErrInvalidPermission)
	}
	if err := service.GrantPermission(ctx, "manager", Permission("system.root")); !errors.Is(err, ErrInvalidPermission) {
		t.Errorf("unknown permission grant error = %v, want %v", err, ErrInvalidPermission)
	}
	if err := service.GrantPermission(ctx, "missing", PermissionUsersRead); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("missing user grant error = %v, want %v", err, ErrUserNotFound)
	}
	if err := service.RevokePermission(ctx, "manager", Permission("system.root")); !errors.Is(err, ErrInvalidPermission) {
		t.Errorf("unknown permission revoke error = %v, want %v", err, ErrInvalidPermission)
	}
	if len(store.users["manager"].Permissions) != 0 {
		t.Errorf("rejected grants were stored: %v", store.users["manager"].Permissions)
	}
}
