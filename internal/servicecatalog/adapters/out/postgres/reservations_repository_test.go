package postgres

import (
	"context"
	"errors"
	"testing"

	platformdatabase "github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

func TestRepositoryReserveAndConsumeStock(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()
	seedWorkOrder(t, db, "wo-1")
	savePart(t, repo, "p1", 5)

	if err := repo.ReserveStock(ctx, "wo-1", "cust-1", []application.StockRequirement{{PartID: "p1", MovementID: "r1", Quantity: 3}}); err != nil {
		t.Fatalf("ReserveStock() error = %v", err)
	}
	if stock, reserved := stockOf(t, repo, "p1"); stock != 5 || reserved != 3 {
		t.Fatalf("after reserve stock/reserved = %d/%d, want 5/3", stock, reserved)
	}
	if err := repo.ConsumeReservedStock(ctx, "wo-1", "mech-1", []application.StockRequirement{{PartID: "p1", MovementID: "c1", Quantity: 2}}); err != nil {
		t.Fatalf("ConsumeReservedStock() error = %v", err)
	}
	if stock, reserved := stockOf(t, repo, "p1"); stock != 3 || reserved != 1 {
		t.Fatalf("after consume stock/reserved = %d/%d, want 3/1", stock, reserved)
	}

	page, err := repo.ListStockMovements(ctx, "p1", firstPage(t))
	if err != nil || page.TotalItems != 2 {
		t.Fatalf("ListStockMovements() = %+v, %v", page, err)
	}
	for _, m := range page.Items {
		if m.WorkOrderID == nil || *m.WorkOrderID != "wo-1" {
			t.Errorf("movement %s work order = %v, want wo-1", m.ID, m.WorkOrderID)
		}
	}
}

func TestRepositoryReservationFailures(t *testing.T) {
	repo, db := newTestRepository(t)
	seedWorkOrder(t, db, "wo-1")
	savePart(t, repo, "p1", 2)
	req := func(part, movementID string, quantity int64) []application.StockRequirement {
		return []application.StockRequirement{{PartID: part, MovementID: movementID, Quantity: quantity}}
	}
	if err := repo.ReserveStock(context.Background(), "wo-1", "c", req("p1", "used", 1)); err != nil {
		t.Fatal(err)
	}

	type op func(context.Context, string, string, []application.StockRequirement) error
	tests := []struct {
		name string
		run  op
		ctx  context.Context
		reqs []application.StockRequirement
		want error
	}{
		{"reserve_more_than_available", repo.ReserveStock, context.Background(), req("p1", "r2", 2), domain.ErrInsufficientStock},
		{"reserve_missing_part", repo.ReserveStock, context.Background(), req("nope", "r3", 1), application.ErrNotFound},
		{"consume_more_than_reserved", repo.ConsumeReservedStock, context.Background(), req("p1", "c2", 2), domain.ErrInsufficientStock},
		{"consume_missing_part", repo.ConsumeReservedStock, context.Background(), req("nope", "c3", 1), application.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run("test_it_should_fail_when_"+tt.name, func(t *testing.T) {
			if err := tt.run(tt.ctx, "wo-1", "u", tt.reqs); !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}

	for name, run := range map[string]op{"reserve": repo.ReserveStock, "consume": repo.ConsumeReservedStock} {
		t.Run("test_it_should_roll_back_"+name+"_when_movement_id_is_reused", func(t *testing.T) {
			err := platformdatabase.NewTransactionManager(db).Within(context.Background(), func(ctx context.Context) error {
				return run(ctx, "wo-1", "u", req("p1", "used", 1))
			})
			if !errors.Is(err, application.ErrDuplicate) {
				t.Errorf("error = %v, want %v", err, application.ErrDuplicate)
			}
			if stock, reserved := stockOf(t, repo, "p1"); stock != 2 || reserved != 1 {
				t.Errorf("stock/reserved = %d/%d, want 2/1", stock, reserved)
			}
		})
	}

	t.Run("test_it_should_fail_when_context_is_canceled", func(t *testing.T) {
		if err := repo.ReserveStock(canceledContext(), "wo-1", "c", req("p1", "x1", 1)); err == nil {
			t.Error("ReserveStock() with canceled context returned nil error")
		}
		if err := repo.ConsumeReservedStock(canceledContext(), "wo-1", "u", req("p1", "x2", 1)); err == nil {
			t.Error("ConsumeReservedStock() with canceled context returned nil error")
		}
	})
}
