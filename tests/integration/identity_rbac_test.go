package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
)

func TestIdentityRBACManagementEndpoints(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()
	for _, user := range []identityapp.User{
		{ID: "integration-manager", Name: "Manager", Email: "manager@example.test", Password: "unused", Status: identityapp.UserStatusActive, Roles: []identityapp.UserRole{identityapp.UserRoleManager}},
		{ID: "integration-attendant", Name: "Attendant", Email: "attendant@example.test", Password: "unused", Status: identityapp.UserStatusActive, Roles: []identityapp.UserRole{identityapp.UserRoleAttendant}},
	} {
		if err := server.identityRepo.Create(ctx, user); err != nil {
			t.Fatalf("create %s fixture: %v", user.Name, err)
		}
	}

	page := call(t, server, http.MethodGet, "/api/v1/admin-users?page=1&per_page=2", nil, http.StatusOK)
	if page["pagination"].(map[string]any)["total_items"] != float64(3) {
		t.Fatalf("user list pagination = %v", page["pagination"])
	}
	user := call(t, server, http.MethodGet, "/api/v1/admin-users/integration-attendant", nil, http.StatusOK)
	if roles := user["roles"].([]any); len(roles) != 1 || roles[0] != string(identityapp.UserRoleAttendant) {
		t.Fatalf("attendant roles = %v", roles)
	}

	call(t, server, http.MethodPut, "/api/v1/admin-users/integration-attendant/roles", map[string]any{"roles": []string{"attendant", "manager"}}, http.StatusOK)
	user = call(t, server, http.MethodGet, "/api/v1/admin-users/integration-attendant", nil, http.StatusOK)
	if roles := user["roles"].([]any); len(roles) != 2 {
		t.Fatalf("updated roles = %v", roles)
	}
	call(t, server, http.MethodPost, "/api/v1/admin-users/integration-attendant/permissions/catalog.products.create", nil, http.StatusNoContent)
	attendantIssuer, err := identityjwt.NewIssuer("integration-secret-with-at-least-32-bytes", "integration", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	attendantToken, _, err := attendantIssuer.Issue("integration-attendant")
	if err != nil {
		t.Fatal(err)
	}
	callWithToken(t, server, attendantToken, http.MethodPost, "/api/v1/parts", map[string]any{
		"sku": "RBAC-ATTENDANT", "internal_code": "RBAC-ATTENDANT", "name": "Attendant part",
		"unit": "piece", "purchase_price_in_cents": 100, "sale_price_in_cents": 250,
		"initial_stock": 1, "created_by": "integration-attendant",
	}, http.StatusCreated)
	call(t, server, http.MethodDelete, "/api/v1/admin-users/integration-attendant/permissions/catalog.products.create", nil, http.StatusNoContent)
	call(t, server, http.MethodPost, "/api/v1/admin-users/integration-attendant/permissions/catalog.products.price.update", nil, http.StatusBadRequest)

	issuer, err := identityjwt.NewIssuer("integration-secret-with-at-least-32-bytes", "integration", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	managerToken, _, err := issuer.Issue("integration-manager")
	if err != nil {
		t.Fatal(err)
	}
	callWithToken(t, server, managerToken, http.MethodGet, "/api/v1/admin-users", nil, http.StatusOK)
	callWithToken(t, server, managerToken, http.MethodPost, "/api/v1/admin-users", map[string]any{
		"name": "Created attendant", "email": "created-attendant@example.test", "password": "a sufficiently long password", "roles": []string{"attendant"},
	}, http.StatusCreated)
	callWithToken(t, server, managerToken, http.MethodPost, "/api/v1/admin-users", map[string]any{
		"name": "Invalid manager target", "email": "invalid-target@example.test", "password": "a sufficiently long password", "roles": []string{"manager"},
	}, http.StatusForbidden)
	callWithToken(t, server, managerToken, http.MethodDelete, "/api/v1/admin-users/integration-attendant", nil, http.StatusForbidden)
	call(t, server, http.MethodPost, "/api/v1/admin-users/integration-attendant/permissions/catalog.products.delete", nil, http.StatusBadRequest)
}
