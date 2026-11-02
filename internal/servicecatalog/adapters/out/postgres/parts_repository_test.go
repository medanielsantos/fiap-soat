package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

func TestRepositoryPartLifecycle(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()
	stocked := newPart(t, "p-b", "SKU-B", 5)
	opening, _ := domain.NewStockMovement("m-open", "p-b", domain.MovementEntry, 5, "initial stock", "admin")
	if err := repo.SavePart(ctx, stocked, &opening); err != nil {
		t.Fatalf("SavePart(with opening) error = %v", err)
	}
	empty := savePart(t, repo, "p-a", 0)
	if err := repo.SavePart(ctx, newPart(t, "p-c", "SKU-B", 0), nil); !errors.Is(err, application.ErrDuplicate) {
		t.Errorf("SavePart(duplicate sku) error = %v, want %v", err, application.ErrDuplicate)
	}

	movements, err := repo.ListStockMovements(ctx, "p-b", firstPage(t))
	if err != nil || movements.TotalItems != 1 || movements.Items[0].Description != "initial stock" || movements.Items[0].Type != "entry" {
		t.Fatalf("opening movement = %+v, %v", movements, err)
	}
	page, err := repo.ListParts(ctx, firstPage(t))
	if err != nil || page.TotalItems != 2 || page.Items[0].ID() != "p-a" || page.Items[1].Stock() != 5 {
		t.Fatalf("ListParts() = %+v, %v", page, err)
	}

	changed, _ := domain.NewPart(empty.ID(), "SKU-A2", "P-A2", "Renamed", "", "box", 600, 1500, 0, 0, false, "editor")
	if err := repo.UpdatePart(ctx, changed); err != nil {
		t.Fatalf("UpdatePart() error = %v", err)
	}
	got, _ := repo.FindPart(ctx, "p-a")
	if got.SKU() != "SKU-A2" || got.Unit() != "box" || got.SalePriceInCents() != 1500 || got.Active() {
		t.Errorf("after update = %+v", got)
	}
	if err := repo.UpdatePart(ctx, newPart(t, "p-a", "SKU-B", 0)); !errors.Is(err, application.ErrDuplicate) {
		t.Errorf("UpdatePart(duplicate sku) error = %v, want %v", err, application.ErrDuplicate)
	}
	if err := repo.UpdatePart(ctx, newPart(t, "missing", "SKU-Z", 0)); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("UpdatePart(missing) error = %v, want %v", err, application.ErrNotFound)
	}
}

func TestRepositoryDeletePart(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()
	savePart(t, repo, "stocked", 3)
	savePart(t, repo, "empty", 0)

	if err := repo.DeletePart(ctx, "stocked"); !errors.Is(err, domain.ErrPartHasStock) {
		t.Errorf("DeletePart(stocked) error = %v, want %v", err, domain.ErrPartHasStock)
	}
	if err := repo.DeletePart(ctx, "empty"); err != nil {
		t.Fatalf("DeletePart(empty) error = %v", err)
	}
	if _, err := repo.FindPart(ctx, "empty"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("FindPart(deleted) error = %v, want %v", err, application.ErrNotFound)
	}
	if err := repo.DeletePart(ctx, "empty"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("DeletePart(deleted) error = %v, want %v", err, application.ErrNotFound)
	}
	// soft-deleted SKU may be reused
	if err := repo.SavePart(ctx, newPart(t, "again", "SKU-empty", 0), nil); err != nil {
		t.Errorf("SavePart(reused sku) error = %v", err)
	}
}

func TestRepositoryPartCorruptRowsAndDriverErrors(t *testing.T) {
	repo, db := newTestRepository(t)
	if err := db.Exec(`INSERT INTO parts (id, sku, internal_code, name, unit, sale_price_in_cents) VALUES ('bad', 'SKU-X', 'P-X', 'Bad', ' ', 100)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindPart(context.Background(), "bad"); !errors.Is(err, domain.ErrInvalidUnit) {
		t.Errorf("FindPart() error = %v, want %v", err, domain.ErrInvalidUnit)
	}
	if _, err := repo.ListParts(context.Background(), firstPage(t)); !errors.Is(err, domain.ErrInvalidUnit) {
		t.Errorf("ListParts() error = %v, want %v", err, domain.ErrInvalidUnit)
	}
	if _, err := repo.ListParts(canceledContext(), firstPage(t)); err == nil {
		t.Error("ListParts() with canceled context returned nil error")
	}
}
