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
	auth.POST("/login", h.login)
	auth.POST("/password/forgot", h.forgot)
	auth.POST("/password/reset", h.reset)
	api.POST("/admin-users", h.createAdmin, h.RequireAuth(), h.RequirePermission(application.PermissionUsersCreate))
	api.PATCH("/admin-users/:id/status", h.updateStatus, h.RequireAuth(), h.RequirePermission(application.PermissionUsersStatusUpdate))
	api.PATCH("/admin-users/:id/role", h.updateRole, h.RequireAuth(), h.RequirePermission(application.PermissionUsersRoleAssign))
	api.DELETE("/admin-users/:id", h.deleteUser, h.RequireAuth(), h.RequirePermission(application.PermissionUsersDelete))
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type registerRequest struct {
	Name     string               `json:"name"`
	Email    string               `json:"email"`
	Password string               `json:"password"`
	Role     application.UserRole `json:"role"`
}
type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
type statusRequest struct {
	Status application.UserStatus `json:"status"`
}

type roleRequest struct {
	Role application.UserRole `json:"role"`
}

func (h *Handler) createAdmin(c *echo.Context) error {
	var req registerRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	requesterRole, _ := c.Get("role").(application.UserRole)
	if req.Role == "" {
		req.Role = application.UserRoleAttendant
	}
	if requesterRole == application.UserRoleManager && req.Role != application.UserRoleAttendant {
		return problem(c, http.StatusForbidden, "Managers can only create attendants")
	}
	user, err := h.service.Register(c.Request().Context(), req.Name, req.Email, req.Password, req.Role)
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
	if req.Status == application.UserStatusDeleted && !hasPermission(c, application.PermissionUsersDelete) {
		return problem(c, http.StatusForbidden, "Insufficient permissions to delete users")
	}
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
func (h *Handler) updateRole(c *echo.Context) error {
	var req roleRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	requestingUserID, _ := c.Get("user_id").(string)
	user, err := h.service.SetRole(c.Request().Context(), c.Param("id"), requestingUserID, req.Role)
	switch {
	case errors.Is(err, application.ErrUserNotFound):
		return problem(c, http.StatusNotFound, "User not found")
	case errors.Is(err, application.ErrCannotChangeOwnRole):
		return problem(c, http.StatusConflict, "Cannot change your own role")
	case err != nil:
		if strings.Contains(err.Error(), "invalid user role") {
			return problem(c, http.StatusBadRequest, "Invalid user role")
		}
		return problem(c, http.StatusInternalServerError, "Could not update user role")
	default:
		return c.JSON(http.StatusOK, user)
	}
}

func (h *Handler) deleteUser(c *echo.Context) error {
	requestingUserID, _ := c.Get("user_id").(string)
	if err := h.service.DeleteUser(c.Request().Context(), c.Param("id"), requestingUserID); err != nil {
		if errors.Is(err, application.ErrCannotDeleteSelf) {
			return problem(c, http.StatusConflict, "Cannot delete your own user")
		}
		if errors.Is(err, application.ErrUserNotFound) {
			return problem(c, http.StatusNotFound, "User not found")
		}
		return problem(c, http.StatusInternalServerError, "Could not delete user")
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *Handler) registrationError(c *echo.Context, err error) error {
	switch {
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
			role, permissions, active := h.service.ActivePermissions(c.Request().Context(), id)
			if !active {
				return problem(c, 401, "Invalid access token")
			}
			c.Set("user_id", id)
			c.Set("role", role)
			c.Set("permissions", permissions)
			return next(c)
		}
	}
}

func (h *Handler) RequirePermission(permission application.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !hasPermission(c, permission) {
				return problem(c, http.StatusForbidden, "Insufficient permissions")
			}
			return next(c)
		}
	}
}

func hasPermission(c *echo.Context, permission application.Permission) bool {
	permissions, _ := c.Get("permissions").([]application.Permission)
	for _, candidate := range permissions {
		if candidate == permission {
			return true
		}
	}
	return false
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
