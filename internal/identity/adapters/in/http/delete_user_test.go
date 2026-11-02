package identityhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name, target, fail string
		code               int
		title              string
	}{
		{"deletes another user", "target", "", http.StatusNoContent, ""},
		{"rejects self deletion", "admin", "", http.StatusConflict, "Cannot delete your own user"},
		{"missing user", "missing", "", http.StatusNotFound, "User not found"},
		{"store failure", "target", "Delete", http.StatusInternalServerError, "Could not delete user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			token := env.admin(t)
			env.addUser(t, "target", []application.UserRole{application.UserRoleManager})
			if tt.fail != "" {
				env.store.fail[tt.fail] = errStoreFailure
			}
			assertProblem(t, env.do(http.MethodDelete, "/api/v1/admin/users/"+tt.target, token, ""), tt.code, tt.title)
			deleted := env.store.users["target"].DeletedAt != nil
			if deleted != (tt.code == http.StatusNoContent) {
				t.Fatalf("target deleted = %v after status %d", deleted, tt.code)
			}
			if env.store.users["admin"].DeletedAt != nil {
				t.Fatal("requesting admin was deleted")
			}
		})
	}
}
