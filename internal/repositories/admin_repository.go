package repositories

import (
	"errors"
	"fmt"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"gorm.io/gorm"
)

// AdminRepository reads and writes administrator accounts.
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository constructs an AdminRepository.
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// Create stores an administrator.
func (r *AdminRepository) Create(admin *models.AdminUser) error {
	return r.CreateWithDB(r.db, admin)
}

// CreateWithDB stores an administrator with the supplied GORM handle.
func (r *AdminRepository) CreateWithDB(db *gorm.DB, admin *models.AdminUser) error {
	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	return nil
}

// FindByEmail returns an administrator by normalized email.
func (r *AdminRepository) FindByEmail(email string) (*models.AdminUser, error) {
	var admin models.AdminUser
	if err := r.db.Where("email = ?", email).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find admin by email: %w", err)
	}

	return &admin, nil
}

// FindByID returns an administrator by ID.
func (r *AdminRepository) FindByID(id uint) (*models.AdminUser, error) {
	var admin models.AdminUser
	if err := r.db.First(&admin, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find admin by id: %w", err)
	}

	return &admin, nil
}

// Count returns the number of administrator accounts.
func (r *AdminRepository) CountWithDB(db *gorm.DB) (int64, error) {
	var count int64
	if err := db.Model(&models.AdminUser{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count admin users: %w", err)
	}

	return count, nil
}

// UpdatePasswordHash stores a new password hash for an administrator.
func (r *AdminRepository) UpdatePasswordHash(id uint, passwordHash string) error {
	if err := r.db.Model(&models.AdminUser{}).
		Where("id = ?", id).
		Update("password_hash", passwordHash).
		Error; err != nil {
		return fmt.Errorf("update admin password hash: %w", err)
	}

	return nil
}

// UpdateProfile stores editable administrator profile fields.
func (r *AdminRepository) UpdateProfile(admin *models.AdminUser) error {
	if err := r.db.Model(admin).Select("name", "email", "avatar_url").Updates(admin).Error; err != nil {
		return fmt.Errorf("update admin profile: %w", err)
	}

	return nil
}
