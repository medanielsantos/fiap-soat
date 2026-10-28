package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"gorm.io/gorm"
)

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (repository *CustomerRepository) Save(ctx context.Context, customer domain.Customer) error {
	return translateError(repository.db.WithContext(ctx).Create(recordFromDomain(customer)).Error)
}

func (repository *CustomerRepository) FindByID(ctx context.Context, id string) (domain.Customer, error) {
	var record customerRecord
	err := repository.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if err != nil {
		return domain.Customer{}, translateError(err)
	}
	return record.toDomain()
}

func (repository *CustomerRepository) List(ctx context.Context) ([]domain.Customer, error) {
	var records []customerRecord
	if err := repository.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, translateError(err)
	}
	customers := make([]domain.Customer, 0, len(records))
	for _, record := range records {
		customer, err := record.toDomain()
		if err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	return customers, nil
}

func (repository *CustomerRepository) Update(ctx context.Context, customer domain.Customer) error {
	result := repository.db.WithContext(ctx).Model(&customerRecord{}).Where("id = ?", customer.ID()).Updates(map[string]any{
		"document":   customer.Document().String(),
		"name":       customer.Name(),
		"email":      customer.Email(),
		"phone":      customer.Phone(),
		"status":     string(customer.Status()),
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return application.ErrCustomerNotFound
	}
	return nil
}

func (repository *CustomerRepository) Delete(ctx context.Context, id string) error {
	result := repository.db.WithContext(ctx).Delete(&customerRecord{}, "id = ?", id)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return application.ErrCustomerNotFound
	}
	return nil
}

func recordFromDomain(customer domain.Customer) *customerRecord {
	return &customerRecord{
		ID: customer.ID(), Document: customer.Document().String(), Name: customer.Name(),
		Email: customer.Email(), Phone: customer.Phone(), Status: string(customer.Status()),
	}
}

func (record customerRecord) toDomain() (domain.Customer, error) {
	document, err := domain.NewDocument(record.Document)
	if err != nil {
		return domain.Customer{}, err
	}
	return domain.RestoreCustomer(record.ID, document, record.Name, record.Email, record.Phone, domain.CustomerStatus(record.Status))
}

func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return application.ErrCustomerNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return application.ErrDuplicateDocument
	}
	return err
}

type customerRecord struct {
	ID        string `gorm:"column:id;primaryKey;type:text"`
	Document  string `gorm:"column:document;not null;uniqueIndex:ux_customers_document"`
	Name      string `gorm:"column:name;not null"`
	Email     string `gorm:"column:email;not null"`
	Phone     string `gorm:"column:phone;not null"`
	Status    string `gorm:"column:status;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (customerRecord) TableName() string { return "customers" }
