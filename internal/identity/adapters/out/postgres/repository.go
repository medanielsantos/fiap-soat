package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type userRow struct {
	ID                     string     `gorm:"column:id"`
	Name                   string     `gorm:"column:name"`
	Email                  string     `gorm:"column:email"`
	Password               string     `gorm:"column:password"`
	Status                 string     `gorm:"column:status"`
	EmailVerifiedAt        *time.Time `gorm:"column:email_verified_at"`
	RememberToken          *string    `gorm:"column:remember_token"`
	TwoFactorSecret        *string    `gorm:"column:two_factor_secret"`
	TwoFactorRecoveryCodes *string    `gorm:"column:two_factor_recovery_codes"`
	TwoFactorConfirmedAt   *time.Time `gorm:"column:two_factor_confirmed_at"`
	DeletedAt              *time.Time `gorm:"column:deleted_at"`
}

func (userRow) TableName() string { return "users" }

type resetRow struct {
	TokenHash string     `gorm:"column:token_hash"`
	UserID    string     `gorm:"column:user_id"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
}

func (resetRow) TableName() string { return "password_reset_tokens" }

type userRoleRow struct {
	UserID string `gorm:"column:user_id"`
	RoleID string `gorm:"column:role_id"`
}

func (userRoleRow) TableName() string { return "user_roles" }

type userPermissionRow struct {
	UserID       string `gorm:"column:user_id"`
	PermissionID string `gorm:"column:permission_id"`
}

func (userPermissionRow) TableName() string { return "user_permissions" }

func (r *Repository) SeedAuthorization(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles := []struct{ ID, Name, Description string }{
			{"admin", "Admin", "Full access to administrative operations."},
			{"manager", "Manager", "Manages operations without deleting users or assigning roles."},
			{"attendant", "Attendant", "Handles day-to-day work without deleting products or changing prices."},
		}
		for _, role := range roles {
			if err := tx.Exec("INSERT INTO roles (id, name, description) VALUES (?, ?, ?) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW()", role.ID, role.Name, role.Description).Error; err != nil {
				return err
			}
		}
		permissions := []struct{ ID, Name, Description string }{
			{"users.create", "Create users", "Create administrative users."},
			{"users.read", "Read users", "List and view administrative users."},
			{"users.status.update", "Update user status", "Change administrative user status."},
			{"users.role.assign", "Assign roles", "Assign administrative roles to users."},
			{"users.permissions.assign", "Assign direct permissions", "Grant or revoke direct user permissions."},
			{"users.delete", "Delete users", "Soft-delete administrative users."},
			{"catalog.products.create", "Create catalog products", "Create services and parts."},
			{"catalog.products.delete", "Delete catalog products", "Delete services and parts."},
			{"catalog.products.price.update", "Update catalog prices", "Change service and part prices."},
		}
		for _, permission := range permissions {
			if err := tx.Exec("INSERT INTO permissions (id, name, description) VALUES (?, ?, ?) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description", permission.ID, permission.Name, permission.Description).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("DELETE FROM role_permissions").Error; err != nil {
			return err
		}
		grants := map[string][]string{
			"admin":     {"users.create", "users.read", "users.status.update", "users.role.assign", "users.permissions.assign", "users.delete", "catalog.products.create", "catalog.products.delete", "catalog.products.price.update"},
			"manager":   {"users.create", "users.read", "users.status.update", "catalog.products.create", "catalog.products.delete", "catalog.products.price.update"},
			"attendant": {},
		}
		for roleID, permissionIDs := range grants {
			for _, permissionID := range permissionIDs {
				if err := tx.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", roleID, permissionID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *Repository) CreateFirst(ctx context.Context, user application.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(84211977)").Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&userRow{}).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return application.ErrRegistrationClosed
		}
		return createUser(tx, user)
	})
}
func (r *Repository) Create(ctx context.Context, user application.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return createUser(tx, user)
	})
}

func createUser(tx *gorm.DB, user application.User) error {
	if err := tx.Create(&userRow{ID: user.ID, Name: user.Name, Email: user.Email, Password: user.Password, Status: string(user.Status)}).Error; err != nil {
		return err
	}
	for _, role := range user.Roles {
		if err := tx.Create(&userRoleRow{UserID: user.ID, RoleID: string(role)}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (application.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL AND status <> ?", email, string(application.UserStatusDeleted)).First(&row).Error
	if err != nil {
		return application.User{}, err
	}
	user := toUser(row)
	err = r.loadAuthorization(ctx, &user)
	return user, err
}
func (r *Repository) FindByID(ctx context.Context, id string) (application.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return application.User{}, application.ErrUserNotFound
		}
		return application.User{}, err
	}
	user := toUser(row)
	err = r.loadAuthorization(ctx, &user)
	return user, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) (pagination.Page[application.User], error) {
	query := r.db.WithContext(ctx).Model(&userRow{}).Where("deleted_at IS NULL")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return pagination.Page[application.User]{}, err
	}
	var rows []userRow
	if err := query.Order("created_at DESC").Offset(params.Offset()).Limit(params.Limit()).Find(&rows).Error; err != nil {
		return pagination.Page[application.User]{}, err
	}
	users := make([]application.User, 0, len(rows))
	for _, row := range rows {
		user := toUser(row)
		if err := r.loadAuthorization(ctx, &user); err != nil {
			return pagination.Page[application.User]{}, err
		}
		users = append(users, user)
	}
	return pagination.NewPage(users, params, total), nil
}

func (r *Repository) loadAuthorization(ctx context.Context, user *application.User) error {
	var roleIDs []string
	if err := r.db.WithContext(ctx).Table("user_roles").Where("user_id = ?", user.ID).Order("role_id").Pluck("role_id", &roleIDs).Error; err != nil {
		return err
	}
	user.Roles = make([]application.UserRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		user.Roles = append(user.Roles, application.UserRole(roleID))
	}
	var rows []struct {
		PermissionID string `gorm:"column:permission_id"`
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT rp.permission_id
		FROM role_permissions rp
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ?
		UNION
		SELECT up.permission_id
		FROM user_permissions up
		WHERE up.user_id = ?
		ORDER BY permission_id`, user.ID, user.ID).Scan(&rows).Error; err != nil {
		return err
	}
	user.Permissions = make([]application.Permission, 0, len(rows))
	for _, row := range rows {
		user.Permissions = append(user.Permissions, application.Permission(row.PermissionID))
	}
	return nil
}

func (r *Repository) GrantPermissionToUser(ctx context.Context, userID string, permission application.Permission) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&userRow{}).Where("id = ? AND deleted_at IS NULL", userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return application.ErrUserNotFound
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&userPermissionRow{UserID: userID, PermissionID: string(permission)}).Error
	})
}

func (r *Repository) RevokePermissionFromUser(ctx context.Context, userID string, permission application.Permission) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted_at IS NULL", userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return application.ErrUserNotFound
	}
	return r.db.WithContext(ctx).Where("user_id = ? AND permission_id = ?", userID, string(permission)).Delete(&userPermissionRow{}).Error
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status application.UserStatus) error {
	deletedAt := any(nil)
	if status == application.UserStatusDeleted {
		deletedAt = time.Now()
	}
	result := r.db.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{"status": string(status), "updated_at": time.Now(), "deleted_at": deletedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return application.ErrUserNotFound
	}
	return nil
}
func (r *Repository) UpdateRoles(ctx context.Context, id string, roles []application.UserRole) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&userRow{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"updated_at": time.Now()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return application.ErrUserNotFound
		}
		if err := tx.Where("user_id = ?", id).Delete(&userRoleRow{}).Error; err != nil {
			return err
		}
		for _, role := range roles {
			if err := tx.Create(&userRoleRow{UserID: id, RoleID: string(role)}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"status": string(application.UserStatusDeleted), "deleted_at": now, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return application.ErrUserNotFound
	}
	return nil
}
func (r *Repository) CreateReset(ctx context.Context, userID, hash string, expires time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND used_at IS NULL", userID).Delete(&resetRow{}).Error; err != nil {
			return err
		}
		return tx.Create(&resetRow{TokenHash: hash, UserID: userID, ExpiresAt: expires}).Error
	})
}
func (r *Repository) ResetPassword(ctx context.Context, tokenHash, passwordHash string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row resetRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return application.ErrInvalidResetToken
			}
			return err
		}
		result := tx.Model(&userRow{}).Where("id = ? AND status = ? AND deleted_at IS NULL", row.UserID, string(application.UserStatusActive)).Update("password", passwordHash)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return application.ErrInvalidResetToken
		}
		return tx.Model(&resetRow{}).Where("token_hash = ? AND used_at IS NULL", tokenHash).Update("used_at", now).Error
	})
}
func toUser(row userRow) application.User {
	return application.User{ID: row.ID, Name: row.Name, Email: row.Email, Password: row.Password, Status: application.UserStatus(row.Status), DeletedAt: row.DeletedAt}
}

func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}
