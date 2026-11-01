package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (repository *CustomerRepository) FindByEmail(ctx context.Context, email string) (domain.Customer, error) {
	var record customerRecord
	if err := repository.db.WithContext(ctx).Where("lower(email) = ?", email).First(&record).Error; err != nil {
		return domain.Customer{}, translateError(err)
	}
	return record.toDomain()
}

func (repository *CustomerRepository) FindByPlateAndDocumentPrefix(ctx context.Context, plate, prefix string) (domain.Customer, string, error) {
	var record customerRecord
	var vehicleID string
	var result struct {
		customerRecord
		VehicleID string `gorm:"column:vehicle_id"`
	}
	if err := repository.db.WithContext(ctx).Table("customers AS c").Select("c.*, vehicles.id AS vehicle_id").Joins("JOIN vehicles ON vehicles.customer_id = c.id").Where("vehicles.plate = ? AND vehicles.deleted_at IS NULL AND c.deleted_at IS NULL AND c.document LIKE ? AND c.status = ?", plate, prefix+"%", "active").First(&result).Error; err != nil {
		return domain.Customer{}, "", translateError(err)
	}
	record, vehicleID = result.customerRecord, result.VehicleID
	customer, err := record.toDomain()
	return customer, vehicleID, err
}

func (repository *CustomerRepository) CreateCustomerReset(ctx context.Context, customerID, tokenHash string, expires time.Time) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("customer_id = ? AND used_at IS NULL", customerID).Delete(&customerPasswordResetRecord{}).Error; err != nil {
			return err
		}
		return tx.Create(&customerPasswordResetRecord{TokenHash: tokenHash, CustomerID: customerID, ExpiresAt: expires}).Error
	})
}

func (repository *CustomerRepository) ResetCustomerPassword(ctx context.Context, tokenHash, password string, now time.Time) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reset customerPasswordResetRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&reset, "token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return application.ErrInvalidResetToken
			}
			return err
		}
		updated := tx.Model(&customerRecord{}).Where("id = ? AND status = ? AND deleted_at IS NULL", reset.CustomerID, string(domain.CustomerStatusActive)).Update("password", password)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return application.ErrInvalidResetToken
		}
		return tx.Model(&customerPasswordResetRecord{}).Where("token_hash = ? AND used_at IS NULL", tokenHash).Update("used_at", now).Error
	})
}

func (repository *CustomerRepository) List(ctx context.Context, params pagination.Params) (pagination.Page[domain.Customer], error) {
	db := repository.db.WithContext(ctx)
	var totalItems int64
	if err := db.Model(&customerRecord{}).Count(&totalItems).Error; err != nil {
		return pagination.Page[domain.Customer]{}, translateError(err)
	}

	var records []customerRecord
	if err := db.Order("created_at DESC").Limit(params.Limit()).Offset(params.Offset()).Find(&records).Error; err != nil {
		return pagination.Page[domain.Customer]{}, translateError(err)
	}
	customers := make([]domain.Customer, 0, len(records))
	for _, record := range records {
		customer, err := record.toDomain()
		if err != nil {
			return pagination.Page[domain.Customer]{}, err
		}
		customers = append(customers, customer)
	}
	return pagination.NewPage(customers, params, totalItems), nil
}

func (repository *CustomerRepository) Update(ctx context.Context, customer domain.Customer) error {
	result := repository.db.WithContext(ctx).Model(&customerRecord{}).Where("id = ?", customer.ID()).Updates(map[string]any{
		"document":   customer.Document().String(),
		"name":       customer.Name(),
		"email":      customer.Email(),
		"phone":      customer.Phone(),
		"status":     string(customer.Status()),
		"password":   customer.Password(),
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
		Password: customer.Password(),
	}
}

func (record customerRecord) toDomain() (domain.Customer, error) {
	document, err := domain.NewDocument(record.Document)
	if err != nil {
		return domain.Customer{}, err
	}
	return domain.RestoreCustomerWithPassword(record.ID, document, record.Name, record.Email, record.Phone, domain.CustomerStatus(record.Status), record.Password)
}

func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return application.ErrCustomerNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		if pgError.ConstraintName == "ux_customers_email" {
			return application.ErrDuplicateEmail
		}
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
	Password  string `gorm:"column:password;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type customerPasswordResetRecord struct {
	TokenHash  string     `gorm:"primaryKey;type:char(64)"`
	CustomerID string     `gorm:"column:customer_id;not null"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	UsedAt     *time.Time `gorm:"column:used_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (customerPasswordResetRecord) TableName() string { return "customer_password_reset_tokens" }

func (customerRecord) TableName() string { return "customers" }
