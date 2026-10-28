package application

import (
	"context"
	"errors"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

var (
	ErrCustomerNotFound  = errors.New("customer not found")
	ErrDuplicateDocument = errors.New("customer document already exists")
)

type CustomerStore interface {
	Save(context.Context, domain.Customer) error
	FindByID(context.Context, string) (domain.Customer, error)
	List(context.Context, pagination.Params) (pagination.Page[domain.Customer], error)
	Update(context.Context, domain.Customer) error
	Delete(context.Context, string) error
}

type CustomerService struct {
	store       CustomerStore
	idGenerator IDGenerator
}

func NewCustomerService(store CustomerStore, idGenerator IDGenerator) *CustomerService {
	return &CustomerService{store: store, idGenerator: idGenerator}
}

type CustomerInput struct {
	Document string
	Name     string
	Email    string
	Phone    string
	Status   domain.CustomerStatus
}

func (service *CustomerService) Create(ctx context.Context, input CustomerInput) (domain.Customer, error) {
	document, err := domain.NewDocument(input.Document)
	if err != nil {
		return domain.Customer{}, err
	}
	id, err := service.idGenerator.NewID(ctx)
	if err != nil {
		return domain.Customer{}, err
	}
	status := input.Status
	if status == "" {
		status = domain.CustomerStatusActive
	}
	customer, err := domain.RestoreCustomer(id, document, input.Name, input.Email, input.Phone, status)
	if err != nil {
		return domain.Customer{}, err
	}
	if err := service.store.Save(ctx, customer); err != nil {
		return domain.Customer{}, err
	}
	return customer, nil
}

func (service *CustomerService) Get(ctx context.Context, id string) (domain.Customer, error) {
	return service.store.FindByID(ctx, id)
}

func (service *CustomerService) List(ctx context.Context, params pagination.Params) (pagination.Page[domain.Customer], error) {
	return service.store.List(ctx, params)
}

func (service *CustomerService) Update(ctx context.Context, id string, input CustomerInput) (domain.Customer, error) {
	if _, err := service.store.FindByID(ctx, id); err != nil {
		return domain.Customer{}, err
	}
	document, err := domain.NewDocument(input.Document)
	if err != nil {
		return domain.Customer{}, err
	}
	customer, err := domain.RestoreCustomer(id, document, input.Name, input.Email, input.Phone, input.Status)
	if err != nil {
		return domain.Customer{}, err
	}
	if err := service.store.Update(ctx, customer); err != nil {
		return domain.Customer{}, err
	}
	return customer, nil
}

func (service *CustomerService) Delete(ctx context.Context, id string) error {
	return service.store.Delete(ctx, id)
}
