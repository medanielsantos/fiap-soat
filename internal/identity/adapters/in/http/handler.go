package identityhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	identitypostgres "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/postgres"
	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

type Handler struct {
	service   *application.Service
	validator *identityjwt.Issuer
}

func NewHandler(service *application.Service, validator *identityjwt.Issuer) *Handler {
	return &Handler{service: service, validator: validator}
}

func (h *Handler) RegisterRoutes(api *echo.Group) {
	auth := api.Group("/auth")
	auth.POST("/register", h.registerFirst)
	auth.POST("/login", h.login)
	auth.POST("/password/forgot", h.forgot)
	auth.POST("/password/reset", h.reset)
	api.POST("/admin-users", h.createAdmin, h.RequireAuth())
	api.PATCH("/admin-users/:id/status", h.updateStatus, h.RequireAuth())
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
type statusRequest struct {
	Status application.UserStatus `json:"status"`
}

func (h *Handler) registerFirst(c *echo.Context) error {
	var req registerRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	user, err := h.service.Register(c.Request().Context(), req.Name, req.Email, req.Password, true)
	if err != nil {
		return h.registrationError(c, err)
	}
	return c.JSON(http.StatusCreated, user)
}
func (h *Handler) createAdmin(c *echo.Context) error {
	var req registerRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	user, err := h.service.Register(c.Request().Context(), req.Name, req.Email, req.Password, false)
	if err != nil {
		return h.registrationError(c, err)
	}
	return c.JSON(http.StatusCreated, user)
}
func (h *Handler) updateStatus(c *echo.Context) error {
	var req statusRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	requestingUserID, _ := c.Get("user_id").(string)
	user, err := h.service.SetStatus(c.Request().Context(), c.Param("id"), requestingUserID, req.Status)
	switch {
	case errors.Is(err, application.ErrUserNotFound):
		return problem(c, 404, "User not found")
	case errors.Is(err, application.ErrInvalidStatus), errors.Is(err, application.ErrCannotDeactivateSelf):
		return problem(c, 400, "Invalid user status")
	case err != nil:
		return problem(c, 500, "Could not update user status")
	default:
		return c.JSON(http.StatusOK, user)
	}
}
func (h *Handler) registrationError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, application.ErrRegistrationClosed):
		return problem(c, 409, "Initial registration is closed")
	case identitypostgres.IsUniqueViolation(err):
		return problem(c, 409, "Email already registered")
	default:
		if strings.Contains(err.Error(), "invalid user") || strings.Contains(err.Error(), "password") {
			return problem(c, 400, "Invalid user details")
		}
		return problem(c, 500, "Could not register user")
	}
}
func (h *Handler) login(c *echo.Context) error {
	var req credentials
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	session, err := h.service.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return problem(c, 401, "Invalid credentials")
	}
	return c.JSON(http.StatusOK, session)
}
func (h *Handler) forgot(c *echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	if err := h.service.RequestPasswordReset(c.Request().Context(), req.Email); err != nil {
		log.Printf("send password reset email: %v", err)
	}
	return c.NoContent(http.StatusAccepted)
}
func (h *Handler) reset(c *echo.Context) error {
	var req resetRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	if err := h.service.ResetPassword(c.Request().Context(), req.Token, req.Password); err != nil {
		return problem(c, 400, "Invalid or expired reset token")
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *Handler) RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			parts := strings.SplitN(c.Request().Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return problem(c, 401, "Authentication required")
			}
			id, err := h.validator.Validate(parts[1])
			if err != nil {
				return problem(c, 401, "Invalid access token")
			}
			if !h.service.IsActiveUser(c.Request().Context(), id) {
				return problem(c, 401, "Invalid access token")
			}
			c.Set("user_id", id)
			return next(c)
		}
	}
}
func decode(c *echo.Context, target any) error {
	d := json.NewDecoder(c.Request().Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("request must contain one JSON object")
	}
	return nil
}
func problem(c *echo.Context, status int, title string) error {
	return c.JSON(status, map[string]any{"type": "about:blank", "title": title, "status": status, "instance": c.Request().URL.Path})
}
