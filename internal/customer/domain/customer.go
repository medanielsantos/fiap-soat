package domain

import (
	"errors"
	"strings"
)

var (
	ErrInvalidCustomerID = errors.New("customer ID is required")
	ErrInvalidName       = errors.New("customer name is required")
	ErrInvalidEmail      = errors.New("invalid customer email")
	ErrInvalidPhone      = errors.New("customer phone is required")
)

type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "active"
	CustomerStatusInactive CustomerStatus = "inactive"
)

type Customer struct {
	id     string
	doc    Document
	name   string
	email  string
	phone  string
	status CustomerStatus
}

func NewCustomer(id string, document Document, name, email, phone string) (Customer, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)

	switch {
	case id == "":
		return Customer{}, ErrInvalidCustomerID
	case !isValidDocument(document):
		return Customer{}, ErrInvalidDocument
	case name == "":
		return Customer{}, ErrInvalidName
	case !validEmail(email):
		return Customer{}, ErrInvalidEmail
	case phone == "":
		return Customer{}, ErrInvalidPhone
	}

	return Customer{
		id:     id,
		doc:    document,
		name:   name,
		email:  email,
		phone:  phone,
		status: CustomerStatusActive,
	}, nil
}

func isValidDocument(document Document) bool {
	_, err := NewDocument(document.String())
	return err == nil
}

func (c Customer) ID() string             { return c.id }
func (c Customer) Document() Document     { return c.doc }
func (c Customer) Name() string           { return c.name }
func (c Customer) Email() string          { return c.email }
func (c Customer) Phone() string          { return c.phone }
func (c Customer) Status() CustomerStatus { return c.status }
func (c Customer) IsActive() bool         { return c.status == CustomerStatusActive }

func (c *Customer) Deactivate() {
	c.status = CustomerStatusInactive
}

func (c *Customer) Activate() {
	c.status = CustomerStatusActive
}

func validEmail(value string) bool {
	if strings.Count(value, "@") != 1 {
		return false
	}
	parts := strings.SplitN(value, "@", 2)
	return parts[0] != "" && strings.Contains(parts[1], ".") &&
		!strings.HasPrefix(parts[1], ".") && !strings.HasSuffix(parts[1], ".")
}
