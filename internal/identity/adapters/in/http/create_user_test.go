package identityhttp

import (
	"errors"
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

const createPath = "/api/v1/admin/users"

func createBody(roleField string) string {
	return `{"name":"New User","email":"new@example.test","password":"correct horse battery",` + roleField + `}`
}

func TestCreateUser(t *testing.T) {
	env := newTestEnv(t)
	response := env.do(http.MethodPost, createPath, env.admin(t), createBody(`"roles":["manager","attendant"]`))
	assertProblem(t, response, http.StatusCreated, "")
	created := decodeBody[application.UserView](t, response)
	if created.ID != "new-user-1" || created.Email != "new@example.test" || len(created.Roles) != 2 {
		t.Fatalf("created user = %+v", created)
	}
	if stored := env.store.users[created.ID]; stored.Password == "" || stored.Password == "correct horse battery" {
		t.Fatalf("stored password = %q, want a hash", stored.Password)
	}
}

func TestManagerCanOnlyCreateAttendants(t *testing.T) {
	tests := []struct {
		name, roleField string
		code            int
		title           string
	}{
		{"attendant", `"role":"attendant"`, http.StatusCreated, ""},
		{"admin", `"role":"admin"`, http.StatusForbidden, "Managers can only create attendants"},
		{"attendant and manager", `"roles":["attendant","manager"]`, http.StatusForbidden, "Managers can only create attendants"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			manager := env.addUser(t, "manager", []application.UserRole{application.UserRoleManager}, application.PermissionUsersCreate)
			assertProblem(t, env.do(http.MethodPost, createPath, manager, createBody(tt.roleField)), tt.code, tt.title)
		})
	}
}

func TestCreateUserMapsErrors(t *testing.T) {
	tests := []struct {
		name, body string
		storeErr   error
		code       int
		title      string
	}{
		{"malformed json", `{"name":`, nil, http.StatusBadRequest, "Invalid request"},
		{"role and roles", createBody(`"role":"admin","roles":["manager"]`), nil, http.StatusBadRequest, "Invalid user roles"},
		{"unknown role", createBody(`"role":"root"`), nil, http.StatusBadRequest, "Invalid user details"},
		{"short password", `{"name":"New","email":"new@example.test","password":"short","role":"attendant"}`, nil, http.StatusBadRequest, "Invalid user details"},
		{"invalid email", `{"name":"New","email":"nope","password":"correct horse battery","role":"attendant"}`, nil, http.StatusBadRequest, "Invalid user details"},
		{"duplicate email", createBody(`"role":"attendant"`), errors.New(`duplicate key value violates unique constraint "ux_users_email"`), http.StatusConflict, "Email already registered"},
		{"store failure", createBody(`"role":"attendant"`), errStoreFailure, http.StatusInternalServerError, "Could not register user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			if tt.storeErr != nil {
				env.store.fail["Create"] = tt.storeErr
			}
			assertProblem(t, env.do(http.MethodPost, createPath, env.admin(t), tt.body), tt.code, tt.title)
			if _, ok := env.store.users["new-user-1"]; ok {
				t.Fatal("user stored despite error")
			}
		})
	}
}
