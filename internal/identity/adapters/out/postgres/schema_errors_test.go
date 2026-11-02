package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestRepositoryPropagatesQueryErrorsInsideOperations(t *testing.T) {
	params, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	user := testUser("user", application.UserRoleAdmin)
	tests := []struct {
		name, dropTable string
		operation       func(context.Context, *Repository) error
	}{
		{"seed role permissions", "role_permissions", func(ctx context.Context, r *Repository) error { return r.SeedAuthorization(ctx) }},
		{"seed permissions", "permissions", func(ctx context.Context, r *Repository) error { return r.SeedAuthorization(ctx) }},
		{"create first count", "users", func(ctx context.Context, r *Repository) error { return r.CreateFirst(ctx, user) }},
		{"grant permission count", "users", func(ctx context.Context, r *Repository) error {
			return r.GrantPermissionToUser(ctx, user.ID, application.PermissionUsersRead)
		}},
		{"find roles", "user_roles", func(ctx context.Context, r *Repository) error { _, err := r.FindByID(ctx, user.ID); return err }},
		{"list permissions", "user_permissions", func(ctx context.Context, r *Repository) error { _, err := r.List(ctx, params); return err }},
		{"replace roles", "user_roles", func(ctx context.Context, r *Repository) error { return r.UpdateRoles(ctx, user.ID, user.Roles) }},
		{"create reset", "password_reset_tokens", func(ctx context.Context, r *Repository) error {
			return r.CreateReset(ctx, user.ID, resetHash("a"), time.Now().Add(time.Hour))
		}},
		{"reset password", "password_reset_tokens", func(ctx context.Context, r *Repository) error {
			return r.ResetPassword(ctx, resetHash("a"), "hash", time.Now())
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestRepository(t)
			ctx := context.Background()
			mustCreate(t, repository, user)
			if err := repository.db.Exec(`DROP TABLE ` + tt.dropTable + ` CASCADE`).Error; err != nil {
				t.Fatalf("drop %s: %v", tt.dropTable, err)
			}
			if err := tt.operation(ctx, repository); err == nil {
				t.Fatalf("operation succeeded without table %s", tt.dropTable)
			}
		})
	}
}
