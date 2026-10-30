package vehiclehttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

type Handler struct {
	service *application.VehicleService
}

func NewHandler(service *application.VehicleService) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(group *echo.Group) {
	group.POST("", handler.Create)
	group.GET("", handler.List)
	group.GET("/:id", handler.Get)
	group.PUT("/:id", handler.Update)
	group.DELETE("/:id", handler.Delete)
}

type vehicleRequest struct {
	CustomerID string `json:"customer_id"`
	Plate      string `json:"plate"`
	Brand      string `json:"brand"`
	Model      string `json:"model"`
	Year       int    `json:"year"`
}

type vehicleResponse struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Plate      string `json:"plate"`
	Brand      string `json:"brand"`
	Model      string `json:"model"`
	Year       int    `json:"year"`
}

type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance"`
}

func (handler *Handler) Create(c *echo.Context) error {
	var request vehicleRequest
	if err := decodeRequest(c, &request); err != nil {
		return invalidRequest(c, err.Error())
	}
	vehicle, err := handler.service.Create(c.Request().Context(), inputFromRequest(request))
	if err != nil {
		return handleError(c, err)
	}
	c.Response().Header().Set(echo.HeaderLocation, "/api/v1/vehicles/"+vehicle.ID())
	return c.JSON(http.StatusCreated, responseFromDomain(vehicle))
}

func (handler *Handler) List(c *echo.Context) error {
	params, err := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if err != nil {
		return writeProblem(c, http.StatusBadRequest, "Invalid pagination", err.Error())
	}
	filter := application.ListFilter{CustomerID: c.QueryParam("customer_id")}
	page, err := handler.service.List(c.Request().Context(), filter, params)
	if err != nil {
		return handleError(c, err)
	}
	if page.TotalItems == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	response := make([]vehicleResponse, 0, len(page.Items))
	for _, vehicle := range page.Items {
		response = append(response, responseFromDomain(vehicle))
	}
	responsePage := pagination.NewPage(response, params, page.TotalItems)
	return c.JSON(http.StatusOK, pagination.NewResponse(responsePage, c.Request().URL.String()))
}

func (handler *Handler) Get(c *echo.Context) error {
	vehicle, err := handler.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(http.StatusOK, responseFromDomain(vehicle))
}

func (handler *Handler) Update(c *echo.Context) error {
	var request vehicleRequest
	if err := decodeRequest(c, &request); err != nil {
		return invalidRequest(c, err.Error())
	}
	vehicle, err := handler.service.Update(c.Request().Context(), c.Param("id"), inputFromRequest(request))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(http.StatusOK, responseFromDomain(vehicle))
}

func (handler *Handler) Delete(c *echo.Context) error {
	if err := handler.service.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return handleError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func inputFromRequest(request vehicleRequest) application.VehicleInput {
	return application.VehicleInput{
		CustomerID: request.CustomerID, Plate: request.Plate, Brand: request.Brand, Model: request.Model, Year: request.Year,
	}
}

func responseFromDomain(vehicle domain.Vehicle) vehicleResponse {
	return vehicleResponse{
		ID: vehicle.ID(), CustomerID: vehicle.CustomerID(), Plate: vehicle.Plate().String(),
		Brand: vehicle.Brand(), Model: vehicle.Model(), Year: vehicle.Year(),
	}
}

func handleError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, application.ErrVehicleNotFound):
		return writeProblem(c, http.StatusNotFound, "Vehicle not found", "The requested vehicle does not exist.")
	case errors.Is(err, application.ErrDuplicatePlate):
		return writeProblem(c, http.StatusConflict, "Vehicle already exists", "A vehicle with this plate already exists.")
	case errors.Is(err, application.ErrCustomerUnavailable):
		return writeProblem(c, http.StatusUnprocessableEntity, "Customer unavailable", "The customer does not exist or is inactive.")
	case errors.Is(err, domain.ErrInvalidPlate), errors.Is(err, domain.ErrInvalidCustomerID),
		errors.Is(err, domain.ErrInvalidBrand), errors.Is(err, domain.ErrInvalidModel),
		errors.Is(err, domain.ErrInvalidYear):
		return writeProblem(c, http.StatusBadRequest, "Invalid vehicle", err.Error())
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

func decodeRequest(c *echo.Context, target *vehicleRequest) error {
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
