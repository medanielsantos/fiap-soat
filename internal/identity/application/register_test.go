package application

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestRegisterAndLoginWithMultipleRoles(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, fixedIDs{"user-1"}, fixedTokens{}, nil, "")
	created, err := service.Register(context.Background(), "Admin", "ADMIN@example.com", "correct horse battery", []UserRole{UserRoleAdmin, UserRoleManager})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Roles) != 2 || created.Email != "admin@example.com" {
		t.Fatalf("created user = %+v", created)
	}
	user := store.users[created.ID]
	if user.Password == "correct horse battery" || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("correct horse battery")) != nil {
		t.Fatal("password was not stored as a valid bcrypt hash")
	}
	session, err := service.Login(context.Background(), created.Email, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if session.AccessToken != "access-user-1" {
		t.Fatalf("access token = %q", session.AccessToken)
	}
	store.users[created.ID] = User{ID: created.ID, Email: user.Email, Password: user.Password, Status: UserStatusSuspended, Roles: user.Roles}
	if _, err := service.Login(context.Background(), created.Email, "correct horse battery"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login for suspended user error = %v", err)
	}
}

func TestSeedAdminCreatesOnlyTheFirstUser(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, fixedIDs{"seed-admin"}, fixedTokens{}, nil, "")
	created, err := service.SeedAdmin(context.Background(), "Seed Admin", "admin@example.test", "initial secure password")
	if err != nil || !created {
		t.Fatalf("first seed = (%v, %v)", created, err)
	}
	user := store.users["seed-admin"]
	if len(user.Roles) != 1 || user.Roles[0] != UserRoleAdmin {
		t.Fatalf("seed roles = %v", user.Roles)
	}
	created, err = service.SeedAdmin(context.Background(), "Other Admin", "other@example.test", "another secure password")
	if err != nil || created {
		t.Fatalf("repeated seed = (%v, %v), want no-op", created, err)
	}
	if len(store.users) != 1 {
		t.Fatalf("users after repeated seed = %d, want 1", len(store.users))
	}
}
