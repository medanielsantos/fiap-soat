package application

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"golang.org/x/crypto/bcrypt"
)

type memoryStore struct {
	users  map[string]User
	resets map[string]memoryReset
}

type memoryReset struct {
	userID  string
	expires time.Time
	used    bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{users: map[string]User{}, resets: map[string]memoryReset{}}
}
func (s *memoryStore) CreateFirst(_ context.Context, user User) error {
	if len(s.users) != 0 {
		return ErrRegistrationClosed
	}
	return s.Create(context.Background(), user)
}
func (s *memoryStore) Create(_ context.Context, user User) error {
	for _, existing := range s.users {
		if existing.Email == user.Email {
			return ErrEmailTaken
		}
	}
	s.users[user.ID] = user
	return nil
}
func (s *memoryStore) FindByEmail(_ context.Context, email string) (User, error) {
	for _, user := range s.users {
		if user.Email == email && user.DeletedAt == nil {
			return user, nil
		}
	}
	return User{}, errors.New("not found")
}
func (s *memoryStore) FindByID(_ context.Context, id string) (User, error) {
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}
func (s *memoryStore) List(_ context.Context, params pagination.Params) (pagination.Page[User], error) {
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return pagination.NewPage(users, params, int64(len(users))), nil
}
func (s *memoryStore) UpdateStatus(_ context.Context, id string, status UserStatus) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	user.Status = status
	if status == UserStatusDeleted {
		now := time.Now()
		user.DeletedAt = &now
	} else {
		user.DeletedAt = nil
	}
	s.users[id] = user
	return nil
}
func (s *memoryStore) UpdateRoles(_ context.Context, id string, roles []UserRole) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	user.Roles = roles
	s.users[id] = user
	return nil
}
func (s *memoryStore) Delete(_ context.Context, id string) error {
	return s.UpdateStatus(context.Background(), id, UserStatusDeleted)
}
func (s *memoryStore) GrantPermissionToUser(_ context.Context, id string, permission Permission) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	user.Permissions = append(user.Permissions, permission)
	s.users[id] = user
	return nil
}
func (s *memoryStore) RevokePermissionFromUser(_ context.Context, id string, permission Permission) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	filtered := user.Permissions[:0]
	for _, current := range user.Permissions {
		if current != permission {
			filtered = append(filtered, current)
		}
	}
	user.Permissions = filtered
	s.users[id] = user
	return nil
}
func (s *memoryStore) CreateReset(_ context.Context, userID, tokenHash string, expires time.Time) error {
	for hash, reset := range s.resets {
		if reset.userID == userID && !reset.used {
			delete(s.resets, hash)
		}
	}
	s.resets[tokenHash] = memoryReset{userID: userID, expires: expires}
	return nil
}
func (s *memoryStore) ResetPassword(_ context.Context, tokenHash, passwordHash string, now time.Time) error {
	reset, ok := s.resets[tokenHash]
	if !ok || reset.used || !reset.expires.After(now) {
		return ErrInvalidResetToken
	}
	user := s.users[reset.userID]
	if user.Status != UserStatusActive {
		return ErrInvalidResetToken
	}
	user.Password = passwordHash
	s.users[user.ID] = user
	reset.used = true
	s.resets[tokenHash] = reset
	return nil
}

type fixedIDs struct{ id string }

func (g fixedIDs) NewID(context.Context) (string, error) { return g.id, nil }

type fixedTokens struct{}

func (fixedTokens) Issue(id string) (string, time.Time, error) {
	return "access-" + id, time.Now().Add(time.Hour), nil
}

type captureMailer struct{ body string }

func (m *captureMailer) Send(_ context.Context, _, _, body string) error { m.body = body; return nil }

func TestRegisterAndLoginWithMultipleRoles(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, fixedIDs{"user-1"}, fixedTokens{}, nil, "")
	created, err := service.Register(context.Background(), "Admin", "ADMIN@example.com", "correct horse battery", []UserRole{UserRoleAdmin, UserRoleManager})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Roles) != 2 || created.Email != "admin@example.com" {
		t.Fatalf("created user = %+v", created)
	}
	user := store.users[created.ID]
	if user.Password == "correct horse battery" || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("correct horse battery")) != nil {
		t.Fatal("password was not stored as a valid bcrypt hash")
	}
	session, err := service.Login(context.Background(), created.Email, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if session.AccessToken != "access-user-1" {
		t.Fatalf("access token = %q", session.AccessToken)
	}
	store.users[created.ID] = User{ID: created.ID, Email: user.Email, Password: user.Password, Status: UserStatusSuspended, Roles: user.Roles}
	if _, err := service.Login(context.Background(), created.Email, "correct horse battery"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login for suspended user error = %v", err)
	}
}

func TestSeedAdminCreatesOnlyTheFirstUser(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, fixedIDs{"seed-admin"}, fixedTokens{}, nil, "")
	created, err := service.SeedAdmin(context.Background(), "Seed Admin", "admin@example.test", "initial secure password")
	if err != nil || !created {
		t.Fatalf("first seed = (%v, %v)", created, err)
	}
	user := store.users["seed-admin"]
	if len(user.Roles) != 1 || user.Roles[0] != UserRoleAdmin {
		t.Fatalf("seed roles = %v", user.Roles)
	}
	created, err = service.SeedAdmin(context.Background(), "Other Admin", "other@example.test", "another secure password")
	if err != nil || created {
		t.Fatalf("repeated seed = (%v, %v), want no-op", created, err)
	}
	if len(store.users) != 1 {
		t.Fatalf("users after repeated seed = %d, want 1", len(store.users))
	}
}

func TestDirectPermissionsAreAppliedAndRevoked(t *testing.T) {
	store := newMemoryStore()
	store.users["attendant"] = User{ID: "attendant", Status: UserStatusActive, Roles: []UserRole{UserRoleAttendant}}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	if err := service.GrantPermission(context.Background(), "attendant", PermissionProductsCreate); err != nil {
		t.Fatal(err)
	}
	roles, permissions, active := service.ActivePermissions(context.Background(), "attendant")
	if !active || len(roles) != 1 || len(permissions) != 1 || permissions[0] != PermissionProductsCreate {
		t.Fatalf("authorization = (%v, %v, %v)", roles, permissions, active)
	}
	if err := service.RevokePermission(context.Background(), "attendant", PermissionProductsCreate); err != nil {
		t.Fatal(err)
	}
	_, permissions, _ = service.ActivePermissions(context.Background(), "attendant")
	if len(permissions) != 0 {
		t.Fatalf("permissions after revoke = %v", permissions)
	}
}

func TestPasswordResetExpiresAndCanOnlyBeUsedOnce(t *testing.T) {
	store := newMemoryStore()
	user := User{ID: "user-1", Name: "Admin", Email: "admin@example.test", Password: "old-hash", Status: UserStatusActive, Roles: []UserRole{UserRoleAdmin}}
	store.users[user.ID] = user
	mailer := &captureMailer{}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, mailer, "https://example.test/reset")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	service.now = func() time.Time { return now }
	if err := service.RequestPasswordReset(context.Background(), user.Email); err != nil {
		t.Fatal(err)
	}
	link, err := url.Parse(mailer.body)
	if err != nil {
		t.Fatal(err)
	}
	token := link.Query().Get("token")
	if token == "" {
		t.Fatal("reset email did not contain a token")
	}
	if err := service.ResetPassword(context.Background(), token, "replacement secure password"); err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(context.Background(), token, "another secure password"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("reused token error = %v", err)
	}
	if err := service.RequestPasswordReset(context.Background(), user.Email); err != nil {
		t.Fatal(err)
	}
	link, err = url.Parse(mailer.body)
	if err != nil {
		t.Fatal(err)
	}
	token = link.Query().Get("token")
	now = now.Add(31 * time.Minute)
	if err := service.ResetPassword(context.Background(), token, "expired secure password"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("expired token error = %v", err)
	}
}

func TestRoleAndPermissionValidation(t *testing.T) {
	if validRoles([]UserRole{UserRoleAdmin, UserRoleAdmin}) {
		t.Fatal("duplicate roles were accepted")
	}
	if validRoles([]UserRole{"unknown"}) {
		t.Fatal("unknown role was accepted")
	}
	if validPermission(Permission("system.root")) {
		t.Fatal("unknown permission was accepted")
	}
	store := newMemoryStore()
	store.users["attendant"] = User{ID: "attendant", Status: UserStatusActive, Roles: []UserRole{UserRoleAttendant}}
	service := NewService(store, fixedIDs{"unused"}, fixedTokens{}, nil, "")
	if err := service.GrantPermission(context.Background(), "attendant", PermissionProductsDelete); !errors.Is(err, ErrInvalidPermission) {
		t.Fatalf("attendant product-delete grant error = %v", err)
	}
}
