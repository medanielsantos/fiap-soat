package identityhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"golang.org/x/crypto/bcrypt"
)

const loginPassword = "correct horse battery"

func addLoginUser(t *testing.T, env *testEnv, status application.UserStatus) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(loginPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	env.store.users["user-1"] = application.User{ID: "user-1", Name: "User", Email: "user@example.test", Password: string(hash), Status: status, Roles: []application.UserRole{application.UserRoleManager}}
}

func TestLoginIssuesAccessToken(t *testing.T) {
	env := newTestEnv(t)
	addLoginUser(t, env, application.UserStatusActive)
	response := env.do(http.MethodPost, "/api/v1/auth/login", "", `{"email":" USER@example.test ","password":"`+loginPassword+`"}`)
	assertProblem(t, response, http.StatusOK, "")
	session := decodeBody[application.Session](t, response)
	if session.User.ID != "user-1" || session.User.Email != "user@example.test" {
		t.Fatalf("session user = %+v", session.User)
	}
	if id, err := env.issuer.Validate(session.AccessToken); err != nil || id != "user-1" {
		t.Fatalf("issued token subject = (%q, %v)", id, err)
	}
}

func TestLoginRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name, body string
		status     application.UserStatus
		code       int
		title      string
	}{
		{"malformed json", `{"email":`, application.UserStatusActive, http.StatusBadRequest, "Invalid request"},
		{"unknown field", `{"email":"user@example.test","password":"x","otp":"1"}`, application.UserStatusActive, http.StatusBadRequest, "Invalid request"},
		{"trailing object", `{"email":"user@example.test","password":"x"}{}`, application.UserStatusActive, http.StatusBadRequest, "Invalid request"},
		{"wrong password", `{"email":"user@example.test","password":"wrong horse battery"}`, application.UserStatusActive, http.StatusUnauthorized, "Invalid credentials"},
		{"unknown email", `{"email":"other@example.test","password":"` + loginPassword + `"}`, application.UserStatusActive, http.StatusUnauthorized, "Invalid credentials"},
		{"suspended user", `{"email":"user@example.test","password":"` + loginPassword + `"}`, application.UserStatusSuspended, http.StatusUnauthorized, "Invalid credentials"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			addLoginUser(t, env, tt.status)
			assertProblem(t, env.do(http.MethodPost, "/api/v1/auth/login", "", tt.body), tt.code, tt.title)
		})
	}
}
