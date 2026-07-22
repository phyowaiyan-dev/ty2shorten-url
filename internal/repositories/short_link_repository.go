package repositories

import (
	"errors"
	"fmt"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"gorm.io/gorm"
)

var ErrDuplicateSlug = errors.New("short link slug already exists")

// ShortLinkRepository reads and writes custom short links.
type ShortLinkRepository struct {
	db *gorm.DB
}

// NewShortLinkRepository constructs a ShortLinkRepository.
func NewShortLinkRepository(db *gorm.DB) *ShortLinkRepository {
	return &ShortLinkRepository{db: db}
}

// FindActiveBySlug returns an active short link by slug.
func (r *ShortLinkRepository) FindActiveBySlug(slug string) (*models.ShortLink, error) {
	var link models.ShortLink
	if err := r.db.Where("slug = ? AND is_active = ?", slug, true).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find active short link: %w", err)
	}

	return &link, nil
}

// List returns short links ordered by newest first.
func (r *ShortLinkRepository) List() ([]models.ShortLink, error) {
	var links []models.ShortLink
	if err := r.db.Order("id DESC").Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list short links: %w", err)
	}
	return links, nil
}

// FindByID returns a short link by ID.
func (r *ShortLinkRepository) FindByID(id uint) (*models.ShortLink, error) {
	var link models.ShortLink
	if err := r.db.First(&link, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find short link by id: %w", err)
	}
	return &link, nil
}

// FindBySlug returns a short link by slug.
func (r *ShortLinkRepository) FindBySlug(slug string) (*models.ShortLink, error) {
	var link models.ShortLink
	if err := r.db.Where("slug = ?", slug).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find short link by slug: %w", err)
	}
	return &link, nil
}

// Create stores a short link.
func (r *ShortLinkRepository) Create(link *models.ShortLink) error {
	if err := r.db.Create(link).Error; err != nil {
		return fmt.Errorf("create short link: %w", err)
	}
	return nil
}

// Update stores editable short link fields.
func (r *ShortLinkRepository) Update(link *models.ShortLink) error {
	if err := r.db.Model(link).Select("title", "slug", "destination", "is_active").Updates(link).Error; err != nil {
		return fmt.Errorf("update short link: %w", err)
	}
	return nil
}

// Delete removes a short link.
func (r *ShortLinkRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.ShortLink{}, id).Error; err != nil {
		return fmt.Errorf("delete short link: %w", err)
	}
	return nil
}

// IncrementClickCount increments a link click counter.
func (r *ShortLinkRepository) IncrementClickCount(id uint) error {
	if err := r.db.Model(&models.ShortLink{}).
		Where("id = ?", id).
		UpdateColumn("click_count", gorm.Expr("click_count + ?", 1)).
		Error; err != nil {
		return fmt.Errorf("increment short link click count: %w", err)
	}

	return nil
}

// ActiveCount returns the number of active links.
func (r *ShortLinkRepository) ActiveCount() (int64, error) {
	var count int64
	if err := r.db.Model(&models.ShortLink{}).Where("is_active = ?", true).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count active short links: %w", err)
	}

	return count, nil
}

// TotalClicks returns the sum of all click counters.
func (r *ShortLinkRepository) TotalClicks() (uint64, error) {
	var total uint64
	if err := r.db.Model(&models.ShortLink{}).Select("COALESCE(SUM(click_count), 0)").Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("sum short link click counts: %w", err)
	}

	return total, nil
}
