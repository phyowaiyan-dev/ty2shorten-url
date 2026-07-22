package repositories

import (
	"errors"
	"fmt"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"gorm.io/gorm"
)

// SettingsRepository reads and writes application settings.
type SettingsRepository struct {
	db *gorm.DB
}

// NewSettingsRepository constructs a SettingsRepository.
func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Current returns the newest application settings row.
func (r *SettingsRepository) Current() (*models.AppSetting, error) {
	return r.CurrentWithDB(r.db)
}

// CurrentWithDB returns the newest settings row with the supplied GORM handle.
func (r *SettingsRepository) CurrentWithDB(db *gorm.DB) (*models.AppSetting, error) {
	var settings models.AppSetting
	if err := db.Order("id DESC").First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load app settings: %w", err)
	}

	return &settings, nil
}

// IsSetupCompleted reports whether initial setup has completed.
func (r *SettingsRepository) IsSetupCompleted() (bool, error) {
	settings, err := r.Current()
	if err != nil {
		return false, err
	}
	if settings == nil {
		return false, nil
	}

	return settings.IsSetupCompleted, nil
}

// CreateWithDB stores application settings with the supplied GORM handle.
func (r *SettingsRepository) CreateWithDB(db *gorm.DB, settings *models.AppSetting) error {
	if err := db.Create(settings).Error; err != nil {
		return fmt.Errorf("create app settings: %w", err)
	}

	return nil
}

// Update stores editable settings without changing setup completion.
func (r *SettingsRepository) Update(settings *models.AppSetting) error {
	if err := r.db.Model(settings).Select(
		"site_name",
		"site_description",
		"android_url",
		"apple_url",
		"default_url",
		"public_base_url",
		"support_email",
	).Updates(settings).Error; err != nil {
		return fmt.Errorf("update app settings: %w", err)
	}

	return nil
}

// UpdateSEO stores editable SEO settings without changing setup completion.
func (r *SettingsRepository) UpdateSEO(settings *models.AppSetting) error {
	if err := r.db.Model(settings).Select(
		"seo_enabled",
		"search_engine_indexing_enabled",
		"robots_enabled",
		"sitemap_enabled",
		"default_meta_title",
		"default_meta_description",
		"default_keywords",
		"canonical_base_url",
		"default_robots_directive",
		"google_site_verification",
		"bing_site_verification",
		"structured_data_enabled",
		"organization_name",
		"organization_url",
		"organization_logo_url",
		"open_graph_site_name",
		"open_graph_locale",
		"open_graph_default_image_url",
		"twitter_card_type",
		"twitter_handle",
		"twitter_default_image_url",
	).Updates(settings).Error; err != nil {
		return fmt.Errorf("update SEO settings: %w", err)
	}
	return nil
}

// UpdateBranding stores editable branding and media settings without changing setup completion.
func (r *SettingsRepository) UpdateBranding(settings *models.AppSetting) error {
	if err := r.db.Model(settings).Select(
		"site_tagline",
		"logo_url",
		"dark_logo_url",
		"favicon_url",
		"apple_touch_icon_url",
		"default_social_image_url",
	).Updates(settings).Error; err != nil {
		return fmt.Errorf("update branding settings: %w", err)
	}
	return nil
}

// UpdateFooter stores editable public footer settings without changing setup completion.
func (r *SettingsRepository) UpdateFooter(settings *models.AppSetting) error {
	if err := r.db.Model(settings).Select(
		"footer_enabled",
		"footer_brand_text",
		"footer_description",
		"copyright_text",
		"support_text",
		"footer_address",
	).Updates(settings).Error; err != nil {
		return fmt.Errorf("update footer settings: %w", err)
	}
	return nil
}

// UpdateLegalContent stores editable public legal page content without changing setup completion.
func (r *SettingsRepository) UpdateLegalContent(settings *models.AppSetting) error {
	if err := r.db.Model(settings).Select(
		"privacy_policy_markdown",
		"terms_markdown",
	).Updates(settings).Error; err != nil {
		return fmt.Errorf("update legal content: %w", err)
	}
	return nil
}
