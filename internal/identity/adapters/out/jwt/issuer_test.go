package jwt

import (
	"strings"
	"testing"
	"time"
)

func TestIssuerIssuesAndValidatesAccessTokens(t *testing.T) {
	issuer, err := NewIssuer("test-secret-with-at-least-32-bytes", "test-suite", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, expiresAt, err := issuer.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("token expiry %v is not in the future", expiresAt)
	}
	userID, err := issuer.Validate(token)
	if err != nil || userID != "user-123" {
		t.Fatalf("validated subject = %q, error = %v", userID, err)
	}
}

func TestIssuerRejectsTamperedAndWrongIssuerTokens(t *testing.T) {
	issuer, err := NewIssuer("test-secret-with-at-least-32-bytes", "test-suite", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	otherIssuer, err := NewIssuer("other-secret-with-at-least-32-bytes", "test-suite", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := otherIssuer.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Validate(token); err == nil {
		t.Fatal("token signed with another key was accepted")
	}
	validToken, _, err := issuer.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	tampered := validToken[:len(validToken)-1] + strings.Repeat("x", 1)
	if _, err := issuer.Validate(tampered); err == nil {
		t.Fatal("tampered token was accepted")
	}
	wrongIssuer, err := NewIssuer("test-secret-with-at-least-32-bytes", "another-issuer", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	wrongIssuerToken, _, err := wrongIssuer.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Validate(wrongIssuerToken); err == nil {
		t.Fatal("token from another issuer was accepted")
	}
}
