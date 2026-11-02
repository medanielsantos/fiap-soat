package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

func TestRepositoryPropagatesDatabaseErrors(t *testing.T) {
	repository := newTestRepository(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	params, err := pagination.NewParams(1, pagination.DefaultPerPage)
	if err != nil {
		t.Fatal(err)
	}
	user := testUser("user", application.UserRoleAdmin)
	operations := map[string]func() error{
		"SeedAuthorization": func() error { return repository.SeedAuthorization(ctx) },
		"CreateFirst":       func() error { return repository.CreateFirst(ctx, user) },
		"Create":            func() error { return repository.Create(ctx, user) },
		"FindByEmail":       func() error { _, err := repository.FindByEmail(ctx, user.Email); return err },
		"FindByID":          func() error { _, err := repository.FindByID(ctx, user.ID); return err },
		"List":              func() error { _, err := repository.List(ctx, params); return err },
		"UpdateStatus":      func() error { return repository.UpdateStatus(ctx, user.ID, application.UserStatusActive) },
		"UpdateRoles":       func() error { return repository.UpdateRoles(ctx, user.ID, user.Roles) },
		"Delete":            func() error { return repository.Delete(ctx, user.ID) },
		"GrantPermission":   func() error { return repository.GrantPermissionToUser(ctx, user.ID, application.PermissionUsersRead) },
		"RevokePermission": func() error {
			return repository.RevokePermissionFromUser(ctx, user.ID, application.PermissionUsersRead)
		},
		"CreateReset":   func() error { return repository.CreateReset(ctx, user.ID, resetHash("a"), time.Now().Add(time.Hour)) },
		"ResetPassword": func() error { return repository.ResetPassword(ctx, resetHash("a"), "hash", time.Now()) },
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			if err := operation(); err == nil {
				t.Fatal("operation succeeded with a canceled context")
			}
		})
	}
}
