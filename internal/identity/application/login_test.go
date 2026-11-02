package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const loginPassword = "correct horse battery"

func loginStore(t *testing.T, user User) *memoryStore {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(loginPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user.Password = string(hash)
	store := newMemoryStore()
	store.users[user.ID] = user
	return store
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	deletedAt := time.Now()
	deleted := activeUser("deleted", UserRoleAdmin)
	deleted.DeletedAt = &deletedAt
	tests := []struct {
		name, email, password string
		user                  User
	}{
		{"unknown email", "missing@example.test", loginPassword, activeUser("user", UserRoleAdmin)},
		{"wrong password", "user@example.test", "wrong horse battery", activeUser("user", UserRoleAdmin)},
		{"inactive user", "inactive@example.test", loginPassword, User{ID: "inactive", Email: "inactive@example.test", Status: UserStatusInactive}},
		{"soft-deleted user", "deleted@example.test", loginPassword, deleted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(deletedVisibleStore{loginStore(t, tt.user)}, fixedIDs{"unused"}, fixedTokens{}, nil, "")
			if _, err := service.Login(context.Background(), tt.email, tt.password); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
			}
		})
	}
}

type deletedVisibleStore struct{ *memoryStore }

func (s deletedVisibleStore) FindByEmail(_ context.Context, email string) (User, error) {
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return User{}, ErrUserNotFound
}

func TestLoginNormalizesEmailAndPropagatesTokenFailure(t *testing.T) {
	store := loginStore(t, activeUser("user", UserRoleAdmin))
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	session, err := service.Login(context.Background(), "  USER@example.test ", loginPassword)
	if err != nil || session.User.ID != "user" {
		t.Fatalf("Login() = (%+v, %v)", session, err)
	}
	failing := NewService(store, fixedIDs{"unused"}, failingTokens{}, nil, "")
	if _, err := failing.Login(context.Background(), "user@example.test", loginPassword); err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() token failure error = %v", err)
	}
}
