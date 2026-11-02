package identityhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestGrantAndRevokePermission(t *testing.T) {
	env := newTestEnv(t)
	token := env.admin(t)
	env.addUser(t, "target", []application.UserRole{application.UserRoleAttendant})
	path := "/api/v1/admin/users/target/permissions/users.read"
	assertProblem(t, env.do(http.MethodPost, path, token, ""), http.StatusNoContent, "")
	if got := env.store.users["target"].Permissions; len(got) != 1 || got[0] != application.PermissionUsersRead {
		t.Fatalf("permissions after grant = %v", got)
	}
	assertProblem(t, env.do(http.MethodDelete, path, token, ""), http.StatusNoContent, "")
	if got := env.store.users["target"].Permissions; len(got) != 0 {
		t.Fatalf("permissions after revoke = %v", got)
	}
}

func TestPermissionEndpointsMapErrors(t *testing.T) {
	tests := []struct {
		name, method, path, fail string
		code                     int
		title                    string
	}{
		{"grant unknown permission", http.MethodPost, "/api/v1/admin/users/target/permissions/system.root", "", http.StatusBadRequest, "Invalid permission"},
		{"grant forbidden for attendant", http.MethodPost, "/api/v1/admin/users/target/permissions/catalog.products.delete", "", http.StatusBadRequest, "Invalid permission"},
		{"grant missing user", http.MethodPost, "/api/v1/admin/users/missing/permissions/users.read", "", http.StatusNotFound, "User not found"},
		{"grant store failure", http.MethodPost, "/api/v1/admin/users/target/permissions/users.read", "GrantPermission", http.StatusInternalServerError, "Could not update user permissions"},
		{"revoke unknown permission", http.MethodDelete, "/api/v1/admin/users/target/permissions/system.root", "", http.StatusBadRequest, "Invalid permission"},
		{"revoke missing user", http.MethodDelete, "/api/v1/admin/users/missing/permissions/users.read", "", http.StatusNotFound, "User not found"},
		{"revoke store failure", http.MethodDelete, "/api/v1/admin/users/target/permissions/users.read", "RevokePermission", http.StatusInternalServerError, "Could not update user permissions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			token := env.admin(t)
			env.addUser(t, "target", []application.UserRole{application.UserRoleAttendant})
			if tt.fail != "" {
				env.store.fail[tt.fail] = errStoreFailure
			}
			assertProblem(t, env.do(tt.method, tt.path, token, ""), tt.code, tt.title)
		})
	}
}
