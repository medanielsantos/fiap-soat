package postgres

import (
	"context"
	"time"

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
	return repository.db.WithContext(ctx).Create(&customerRecord{
		ID:       customer.ID(),
		Document: customer.Document().String(),
		Name:     customer.Name(),
		Email:    customer.Email(),
		Phone:    customer.Phone(),
		Status:   string(customer.Status()),
	}).Error
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
}

func (customerRecord) TableName() string { return "customers" }
