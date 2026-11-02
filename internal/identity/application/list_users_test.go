package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestListUsersReturnsViewsWithoutPasswords(t *testing.T) {
	store := newMemoryStore()
	user := activeUser("user-1", UserRoleManager)
	user.Password = "secret-hash"
	store.users[user.ID] = user
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	params, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.ListUsers(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalItems != 1 || len(page.Items) != 1 || page.Items[0].ID != "user-1" || page.Items[0].Email != user.Email {
		t.Fatalf("ListUsers() = %+v", page)
	}
}

func TestListUsersPropagatesStoreFailure(t *testing.T) {
	service := NewService(newFailingStore("List"), fixedIDs{"unused"}, fixedTokens{}, nil, "")
	params, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListUsers(context.Background(), params); !errors.Is(err, errStoreFailure) {
		t.Fatalf("ListUsers() error = %v, want %v", err, errStoreFailure)
	}
}

func TestGetUser(t *testing.T) {
	store := newMemoryStore()
	store.users["user-1"] = activeUser("user-1", UserRoleAttendant)
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	got, err := service.GetUser(context.Background(), "user-1")
	if err != nil || got.ID != "user-1" || got.Status != UserStatusActive {
		t.Fatalf("GetUser() = (%+v, %v)", got, err)
	}
	if _, err := service.GetUser(context.Background(), "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetUser(missing) error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestActivePermissionsRejectsUnavailableUsers(t *testing.T) {
	store := newMemoryStore()
	store.users["inactive"] = User{ID: "inactive", Status: UserStatusInactive, Roles: []UserRole{UserRoleAdmin}}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	for _, id := range []string{"inactive", "missing"} {
		if roles, permissions, active := service.ActivePermissions(context.Background(), id); active || roles != nil || permissions != nil {
			t.Errorf("ActivePermissions(%q) = (%v, %v, %v), want inactive", id, roles, permissions, active)
		}
	}
	if _, _, active := (&Service{}).ActivePermissions(context.Background(), "any"); active {
		t.Error("service without store reported an active user")
	}
}
