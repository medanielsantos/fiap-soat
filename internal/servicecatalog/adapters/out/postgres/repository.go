package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	platformdatabase "github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type serviceRecord struct {
	ID                       string `gorm:"primaryKey;type:text"`
	InternalCode             string `gorm:"column:internal_code"`
	Name                     string
	Description              string
	SalePriceInCents         int64  `gorm:"column:sale_price_in_cents"`
	EstimatedDurationMinutes int    `gorm:"column:estimated_duration_minutes"`
	PerformedBy              string `gorm:"column:created_by"`
	Active                   bool
	CreatedAt, UpdatedAt     time.Time
	DeletedAt                gorm.DeletedAt `gorm:"index"`
}

func (serviceRecord) TableName() string { return "catalog_services" }

type partRecord struct {
	ID                      string `gorm:"primaryKey;type:text"`
	SKU                     string
	InternalCode            string `gorm:"column:internal_code"`
	Name, Description, Unit string
	PurchasePriceInCents    int64 `gorm:"column:purchase_price_in_cents"`
	SalePriceInCents        int64 `gorm:"column:sale_price_in_cents"`
	Stock                   int64
	ReservedStock           int64  `gorm:"column:reserved_stock"`
	PerformedBy             string `gorm:"column:created_by"`
	Active                  bool
	CreatedAt, UpdatedAt    time.Time
	DeletedAt               gorm.DeletedAt `gorm:"index"`
}

func (partRecord) TableName() string { return "parts" }

type movementRecord struct {
	ID          string `gorm:"primaryKey;type:text"`
	PartID      string `gorm:"column:part_id"`
	Type        string `gorm:"column:movement_type"`
	Quantity    int64
	Description string
	PerformedBy string  `gorm:"column:created_by"`
	WorkOrderID *string `gorm:"column:work_order_id"`
	CreatedAt   time.Time
}

func (movementRecord) TableName() string { return "stock_movements" }

func (r *Repository) SaveService(ctx context.Context, v domain.Service) error {
	return translate(platformdatabase.Current(ctx, r.db).Create(&serviceRecord{ID: v.ID(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), SalePriceInCents: v.SalePriceInCents(), EstimatedDurationMinutes: v.EstimatedDurationMinutes(), PerformedBy: v.PerformedBy(), Active: v.Active()}).Error)
}
func (r *Repository) FindService(ctx context.Context, id string) (domain.Service, error) {
	var x serviceRecord
	if e := platformdatabase.Current(ctx, r.db).First(&x, "id = ?", id).Error; e != nil {
		return domain.Service{}, translate(e)
	}
	return domain.NewService(x.ID, x.InternalCode, x.Name, x.Description, x.SalePriceInCents, x.EstimatedDurationMinutes, x.Active, x.PerformedBy)
}
func (r *Repository) ListServices(ctx context.Context, p pagination.Params) (pagination.Page[domain.Service], error) {
	var xs []serviceRecord
	var n int64
	db := platformdatabase.Current(ctx, r.db).Model(&serviceRecord{})
	if e := db.Count(&n).Error; e != nil {
		return pagination.Page[domain.Service]{}, e
	}
	if e := db.Order("name ASC").Limit(p.Limit()).Offset(p.Offset()).Find(&xs).Error; e != nil {
		return pagination.Page[domain.Service]{}, e
	}
	out := make([]domain.Service, 0, len(xs))
	for _, x := range xs {
		v, e := domain.NewService(x.ID, x.InternalCode, x.Name, x.Description, x.SalePriceInCents, x.EstimatedDurationMinutes, x.Active, x.PerformedBy)
		if e != nil {
			return pagination.Page[domain.Service]{}, e
		}
		out = append(out, v)
	}
	return pagination.NewPage(out, p, n), nil
}
func (r *Repository) UpdateService(ctx context.Context, v domain.Service) error {
	return update(platformdatabase.Current(ctx, r.db).Model(&serviceRecord{}).Where("id = ?", v.ID()).Updates(map[string]any{"internal_code": v.InternalCode(), "name": v.Name(), "description": v.Description(), "sale_price_in_cents": v.SalePriceInCents(), "estimated_duration_minutes": v.EstimatedDurationMinutes(), "updated_by": v.PerformedBy(), "active": v.Active(), "updated_at": time.Now()}))
}
func (r *Repository) DeleteService(ctx context.Context, id string) error {
	return deleteRecord(platformdatabase.Current(ctx, r.db).Delete(&serviceRecord{}, "id = ?", id))
}
func (r *Repository) SavePart(ctx context.Context, v domain.Part, openingMovement *domain.StockMovement) error {
	db := platformdatabase.Current(ctx, r.db)
	return translate(db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&partRecord{ID: v.ID(), SKU: v.SKU(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), PurchasePriceInCents: v.PurchasePriceInCents(), SalePriceInCents: v.SalePriceInCents(), Stock: v.Stock(), PerformedBy: v.PerformedBy(), Active: v.Active()}).Error; err != nil {
			return err
		}
		if openingMovement != nil {
			return tx.Create(&movementRecord{ID: openingMovement.ID(), PartID: openingMovement.PartID(), Type: string(openingMovement.Type()), Quantity: openingMovement.Quantity(), Description: openingMovement.Description(), PerformedBy: openingMovement.PerformedBy()}).Error
		}
		return nil
	}))
}
func (r *Repository) FindPart(ctx context.Context, id string) (domain.Part, error) {
	var x partRecord
	if e := platformdatabase.Current(ctx, r.db).First(&x, "id = ?", id).Error; e != nil {
		return domain.Part{}, translate(e)
	}
	return domain.NewPart(x.ID, x.SKU, x.InternalCode, x.Name, x.Description, x.Unit, x.PurchasePriceInCents, x.SalePriceInCents, x.Stock, x.ReservedStock, x.Active, x.PerformedBy)
}
func (r *Repository) ListParts(ctx context.Context, p pagination.Params) (pagination.Page[domain.Part], error) {
	var xs []partRecord
	var n int64
	db := platformdatabase.Current(ctx, r.db).Model(&partRecord{})
	if e := db.Count(&n).Error; e != nil {
		return pagination.Page[domain.Part]{}, e
	}
	if e := db.Order("name ASC").Limit(p.Limit()).Offset(p.Offset()).Find(&xs).Error; e != nil {
		return pagination.Page[domain.Part]{}, e
	}
	out := make([]domain.Part, 0, len(xs))
	for _, x := range xs {
		v, e := domain.NewPart(x.ID, x.SKU, x.InternalCode, x.Name, x.Description, x.Unit, x.PurchasePriceInCents, x.SalePriceInCents, x.Stock, x.ReservedStock, x.Active, x.PerformedBy)
		if e != nil {
			return pagination.Page[domain.Part]{}, e
		}
		out = append(out, v)
	}
	return pagination.NewPage(out, p, n), nil
}
func (r *Repository) UpdatePart(ctx context.Context, v domain.Part) error {
	return update(platformdatabase.Current(ctx, r.db).Model(&partRecord{}).Where("id = ?", v.ID()).Updates(map[string]any{"sku": v.SKU(), "internal_code": v.InternalCode(), "name": v.Name(), "description": v.Description(), "unit": v.Unit(), "purchase_price_in_cents": v.PurchasePriceInCents(), "sale_price_in_cents": v.SalePriceInCents(), "updated_by": v.PerformedBy(), "active": v.Active(), "updated_at": time.Now()}))
}
func (r *Repository) DeletePart(ctx context.Context, id string) error {
	db := platformdatabase.Current(ctx, r.db)
	return translate(db.Transaction(func(tx *gorm.DB) error {
		var part partRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&part, "id = ?", id).Error; err != nil {
			return translate(err)
		}
		if part.Stock > 0 || part.ReservedStock > 0 {
			return domain.ErrPartHasStock
		}
		return deleteRecord(tx.Delete(&partRecord{}, "id = ?", id))
	}))
}
func (r *Repository) MoveStock(ctx context.Context, m domain.StockMovement) (domain.Part, error) {
	e := platformdatabase.Current(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		delta := m.Quantity()
		if m.Type() == domain.MovementExit {
			delta = -delta
		}
		q := tx.Model(&partRecord{}).Where("id = ?", m.PartID())
		if delta < 0 {
			q = q.Where("stock - reserved_stock >= ?", m.Quantity())
		}
		res := q.Updates(map[string]any{"stock": gorm.Expr("stock + ?", delta), "updated_at": time.Now(), "updated_by": m.PerformedBy()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			var exists int64
			if er := tx.Model(&partRecord{}).Where("id = ?", m.PartID()).Count(&exists).Error; er != nil {
				return er
			}
			if exists == 0 {
				return application.ErrNotFound
			}
			return domain.ErrInsufficientStock
		}
		return tx.Create(&movementRecord{ID: m.ID(), PartID: m.PartID(), Type: string(m.Type()), Quantity: m.Quantity(), Description: m.Description(), PerformedBy: m.PerformedBy()}).Error
	})
	if e != nil {
		return domain.Part{}, translate(e)
	}
	return r.FindPart(ctx, m.PartID())
}

func (r *Repository) ListStockMovements(ctx context.Context, partID string, p pagination.Params) (pagination.Page[application.StockMovement], error) {
	db := platformdatabase.Current(ctx, r.db).Model(&movementRecord{}).Where("part_id = ?", partID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return pagination.Page[application.StockMovement]{}, err
	}
	var rows []movementRecord
	if err := db.Order("created_at DESC, id DESC").Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return pagination.Page[application.StockMovement]{}, err
	}
	items := make([]application.StockMovement, 0, len(rows))
	for _, row := range rows {
		items = append(items, application.StockMovement{ID: row.ID, PartID: row.PartID, Type: row.Type, Quantity: row.Quantity, Description: row.Description, PerformedBy: row.PerformedBy, WorkOrderID: row.WorkOrderID, CreatedAt: row.CreatedAt})
	}
	return pagination.NewPage(items, p, total), nil
}

func (r *Repository) ReserveStock(ctx context.Context, orderID, clientID string, requirements []application.StockRequirement) error {
	db := platformdatabase.Current(ctx, r.db)
	for _, requirement := range requirements {
		result := db.Model(&partRecord{}).Where("id = ? AND stock - reserved_stock >= ?", requirement.PartID, requirement.Quantity).
			Updates(map[string]any{"reserved_stock": gorm.Expr("reserved_stock + ?", requirement.Quantity), "updated_at": time.Now(), "updated_by": clientID})
		if result.Error != nil {
			return translate(result.Error)
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := db.Model(&partRecord{}).Where("id = ?", requirement.PartID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return application.ErrNotFound
			}
			return domain.ErrInsufficientStock
		}
		orderRef := orderID
		if err := db.Create(&movementRecord{ID: requirement.MovementID, PartID: requirement.PartID, Type: "reservation", Quantity: requirement.Quantity, Description: "reserved for work order", PerformedBy: clientID, WorkOrderID: &orderRef}).Error; err != nil {
			return translate(err)
		}
	}
	return nil
}

func (r *Repository) ConsumeReservedStock(ctx context.Context, orderID, userID string, requirements []application.StockRequirement) error {
	db := platformdatabase.Current(ctx, r.db)
	for _, requirement := range requirements {
		result := db.Model(&partRecord{}).Where("id = ? AND reserved_stock >= ? AND stock >= ?", requirement.PartID, requirement.Quantity, requirement.Quantity).
			Updates(map[string]any{"reserved_stock": gorm.Expr("reserved_stock - ?", requirement.Quantity), "stock": gorm.Expr("stock - ?", requirement.Quantity), "updated_at": time.Now(), "updated_by": userID})
		if result.Error != nil {
			return translate(result.Error)
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := db.Model(&partRecord{}).Where("id = ?", requirement.PartID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return application.ErrNotFound
			}
			return domain.ErrInsufficientStock
		}
		orderRef := orderID
		if err := db.Create(&movementRecord{ID: requirement.MovementID, PartID: requirement.PartID, Type: "consumption", Quantity: requirement.Quantity, Description: "consumed for work order", PerformedBy: userID, WorkOrderID: &orderRef}).Error; err != nil {
			return translate(err)
		}
	}
	return nil
}
func update(res *gorm.DB) error {
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return application.ErrNotFound
	}
	return nil
}
func deleteRecord(res *gorm.DB) error {
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return application.ErrNotFound
	}
	return nil
}
func translate(e error) error {
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return application.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && pg.Code == "23505" {
		return application.ErrDuplicate
	}
	return e
}
