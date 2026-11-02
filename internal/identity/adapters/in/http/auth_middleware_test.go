package identityhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestRequireAuthRejectsUnauthenticatedRequests(t *testing.T) {
	env := newTestEnv(t)
	inactive := env.addUser(t, "inactive", []application.UserRole{application.UserRoleAdmin}, allPermissions...)
	user := env.store.users["inactive"]
	user.Status = application.UserStatusSuspended
	env.store.users["inactive"] = user
	orphanToken, _, err := env.issuer.Issue("missing")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, authorization, title string
	}{
		{"missing header", "", "Authentication required"},
		{"non bearer scheme", "Basic YWRtaW46cGFzcw==", "Authentication required"},
		{"bearer without token", "Bearer", "Authentication required"},
		{"malformed token", "Bearer not-a-jwt", "Invalid access token"},
		{"inactive user", "Bearer " + inactive, "Invalid access token"},
		{"unknown user", "Bearer " + orphanToken, "Invalid access token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := env.doWithAuthorization(http.MethodGet, "/api/v1/admin/users", tt.authorization)
			assertProblem(t, response, http.StatusUnauthorized, tt.title)
		})
	}
}

func TestRequireAuthAcceptsCaseInsensitiveScheme(t *testing.T) {
	env := newTestEnv(t)
	token := env.admin(t)
	response := env.doWithAuthorization(http.MethodGet, "/api/v1/admin/users", "bearer "+token)
	assertProblem(t, response, http.StatusOK, "")
}

func TestRequirePermissionDeniesMissingPermission(t *testing.T) {
	env := newTestEnv(t)
	attendant := env.addUser(t, "attendant", []application.UserRole{application.UserRoleAttendant})
	reader := env.addUser(t, "reader", []application.UserRole{application.UserRoleManager}, application.PermissionUsersRead)
	env.addUser(t, "target", []application.UserRole{application.UserRoleAttendant})
	tests := []struct {
		name, method, path, token, body string
	}{
		{"list users", http.MethodGet, "/api/v1/admin/users", attendant, ""},
		{"create user", http.MethodPost, "/api/v1/admin/users", reader, `{}`},
		{"update status", http.MethodPatch, "/api/v1/admin/users/target/status", reader, `{"status":"inactive"}`},
		{"assign roles", http.MethodPut, "/api/v1/admin/users/target/roles", reader, `{"roles":["admin"]}`},
		{"grant permission", http.MethodPost, "/api/v1/admin/users/target/permissions/users.read", reader, ""},
		{"delete user", http.MethodDelete, "/api/v1/admin/users/target", reader, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProblem(t, env.do(tt.method, tt.path, tt.token, tt.body), http.StatusForbidden, "Insufficient permissions")
		})
	}
	if env.store.users["target"].Status != application.UserStatusActive || len(env.store.users["target"].Roles) != 1 {
		t.Fatalf("denied requests changed the target: %+v", env.store.users["target"])
	}
}
