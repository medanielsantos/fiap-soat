package identityhttp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestListUsers(t *testing.T) {
	env := newTestEnv(t)
	token := env.admin(t)
	env.addUser(t, "attendant", []application.UserRole{application.UserRoleAttendant})
	response := env.do(http.MethodGet, "/api/v1/admin/users?page=1&per_page=10", token, "")
	assertProblem(t, response, http.StatusOK, "")
	body := decodeBody[pagination.Response[application.UserView]](t, response)
	if len(body.Data) != 2 || body.Pagination.TotalItems != 2 || body.Pagination.PerPage != 10 {
		t.Fatalf("list response = %+v", body)
	}
}

func TestListUsersErrors(t *testing.T) {
	env := newTestEnv(t)
	token := env.admin(t)
	assertProblem(t, env.do(http.MethodGet, "/api/v1/admin/users?page=0", token, ""), http.StatusBadRequest, "Invalid pagination")
	env.store.fail["List"] = errStoreFailure
	assertProblem(t, env.do(http.MethodGet, "/api/v1/admin/users", token, ""), http.StatusInternalServerError, "Could not list users")
}

func TestGetUser(t *testing.T) {
	env := newTestEnv(t)
	token := env.admin(t)
	env.addUser(t, "target", []application.UserRole{application.UserRoleManager})
	response := env.do(http.MethodGet, "/api/v1/admin/users/target", token, "")
	assertProblem(t, response, http.StatusOK, "")
	if user := decodeBody[application.UserView](t, response); user.ID != "target" || user.Email != "target@example.test" {
		t.Fatalf("user = %+v", user)
	}
	if body := response.Body.String(); strings.Contains(body, "password") {
		t.Fatalf("response exposes password: %s", body)
	}
	assertProblem(t, env.do(http.MethodGet, "/api/v1/admin/users/missing", token, ""), http.StatusNotFound, "User not found")
	env.store.fail["FindByID:target"] = errStoreFailure
	assertProblem(t, env.do(http.MethodGet, "/api/v1/admin/users/target", token, ""), http.StatusInternalServerError, "Could not retrieve user")
}
