package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

func TestRepositoryServiceLifecycle(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()
	b := newService(t, "s-b", "S-B", "Brakes")
	a := newService(t, "s-a", "S-A", "Alignment")
	for _, s := range []domain.Service{b, a} {
		if err := repo.SaveService(ctx, s); err != nil {
			t.Fatalf("SaveService(%s) error = %v", s.ID(), err)
		}
	}
	if err := repo.SaveService(ctx, newService(t, "s-c", "S-A", "Dup")); !errors.Is(err, application.ErrDuplicate) {
		t.Errorf("SaveService(duplicate code) error = %v, want %v", err, application.ErrDuplicate)
	}

	got, err := repo.FindService(ctx, "s-a")
	if err != nil || got.Name() != "Alignment" || got.PerformedBy() != "admin" || got.SalePriceInCents() != 5000 {
		t.Fatalf("FindService() = %+v, %v", got, err)
	}
	page, err := repo.ListServices(ctx, firstPage(t))
	if err != nil || page.TotalItems != 2 || page.Items[0].ID() != "s-a" {
		t.Fatalf("ListServices() = %+v, %v; want 2 ordered by name", page, err)
	}

	updated, _ := domain.NewService("s-a", "S-A2", "Alignment 3D", "", 7000, 45, false, "editor")
	if err := repo.UpdateService(ctx, updated); err != nil {
		t.Fatalf("UpdateService() error = %v", err)
	}
	got, _ = repo.FindService(ctx, "s-a")
	if got.InternalCode() != "S-A2" || got.SalePriceInCents() != 7000 || got.Active() {
		t.Errorf("after update = %+v", got)
	}
	var updatedBy string
	db.Raw(`SELECT updated_by FROM catalog_services WHERE id = 's-a'`).Scan(&updatedBy)
	if updatedBy != "editor" {
		t.Errorf("updated_by = %q, want editor", updatedBy)
	}
	if err := repo.UpdateService(ctx, newService(t, "s-b", "S-A2", "Brakes")); !errors.Is(err, application.ErrDuplicate) {
		t.Errorf("UpdateService(duplicate code) error = %v, want %v", err, application.ErrDuplicate)
	}

	if err := repo.DeleteService(ctx, "s-a"); err != nil {
		t.Fatalf("DeleteService() error = %v", err)
	}
	if _, err := repo.FindService(ctx, "s-a"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("FindService(deleted) error = %v, want %v", err, application.ErrNotFound)
	}
	if err := repo.DeleteService(ctx, "s-a"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("DeleteService(deleted) error = %v, want %v", err, application.ErrNotFound)
	}
	if err := repo.UpdateService(ctx, updated); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("UpdateService(deleted) error = %v, want %v", err, application.ErrNotFound)
	}
}

func TestRepositoryServiceCorruptRows(t *testing.T) {
	repo, db := newTestRepository(t)
	if err := db.Exec(`INSERT INTO catalog_services (id, internal_code, name, sale_price_in_cents, estimated_duration_minutes) VALUES ('bad', 'S-X', '  ', 100, 10)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindService(context.Background(), "bad"); !errors.Is(err, domain.ErrInvalidName) {
		t.Errorf("FindService() error = %v, want %v", err, domain.ErrInvalidName)
	}
	if _, err := repo.ListServices(context.Background(), firstPage(t)); !errors.Is(err, domain.ErrInvalidName) {
		t.Errorf("ListServices() error = %v, want %v", err, domain.ErrInvalidName)
	}
}

func TestRepositoryServiceDriverErrors(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := canceledContext()
	if _, err := repo.ListServices(ctx, firstPage(t)); err == nil {
		t.Error("ListServices() with canceled context returned nil error")
	}
	if err := repo.DeleteService(ctx, "x"); err == nil {
		t.Error("DeleteService() with canceled context returned nil error")
	}
}
