package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"golang.org/x/crypto/bcrypt"
)

type credentialCustomerStore struct {
	*memoryCustomerStore
	byEmail      map[string]domain.Customer
	byPlate      map[string]plateMatch
	findErr      error
	resetCreated []resetRequest
	resetErr     error
	resetHash    string
	resetToken   string
	resetAt      time.Time
}

type plateMatch struct {
	customer  domain.Customer
	vehicleID string
}

type resetRequest struct {
	customerID, tokenHash string
	expires               time.Time
}

func newCredentialCustomerStore() *credentialCustomerStore {
	return &credentialCustomerStore{
		memoryCustomerStore: newMemoryCustomerStore(),
		byEmail:             map[string]domain.Customer{},
		byPlate:             map[string]plateMatch{},
	}
}

func (store *credentialCustomerStore) FindByEmail(_ context.Context, email string) (domain.Customer, error) {
	if store.findErr != nil {
		return domain.Customer{}, store.findErr
	}
	customer, ok := store.byEmail[email]
	if !ok {
		return domain.Customer{}, ErrCustomerNotFound
	}
	return customer, nil
}

func (store *credentialCustomerStore) FindByPlateAndDocumentPrefix(_ context.Context, plate, prefix string) (domain.Customer, string, error) {
	match, ok := store.byPlate[plate+"|"+prefix]
	if !ok {
		return domain.Customer{}, "", ErrCustomerNotFound
	}
	return match.customer, match.vehicleID, nil
}

func (store *credentialCustomerStore) CreateCustomerReset(_ context.Context, customerID, tokenHash string, expires time.Time) error {
	if store.resetErr != nil {
		return store.resetErr
	}
	store.resetCreated = append(store.resetCreated, resetRequest{customerID: customerID, tokenHash: tokenHash, expires: expires})
	return nil
}

func (store *credentialCustomerStore) ResetCustomerPassword(_ context.Context, tokenHash, password string, now time.Time) error {
	store.resetToken, store.resetHash, store.resetAt = tokenHash, password, now
	return store.resetErr
}

type credentialOnlyStore struct {
	*memoryCustomerStore
	customer domain.Customer
}

func (store credentialOnlyStore) FindByEmail(context.Context, string) (domain.Customer, error) {
	return store.customer, nil
}

func (store credentialOnlyStore) FindByPlateAndDocumentPrefix(context.Context, string, string) (domain.Customer, string, error) {
	return store.customer, "", nil
}

type recordingMailer struct {
	to, subject, body string
	err               error
}

func (mailer *recordingMailer) Send(_ context.Context, to, subject, body string) error {
	mailer.to, mailer.subject, mailer.body = to, subject, body
	return mailer.err
}

var errStoreUnavailable = errors.New("store unavailable")

func newTestCustomer(t *testing.T, id, document, email string, status domain.CustomerStatus, password string) domain.Customer {
	t.Helper()
	doc, err := domain.NewDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	hash := ""
	if password != "" {
		raw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		hash = string(raw)
	}
	customer, err := domain.RestoreCustomerWithPassword(id, doc, "Ana Silva", email, "123", status, hash)
	if err != nil {
		t.Fatal(err)
	}
	return customer
}
