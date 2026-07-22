package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"gorm.io/gorm"
)

// AuditRepository stores and reads audit log entries.
type AuditRepository struct {
	db *gorm.DB
}

// AuditListFilter constrains audit log list queries.
type AuditListFilter struct {
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

// NewAuditRepository constructs an AuditRepository.
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create appends an audit log entry.
func (r *AuditRepository) Create(log *models.AuditLog) error {
	if err := r.db.Create(log).Error; err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// List returns recent audit log entries.
func (r *AuditRepository) List(limit int) ([]models.AuditLog, error) {
	return r.Search(AuditListFilter{Limit: limit})
}

// Search returns recent audit log entries matching safe filter values.
func (r *AuditRepository) Search(filter AuditListFilter) ([]models.AuditLog, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := r.db.Model(&models.AuditLog{})
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at < ?", *filter.CreatedTo)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	var logs []models.AuditLog
	if err := query.Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, nil
}

// FindByID returns one audit log entry.
func (r *AuditRepository) FindByID(id uint) (*models.AuditLog, error) {
	var log models.AuditLog
	if err := r.db.First(&log, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find audit log: %w", err)
	}
	return &log, nil
}
