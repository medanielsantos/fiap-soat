package application

import (
	"context"
	"fmt"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
)

type CustomerRepository interface {
	Save(ctx context.Context, customer domain.Customer) error
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type RegisterCustomerCommand struct {
	Document string
	Name     string
	Email    string
	Phone    string
}

type RegisterCustomer struct {
	repository  CustomerRepository
	idGenerator IDGenerator
}

func NewRegisterCustomer(repository CustomerRepository, idGenerator IDGenerator) *RegisterCustomer {
	return &RegisterCustomer{repository: repository, idGenerator: idGenerator}
}

func (useCase *RegisterCustomer) Execute(ctx context.Context, command RegisterCustomerCommand) (domain.Customer, error) {
	document, err := domain.NewDocument(command.Document)
	if err != nil {
		return domain.Customer{}, err
	}

	id, err := useCase.idGenerator.NewID(ctx)
	if err != nil {
		return domain.Customer{}, fmt.Errorf("generate customer ID: %w", err)
	}

	customer, err := domain.NewCustomer(id, document, command.Name, command.Email, command.Phone)
	if err != nil {
		return domain.Customer{}, err
	}

	if err := useCase.repository.Save(ctx, customer); err != nil {
		return domain.Customer{}, fmt.Errorf("save customer: %w", err)
	}

	return customer, nil
}
