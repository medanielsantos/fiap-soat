package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
	"gorm.io/gorm"
)

type VehicleRepository struct {
	db *gorm.DB
}

func NewVehicleRepository(db *gorm.DB) *VehicleRepository {
	return &VehicleRepository{db: db}
}

func (repository *VehicleRepository) Save(ctx context.Context, vehicle domain.Vehicle) error {
	return translateError(repository.db.WithContext(ctx).Create(recordFromDomain(vehicle)).Error)
}

func (repository *VehicleRepository) FindByID(ctx context.Context, id string) (domain.Vehicle, error) {
	var record vehicleRecord
	err := repository.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if err != nil {
		return domain.Vehicle{}, translateError(err)
	}
	return record.toDomain()
}

func (repository *VehicleRepository) List(ctx context.Context, filter application.ListFilter, params pagination.Params) (pagination.Page[domain.Vehicle], error) {
	db := repository.db.WithContext(ctx).Model(&vehicleRecord{})
	if filter.CustomerID != "" {
		db = db.Where("customer_id = ?", filter.CustomerID)
	}
	var totalItems int64
	if err := db.Count(&totalItems).Error; err != nil {
		return pagination.Page[domain.Vehicle]{}, translateError(err)
	}

	var records []vehicleRecord
	if err := db.Order("created_at DESC").Limit(params.Limit()).Offset(params.Offset()).Find(&records).Error; err != nil {
		return pagination.Page[domain.Vehicle]{}, translateError(err)
	}
	vehicles := make([]domain.Vehicle, 0, len(records))
	for _, record := range records {
		vehicle, err := record.toDomain()
		if err != nil {
			return pagination.Page[domain.Vehicle]{}, err
		}
		vehicles = append(vehicles, vehicle)
	}
	return pagination.NewPage(vehicles, params, totalItems), nil
}

func (repository *VehicleRepository) Update(ctx context.Context, vehicle domain.Vehicle) error {
	result := repository.db.WithContext(ctx).Model(&vehicleRecord{}).Where("id = ?", vehicle.ID()).Updates(map[string]any{
		"customer_id": vehicle.CustomerID(),
		"plate":       vehicle.Plate().String(),
		"brand":       vehicle.Brand(),
		"model":       vehicle.Model(),
		"year":        vehicle.Year(),
		"updated_at":  time.Now(),
	})
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return application.ErrVehicleNotFound
	}
	return nil
}

func (repository *VehicleRepository) Delete(ctx context.Context, id string) error {
	result := repository.db.WithContext(ctx).Delete(&vehicleRecord{}, "id = ?", id)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return application.ErrVehicleNotFound
	}
	return nil
}

func recordFromDomain(vehicle domain.Vehicle) *vehicleRecord {
	return &vehicleRecord{
		ID: vehicle.ID(), CustomerID: vehicle.CustomerID(), Plate: vehicle.Plate().String(),
		Brand: vehicle.Brand(), Model: vehicle.Model(), Year: vehicle.Year(),
	}
}

func (record vehicleRecord) toDomain() (domain.Vehicle, error) {
	plate, err := domain.NewPlate(record.Plate)
	if err != nil {
		return domain.Vehicle{}, err
	}
	return domain.NewVehicle(record.ID, record.CustomerID, plate, record.Brand, record.Model, record.Year)
}

func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return application.ErrVehicleNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "23505":
			return application.ErrDuplicatePlate
		case "23503":
			return application.ErrCustomerUnavailable
		}
	}
	return err
}

type vehicleRecord struct {
	ID         string `gorm:"column:id;primaryKey;type:text"`
	CustomerID string `gorm:"column:customer_id;not null;index"`
	Plate      string `gorm:"column:plate;not null;uniqueIndex:ux_vehicles_plate"`
	Brand      string `gorm:"column:brand;not null"`
	Model      string `gorm:"column:model;not null"`
	Year       int    `gorm:"column:year;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (vehicleRecord) TableName() string { return "vehicles" }
