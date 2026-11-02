package application

import (
	"context"
	"errors"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
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
