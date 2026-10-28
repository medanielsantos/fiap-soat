package customerhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

type Handler struct {
	service *application.CustomerService
}

func NewHandler(service *application.CustomerService) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(group *echo.Group) {
	group.POST("", handler.Create)
	group.GET("", handler.List)
	group.GET("/:id", handler.Get)
	group.PUT("/:id", handler.Update)
	group.DELETE("/:id", handler.Delete)
}

type customerRequest struct {
	Document string                `json:"document"`
	Name     string                `json:"name"`
	Email    string                `json:"email"`
	Phone    string                `json:"phone"`
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

func (handler *Handler) Create(c *echo.Context) error {
	var request customerRequest
	if err := decodeRequest(c, &request); err != nil {
		return invalidRequest(c, err.Error())
	}
	customer, err := handler.service.Create(c.Request().Context(), application.CustomerInput{
		Document: request.Document, Name: request.Name, Email: request.Email, Phone: request.Phone, Status: request.Status,
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
		Document: request.Document, Name: request.Name, Email: request.Email, Phone: request.Phone, Status: request.Status,
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
	case errors.Is(err, domain.ErrInvalidDocument), errors.Is(err, domain.ErrInvalidName),
		errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidPhone),
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
