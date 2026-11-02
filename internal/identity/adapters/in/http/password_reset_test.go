package identityhttp

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestForgotPasswordSendsResetLink(t *testing.T) {
	env := newTestEnv(t)
	env.addUser(t, "user", []application.UserRole{application.UserRoleManager})
	assertProblem(t, env.do(http.MethodPost, "/api/v1/auth/password/forgot", "", `{"email":"USER@example.test"}`), http.StatusAccepted, "")
	link, err := url.Parse(env.mailer.body)
	if err != nil || env.mailer.to != "user@example.test" || link.Query().Get("token") == "" {
		t.Fatalf("reset mail to %q body %q (parse err %v)", env.mailer.to, env.mailer.body, err)
	}
	if len(env.store.resetHashes) != 1 || env.store.resetHashes[0] == link.Query().Get("token") {
		t.Fatalf("stored reset hashes = %v, want one hash differing from the plain token", env.store.resetHashes)
	}
}

func TestForgotPasswordDoesNotRevealFailures(t *testing.T) {
	tests := []struct {
		name, email string
		mailErr     error
	}{
		{"unknown email", "missing@example.test", nil},
		{"mailer failure", "user@example.test", errors.New("smtp down")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.addUser(t, "user", []application.UserRole{application.UserRoleManager})
			env.mailer.err = tt.mailErr
			assertProblem(t, env.do(http.MethodPost, "/api/v1/auth/password/forgot", "", `{"email":"`+tt.email+`"}`), http.StatusAccepted, "")
		})
	}
	env := newTestEnv(t)
	assertProblem(t, env.do(http.MethodPost, "/api/v1/auth/password/forgot", "", `{"mail":"x"}`), http.StatusBadRequest, "Invalid request")
}

func TestResetPassword(t *testing.T) {
	tests := []struct {
		name, body string
		storeErr   error
		code       int
		title      string
	}{
		{"valid token", `{"token":"abc","password":"replacement secure password"}`, nil, http.StatusNoContent, ""},
		{"malformed json", `{"token":`, nil, http.StatusBadRequest, "Invalid request"},
		{"short password", `{"token":"abc","password":"short"}`, nil, http.StatusBadRequest, "Invalid or expired reset token"},
		{"invalid token", `{"token":"abc","password":"replacement secure password"}`, application.ErrInvalidResetToken, http.StatusBadRequest, "Invalid or expired reset token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			if tt.storeErr != nil {
				env.store.fail["ResetPassword"] = tt.storeErr
			}
			assertProblem(t, env.do(http.MethodPost, "/api/v1/auth/password/reset", "", tt.body), tt.code, tt.title)
		})
	}
}
