package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrRegistrationClosed   = errors.New("initial registration is closed")
	ErrEmailTaken           = errors.New("email already registered")
	ErrInvalidResetToken    = errors.New("invalid password reset token")
	ErrInvalidStatus        = errors.New("invalid user status")
	ErrUserNotFound         = errors.New("user not found")
	ErrCannotDeactivateSelf = errors.New("cannot deactivate the current user")
	ErrCannotDeleteSelf     = errors.New("cannot delete the current user")
	ErrCannotChangeOwnRole  = errors.New("cannot change the current user's role")
	ErrInvalidRole          = errors.New("invalid user role")
	ErrInvalidPermission    = errors.New("invalid permission")
)

type UserStatus string
type UserRole string
type Permission string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusPending   UserStatus = "pending"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
	UserStatusDeleted   UserStatus = "deleted"
)

const (
	PermissionUsersCreate           Permission = "users.create"
	PermissionUsersStatusUpdate     Permission = "users.status.update"
	PermissionUsersRoleAssign       Permission = "users.role.assign"
	PermissionUsersPermissionAssign Permission = "users.permissions.assign"
	PermissionUsersDelete           Permission = "users.delete"
	PermissionUsersRead             Permission = "users.read"
	PermissionProductsCreate        Permission = "catalog.products.create"
	PermissionProductsDelete        Permission = "catalog.products.delete"
	PermissionProductsPriceUpdate   Permission = "catalog.products.price.update"
)

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleManager   UserRole = "manager"
	UserRoleAttendant UserRole = "attendant"
)

type User struct {
	ID, Name, Email, Password string
	Status                    UserStatus
	Roles                     []UserRole
	Permissions               []Permission
	DeletedAt                 *time.Time
}

type Store interface {
	CreateFirst(context.Context, User) error
	Create(context.Context, User) error
	FindByEmail(context.Context, string) (User, error)
	FindByID(context.Context, string) (User, error)
	List(context.Context, pagination.Params) (pagination.Page[User], error)
	UpdateStatus(context.Context, string, UserStatus) error
	UpdateRoles(context.Context, string, []UserRole) error
	Delete(context.Context, string) error
	GrantPermissionToUser(context.Context, string, Permission) error
	RevokePermissionFromUser(context.Context, string, Permission) error
	CreateReset(context.Context, string, string, time.Time) error
	ResetPassword(context.Context, string, string, time.Time) error
}

type IDGenerator interface {
	NewID(context.Context) (string, error)
}
type TokenIssuer interface {
	Issue(string) (string, time.Time, error)
}
type Mailer interface {
	Send(context.Context, string, string, string) error
}

type Service struct {
	store    Store
	ids      IDGenerator
	tokens   TokenIssuer
	mailer   Mailer
	resetURL string
	now      func() time.Time
}

func NewService(store Store, ids IDGenerator, tokens TokenIssuer, mailer Mailer, resetURL string) *Service {
	return &Service{store: store, ids: ids, tokens: tokens, mailer: mailer, resetURL: strings.TrimRight(resetURL, "/"), now: time.Now}
}

type UserView struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Email  string     `json:"email"`
	Status UserStatus `json:"status"`
	Roles  []UserRole `json:"roles"`
}
type Session struct {
	User        UserView  `json:"user"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (s *Service) Register(ctx context.Context, name, email, password string, roles []UserRole) (UserView, error) {
	if !validRoles(roles) {
		return UserView{}, ErrInvalidRole
	}
	user, err := s.newUser(ctx, name, email, password, roles)
	if err != nil {
		return UserView{}, err
	}
	if err := s.store.Create(ctx, user); err != nil {
		return UserView{}, err
	}
	return view(user), nil
}

func (s *Service) SeedAdmin(ctx context.Context, name, email, password string) (bool, error) {
	user, err := s.newUser(ctx, name, email, password, []UserRole{UserRoleAdmin})
	if err != nil {
		return false, err
	}
	if err := s.store.CreateFirst(ctx, user); errors.Is(err, ErrRegistrationClosed) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) newUser(ctx context.Context, name, email, password string, roles []UserRole) (User, error) {
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	parsedEmail, emailErr := mail.ParseAddress(email)
	if name == "" || len(name) > 120 || emailErr != nil || parsedEmail.Address != email || len(email) > 254 || strings.ContainsAny(email, "\r\n") || len(password) < 12 || len(password) > 72 {
		return User{}, errors.New("invalid user details or password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	id, err := s.ids.NewID(ctx)
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Name: name, Email: email, Password: string(hash), Status: UserStatusActive, Roles: roles}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	user, err := s.store.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || user.Status != UserStatusActive || user.DeletedAt != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return Session{}, ErrInvalidCredentials
	}
	token, expiry, err := s.tokens.Issue(user.ID)
	if err != nil {
		return Session{}, err
	}
	return Session{User: view(user), AccessToken: token, ExpiresAt: expiry}, nil
}

func (s *Service) ActivePermissions(ctx context.Context, id string) ([]UserRole, []Permission, bool) {
	if s.store == nil {
		return nil, nil, false
	}
	user, err := s.store.FindByID(ctx, id)
	if err != nil || user.Status != UserStatusActive || user.DeletedAt != nil {
		return nil, nil, false
	}
	return user.Roles, user.Permissions, true
}

func (s *Service) SetStatus(ctx context.Context, id, requestingUserID string, status UserStatus) (UserView, error) {
	if status != UserStatusActive && status != UserStatusInactive && status != UserStatusPending && status != UserStatusSuspended && status != UserStatusBanned && status != UserStatusDeleted {
		return UserView{}, ErrInvalidStatus
	}
	if id == requestingUserID && status != UserStatusActive {
		return UserView{}, ErrCannotDeactivateSelf
	}
	if err := s.store.UpdateStatus(ctx, id, status); err != nil {
		return UserView{}, err
	}
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	return view(user), nil
}

func (s *Service) SetRoles(ctx context.Context, id, requestingUserID string, roles []UserRole) (UserView, error) {
	if !validRoles(roles) {
		return UserView{}, ErrInvalidRole
	}
	if id == requestingUserID && !hasRole(roles, UserRoleAdmin) {
		return UserView{}, ErrCannotChangeOwnRole
	}
	if err := s.store.UpdateRoles(ctx, id, roles); err != nil {
		return UserView{}, err
	}
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	return view(user), nil
}

func (s *Service) ListUsers(ctx context.Context, params pagination.Params) (pagination.Page[UserView], error) {
	page, err := s.store.List(ctx, params)
	if err != nil {
		return pagination.Page[UserView]{}, err
	}
	views := make([]UserView, 0, len(page.Items))
	for _, user := range page.Items {
		views = append(views, view(user))
	}
	return pagination.NewPage(views, params, page.TotalItems), nil
}

func (s *Service) GetUser(ctx context.Context, id string) (UserView, error) {
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	return view(user), nil
}

func (s *Service) GrantPermission(ctx context.Context, userID string, permission Permission) error {
	if !validPermission(permission) {
		return ErrInvalidPermission
	}
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if hasRole(user.Roles, UserRoleManager) && permission == PermissionUsersDelete {
		return ErrInvalidPermission
	}
	if hasRole(user.Roles, UserRoleAttendant) && (permission == PermissionProductsDelete || permission == PermissionProductsPriceUpdate) {
		return ErrInvalidPermission
	}
	return s.store.GrantPermissionToUser(ctx, userID, permission)
}

func (s *Service) RevokePermission(ctx context.Context, userID string, permission Permission) error {
	if !validPermission(permission) {
		return ErrInvalidPermission
	}
	return s.store.RevokePermissionFromUser(ctx, userID, permission)
}

func (s *Service) DeleteUser(ctx context.Context, id, requestingUserID string) error {
	if id == requestingUserID {
		return ErrCannotDeleteSelf
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	if s.resetURL == "" {
		return errors.New("PASSWORD_RESET_URL is required")
	}
	user, err := s.store.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || user.Status != UserStatusActive || user.DeletedAt != nil {
		return nil
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	plain := base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(plain))
	if err := s.store.CreateReset(ctx, user.ID, hex.EncodeToString(digest[:]), s.now().Add(30*time.Minute)); err != nil {
		return err
	}
	if s.mailer == nil {
		return errors.New("SMTP mail adapter is not configured")
	}
	resetLink, err := url.Parse(s.resetURL)
	if err != nil {
		return errors.New("PASSWORD_RESET_URL is invalid")
	}
	query := resetLink.Query()
	query.Set("token", plain)
	resetLink.RawQuery = query.Encode()
	return s.mailer.Send(ctx, user.Email, "Redefinição de senha", resetLink.String())
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if len(password) < 12 || len(password) > 72 {
		return errors.New("password must contain between 12 and 72 bytes")
	}
	digest := sha256.Sum256([]byte(token))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.ResetPassword(ctx, hex.EncodeToString(digest[:]), string(hash), s.now())
}

func validRoles(roles []UserRole) bool {
	if len(roles) == 0 {
		return false
	}
	seen := make(map[UserRole]struct{}, len(roles))
	for _, role := range roles {
		if role != UserRoleAdmin && role != UserRoleManager && role != UserRoleAttendant {
			return false
		}
		if _, exists := seen[role]; exists {
			return false
		}
		seen[role] = struct{}{}
	}
	return true
}

func hasRole(roles []UserRole, target UserRole) bool {
	for _, role := range roles {
		if role == target {
			return true
		}
	}
	return false
}

func validPermission(permission Permission) bool {
	switch permission {
	case PermissionUsersCreate, PermissionUsersStatusUpdate, PermissionUsersRoleAssign, PermissionUsersPermissionAssign,
		PermissionUsersDelete, PermissionUsersRead, PermissionProductsCreate, PermissionProductsDelete, PermissionProductsPriceUpdate:
		return true
	default:
		return false
	}
}

func view(user User) UserView {
	return UserView{ID: user.ID, Name: user.Name, Email: user.Email, Status: user.Status, Roles: user.Roles}
}
