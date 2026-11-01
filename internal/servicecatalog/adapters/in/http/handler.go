package servicecataloghttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

type Handler struct{ service *application.CatalogService }

func NewHandler(s *application.CatalogService) *Handler { return &Handler{service: s} }
func (h *Handler) RegisterRoutes(g *echo.Group, requireProductCreate, requireProductDelete echo.MiddlewareFunc) {
	s := g.Group("/services")
	s.POST("", h.createService, requireProductCreate)
	s.GET("", h.listServices)
	s.GET("/:id", h.getService)
	s.PUT("/:id", h.updateService)
	s.DELETE("/:id", h.deleteService, requireProductDelete)
	p := g.Group("/parts")
	p.POST("", h.createPart, requireProductCreate)
	p.GET("", h.listParts)
	p.GET("/:id", h.getPart)
	p.PUT("/:id", h.updatePart)
	p.DELETE("/:id", h.deletePart, requireProductDelete)
	p.POST("/:id/stock-movements", h.moveStock)
	p.GET("/:id/stock-movements", h.listStockMovements)
}

type serviceDTO struct {
	ID                       string `json:"id"`
	InternalCode             string `json:"internal_code"`
	Name                     string `json:"name"`
	Description              string `json:"description"`
	SalePriceInCents         int64  `json:"sale_price_in_cents"`
	EstimatedDurationMinutes int    `json:"estimated_duration_minutes"`
	PerformedBy              string `json:"created_by"`
	Active                   bool   `json:"active"`
}
type partDTO struct {
	ID                   string `json:"id"`
	SKU                  string `json:"sku"`
	InternalCode         string `json:"internal_code"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Unit                 string `json:"unit"`
	PurchasePriceInCents int64  `json:"purchase_price_in_cents"`
	SalePriceInCents     int64  `json:"sale_price_in_cents"`
	Stock                int64  `json:"stock"`
	ReservedStock        int64  `json:"reserved_stock"`
	AvailableStock       int64  `json:"available_stock"`
	PerformedBy          string `json:"created_by"`
	Active               bool   `json:"active"`
}
type serviceInput struct {
	InternalCode             string `json:"internal_code"`
	Name                     string `json:"name"`
	Description              string `json:"description"`
	SalePriceInCents         int64  `json:"sale_price_in_cents"`
	EstimatedDurationMinutes int    `json:"estimated_duration_minutes"`
	CreatedBy                string `json:"created_by"`
	UpdatedBy                string `json:"updated_by"`
	Active                   *bool  `json:"active"`
}
type partInput struct {
	SKU                  string `json:"sku"`
	InternalCode         string `json:"internal_code"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Unit                 string `json:"unit"`
	PurchasePriceInCents int64  `json:"purchase_price_in_cents"`
	SalePriceInCents     int64  `json:"sale_price_in_cents"`
	InitialStock         int64  `json:"initial_stock"`
	CreatedBy            string `json:"created_by"`
	UpdatedBy            string `json:"updated_by"`
	Active               *bool  `json:"active"`
}
type movementInput struct {
	Type        domain.MovementType `json:"type"`
	Quantity    int64               `json:"quantity"`
	Description string              `json:"description"`
	PerformedBy string              `json:"created_by"`
}
type movementDTO struct {
	ID          string    `json:"id"`
	PartID      string    `json:"part_id"`
	Type        string    `json:"type"`
	Quantity    int64     `json:"quantity"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	WorkOrderID *string   `json:"work_order_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance"`
}

func active(v *bool) bool { return v == nil || *v }
func (h *Handler) createService(c *echo.Context) error {
	var x serviceInput
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	v, e := h.service.CreateService(c.Request().Context(), application.ServiceInput{InternalCode: x.InternalCode, Name: x.Name, Description: x.Description, SalePriceInCents: x.SalePriceInCents, EstimatedDurationMinutes: x.EstimatedDurationMinutes, PerformedBy: authenticatedUserID(c), Active: active(x.Active)})
	if e != nil {
		return fail(c, e)
	}
	c.Response().Header().Set(echo.HeaderLocation, "/api/v1/services/"+v.ID())
	return c.JSON(http.StatusCreated, serviceDTO{ID: v.ID(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), SalePriceInCents: v.SalePriceInCents(), EstimatedDurationMinutes: v.EstimatedDurationMinutes(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) listServices(c *echo.Context) error {
	p, e := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if e != nil {
		return bad(c, e)
	}
	page, e := h.service.ListServices(c.Request().Context(), p)
	if e != nil {
		return fail(c, e)
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	out := make([]serviceDTO, 0, len(page.Items))
	for _, v := range page.Items {
		out = append(out, serviceDTO{ID: v.ID(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), SalePriceInCents: v.SalePriceInCents(), EstimatedDurationMinutes: v.EstimatedDurationMinutes(), PerformedBy: v.PerformedBy(), Active: v.Active()})
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(pagination.NewPage(out, p, page.TotalItems), c.Request().URL.String()))
}
func (h *Handler) getService(c *echo.Context) error {
	v, e := h.service.GetService(c.Request().Context(), c.Param("id"))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, serviceDTO{ID: v.ID(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), SalePriceInCents: v.SalePriceInCents(), EstimatedDurationMinutes: v.EstimatedDurationMinutes(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) updateService(c *echo.Context) error {
	var x serviceInput
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	if !hasPermission(c, identityapp.PermissionProductsPriceUpdate) {
		current, err := h.service.GetService(c.Request().Context(), c.Param("id"))
		if err != nil {
			return fail(c, err)
		}
		if current.SalePriceInCents() != x.SalePriceInCents {
			return write(c, http.StatusForbidden, "Insufficient permissions", "Attendants cannot change prices.")
		}
	}
	v, e := h.service.UpdateService(c.Request().Context(), c.Param("id"), application.ServiceInput{InternalCode: x.InternalCode, Name: x.Name, Description: x.Description, SalePriceInCents: x.SalePriceInCents, EstimatedDurationMinutes: x.EstimatedDurationMinutes, PerformedBy: authenticatedUserID(c), Active: active(x.Active)})
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, serviceDTO{ID: v.ID(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), SalePriceInCents: v.SalePriceInCents(), EstimatedDurationMinutes: v.EstimatedDurationMinutes(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) deleteService(c *echo.Context) error {
	if e := h.service.DeleteService(c.Request().Context(), c.Param("id")); e != nil {
		return fail(c, e)
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *Handler) createPart(c *echo.Context) error {
	var x partInput
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	v, e := h.service.CreatePart(c.Request().Context(), application.PartInput{SKU: x.SKU, InternalCode: x.InternalCode, Name: x.Name, Description: x.Description, Unit: x.Unit, PurchasePriceInCents: x.PurchasePriceInCents, SalePriceInCents: x.SalePriceInCents, InitialStock: x.InitialStock, PerformedBy: authenticatedUserID(c), Active: active(x.Active)})
	if e != nil {
		return fail(c, e)
	}
	c.Response().Header().Set(echo.HeaderLocation, "/api/v1/parts/"+v.ID())
	return c.JSON(http.StatusCreated, partDTO{ID: v.ID(), SKU: v.SKU(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), PurchasePriceInCents: v.PurchasePriceInCents(), SalePriceInCents: v.SalePriceInCents(), Stock: v.Stock(), ReservedStock: v.ReservedStock(), AvailableStock: v.AvailableStock(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) listParts(c *echo.Context) error {
	p, e := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if e != nil {
		return bad(c, e)
	}
	page, e := h.service.ListParts(c.Request().Context(), p)
	if e != nil {
		return fail(c, e)
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	out := make([]partDTO, 0, len(page.Items))
	for _, v := range page.Items {
		out = append(out, partDTO{ID: v.ID(), SKU: v.SKU(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), PurchasePriceInCents: v.PurchasePriceInCents(), SalePriceInCents: v.SalePriceInCents(), Stock: v.Stock(), ReservedStock: v.ReservedStock(), AvailableStock: v.AvailableStock(), PerformedBy: v.PerformedBy(), Active: v.Active()})
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(pagination.NewPage(out, p, page.TotalItems), c.Request().URL.String()))
}
func (h *Handler) getPart(c *echo.Context) error {
	v, e := h.service.GetPart(c.Request().Context(), c.Param("id"))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, partDTO{ID: v.ID(), SKU: v.SKU(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), PurchasePriceInCents: v.PurchasePriceInCents(), SalePriceInCents: v.SalePriceInCents(), Stock: v.Stock(), ReservedStock: v.ReservedStock(), AvailableStock: v.AvailableStock(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) updatePart(c *echo.Context) error {
	var x partInput
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	if !hasPermission(c, identityapp.PermissionProductsPriceUpdate) {
		current, err := h.service.GetPart(c.Request().Context(), c.Param("id"))
		if err != nil {
			return fail(c, err)
		}
		if current.PurchasePriceInCents() != x.PurchasePriceInCents || current.SalePriceInCents() != x.SalePriceInCents {
			return write(c, http.StatusForbidden, "Insufficient permissions", "Attendants cannot change prices.")
		}
	}
	v, e := h.service.UpdatePart(c.Request().Context(), c.Param("id"), application.PartInput{SKU: x.SKU, InternalCode: x.InternalCode, Name: x.Name, Description: x.Description, Unit: x.Unit, PurchasePriceInCents: x.PurchasePriceInCents, SalePriceInCents: x.SalePriceInCents, PerformedBy: authenticatedUserID(c), Active: active(x.Active)})
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, partDTO{ID: v.ID(), SKU: v.SKU(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), PurchasePriceInCents: v.PurchasePriceInCents(), SalePriceInCents: v.SalePriceInCents(), Stock: v.Stock(), ReservedStock: v.ReservedStock(), AvailableStock: v.AvailableStock(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) deletePart(c *echo.Context) error {
	if e := h.service.DeletePart(c.Request().Context(), c.Param("id")); e != nil {
		return fail(c, e)
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *Handler) moveStock(c *echo.Context) error {
	var x movementInput
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	v, e := h.service.MoveStock(c.Request().Context(), c.Param("id"), x.Type, x.Quantity, x.Description, authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, partDTO{ID: v.ID(), SKU: v.SKU(), InternalCode: v.InternalCode(), Name: v.Name(), Description: v.Description(), Unit: v.Unit(), PurchasePriceInCents: v.PurchasePriceInCents(), SalePriceInCents: v.SalePriceInCents(), Stock: v.Stock(), ReservedStock: v.ReservedStock(), AvailableStock: v.AvailableStock(), PerformedBy: v.PerformedBy(), Active: v.Active()})
}
func (h *Handler) listStockMovements(c *echo.Context) error {
	p, e := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if e != nil {
		return bad(c, e)
	}
	page, e := h.service.ListStockMovements(c.Request().Context(), c.Param("id"), p)
	if e != nil {
		return fail(c, e)
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	out := make([]movementDTO, 0, len(page.Items))
	for _, m := range page.Items {
		out = append(out, movementDTO{ID: m.ID, PartID: m.PartID, Type: m.Type, Quantity: m.Quantity, Description: m.Description, CreatedBy: m.PerformedBy, WorkOrderID: m.WorkOrderID, CreatedAt: m.CreatedAt})
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(pagination.NewPage(out, p, page.TotalItems), c.Request().URL.String()))
}
func authenticatedUserID(c *echo.Context) string {
	userID, _ := c.Get("user_id").(string)
	return userID
}
func hasPermission(c *echo.Context, permission identityapp.Permission) bool {
	permissions, _ := c.Get("permissions").([]identityapp.Permission)
	for _, candidate := range permissions {
		if candidate == permission {
			return true
		}
	}
	return false
}
func decode(c *echo.Context, v any) error {
	d := json.NewDecoder(c.Request().Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
func bad(c *echo.Context, e error) error {
	return write(c, http.StatusBadRequest, "Invalid request", e.Error())
}
func fail(c *echo.Context, e error) error {
	switch {
	case errors.Is(e, application.ErrNotFound):
		return write(c, http.StatusNotFound, "Catalog item not found", "The requested catalog item does not exist.")
	case errors.Is(e, application.ErrDuplicate):
		return write(c, http.StatusConflict, "Catalog item already exists", "A catalog item with this identifier already exists.")
	case errors.Is(e, domain.ErrInsufficientStock):
		return write(c, http.StatusConflict, "Insufficient stock", e.Error())
	case errors.Is(e, domain.ErrPartHasStock):
		return write(c, http.StatusConflict, "Part still has stock", e.Error())
	case errors.Is(e, domain.ErrInvalidID), errors.Is(e, domain.ErrInvalidName), errors.Is(e, domain.ErrInvalidInternalCode), errors.Is(e, domain.ErrInvalidSKU), errors.Is(e, domain.ErrInvalidPrice), errors.Is(e, domain.ErrInvalidPurchasePrice), errors.Is(e, domain.ErrInvalidDuration), errors.Is(e, domain.ErrInvalidUnit), errors.Is(e, domain.ErrInvalidStockChange), errors.Is(e, domain.ErrInvalidMovement):
		return write(c, http.StatusBadRequest, "Invalid catalog item", e.Error())
	default:
		return write(c, http.StatusInternalServerError, "Internal server error", "The request could not be completed.")
	}
}
func write(c *echo.Context, status int, title, detail string) error {
	body, _ := json.Marshal(problem{Type: "about:blank", Title: title, Status: status, Detail: detail, Instance: c.Request().URL.Path})
	return c.Blob(status, "application/problem+json", body)
}
