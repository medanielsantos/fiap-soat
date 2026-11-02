package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrDuplicateDocument  = errors.New("customer document already exists")
	ErrDuplicateEmail     = errors.New("customer email already exists")
	ErrInvalidPassword    = errors.New("customer password must contain between 12 and 72 bytes")
	ErrInvalidCredentials = errors.New("invalid customer credentials")
	ErrInvalidResetToken  = errors.New("invalid or expired customer password reset token")
)

type CustomerStore interface {
	Save(context.Context, domain.Customer) error
	FindByID(context.Context, string) (domain.Customer, error)
	List(context.Context, pagination.Params) (pagination.Page[domain.Customer], error)
	Update(context.Context, domain.Customer) error
	Delete(context.Context, string) error
}

type CustomerCredentialStore interface {
	FindByEmail(context.Context, string) (domain.Customer, error)
	FindByPlateAndDocumentPrefix(context.Context, string, string) (domain.Customer, string, error)
}

type CustomerPasswordResetStore interface {
	CreateCustomerReset(context.Context, string, string, time.Time) error
	ResetCustomerPassword(context.Context, string, string, time.Time) error
}

type CustomerMailer interface {
	Send(context.Context, string, string, string) error
}
type CustomerResetOptions struct {
	Mailer CustomerMailer
	URL    string
}

type CustomerService struct {
	store       CustomerStore
	idGenerator IDGenerator
	mailer      CustomerMailer
	resetURL    string
	now         func() time.Time
}

func NewCustomerService(store CustomerStore, idGenerator IDGenerator, options ...CustomerResetOptions) *CustomerService {
	service := &CustomerService{store: store, idGenerator: idGenerator, now: time.Now}
	if len(options) > 0 {
		service.mailer, service.resetURL = options[0].Mailer, strings.TrimRight(options[0].URL, "/")
	}
	return service
}

type CustomerInput struct {
	Document string
	Name     string
	Email    string
	Phone    string
	Password string
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
	hash, err := hashPassword(input.Password)
	if err != nil {
		return domain.Customer{}, err
	}
	customer, err := domain.RestoreCustomerWithPassword(id, document, input.Name, input.Email, input.Phone, status, hash)
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
	existing, err := service.store.FindByID(ctx, id)
	if err != nil {
		return domain.Customer{}, err
	}
	document, err := domain.NewDocument(input.Document)
	if err != nil {
		return domain.Customer{}, err
	}
	hash := existing.Password()
	if input.Password != "" {
		hash, err = hashPassword(input.Password)
		if err != nil {
			return domain.Customer{}, err
		}
	}
	customer, err := domain.RestoreCustomerWithPassword(id, document, input.Name, input.Email, input.Phone, input.Status, hash)
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

func (service *CustomerService) Authenticate(ctx context.Context, email, password string) (domain.Customer, error) {
	store, ok := service.store.(CustomerCredentialStore)
	if !ok {
		return domain.Customer{}, ErrInvalidCredentials
	}
	customer, err := store.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || !customer.IsActive() || customer.Password() == "" || bcrypt.CompareHashAndPassword([]byte(customer.Password()), []byte(password)) != nil {
		return domain.Customer{}, ErrInvalidCredentials
	}
	return customer, nil
}

func (service *CustomerService) AuthenticateVehicle(ctx context.Context, plate, prefix string) (domain.Customer, string, error) {
	store, ok := service.store.(CustomerCredentialStore)
	if !ok {
		return domain.Customer{}, "", ErrInvalidCredentials
	}
	var digits strings.Builder
	for _, r := range prefix {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		} else if r != '.' && r != '-' && r != '/' && r != ' ' {
			return domain.Customer{}, "", errors.New("invalid access credentials")
		}
	}
	if digits.Len() != 5 {
		return domain.Customer{}, "", errors.New("invalid access credentials")
	}
	normalizedPlate := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(plate)))
	customer, vehicleID, err := store.FindByPlateAndDocumentPrefix(ctx, normalizedPlate, digits.String())
	if err != nil || !customer.IsActive() || !strings.HasPrefix(customer.Document().String(), digits.String()) {
		return domain.Customer{}, "", errors.New("invalid access credentials")
	}
	return customer, vehicleID, nil
}

func (service *CustomerService) RequestPasswordReset(ctx context.Context, email string) error {
	if service.resetURL == "" {
		return errors.New("CUSTOMER_PASSWORD_RESET_URL is required")
	}
	store, ok := service.store.(CustomerCredentialStore)
	if !ok {
		return errors.New("customer credential store is unavailable")
	}
	customer, err := store.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || !customer.IsActive() {
		return nil
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(token))
	resetStore, ok := service.store.(CustomerPasswordResetStore)
	if !ok {
		return errors.New("customer password reset store is unavailable")
	}
	if err := resetStore.CreateCustomerReset(ctx, customer.ID(), hex.EncodeToString(digest[:]), service.now().Add(30*time.Minute)); err != nil {
		return err
	}
	if service.mailer == nil {
		return errors.New("SMTP mail adapter is not configured")
	}
	link, err := url.Parse(service.resetURL)
	if err != nil {
		return errors.New("CUSTOMER_PASSWORD_RESET_URL is invalid")
	}
	query := link.Query()
	query.Set("token", token)
	link.RawQuery = query.Encode()
	return service.mailer.Send(ctx, customer.Email(), "Redefinição de senha", link.String())
}

func (service *CustomerService) ResetPassword(ctx context.Context, token, password string) error {
	if len(password) < 12 || len(password) > 72 {
		return ErrInvalidPassword
	}
	resetStore, ok := service.store.(CustomerPasswordResetStore)
	if !ok {
		return ErrInvalidResetToken
	}
	digest := sha256.Sum256([]byte(token))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return resetStore.ResetCustomerPassword(ctx, hex.EncodeToString(digest[:]), string(hash), service.now())
}

func hashPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	if len(password) < 12 || len(password) > 72 {
		return "", ErrInvalidPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
