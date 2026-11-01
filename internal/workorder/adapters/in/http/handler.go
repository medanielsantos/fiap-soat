package workorderhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.POST("/:id/diagnosis", h.diagnosis)
	g.POST("/:id/items", h.addItem)
	g.DELETE("/:id/items/:item_id", h.removeItem)
	g.POST("/:id/quotes", h.submitQuote)
	g.POST("/:id/quotes/approve", h.approve)
	g.POST("/:id/quotes/reject", h.reject)
	g.POST("/:id/consumptions", h.consume)
	g.POST("/:id/complete", h.complete)
	g.POST("/:id/deliver", h.deliver)
}

type createRequest struct {
	CustomerID string `json:"customer_id"`
	VehicleID  string `json:"vehicle_id"`
}
type diagnosisRequest struct {
	Diagnosis string `json:"diagnosis"`
}
type itemRequest struct {
	Type      domain.ItemType `json:"type"`
	CatalogID string          `json:"catalog_id"`
	Quantity  int64           `json:"quantity"`
}
type userRequest struct {
}
type clientRequest struct {
	ClientID string `json:"client_id"`
}
type rejectRequest struct {
	ClientID string `json:"client_id"`
	Note     string `json:"note"`
}
type consumeRequest struct {
	PartID   string `json:"part_id"`
	Quantity int64  `json:"quantity"`
}
type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance"`
}

func (h *Handler) create(c *echo.Context) error {
	var x createRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.Create(c.Request().Context(), x.CustomerID, x.VehicleID, authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	c.Response().Header().Set(echo.HeaderLocation, "/api/v1/work-orders/"+o.ID)
	return c.JSON(http.StatusCreated, o)
}
func (h *Handler) list(c *echo.Context) error {
	p, e := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if e != nil {
		return bad(c, e)
	}
	page, e := h.service.List(c.Request().Context(), c.QueryParam("status"), p)
	if e != nil {
		return fail(c, e)
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(page, c.Request().URL.String()))
}
func (h *Handler) get(c *echo.Context) error {
	o, e := h.service.Get(c.Request().Context(), c.Param("id"))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) diagnosis(c *echo.Context) error {
	var x diagnosisRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.Diagnosis(c.Request().Context(), c.Param("id"), x.Diagnosis, authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) addItem(c *echo.Context) error {
	var x itemRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.AddItem(c.Request().Context(), c.Param("id"), x.Type, x.CatalogID, x.Quantity, authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusCreated, o)
}
func (h *Handler) removeItem(c *echo.Context) error {
	o, e := h.service.RemoveItem(c.Request().Context(), c.Param("id"), c.Param("item_id"), authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) submitQuote(c *echo.Context) error {
	var x userRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.SubmitQuote(c.Request().Context(), c.Param("id"), authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) approve(c *echo.Context) error {
	var x clientRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.ApproveQuote(c.Request().Context(), c.Param("id"), x.ClientID)
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) reject(c *echo.Context) error {
	var x rejectRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.RejectQuote(c.Request().Context(), c.Param("id"), x.ClientID, x.Note)
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) consume(c *echo.Context) error {
	var x consumeRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.Consume(c.Request().Context(), c.Param("id"), x.PartID, x.Quantity, authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) complete(c *echo.Context) error {
	var x userRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.Complete(c.Request().Context(), c.Param("id"), authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func (h *Handler) deliver(c *echo.Context) error {
	var x userRequest
	if e := decode(c, &x); e != nil {
		return bad(c, e)
	}
	o, e := h.service.Deliver(c.Request().Context(), c.Param("id"), authenticatedUserID(c))
	if e != nil {
		return fail(c, e)
	}
	return c.JSON(http.StatusOK, o)
}
func authenticatedUserID(c *echo.Context) string {
	userID, _ := c.Get("user_id").(string)
	return userID
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
		return write(c, http.StatusNotFound, "Work order not found", "The requested work order does not exist.")
	case errors.Is(e, application.ErrCustomerOrVehicleUnavailable), errors.Is(e, application.ErrCatalogItemUnavailable):
		return write(c, http.StatusUnprocessableEntity, "Related resource unavailable", e.Error())
	case errors.Is(e, domain.ErrInvalidTransition), errors.Is(e, domain.ErrQuoteUnavailable), errors.Is(e, domain.ErrInsufficientConsumption), errors.Is(e, domain.ErrItemsNotConsumed), errors.Is(e, domain.ErrQuoteEmpty), errors.Is(e, domain.ErrInvalidItem):
		return write(c, http.StatusConflict, "Work order state conflict", e.Error())
	case errors.Is(e, domain.ErrInvalidDiagnosis), errors.Is(e, domain.ErrInvalidPrincipal), errors.Is(e, domain.ErrInvalidQuantity), errors.Is(e, domain.ErrInvalidOrder), errors.Is(e, domain.ErrAmountOverflow):
		return write(c, http.StatusBadRequest, "Invalid work order", e.Error())
	case errors.Is(e, application.ErrInsufficientStock):
		return write(c, http.StatusConflict, "Insufficient stock", e.Error())
	default:
		return write(c, http.StatusInternalServerError, "Internal server error", "The request could not be completed.")
	}
}
func write(c *echo.Context, status int, title, detail string) error {
	body, _ := json.Marshal(problem{Type: "about:blank", Title: title, Status: status, Detail: detail, Instance: c.Request().URL.Path})
	return c.Blob(status, "application/problem+json", body)
}
