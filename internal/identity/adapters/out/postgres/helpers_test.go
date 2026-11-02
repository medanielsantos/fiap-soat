package postgres

import (
	"context"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	repository := NewRepository(testdb.Open(t))
	if err := repository.SeedAuthorization(context.Background()); err != nil {
		t.Fatalf("seed authorization: %v", err)
	}
	return repository
}

func testUser(id string, roles ...application.UserRole) application.User {
	return application.User{ID: id, Name: "User " + id, Email: id + "@example.test", Password: "hash-" + id, Status: application.UserStatusActive, Roles: roles}
}

func mustCreate(t *testing.T, repository *Repository, user application.User) {
	t.Helper()
	if err := repository.Create(context.Background(), user); err != nil {
		t.Fatalf("create user %s: %v", user.ID, err)
	}
}

func hasPermission(permissions []application.Permission, target application.Permission) bool {
	for _, permission := range permissions {
		if permission == target {
			return true
		}
	}
	return false
}
