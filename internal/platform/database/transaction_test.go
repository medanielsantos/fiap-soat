package database

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/testdb"
	"gorm.io/gorm"
)

func newTransactionFixture(t *testing.T) (*gorm.DB, *TransactionManager) {
	t.Helper()
	db := testdb.Open(t)
	if err := db.Exec("CREATE TABLE tx_probe (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	return db, NewTransactionManager(db)
}

func countProbes(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Table("tx_probe").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestWithinCommitsWorkDoneThroughCurrent(t *testing.T) {
	db, manager := newTransactionFixture(t)

	err := manager.Within(context.Background(), func(ctx context.Context) error {
		return Current(ctx, db).Exec("INSERT INTO tx_probe (value) VALUES ('committed')").Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := countProbes(t, db); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
}

func TestWithinRollsBackWhenActionFails(t *testing.T) {
	db, manager := newTransactionFixture(t)
	boom := errors.New("boom")

	err := manager.Within(context.Background(), func(ctx context.Context) error {
		if err := Current(ctx, db).Exec("INSERT INTO tx_probe (value) VALUES ('discarded')").Error; err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if got := countProbes(t, db); got != 0 {
		t.Fatalf("rows = %d, want 0 after rollback", got)
	}
}

func TestCurrentFallsBackOutsideTransaction(t *testing.T) {
	db, _ := newTransactionFixture(t)
	if err := Current(context.Background(), db).Exec("INSERT INTO tx_probe (value) VALUES ('direct')").Error; err != nil {
		t.Fatal(err)
	}
	if got := countProbes(t, db); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
}
