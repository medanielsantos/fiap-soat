package identityhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestUpdateRoles(t *testing.T) {
	tests := []struct {
		name, target, body, fail string
		code                     int
		title                    string
	}{
		{"assigns roles", "target", `{"roles":["manager","attendant"]}`, "", http.StatusOK, ""},
		{"malformed json", "target", `{"roles":"admin"}`, "", http.StatusBadRequest, "Invalid request"},
		{"unknown role", "target", `{"roles":["root"]}`, "", http.StatusBadRequest, "Invalid user role"},
		{"duplicate role", "target", `{"roles":["manager","manager"]}`, "", http.StatusBadRequest, "Invalid user role"},
		{"drops own admin role", "admin", `{"roles":["manager"]}`, "", http.StatusConflict, "Cannot change your own role"},
		{"missing user", "missing", `{"roles":["manager"]}`, "", http.StatusNotFound, "User not found"},
		{"store failure", "target", `{"roles":["manager"]}`, "UpdateRoles", http.StatusInternalServerError, "Could not update user role"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			token := env.admin(t)
			env.addUser(t, "target", []application.UserRole{application.UserRoleAttendant})
			if tt.fail != "" {
				env.store.fail[tt.fail] = errStoreFailure
			}
			response := env.do(http.MethodPut, "/api/v1/admin/users/"+tt.target+"/roles", token, tt.body)
			assertProblem(t, response, tt.code, tt.title)
			wantRoles := 1
			if tt.code == http.StatusOK {
				wantRoles = len(decodeBody[application.UserView](t, response).Roles)
			}
			if got := len(env.store.users["target"].Roles); got != wantRoles {
				t.Fatalf("target roles = %v, want %d roles", env.store.users["target"].Roles, wantRoles)
			}
		})
	}
}
