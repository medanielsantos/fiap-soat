package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

func movement(t *testing.T, id, partID string, kind domain.MovementType, quantity int64) domain.StockMovement {
	t.Helper()
	m, err := domain.NewStockMovement(id, partID, kind, quantity, "manual", "clerk")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRepositoryMoveStock(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()
	savePart(t, repo, "p1", 5)
	if err := db.Exec(`UPDATE parts SET reserved_stock = 2 WHERE id = 'p1'`).Error; err != nil {
		t.Fatal(err)
	}

	got, err := repo.MoveStock(ctx, movement(t, "m1", "p1", domain.MovementEntry, 4))
	if err != nil || got.Stock() != 9 || got.ReservedStock() != 2 {
		t.Fatalf("MoveStock(entry) = stock %d reserved %d, %v; want 9/2", got.Stock(), got.ReservedStock(), err)
	}
	got, err = repo.MoveStock(ctx, movement(t, "m2", "p1", domain.MovementExit, 7))
	if err != nil || got.Stock() != 2 || got.AvailableStock() != 0 {
		t.Fatalf("MoveStock(exit) = stock %d available %d, %v; want 2/0", got.Stock(), got.AvailableStock(), err)
	}

	tests := []struct {
		name string
		m    domain.StockMovement
		want error
	}{
		{"exit_exceeds_available_stock", movement(t, "m3", "p1", domain.MovementExit, 1), domain.ErrInsufficientStock},
		{"part_is_missing", movement(t, "m4", "missing", domain.MovementEntry, 1), application.ErrNotFound},
		{"movement_id_is_reused", movement(t, "m1", "p1", domain.MovementEntry, 1), application.ErrDuplicate},
	}
	for _, tt := range tests {
		t.Run("test_it_should_fail_when_"+tt.name, func(t *testing.T) {
			if _, err := repo.MoveStock(ctx, tt.m); !errors.Is(err, tt.want) {
				t.Errorf("MoveStock() error = %v, want %v", err, tt.want)
			}
			if stock, _ := stockOf(t, repo, "p1"); stock != 2 {
				t.Errorf("stock = %d after failed movement, want 2 (rolled back)", stock)
			}
		})
	}

	t.Run("test_it_should_list_movements_newest_first_with_pagination", func(t *testing.T) {
		page, err := repo.ListStockMovements(ctx, "p1", firstPage(t))
		if err != nil || page.TotalItems != 2 || page.Items[0].ID != "m2" || page.Items[0].Quantity != 7 || page.Items[0].PerformedBy != "clerk" {
			t.Fatalf("ListStockMovements() = %+v, %v", page, err)
		}
		if _, err := repo.ListStockMovements(canceledContext(), "p1", firstPage(t)); err == nil {
			t.Error("ListStockMovements() with canceled context returned nil error")
		}
	})

	t.Run("test_it_should_fail_when_context_is_canceled", func(t *testing.T) {
		if _, err := repo.MoveStock(canceledContext(), movement(t, "m9", "p1", domain.MovementEntry, 1)); err == nil {
			t.Error("MoveStock() with canceled context returned nil error")
		}
	})
}
