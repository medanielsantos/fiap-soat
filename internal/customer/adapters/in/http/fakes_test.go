package customerhttp

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

var errBackend = errors.New("backend unavailable")

type clientCustomerStore struct {
	*handlerCustomerStore
	plates    map[string]string
	owner     map[string]string
	listErr   error
	updateErr error
	resetErr  error
	resets    int
}

func newClientCustomerStore() *clientCustomerStore {
	return &clientCustomerStore{
		handlerCustomerStore: &handlerCustomerStore{customers: map[string]domain.Customer{}},
		plates:               map[string]string{}, owner: map[string]string{},
	}
}

func (store *clientCustomerStore) List(ctx context.Context, params pagination.Params) (pagination.Page[domain.Customer], error) {
	if store.listErr != nil {
		return pagination.Page[domain.Customer]{}, store.listErr
	}
	return store.handlerCustomerStore.List(ctx, params)
}

func (store *clientCustomerStore) Update(ctx context.Context, customer domain.Customer) error {
	if store.updateErr != nil {
		return store.updateErr
	}
	return store.handlerCustomerStore.Update(ctx, customer)
}

func (store *clientCustomerStore) FindByEmail(_ context.Context, email string) (domain.Customer, error) {
	for _, customer := range store.customers {
		if customer.Email() == email {
			return customer, nil
		}
	}
	return domain.Customer{}, application.ErrCustomerNotFound
}

func (store *clientCustomerStore) FindByPlateAndDocumentPrefix(_ context.Context, plate, _ string) (domain.Customer, string, error) {
	customer, ok := store.customers[store.owner[plate]]
	if !ok {
		return domain.Customer{}, "", application.ErrCustomerNotFound
	}
	return customer, store.plates[plate], nil
}

func (store *clientCustomerStore) CreateCustomerReset(context.Context, string, string, time.Time) error {
	store.resets++
	return nil
}

func (store *clientCustomerStore) ResetCustomerPassword(context.Context, string, string, time.Time) error {
	return store.resetErr
}

type fakeLimiter struct {
	mu      sync.Mutex
	deny    bool
	err     error
	keys    []string
	windows []time.Duration
}

func (limiter *fakeLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.keys = append(limiter.keys, key)
	limiter.windows = append(limiter.windows, window)
	return !limiter.deny, limiter.err
}

type fakeMailer struct {
	sent int
	err  error
}

func (mailer *fakeMailer) Send(context.Context, string, string, string) error {
	mailer.sent++
	return mailer.err
}
