package identityhttp

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

var errStoreFailure = errors.New("store failure")

type fakeStore struct {
	users       map[string]application.User
	fail        map[string]error
	resetHashes []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]application.User{}, fail: map[string]error{}}
}

func (s *fakeStore) failure(op, id string) error {
	if err := s.fail[op+":"+id]; err != nil {
		return err
	}
	return s.fail[op]
}

func (s *fakeStore) lookup(op, id string) (application.User, error) {
	if err := s.failure(op, id); err != nil {
		return application.User{}, err
	}
	user, ok := s.users[id]
	if !ok {
		return application.User{}, application.ErrUserNotFound
	}
	return user, nil
}

func (s *fakeStore) CreateFirst(ctx context.Context, user application.User) error {
	return s.Create(ctx, user)
}
func (s *fakeStore) Create(_ context.Context, user application.User) error {
	if err := s.failure("Create", user.ID); err != nil {
		return err
	}
	s.users[user.ID] = user
	return nil
}
func (s *fakeStore) FindByEmail(_ context.Context, email string) (application.User, error) {
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return application.User{}, application.ErrUserNotFound
}
func (s *fakeStore) FindByID(_ context.Context, id string) (application.User, error) {
	return s.lookup("FindByID", id)
}
func (s *fakeStore) List(_ context.Context, params pagination.Params) (pagination.Page[application.User], error) {
	if err := s.failure("List", ""); err != nil {
		return pagination.Page[application.User]{}, err
	}
	users := make([]application.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return pagination.NewPage(users, params, int64(len(users))), nil
}
func (s *fakeStore) UpdateStatus(_ context.Context, id string, status application.UserStatus) error {
	user, err := s.lookup("UpdateStatus", id)
	if err != nil {
		return err
	}
	user.Status = status
	s.users[id] = user
	return nil
}
func (s *fakeStore) UpdateRoles(_ context.Context, id string, roles []application.UserRole) error {
	user, err := s.lookup("UpdateRoles", id)
	if err != nil {
		return err
	}
	user.Roles = roles
	s.users[id] = user
	return nil
}
func (s *fakeStore) Delete(_ context.Context, id string) error {
	user, err := s.lookup("Delete", id)
	if err != nil {
		return err
	}
	now := time.Now()
	user.Status, user.DeletedAt = application.UserStatusDeleted, &now
	s.users[id] = user
	return nil
}
func (s *fakeStore) GrantPermissionToUser(_ context.Context, id string, permission application.Permission) error {
	user, err := s.lookup("GrantPermission", id)
	if err != nil {
		return err
	}
	user.Permissions = append(user.Permissions, permission)
	s.users[id] = user
	return nil
}
func (s *fakeStore) RevokePermissionFromUser(_ context.Context, id string, permission application.Permission) error {
	user, err := s.lookup("RevokePermission", id)
	if err != nil {
		return err
	}
	kept := make([]application.Permission, 0, len(user.Permissions))
	for _, current := range user.Permissions {
		if current != permission {
			kept = append(kept, current)
		}
	}
	user.Permissions = kept
	s.users[id] = user
	return nil
}
func (s *fakeStore) CreateReset(_ context.Context, _, hash string, _ time.Time) error {
	s.resetHashes = append(s.resetHashes, hash)
	return s.failure("CreateReset", "")
}
func (s *fakeStore) ResetPassword(context.Context, string, string, time.Time) error {
	return s.failure("ResetPassword", "")
}

type sequentialIDs struct{ next int }

func (g *sequentialIDs) NewID(context.Context) (string, error) {
	g.next++
	return "new-user-" + strconv.Itoa(g.next), nil
}

type fakeMailer struct {
	to, body string
	err      error
}

func (m *fakeMailer) Send(_ context.Context, to, _, body string) error {
	m.to, m.body = to, body
	return m.err
}
