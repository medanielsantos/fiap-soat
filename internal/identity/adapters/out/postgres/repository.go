package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/identity/application"
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

func (userRow) TableName() string { return "admin_users" }

type resetRow struct {
	TokenHash string     `gorm:"column:token_hash"`
	UserID    string     `gorm:"column:user_id"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
}

func (resetRow) TableName() string { return "password_reset_tokens" }

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
		return tx.Create(&userRow{ID: user.ID, Name: user.Name, Email: user.Email, Password: user.Password, Status: string(user.Status)}).Error
	})
}
func (r *Repository) Create(ctx context.Context, user application.User) error {
	return r.db.WithContext(ctx).Create(&userRow{ID: user.ID, Name: user.Name, Email: user.Email, Password: user.Password, Status: string(user.Status)}).Error
}
func (r *Repository) FindByEmail(ctx context.Context, email string) (application.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL AND status <> ?", email, string(application.UserStatusDeleted)).First(&row).Error
	return toUser(row), err
}
func (r *Repository) FindByID(ctx context.Context, id string) (application.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	return toUser(row), err
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
