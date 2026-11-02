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
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
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
	admin := api.Group("/admin")
	admin.POST("/users", h.createAdmin, h.RequireAuth(), h.RequirePermission(application.PermissionUsersCreate))
	admin.GET("/users", h.listUsers, h.RequireAuth(), h.RequirePermission(application.PermissionUsersRead))
	admin.GET("/users/:id", h.getUser, h.RequireAuth(), h.RequirePermission(application.PermissionUsersRead))
	admin.PATCH("/users/:id/status", h.updateStatus, h.RequireAuth(), h.RequirePermission(application.PermissionUsersStatusUpdate))
	admin.PUT("/users/:id/roles", h.updateRoles, h.RequireAuth(), h.RequirePermission(application.PermissionUsersRoleAssign))
	admin.POST("/users/:id/permissions/:permission", h.grantPermission, h.RequireAuth(), h.RequirePermission(application.PermissionUsersPermissionAssign))
	admin.DELETE("/users/:id/permissions/:permission", h.revokePermission, h.RequireAuth(), h.RequirePermission(application.PermissionUsersPermissionAssign))
	admin.DELETE("/users/:id", h.deleteUser, h.RequireAuth(), h.RequirePermission(application.PermissionUsersDelete))
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type registerRequest struct {
	Name     string                 `json:"name"`
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Role     application.UserRole   `json:"role,omitempty"`
	Roles    []application.UserRole `json:"roles,omitempty"`
}
type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
type statusRequest struct {
	Status application.UserStatus `json:"status"`
}

type rolesRequest struct {
	Roles []application.UserRole `json:"roles"`
}

func (h *Handler) createAdmin(c *echo.Context) error {
	var req registerRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	roles, err := requestedRoles(req.Role, req.Roles)
	if err != nil {
		return problem(c, http.StatusBadRequest, "Invalid user roles")
	}
	requesterRoles, _ := c.Get("roles").([]application.UserRole)
	if hasRole(requesterRoles, application.UserRoleManager) && (len(roles) != 1 || roles[0] != application.UserRoleAttendant) {
		return problem(c, http.StatusForbidden, "Managers can only create attendants")
	}
	user, err := h.service.Register(c.Request().Context(), req.Name, req.Email, req.Password, roles)
	if err != nil {
		return h.registrationError(c, err)
	}
	return c.JSON(http.StatusCreated, user)
}
func (h *Handler) listUsers(c *echo.Context) error {
	params, err := pagination.Parse(c.QueryParam("page"), c.QueryParam("per_page"))
	if err != nil {
		return problem(c, http.StatusBadRequest, "Invalid pagination")
	}
	page, err := h.service.ListUsers(c.Request().Context(), params)
	if err != nil {
		return problem(c, http.StatusInternalServerError, "Could not list users")
	}
	return c.JSON(http.StatusOK, pagination.NewResponse(page, c.Request().URL.String()))
}

func (h *Handler) getUser(c *echo.Context) error {
	user, err := h.service.GetUser(c.Request().Context(), c.Param("id"))
	if errors.Is(err, application.ErrUserNotFound) {
		return problem(c, http.StatusNotFound, "User not found")
	}
	if err != nil {
		return problem(c, http.StatusInternalServerError, "Could not retrieve user")
	}
	return c.JSON(http.StatusOK, user)
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
func (h *Handler) updateRoles(c *echo.Context) error {
	var req rolesRequest
	if err := decode(c, &req); err != nil {
		return problem(c, 400, "Invalid request")
	}
	requestingUserID, _ := c.Get("user_id").(string)
	user, err := h.service.SetRoles(c.Request().Context(), c.Param("id"), requestingUserID, req.Roles)
	switch {
	case errors.Is(err, application.ErrUserNotFound):
		return problem(c, http.StatusNotFound, "User not found")
	case errors.Is(err, application.ErrCannotChangeOwnRole):
		return problem(c, http.StatusConflict, "Cannot change your own role")
	case err != nil:
		if errors.Is(err, application.ErrInvalidRole) {
			return problem(c, http.StatusBadRequest, "Invalid user role")
		}
		return problem(c, http.StatusInternalServerError, "Could not update user role")
	default:
		return c.JSON(http.StatusOK, user)
	}
}

func (h *Handler) grantPermission(c *echo.Context) error {
	permission := application.Permission(c.Param("permission"))
	if err := h.service.GrantPermission(c.Request().Context(), c.Param("id"), permission); err != nil {
		return permissionError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) revokePermission(c *echo.Context) error {
	permission := application.Permission(c.Param("permission"))
	if err := h.service.RevokePermission(c.Request().Context(), c.Param("id"), permission); err != nil {
		return permissionError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func permissionError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, application.ErrInvalidPermission):
		return problem(c, http.StatusBadRequest, "Invalid permission")
	case errors.Is(err, application.ErrUserNotFound):
		return problem(c, http.StatusNotFound, "User not found")
	default:
		return problem(c, http.StatusInternalServerError, "Could not update user permissions")
	}
}

func requestedRoles(role application.UserRole, roles []application.UserRole) ([]application.UserRole, error) {
	if role != "" {
		if len(roles) != 0 {
			return nil, application.ErrInvalidRole
		}
		return []application.UserRole{role}, nil
	}
	return roles, nil
}

func hasRole(roles []application.UserRole, role application.UserRole) bool {
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}
	return false
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
			roles, permissions, active := h.service.ActivePermissions(c.Request().Context(), id)
			if !active {
				return problem(c, 401, "Invalid access token")
			}
			c.Set("user_id", id)
			c.Set("roles", roles)
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
