package database

import (
	"context"

	"gorm.io/gorm"
)

type transactionKey struct{}

// Current returns the transaction bound to ctx, or the supplied database when
// the call is outside an application transaction.
func Current(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}

type TransactionManager struct{ db *gorm.DB }

func NewTransactionManager(db *gorm.DB) *TransactionManager { return &TransactionManager{db: db} }

func (manager *TransactionManager) Within(ctx context.Context, action func(context.Context) error) error {
	return manager.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return action(context.WithValue(ctx, transactionKey{}, tx))
	})
}
