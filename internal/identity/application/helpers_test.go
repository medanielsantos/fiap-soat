package application

import (
	"context"
	"errors"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

var errStoreFailure = errors.New("store failure")

type failingStore struct {
	*memoryStore
	fail map[string]bool
}

func newFailingStore(ops ...string) *failingStore {
	fail := make(map[string]bool, len(ops))
	for _, op := range ops {
		fail[op] = true
	}
	return &failingStore{memoryStore: newMemoryStore(), fail: fail}
}

func (s *failingStore) Create(ctx context.Context, user User) error {
	if s.fail["Create"] {
		return errStoreFailure
	}
	return s.memoryStore.Create(ctx, user)
}
func (s *failingStore) CreateFirst(ctx context.Context, user User) error {
	if s.fail["CreateFirst"] {
		return errStoreFailure
	}
	return s.memoryStore.CreateFirst(ctx, user)
}
func (s *failingStore) FindByID(ctx context.Context, id string) (User, error) {
	if s.fail["FindByID"] {
		return User{}, errStoreFailure
	}
	return s.memoryStore.FindByID(ctx, id)
}
func (s *failingStore) List(ctx context.Context, params pagination.Params) (pagination.Page[User], error) {
	if s.fail["List"] {
		return pagination.Page[User]{}, errStoreFailure
	}
	return s.memoryStore.List(ctx, params)
}
func (s *failingStore) CreateReset(ctx context.Context, userID, hash string, expires time.Time) error {
	if s.fail["CreateReset"] {
		return errStoreFailure
	}
	return s.memoryStore.CreateReset(ctx, userID, hash, expires)
}

type failingIDs struct{}

func (failingIDs) NewID(context.Context) (string, error) { return "", errors.New("id failure") }

type failingTokens struct{}

func (failingTokens) Issue(string) (string, time.Time, error) {
	return "", time.Time{}, errors.New("token failure")
}

func activeUser(id string, roles ...UserRole) User {
	return User{ID: id, Name: id, Email: id + "@example.test", Status: UserStatusActive, Roles: roles}
}
