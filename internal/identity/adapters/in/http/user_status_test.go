package identityhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestUpdateStatus(t *testing.T) {
	statusUpdater := []application.Permission{application.PermissionUsersStatusUpdate}
	tests := []struct {
		name, target, body string
		permissions        []application.Permission
		fail               string
		code               int
		title              string
	}{
		{"suspends user", "target", `{"status":"suspended"}`, statusUpdater, "", http.StatusOK, ""},
		{"deletes with delete permission", "target", `{"status":"deleted"}`, allPermissions, "", http.StatusOK, ""},
		{"delete requires delete permission", "target", `{"status":"deleted"}`, statusUpdater, "", http.StatusForbidden, "Insufficient permissions to delete users"},
		{"malformed json", "target", `{"status":1}`, statusUpdater, "", http.StatusBadRequest, "Invalid request"},
		{"unknown status", "target", `{"status":"archived"}`, statusUpdater, "", http.StatusBadRequest, "Invalid user status"},
		{"self deactivation", "requester", `{"status":"inactive"}`, statusUpdater, "", http.StatusBadRequest, "Invalid user status"},
		{"missing user", "missing", `{"status":"inactive"}`, statusUpdater, "", http.StatusNotFound, "User not found"},
		{"store failure", "target", `{"status":"inactive"}`, statusUpdater, "UpdateStatus", http.StatusInternalServerError, "Could not update user status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			token := env.addUser(t, "requester", []application.UserRole{application.UserRoleManager}, tt.permissions...)
			env.addUser(t, "target", []application.UserRole{application.UserRoleAttendant})
			if tt.fail != "" {
				env.store.fail[tt.fail] = errStoreFailure
			}
			response := env.do(http.MethodPatch, "/api/v1/admin/users/"+tt.target+"/status", token, tt.body)
			assertProblem(t, response, tt.code, tt.title)
			want := application.UserStatusActive
			if tt.code == http.StatusOK {
				want = decodeBody[application.UserView](t, response).Status
			}
			if got := env.store.users["target"].Status; got != want {
				t.Fatalf("target status = %q, want %q", got, want)
			}
		})
	}
}
