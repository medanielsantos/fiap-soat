package customerhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/platform/ratelimit"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	workorderapp "github.com/medanielsantos/fiap-soat/internal/workorder/application"
	workorderdomain "github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

type Handler struct {
	service     *application.CustomerService
	vehicles    *vehicleapp.VehicleService
	issuer      *identityjwt.Issuer
	workOrders  *workorderapp.Service
	rateLimiter ratelimit.Limiter
}

type ClientDependencies struct {
	Vehicles    *vehicleapp.VehicleService
	Issuer      *identityjwt.Issuer
	WorkOrders  *workorderapp.Service
	RateLimiter ratelimit.Limiter
}

func NewHandler(service *application.CustomerService, client ...ClientDependencies) *Handler {
	handler := &Handler{service: service}
	if len(client) > 0 {
		handler.vehicles, handler.issuer, handler.workOrders = client[0].Vehicles, client[0].Issuer, client[0].WorkOrders
		handler.rateLimiter = client[0].RateLimiter
	}
	return handler
}

func (handler *Handler) RegisterRoutes(group *echo.Group) {
	group.POST("", handler.Create)
	group.GET("", handler.List)
	group.GET("/:id", handler.Get)
	group.PUT("/:id", handler.Update)
	group.DELETE("/:id", handler.Delete)
}

func (handler *Handler) RegisterClientRoutes(group *echo.Group) {
	group.POST("/client/auth/login", handler.clientLogin)
	group.POST("/client/auth/plate", handler.plateLogin)
	group.POST("/client/auth/password/forgot", handler.clientForgotPassword)
	group.POST("/client/auth/password/reset", handler.clientResetPassword)
	group.GET("/client/vehicles", handler.clientVehicles, handler.requireClient())
	group.GET("/client/work-orders", handler.clientWorkOrders, handler.requireClient())
	group.GET("/client/work-orders/:id", handler.clientWorkOrder, handler.requireClient())
	group.POST("/client/work-orders/:id/quotes/approve", handler.clientApprove, handler.requireClient())
	group.POST("/client/work-orders/:id/quotes/reject", handler.clientReject, handler.requireClient())
}

type customerRequest struct {
	Document string                `json:"document"`
	Name     string                `json:"name"`
	Email    string                `json:"email"`
	Phone    string                `json:"phone"`
	Password string                `json:"password,omitempty"`
	Status   domain.CustomerStatus `json:"status"`
}

type customerResponse struct {
	ID       string                `json:"id"`
	Document string                `json:"document"`
	Name     string                `json:"name"`
	Email    string                `json:"email"`
	Phone    string                `json:"phone"`
	Status   domain.CustomerStatus `json:"status"`
}

type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance"`
}

type clientLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type plateLoginRequest struct {
	Plate          string `json:"plate"`
	DocumentPrefix string `json:"document_prefix"`
}
type clientResetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
type clientSession struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}
type clientQuote struct {
	Version      int               `json:"version"`
	TotalInCents int64             `json:"total_in_cents"`
	Decision     string            `json:"decision"`
	DecidedAt    *time.Time        `json:"decided_at,omitempty"`
	Items        []clientQuoteItem `json:"items"`
}
type clientQuoteItem struct {
	Type                     workorderdomain.ItemType `json:"type"`
	Code                     string                   `json:"code"`
	Description              string                   `json:"description"`
	Unit                     string                   `json:"unit"`
	Quantity                 int64                    `json:"quantity"`
	UnitPriceInCents         int64                    `json:"unit_price_in_cents"`
	SubtotalInCents          int64                    `json:"subtotal_in_cents"`
	EstimatedDurationMinutes int                      `json:"estimated_duration_minutes,omitempty"`
}
type clientWorkOrderItem struct {
	Type                     workorderdomain.ItemType `json:"type"`
	Code                     string                   `json:"code"`
	Description              string                   `json:"description"`
	Unit                     string                   `json:"unit"`
	Quantity                 int64                    `json:"quantity"`
	UnitPriceInCents         int64                    `json:"unit_price_in_cents"`
	ConsumedQuantity         int64                    `json:"consumed_quantity"`
	EstimatedDurationMinutes int                      `json:"estimated_duration_minutes,omitempty"`
}
type clientVehicle struct {
	ID    string `json:"id"`
	Plate string `json:"plate"`
	Brand string `json:"brand"`
	Model string `json:"model"`
	Year  int    `json:"year"`
}
type clientWorkOrder struct {
	ID        string                 `json:"id"`
	Status    workorderdomain.Status `json:"status"`
	Diagnosis string                 `json:"diagnosis"`
	Items     []clientWorkOrderItem  `json:"items"`
	Quotes    []clientQuote          `json:"quotes"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

const (
	clientAuthLimit       = 5
	clientAPIRequestLimit = 60
)

func (handler *Handler) clientLogin(c *echo.Context) error {
	if err := handler.enforceRateLimit(c, "client:auth:"+c.RealIP(), clientAuthLimit); err != nil {
		return err
	}
	var req clientLoginRequest
	if err := decodeGeneric(c, &req); err != nil {
		return invalidRequest(c, "Invalid credentials")
	}
	customer, err := handler.service.Authenticate(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return writeProblem(c, http.StatusUnauthorized, "Invalid credentials", "The credentials are invalid.")
	}
	token, expiry, err := handler.issuer.IssueClient(customer.ID(), "")
	if err != nil {
		return writeProblem(c, 500, "Internal server error", "The request could not be completed.")
	}
	return c.JSON(http.StatusOK, clientSession{AccessToken: token, ExpiresAt: expiry})
}

func (handler *Handler) plateLogin(c *echo.Context) error {
	if err := handler.enforceRateLimit(c, "client:auth:"+c.RealIP(), clientAuthLimit); err != nil {
		return err
	}
	var req plateLoginRequest
	if err := decodeGeneric(c, &req); err != nil {
		return invalidRequest(c, "Invalid credentials")
	}
	customer, vehicleID, err := handler.service.AuthenticateVehicle(c.Request().Context(), req.Plate, req.DocumentPrefix)
	if err != nil {
		return writeProblem(c, http.StatusUnauthorized, "Invalid credentials", "The credentials are invalid.")
	}
	token, expiry, err := handler.issuer.IssueClient(customer.ID(), vehicleID)
	if err != nil {
		return writeProblem(c, 500, "Internal server error", "The request could not be completed.")
	}
	return c.JSON(http.StatusOK, clientSession{AccessToken: token, ExpiresAt: expiry})
}

func (handler *Handler) clientForgotPassword(c *echo.Context) error {
	if err := handler.enforceRateLimit(c, "client:auth:"+c.RealIP(), clientAuthLimit); err != nil {
		return err
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeGeneric(c, &req); err != nil {
		return invalidRequest(c, "Invalid request")
	}
	if err := handler.service.RequestPasswordReset(c.Request().Context(), req.Email); err != nil {
		log.Printf("send customer password reset email: %v", err)
	}
	return c.NoContent(http.StatusAccepted)
}

func (handler *Handler) clientResetPassword(c *echo.Context) error {
	if err := handler.enforceRateLimit(c, "client:auth:"+c.RealIP(), clientAuthLimit); err != nil {
		return err
	}
	var req clientResetRequest
	if err := decodeGeneric(c, &req); err != nil {
		return invalidRequest(c, "Invalid request")
	}
	if err := handler.service.ResetPassword(c.Request().Context(), req.Token, req.Password); err != nil {
		return writeProblem(c, http.StatusBadRequest, "Invalid reset request", "The token is invalid, expired, or the password does not meet requirements.")
	}
	return c.NoContent(http.StatusNoContent)
}

func (handler *Handler) requireClient() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			parts := strings.SplitN(c.Request().Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return writeProblem(c, 401, "Authentication required", "A client access token is required.")
			}
			customerID, vehicleID, err := handler.issuer.ValidateClient(parts[1])
			if err != nil {
				return writeProblem(c, 401, "Invalid access token", "The token is invalid or expired.")
			}
			customer, err := handler.service.Get(c.Request().Context(), customerID)
			if err != nil || !customer.IsActive() {
				return writeProblem(c, 401, "Invalid access token", "The token is invalid or expired.")
			}
			if err := handler.enforceRateLimit(c, "client:api:"+customerID+":"+c.RealIP(), clientAPIRequestLimit); err != nil {
				return err
			}
			c.Set("client_id", customerID)
			c.Set("vehicle_id", vehicleID)
			return next(c)
		}
	}
}

func (handler *Handler) clientVehicles(c *echo.Context) error {
	customerID, _ := c.Get("client_id").(string)
	vehicleID, _ := c.Get("vehicle_id").(string)
	params, err := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if err != nil {
		return writeProblem(c, 400, "Invalid pagination", err.Error())
	}
	filter := vehicleapp.ListFilter{CustomerID: customerID, VehicleID: vehicleID}
	page, err := handler.vehicles.List(c.Request().Context(), filter, params)
	if err != nil {
		return writeProblem(c, 500, "Internal server error", "The request could not be completed.")
	}
	vehicles := make([]clientVehicle, 0, len(page.Items))
	for _, vehicle := range page.Items {
		vehicles = append(vehicles, clientVehicle{ID: vehicle.ID(), Plate: vehicle.Plate().String(), Brand: vehicle.Brand(), Model: vehicle.Model(), Year: vehicle.Year()})
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(pagination.NewPage(vehicles, params, page.TotalItems), c.Request().URL.String()))
}

func (handler *Handler) clientWorkOrders(c *echo.Context) error {
	customerID, _ := c.Get("client_id").(string)
	vehicleID, _ := c.Get("vehicle_id").(string)
	params, err := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if err != nil {
		return writeProblem(c, 400, "Invalid pagination", err.Error())
	}
	page, err := handler.workOrders.ListForClient(c.Request().Context(), customerID, vehicleID, params)
	if err != nil {
		return writeProblem(c, 500, "Internal server error", "The request could not be completed.")
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	orders := make([]clientWorkOrder, 0, len(page.Items))
	for _, order := range page.Items {
		orders = append(orders, clientWorkOrderFromDomain(order))
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(pagination.NewPage(orders, params, page.TotalItems), c.Request().URL.String()))
}

func (handler *Handler) clientWorkOrder(c *echo.Context) error {
	order, err := handler.clientOrder(c)
	if err != nil {
		return writeProblem(c, http.StatusNotFound, "Work order not found", "The requested work order does not exist.")
	}
	return c.JSON(http.StatusOK, clientWorkOrderFromDomain(order))
}

func (handler *Handler) clientApprove(c *echo.Context) error {
	if _, err := handler.clientOrder(c); err != nil {
		return writeProblem(c, 404, "Work order not found", "The requested work order does not exist.")
	}
	customerID, _ := c.Get("client_id").(string)
	order, err := handler.workOrders.ApproveQuote(c.Request().Context(), c.Param("id"), customerID)
	if err != nil {
		return writeProblem(c, 409, "Quote cannot be approved", "The quote is unavailable or stock is insufficient.")
	}
	return c.JSON(http.StatusOK, clientWorkOrderFromDomain(order))
}

func (handler *Handler) clientReject(c *echo.Context) error {
	if _, err := handler.clientOrder(c); err != nil {
		return writeProblem(c, 404, "Work order not found", "The requested work order does not exist.")
	}
	var req struct {
		Note string `json:"note"`
	}
	if c.Request().ContentLength != 0 {
		if err := decodeGeneric(c, &req); err != nil {
			return invalidRequest(c, err.Error())
		}
	}
	customerID, _ := c.Get("client_id").(string)
	order, err := handler.workOrders.RejectQuote(c.Request().Context(), c.Param("id"), customerID, req.Note)
	if err != nil {
		return writeProblem(c, 409, "Quote cannot be rejected", "The quote is unavailable.")
	}
	return c.JSON(http.StatusOK, clientWorkOrderFromDomain(order))
}

func clientWorkOrderFromDomain(order workorderdomain.WorkOrder) clientWorkOrder {
	out := clientWorkOrder{ID: order.ID, Status: order.Status, Diagnosis: order.Diagnosis, Items: make([]clientWorkOrderItem, 0, len(order.Items)), Quotes: make([]clientQuote, 0, len(order.Quotes)), CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt}
	for _, item := range order.Items {
		out.Items = append(out.Items, clientWorkOrderItem{Type: item.Type, Code: item.Code, Description: item.Description, Unit: item.Unit, Quantity: item.Quantity, UnitPriceInCents: item.UnitPriceInCents, ConsumedQuantity: item.ConsumedQuantity, EstimatedDurationMinutes: item.EstimatedDurationMinutes})
	}
	for _, quote := range order.Quotes {
		items := make([]clientQuoteItem, 0, len(quote.Items))
		for _, quoteItem := range quote.Items {
			item := quoteItem.Item
			items = append(items, clientQuoteItem{Type: item.Type, Code: item.Code, Description: item.Description, Unit: item.Unit, Quantity: item.Quantity, UnitPriceInCents: item.UnitPriceInCents, SubtotalInCents: quoteItem.SubtotalInCents, EstimatedDurationMinutes: item.EstimatedDurationMinutes})
		}
		out.Quotes = append(out.Quotes, clientQuote{Version: quote.Version, TotalInCents: quote.TotalInCents, Decision: quote.Decision, DecidedAt: quote.DecidedAt, Items: items})
	}
	return out
}

func (handler *Handler) clientOrder(c *echo.Context) (workorderdomain.WorkOrder, error) {
	customerID, _ := c.Get("client_id").(string)
	vehicleID, _ := c.Get("vehicle_id").(string)
	return handler.workOrders.GetForClient(c.Request().Context(), c.Param("id"), customerID, vehicleID)
}

func (handler *Handler) enforceRateLimit(c *echo.Context, key string, limit int) error {
	if handler.rateLimiter == nil {
		return writeProblem(c, http.StatusServiceUnavailable, "Client access unavailable", "Rate limiting is temporarily unavailable.")
	}
	allowed, err := handler.rateLimiter.Allow(c.Request().Context(), key, limit, time.Minute)
	if err != nil {
		return writeProblem(c, http.StatusServiceUnavailable, "Client access unavailable", "Rate limiting is temporarily unavailable.")
	}
	if !allowed {
		c.Response().Header().Set(echo.HeaderRetryAfter, "60")
		return writeProblem(c, http.StatusTooManyRequests, "Too many requests", "Try again later.")
	}
	return nil
}

func (handler *Handler) Create(c *echo.Context) error {
	var request customerRequest
	if err := decodeRequest(c, &request); err != nil {
		return invalidRequest(c, err.Error())
	}
	customer, err := handler.service.Create(c.Request().Context(), application.CustomerInput{
		Document: request.Document, Name: request.Name, Email: request.Email, Phone: request.Phone, Password: request.Password, Status: request.Status,
	})
	if err != nil {
		return handleError(c, err)
	}
	c.Response().Header().Set(echo.HeaderLocation, "/api/v1/customers/"+customer.ID())
	return c.JSON(http.StatusCreated, responseFromDomain(customer))
}

func (handler *Handler) List(c *echo.Context) error {
	params, err := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if err != nil {
		return writeProblem(c, http.StatusBadRequest, "Invalid pagination", err.Error())
	}
	page, err := handler.service.List(c.Request().Context(), params)
	if err != nil {
		return handleError(c, err)
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	response := make([]customerResponse, 0, len(page.Items))
	for _, customer := range page.Items {
		response = append(response, responseFromDomain(customer))
	}
	responsePage := pagination.NewPage(response, params, page.TotalItems)
	return c.JSON(http.StatusOK, pagination.NewResponse(responsePage, c.Request().URL.String()))
}

func (handler *Handler) Get(c *echo.Context) error {
	customer, err := handler.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(http.StatusOK, responseFromDomain(customer))
}

func (handler *Handler) Update(c *echo.Context) error {
	var request customerRequest
	if err := decodeRequest(c, &request); err != nil {
		return invalidRequest(c, err.Error())
	}
	customer, err := handler.service.Update(c.Request().Context(), c.Param("id"), application.CustomerInput{
		Document: request.Document, Name: request.Name, Email: request.Email, Phone: request.Phone, Password: request.Password, Status: request.Status,
	})
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(http.StatusOK, responseFromDomain(customer))
}

func (handler *Handler) Delete(c *echo.Context) error {
	if err := handler.service.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return handleError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func responseFromDomain(customer domain.Customer) customerResponse {
	return customerResponse{
		ID: customer.ID(), Document: maskDocument(customer.Document().String()), Name: customer.Name(),
		Email: customer.Email(), Phone: customer.Phone(), Status: customer.Status(),
	}
}

func maskDocument(value string) string {
	if len(value) == 11 {
		return "***.***.***-" + value[9:]
	}
	if len(value) == 14 {
		return "**.***.***/****-" + value[12:]
	}
	return ""
}

func handleError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, application.ErrCustomerNotFound):
		return writeProblem(c, http.StatusNotFound, "Customer not found", "The requested customer does not exist.")
	case errors.Is(err, application.ErrDuplicateDocument):
		return writeProblem(c, http.StatusConflict, "Customer already exists", "A customer with this document already exists.")
	case errors.Is(err, application.ErrDuplicateEmail):
		return writeProblem(c, http.StatusConflict, "Email already registered", "A customer with this email already exists.")
	case errors.Is(err, domain.ErrInvalidDocument), errors.Is(err, domain.ErrInvalidName),
		errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidPhone), errors.Is(err, application.ErrInvalidPassword),
		errors.Is(err, domain.ErrInvalidStatus):
		return writeProblem(c, http.StatusBadRequest, "Invalid customer", err.Error())
	default:
		return writeProblem(c, http.StatusInternalServerError, "Internal server error", "The request could not be completed.")
	}
}

func invalidRequest(c *echo.Context, detail string) error {
	return writeProblem(c, http.StatusBadRequest, "Invalid request", detail)
}

func writeProblem(c *echo.Context, status int, title, detail string) error {
	body, _ := json.Marshal(problem{
		Type: "about:blank", Title: title, Status: status, Detail: detail, Instance: c.Request().URL.Path,
	})
	return c.Blob(status, "application/problem+json", body)
}

func decodeRequest(c *echo.Context, target *customerRequest) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func decodeGeneric(c *echo.Context, target any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
