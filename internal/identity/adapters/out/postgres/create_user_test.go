package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestCreateFirstOnlyAcceptsTheFirstUser(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	if err := repository.CreateFirst(ctx, testUser("first", application.UserRoleAdmin)); err != nil {
		t.Fatalf("CreateFirst() first user: %v", err)
	}
	if err := repository.CreateFirst(ctx, testUser("second", application.UserRoleAdmin)); !errors.Is(err, application.ErrRegistrationClosed) {
		t.Fatalf("CreateFirst() second user error = %v, want %v", err, application.ErrRegistrationClosed)
	}
	if _, err := repository.FindByID(ctx, "second"); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("second user persisted: %v", err)
	}
}

func TestCreateRejectsDuplicateEmailAndUnknownRole(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	mustCreate(t, repository, testUser("user-1", application.UserRoleManager))
	duplicate := testUser("user-2", application.UserRoleManager)
	duplicate.Email = "user-1@example.test"
	if err := repository.Create(ctx, duplicate); !IsUniqueViolation(err) {
		t.Fatalf("Create() duplicate email error = %v, want unique violation", err)
	}
	if err := repository.Create(ctx, testUser("user-3", application.UserRole("root"))); err == nil || IsUniqueViolation(err) {
		t.Fatalf("Create() unknown role error = %v, want foreign key failure", err)
	}
	if _, err := repository.FindByID(ctx, "user-3"); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("user with unknown role was not rolled back: %v", err)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("connection refused"), false},
		{errors.New(`ERROR: duplicate key value violates UNIQUE constraint "ux_users_email"`), true},
	}
	for _, tt := range tests {
		if got := IsUniqueViolation(tt.err); got != tt.want {
			t.Errorf("IsUniqueViolation(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
