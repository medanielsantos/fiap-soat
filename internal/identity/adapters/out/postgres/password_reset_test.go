package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func resetHash(seed string) string { return strings.Repeat(seed, 64)[:64] }

func TestResetPasswordConsumesTokenOnce(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	mustCreate(t, repository, testUser("user", application.UserRoleAdmin))
	if err := repository.CreateReset(ctx, "user", resetHash("a"), now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateReset(ctx, "user", resetHash("b"), now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repository.ResetPassword(ctx, resetHash("a"), "new-hash", now); !errors.Is(err, application.ErrInvalidResetToken) {
		t.Fatalf("superseded token error = %v, want %v", err, application.ErrInvalidResetToken)
	}
	if err := repository.ResetPassword(ctx, resetHash("b"), "new-hash", now); err != nil {
		t.Fatalf("ResetPassword(): %v", err)
	}
	user, err := repository.FindByID(ctx, "user")
	if err != nil || user.Password != "new-hash" {
		t.Fatalf("password after reset = (%q, %v)", user.Password, err)
	}
	if err := repository.ResetPassword(ctx, resetHash("b"), "other-hash", now); !errors.Is(err, application.ErrInvalidResetToken) {
		t.Fatalf("reused token error = %v, want %v", err, application.ErrInvalidResetToken)
	}
}

func TestResetPasswordRejectsExpiredTokensAndInactiveUsers(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	mustCreate(t, repository, testUser("expired", application.UserRoleAdmin))
	mustCreate(t, repository, testUser("suspended", application.UserRoleAdmin))
	if err := repository.CreateReset(ctx, "expired", resetHash("c"), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateReset(ctx, "suspended", resetHash("d"), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpdateStatus(ctx, "suspended", application.UserStatusSuspended); err != nil {
		t.Fatal(err)
	}
	for _, hash := range []string{resetHash("c"), resetHash("d"), resetHash("e")} {
		if err := repository.ResetPassword(ctx, hash, "new-hash", now); !errors.Is(err, application.ErrInvalidResetToken) {
			t.Errorf("ResetPassword(%s) error = %v, want %v", hash[:1], err, application.ErrInvalidResetToken)
		}
	}
	user, err := repository.FindByID(ctx, "suspended")
	if err != nil || user.Password != "hash-suspended" {
		t.Fatalf("suspended user's password changed: (%q, %v)", user.Password, err)
	}
	if err := repository.CreateReset(ctx, "missing", resetHash("f"), now.Add(time.Hour)); err == nil {
		t.Fatal("CreateReset() accepted an unknown user")
	}
}
