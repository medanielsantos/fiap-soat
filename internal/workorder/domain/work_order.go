package domain

import (
	"errors"
	"math"
	"strings"
	"time"
)

type Status string

const (
	Received         Status = "received"
	Diagnosis        Status = "diagnosis"
	AwaitingApproval Status = "awaiting_approval"
	InProgress       Status = "in_progress"
	Completed        Status = "completed"
	Delivered        Status = "delivered"
)

var (
	ErrInvalidOrder            = errors.New("work order ID, customer ID and vehicle ID are required")
	ErrInvalidTransition       = errors.New("work order transition is not allowed")
	ErrInvalidItem             = errors.New("work order item is invalid")
	ErrInvalidDiagnosis        = errors.New("diagnosis is required")
	ErrQuoteEmpty              = errors.New("cannot submit an empty quote")
	ErrQuoteUnavailable        = errors.New("there is no pending quote")
	ErrInvalidQuantity         = errors.New("quantity must be positive")
	ErrInsufficientConsumption = errors.New("consumption exceeds the remaining reserved quantity")
	ErrItemsNotConsumed        = errors.New("all reserved parts must be consumed before completion")
	ErrAmountOverflow          = errors.New("quote amount exceeds supported range")
)

type ItemType string

const (
	ServiceItem ItemType = "service"
	PartItem    ItemType = "part"
)

type Item struct {
	ID                       string   `json:"id"`
	Type                     ItemType `json:"type"`
	CatalogID                string   `json:"catalog_id"`
	Code                     string   `json:"code"`
	Description              string   `json:"description"`
	Unit                     string   `json:"unit"`
	Quantity                 int64    `json:"quantity"`
	UnitPriceInCents         int64    `json:"unit_price_in_cents"`
	EstimatedDurationMinutes int      `json:"estimated_duration_minutes"`
	ConsumedQuantity         int64    `json:"consumed_quantity"`
}

func NewItem(id string, itemType ItemType, catalogID, code, description, unit string, quantity, price int64, duration int) (Item, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(catalogID) == "" || (itemType != ServiceItem && itemType != PartItem) || strings.TrimSpace(code) == "" || strings.TrimSpace(description) == "" || quantity <= 0 || price <= 0 || duration < 0 {
		return Item{}, ErrInvalidItem
	}
	if itemType == PartItem && strings.TrimSpace(unit) == "" {
		return Item{}, ErrInvalidItem
	}
	if itemType == ServiceItem {
		unit = "service"
	}
	return Item{ID: id, Type: itemType, CatalogID: catalogID, Code: code, Description: description, Unit: unit, Quantity: quantity, UnitPriceInCents: price, EstimatedDurationMinutes: duration}, nil
}
func (i Item) SubtotalInCents() (int64, error) {
	if i.Quantity > math.MaxInt64/i.UnitPriceInCents {
		return 0, ErrAmountOverflow
	}
	return i.Quantity * i.UnitPriceInCents, nil
}

type QuoteItem struct {
	Item            Item  `json:"item"`
	SubtotalInCents int64 `json:"subtotal_in_cents"`
}
type Quote struct {
	ID           string      `json:"id"`
	Version      int         `json:"version"`
	TotalInCents int64       `json:"total_in_cents"`
	Decision     string      `json:"decision"`
	CreatedBy    string      `json:"created_by,omitempty"`
	DecidedBy    string      `json:"decided_by,omitempty"`
	DecidedAt    *time.Time  `json:"decided_at,omitempty"`
	Items        []QuoteItem `json:"items"`
}
type HistoryEntry struct {
	ID        string    `json:"id"`
	From      *Status   `json:"from_status,omitempty"`
	To        Status    `json:"to_status"`
	ActorID   string    `json:"actor_id,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkOrder struct {
	ID           string         `json:"id"`
	CustomerID   string         `json:"customer_id"`
	VehicleID    string         `json:"vehicle_id"`
	Status       Status         `json:"status"`
	Diagnosis    string         `json:"diagnosis"`
	QuoteVersion int            `json:"quote_version"`
	Items        []Item         `json:"items"`
	Quotes       []Quote        `json:"quotes,omitempty"`
	History      []HistoryEntry `json:"history,omitempty"`
	CreatedBy    string         `json:"created_by,omitempty"`
	UpdatedBy    string         `json:"updated_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func NewWorkOrder(id, customerID, vehicleID, actor string, now time.Time) (WorkOrder, HistoryEntry, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(customerID) == "" || strings.TrimSpace(vehicleID) == "" {
		return WorkOrder{}, HistoryEntry{}, ErrInvalidOrder
	}
	o := WorkOrder{ID: id, CustomerID: customerID, VehicleID: vehicleID, Status: Received, Items: []Item{}, Quotes: []Quote{}, CreatedBy: actor, UpdatedBy: actor, CreatedAt: now, UpdatedAt: now}
	return o, o.event(nil, Received, actor, "work order created", now), nil
}
func (o *WorkOrder) SetDiagnosis(text, actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != Received && o.Status != Diagnosis {
		return HistoryEntry{}, ErrInvalidTransition
	}
	if strings.TrimSpace(text) == "" {
		return HistoryEntry{}, ErrInvalidDiagnosis
	}
	from := o.Status
	o.Diagnosis = strings.TrimSpace(text)
	o.Status = Diagnosis
	return o.changed(from, Diagnosis, actor, "diagnosis recorded", now), nil
}
func (o *WorkOrder) AddItem(item Item, actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != Diagnosis {
		return HistoryEntry{}, ErrInvalidTransition
	}
	for _, existing := range o.Items {
		if existing.ID == item.ID {
			return HistoryEntry{}, ErrInvalidItem
		}
	}
	o.Items = append(o.Items, item)
	o.UpdatedBy = actor
	o.UpdatedAt = now
	return o.event(nil, o.Status, actor, "item added", now), nil
}
func (o *WorkOrder) RemoveItem(id, actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != Diagnosis {
		return HistoryEntry{}, ErrInvalidTransition
	}
	for i, item := range o.Items {
		if item.ID == id {
			o.Items = append(o.Items[:i], o.Items[i+1:]...)
			o.UpdatedBy = actor
			o.UpdatedAt = now
			return o.event(nil, o.Status, actor, "item removed", now), nil
		}
	}
	return HistoryEntry{}, ErrInvalidItem
}
func (o *WorkOrder) SubmitQuote(id, actor string, now time.Time) (Quote, HistoryEntry, error) {
	if o.Status != Diagnosis || len(o.Items) == 0 {
		return Quote{}, HistoryEntry{}, ErrQuoteEmpty
	}
	q := Quote{ID: id, Version: o.QuoteVersion + 1, Decision: "pending", CreatedBy: actor, Items: make([]QuoteItem, 0, len(o.Items))}
	for _, item := range o.Items {
		subtotal, e := item.SubtotalInCents()
		if e != nil {
			return Quote{}, HistoryEntry{}, e
		}
		if q.TotalInCents > math.MaxInt64-subtotal {
			return Quote{}, HistoryEntry{}, ErrAmountOverflow
		}
		q.TotalInCents += subtotal
		q.Items = append(q.Items, QuoteItem{Item: item, SubtotalInCents: subtotal})
	}
	o.QuoteVersion = q.Version
	o.Quotes = append(o.Quotes, q)
	from := o.Status
	o.Status = AwaitingApproval
	entry := o.changed(from, AwaitingApproval, actor, "quote submitted", now)
	return q, entry, nil
}
func (o *WorkOrder) DecideQuote(approve bool, actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != AwaitingApproval || len(o.Quotes) == 0 {
		return HistoryEntry{}, ErrQuoteUnavailable
	}
	q := &o.Quotes[len(o.Quotes)-1]
	if q.Decision != "pending" || q.Version != o.QuoteVersion {
		return HistoryEntry{}, ErrQuoteUnavailable
	}
	q.DecidedAt = &now
	q.DecidedBy = actor
	from := o.Status
	if approve {
		q.Decision = "approved"
		o.Status = InProgress
	} else {
		q.Decision = "rejected"
		o.Status = Diagnosis
	}
	note := "quote rejected"
	if approve {
		note = "quote approved"
	}
	return o.changed(from, o.Status, actor, note, now), nil
}
func (o *WorkOrder) Consume(partID string, quantity int64, actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != InProgress {
		return HistoryEntry{}, ErrInvalidTransition
	}
	if quantity <= 0 {
		return HistoryEntry{}, ErrInvalidQuantity
	}
	remaining := int64(0)
	for _, item := range o.Items {
		if item.Type == PartItem && item.CatalogID == partID {
			remaining += item.Quantity - item.ConsumedQuantity
		}
	}
	if quantity > remaining {
		return HistoryEntry{}, ErrInsufficientConsumption
	}
	toConsume := quantity
	for i := range o.Items {
		item := &o.Items[i]
		if item.Type != PartItem || item.CatalogID != partID {
			continue
		}
		available := item.Quantity - item.ConsumedQuantity
		take := available
		if take > toConsume {
			take = toConsume
		}
		item.ConsumedQuantity += take
		toConsume -= take
		if toConsume == 0 {
			break
		}
	}
	o.UpdatedBy = actor
	o.UpdatedAt = now
	return o.event(nil, o.Status, actor, "reserved parts consumed", now), nil
}
func (o *WorkOrder) Complete(actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != InProgress {
		return HistoryEntry{}, ErrInvalidTransition
	}
	for _, item := range o.Items {
		if item.Type == PartItem && item.ConsumedQuantity != item.Quantity {
			return HistoryEntry{}, ErrItemsNotConsumed
		}
	}
	return o.changed(o.Status, Completed, actor, "work completed", now), nil
}
func (o *WorkOrder) Deliver(actor string, now time.Time) (HistoryEntry, error) {
	if o.Status != Completed {
		return HistoryEntry{}, ErrInvalidTransition
	}
	return o.changed(o.Status, Delivered, actor, "vehicle delivered", now), nil
}
func (o *WorkOrder) RestoreTransition(status Status, actor, note string, now time.Time) {
	o.Status = status
	o.UpdatedBy = actor
	o.UpdatedAt = now
}
func (o *WorkOrder) event(from *Status, to Status, actor, note string, now time.Time) HistoryEntry {
	return HistoryEntry{To: to, ActorID: actor, Note: note, CreatedAt: now}
}
func (o *WorkOrder) changed(from, to Status, actor, note string, now time.Time) HistoryEntry {
	o.Status = to
	o.UpdatedBy = actor
	o.UpdatedAt = now
	f := from
	return o.event(&f, to, actor, note, now)
}
