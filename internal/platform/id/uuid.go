package id

import (
	"context"
	"uuid"
)

type UUIDGenerator struct{}

func (UUIDGenerator) NewID(context.Context) (string, error) {
	return uuid.NewV7().String(), nil
}
