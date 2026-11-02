package application

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
)

func TestPasswordResetExpiresAndCanOnlyBeUsedOnce(t *testing.T) {
	store := newMemoryStore()
	user := User{ID: "user-1", Name: "Admin", Email: "admin@example.test", Password: "old-hash", Status: UserStatusActive, Roles: []UserRole{UserRoleAdmin}}
	store.users[user.ID] = user
	mailer := &captureMailer{}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, mailer, "https://example.test/reset")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	service.now = func() time.Time { return now }
	if err := service.RequestPasswordReset(context.Background(), user.Email); err != nil {
		t.Fatal(err)
	}
	link, err := url.Parse(mailer.body)
	if err != nil {
		t.Fatal(err)
	}
	token := link.Query().Get("token")
	if token == "" {
		t.Fatal("reset email did not contain a token")
	}
	if err := service.ResetPassword(context.Background(), token, "replacement secure password"); err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(context.Background(), token, "another secure password"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("reused token error = %v", err)
	}
	if err := service.RequestPasswordReset(context.Background(), user.Email); err != nil {
		t.Fatal(err)
	}
	link, err = url.Parse(mailer.body)
	if err != nil {
		t.Fatal(err)
	}
	token = link.Query().Get("token")
	now = now.Add(31 * time.Minute)
	if err := service.ResetPassword(context.Background(), token, "expired secure password"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("expired token error = %v", err)
	}
}
