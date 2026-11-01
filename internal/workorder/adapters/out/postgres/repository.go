package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type orderRecord struct {
	ID                   string `gorm:"primaryKey;type:text"`
	CustomerID           string `gorm:"column:customer_id"`
	VehicleID            string `gorm:"column:vehicle_id"`
	Status               string
	Diagnosis            string
	QuoteVersion         int    `gorm:"column:quote_version"`
	CreatedBy            string `gorm:"column:created_by"`
	UpdatedBy            string `gorm:"column:updated_by"`
	CreatedAt, UpdatedAt time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (orderRecord) TableName() string { return "work_orders" }

type itemRecord struct {
	ID                       string `gorm:"primaryKey;type:text"`
	WorkOrderID              string `gorm:"column:work_order_id"`
	Type                     string `gorm:"column:item_type"`
	CatalogID                string `gorm:"column:catalog_id"`
	Code, Description, Unit  string
	Quantity                 int64
	UnitPriceInCents         int64 `gorm:"column:unit_price_in_cents"`
	EstimatedDurationMinutes int   `gorm:"column:estimated_duration_minutes"`
	ConsumedQuantity         int64 `gorm:"column:consumed_quantity"`
	CreatedAt                time.Time
}

func (itemRecord) TableName() string { return "work_order_items" }

type quoteRecord struct {
	ID                string `gorm:"primaryKey;type:text"`
	WorkOrderID       string `gorm:"column:work_order_id"`
	Version           int
	TotalInCents      int64 `gorm:"column:total_in_cents"`
	Decision          string
	CreatedBy         string     `gorm:"column:created_by"`
	DecidedByClientID *string    `gorm:"column:decided_by_client_id"`
	DecidedAt         *time.Time `gorm:"column:decided_at"`
	CreatedAt         time.Time
}

func (quoteRecord) TableName() string { return "work_order_quotes" }

type quoteItemRecord struct {
	ID                       string `gorm:"primaryKey;type:text"`
	QuoteID                  string `gorm:"column:quote_id"`
	Type                     string `gorm:"column:item_type"`
	CatalogID                string `gorm:"column:catalog_id"`
	Code, Description, Unit  string
	Quantity                 int64
	UnitPriceInCents         int64 `gorm:"column:unit_price_in_cents"`
	SubtotalInCents          int64 `gorm:"column:subtotal_in_cents"`
	EstimatedDurationMinutes int   `gorm:"column:estimated_duration_minutes"`
}

func (quoteItemRecord) TableName() string { return "work_order_quote_items" }

type historyRecord struct {
	ID          string  `gorm:"primaryKey;type:text"`
	WorkOrderID string  `gorm:"column:work_order_id"`
	FromStatus  *string `gorm:"column:from_status"`
	ToStatus    string  `gorm:"column:to_status"`
	UserID      *string `gorm:"column:user_id"`
	ClientID    *string `gorm:"column:client_id"`
	Note        string
	CreatedAt   time.Time
}

func (historyRecord) TableName() string { return "work_order_status_history" }

func (r *Repository) Create(ctx context.Context, o domain.WorkOrder, event domain.HistoryEntry) error {
	db := database.Current(ctx, r.db)
	if err := db.Create(orderFromDomain(o)).Error; err != nil {
		return err
	}
	if err := r.saveItems(db, o); err != nil {
		return err
	}
	return db.Create(historyFromDomain(o.ID, event)).Error
}
func (r *Repository) Find(ctx context.Context, id string) (domain.WorkOrder, error) {
	db := database.Current(ctx, r.db)
	var row orderRecord
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.WorkOrder{}, application.ErrNotFound
		}
		return domain.WorkOrder{}, err
	}
	return r.load(db, row)
}
func (r *Repository) List(ctx context.Context, status string, p pagination.Params) (pagination.Page[domain.WorkOrder], error) {
	db := database.Current(ctx, r.db).Model(&orderRecord{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return pagination.Page[domain.WorkOrder]{}, err
	}
	var rows []orderRecord
	if err := db.Order("created_at DESC").Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return pagination.Page[domain.WorkOrder]{}, err
	}
	items := make([]domain.WorkOrder, 0, len(rows))
	for _, row := range rows {
		o, e := r.load(database.Current(ctx, r.db), row)
		if e != nil {
			return pagination.Page[domain.WorkOrder]{}, e
		}
		items = append(items, o)
	}
	return pagination.NewPage(items, p, total), nil
}
func (r *Repository) Save(ctx context.Context, o domain.WorkOrder, event domain.HistoryEntry) error {
	db := database.Current(ctx, r.db)
	res := db.Model(&orderRecord{}).Where("id = ?", o.ID).Updates(map[string]any{"status": string(o.Status), "diagnosis": o.Diagnosis, "quote_version": o.QuoteVersion, "updated_by": o.UpdatedBy, "updated_at": o.UpdatedAt})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return application.ErrNotFound
	}
	if err := db.Where("work_order_id = ?", o.ID).Delete(&itemRecord{}).Error; err != nil {
		return err
	}
	if err := r.saveItems(db, o); err != nil {
		return err
	}
	if err := r.saveQuotes(db, o); err != nil {
		return err
	}
	return db.Create(historyFromDomain(o.ID, event)).Error
}
func (r *Repository) load(db *gorm.DB, row orderRecord) (domain.WorkOrder, error) {
	o := domain.WorkOrder{ID: row.ID, CustomerID: row.CustomerID, VehicleID: row.VehicleID, Status: domain.Status(row.Status), Diagnosis: row.Diagnosis, QuoteVersion: row.QuoteVersion, CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Items: []domain.Item{}, Quotes: []domain.Quote{}, History: []domain.HistoryEntry{}}
	var ir []itemRecord
	if e := db.Where("work_order_id = ?", row.ID).Order("created_at, id").Find(&ir).Error; e != nil {
		return o, e
	}
	for _, x := range ir {
		o.Items = append(o.Items, domain.Item{ID: x.ID, Type: domain.ItemType(x.Type), CatalogID: x.CatalogID, Code: x.Code, Description: x.Description, Unit: x.Unit, Quantity: x.Quantity, UnitPriceInCents: x.UnitPriceInCents, EstimatedDurationMinutes: x.EstimatedDurationMinutes, ConsumedQuantity: x.ConsumedQuantity})
	}
	var qr []quoteRecord
	if e := db.Where("work_order_id = ?", row.ID).Order("version").Find(&qr).Error; e != nil {
		return o, e
	}
	for _, x := range qr {
		decidedByClientID := ""
		if x.DecidedByClientID != nil {
			decidedByClientID = *x.DecidedByClientID
		}
		q := domain.Quote{ID: x.ID, Version: x.Version, TotalInCents: x.TotalInCents, Decision: x.Decision, CreatedBy: x.CreatedBy, DecidedByClientID: decidedByClientID, DecidedAt: x.DecidedAt, Items: []domain.QuoteItem{}}
		var qis []quoteItemRecord
		if e := db.Where("quote_id = ?", x.ID).Find(&qis).Error; e != nil {
			return o, e
		}
		for _, qi := range qis {
			it := domain.Item{Type: domain.ItemType(qi.Type), CatalogID: qi.CatalogID, Code: qi.Code, Description: qi.Description, Unit: qi.Unit, Quantity: qi.Quantity, UnitPriceInCents: qi.UnitPriceInCents, EstimatedDurationMinutes: qi.EstimatedDurationMinutes}
			q.Items = append(q.Items, domain.QuoteItem{Item: it, SubtotalInCents: qi.SubtotalInCents})
		}
		o.Quotes = append(o.Quotes, q)
	}
	var hr []historyRecord
	if e := db.Where("work_order_id = ?", row.ID).Order("created_at,id").Find(&hr).Error; e != nil {
		return o, e
	}
	for _, x := range hr {
		var from *domain.Status
		if x.FromStatus != nil {
			v := domain.Status(*x.FromStatus)
			from = &v
		}
		userID, clientID := "", ""
		if x.UserID != nil {
			userID = *x.UserID
		}
		if x.ClientID != nil {
			clientID = *x.ClientID
		}
		o.History = append(o.History, domain.HistoryEntry{ID: x.ID, From: from, To: domain.Status(x.ToStatus), UserID: userID, ClientID: clientID, Note: x.Note, CreatedAt: x.CreatedAt})
	}
	return o, nil
}
func (r *Repository) saveItems(db *gorm.DB, o domain.WorkOrder) error {
	for _, x := range o.Items {
		row := itemRecord{ID: x.ID, WorkOrderID: o.ID, Type: string(x.Type), CatalogID: x.CatalogID, Code: x.Code, Description: x.Description, Unit: x.Unit, Quantity: x.Quantity, UnitPriceInCents: x.UnitPriceInCents, EstimatedDurationMinutes: x.EstimatedDurationMinutes, ConsumedQuantity: x.ConsumedQuantity}
		if e := db.Create(&row).Error; e != nil {
			return e
		}
	}
	return nil
}
func (r *Repository) saveQuotes(db *gorm.DB, o domain.WorkOrder) error {
	for _, q := range o.Quotes {
		var decidedByClientID *string
		if q.DecidedByClientID != "" {
			value := q.DecidedByClientID
			decidedByClientID = &value
		}
		var existing quoteRecord
		err := db.First(&existing, "id = ?", q.ID).Error
		if err == nil {
			if err := db.Model(&quoteRecord{}).Where("id = ?", q.ID).Updates(map[string]any{"decision": q.Decision, "decided_by_client_id": decidedByClientID, "decided_at": q.DecidedAt}).Error; err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := quoteRecord{ID: q.ID, WorkOrderID: o.ID, Version: q.Version, TotalInCents: q.TotalInCents, Decision: q.Decision, CreatedBy: q.CreatedBy, DecidedByClientID: decidedByClientID, DecidedAt: q.DecidedAt}
		if e := db.Create(&row).Error; e != nil {
			return e
		}
		for i, qi := range q.Items {
			id := fmt.Sprintf("%s-%d", q.ID, i+1)
			x := quoteItemRecord{ID: id, QuoteID: q.ID, Type: string(qi.Item.Type), CatalogID: qi.Item.CatalogID, Code: qi.Item.Code, Description: qi.Item.Description, Unit: qi.Item.Unit, Quantity: qi.Item.Quantity, UnitPriceInCents: qi.Item.UnitPriceInCents, SubtotalInCents: qi.SubtotalInCents, EstimatedDurationMinutes: qi.Item.EstimatedDurationMinutes}
			if e := db.Create(&x).Error; e != nil {
				return e
			}
		}
	}
	return nil
}
func orderFromDomain(o domain.WorkOrder) *orderRecord {
	return &orderRecord{ID: o.ID, CustomerID: o.CustomerID, VehicleID: o.VehicleID, Status: string(o.Status), Diagnosis: o.Diagnosis, QuoteVersion: o.QuoteVersion, CreatedBy: o.CreatedBy, UpdatedBy: o.UpdatedBy, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
}
func historyFromDomain(id string, e domain.HistoryEntry) *historyRecord {
	var from *string
	if e.From != nil {
		x := string(*e.From)
		from = &x
	}
	var userID, clientID *string
	if e.UserID != "" {
		value := e.UserID
		userID = &value
	}
	if e.ClientID != "" {
		value := e.ClientID
		clientID = &value
	}
	return &historyRecord{ID: e.ID, WorkOrderID: id, FromStatus: from, ToStatus: string(e.To), UserID: userID, ClientID: clientID, Note: e.Note, CreatedAt: e.CreatedAt}
}
