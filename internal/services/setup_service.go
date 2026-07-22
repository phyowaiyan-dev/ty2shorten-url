package services

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	// ErrSetupCompleted is returned when initial setup has already completed.
	ErrSetupCompleted = errors.New("setup has already been completed")
	// ErrInvalidSetupForm is returned when setup input fails validation.
	ErrInvalidSetupForm = errors.New("invalid setup form")
)

// SetupForm contains submitted first-run setup values.
type SetupForm struct {
	SiteName        string
	SiteDescription string
	AdminName       string
	AdminEmail      string
	AdminPassword   string
	PasswordConfirm string
	AndroidURL      string
	AppleURL        string
	DefaultURL      string
	PublicBaseURL   string
	SupportEmail    string
}

// SetupValidationError contains friendly field errors for the setup form.
type SetupValidationError struct {
	FieldErrors map[string]string
}

func (e SetupValidationError) Error() string {
	return ErrInvalidSetupForm.Error()
}

// SetupService owns first-run setup rules and persistence.
type SetupService struct {
	db             *gorm.DB
	admins         *repositories.AdminRepository
	settings       *repositories.SettingsRepository
	isProduction   bool
	defaultBaseURL string
}

// NewSetupService constructs a SetupService.
func NewSetupService(db *gorm.DB, admins *repositories.AdminRepository, settings *repositories.SettingsRepository, isProduction bool, defaultBaseURL string) *SetupService {
	return &SetupService{
		db:             db,
		admins:         admins,
		settings:       settings,
		isProduction:   isProduction,
		defaultBaseURL: defaultBaseURL,
	}
}

// IsCompleted reports whether first-run setup has completed.
func (s *SetupService) IsCompleted() (bool, error) {
	return s.settings.IsSetupCompleted()
}

// Complete validates and stores the initial administrator and settings.
func (s *SetupService) Complete(form SetupForm) error {
	normalized, validationErr := s.validate(form)
	if validationErr != nil {
		return validationErr
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(form.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		settings, err := s.settings.CurrentWithDB(tx)
		if err != nil {
			return err
		}
		if settings != nil && settings.IsSetupCompleted {
			return ErrSetupCompleted
		}

		adminCount, err := s.admins.CountWithDB(tx)
		if err != nil {
			return err
		}
		if adminCount > 0 {
			return ErrSetupCompleted
		}

		admin := &models.AdminUser{
			Name:         normalized.AdminName,
			Email:        normalized.AdminEmail,
			PasswordHash: string(passwordHash),
		}
		if err := s.admins.CreateWithDB(tx, admin); err != nil {
			return err
		}

		appSettings := &models.AppSetting{
			SiteName:         normalized.SiteName,
			SiteDescription:  normalized.SiteDescription,
			AndroidURL:       normalized.AndroidURL,
			AppleURL:         normalized.AppleURL,
			DefaultURL:       normalized.DefaultURL,
			PublicBaseURL:    normalized.PublicBaseURL,
			SupportEmail:     normalized.SupportEmail,
			IsSetupCompleted: true,
		}
		if err := s.settings.CreateWithDB(tx, appSettings); err != nil {
			return err
		}

		return nil
	})
}

func (s *SetupService) validate(form SetupForm) (SetupForm, error) {
	normalized := SetupForm{
		SiteName:        strings.TrimSpace(form.SiteName),
		SiteDescription: strings.TrimSpace(form.SiteDescription),
		AdminName:       strings.TrimSpace(form.AdminName),
		AdminEmail:      normalizeEmail(form.AdminEmail),
		AdminPassword:   form.AdminPassword,
		PasswordConfirm: form.PasswordConfirm,
		AndroidURL:      strings.TrimSpace(form.AndroidURL),
		AppleURL:        strings.TrimSpace(form.AppleURL),
		DefaultURL:      strings.TrimSpace(form.DefaultURL),
		PublicBaseURL:   strings.TrimSpace(form.PublicBaseURL),
		SupportEmail:    normalizeEmail(form.SupportEmail),
	}
	if normalized.PublicBaseURL == "" {
		normalized.PublicBaseURL = s.defaultBaseURL
	}

	fieldErrors := map[string]string{}
	requireText(fieldErrors, "site_name", normalized.SiteName, "Enter a site name.")
	requireText(fieldErrors, "site_description", normalized.SiteDescription, "Enter a site description.")
	requireText(fieldErrors, "admin_name", normalized.AdminName, "Enter the administrator name.")
	requireText(fieldErrors, "admin_email", normalized.AdminEmail, "Enter the administrator email.")

	if normalized.AdminEmail != "" && !validEmail(normalized.AdminEmail) {
		fieldErrors["admin_email"] = "Enter a valid administrator email."
	}
	if len(normalized.AdminPassword) < 10 {
		fieldErrors["admin_password"] = "Use a password with at least 10 characters."
	}
	if normalized.AdminPassword != normalized.PasswordConfirm {
		fieldErrors["password_confirm"] = "Passwords must match."
	}
	validateRequiredURL(fieldErrors, "android_url", normalized.AndroidURL, s.isProduction)
	validateRequiredURL(fieldErrors, "apple_url", normalized.AppleURL, s.isProduction)
	validateOptionalURL(fieldErrors, "default_url", normalized.DefaultURL, s.isProduction)
	validateRequiredURL(fieldErrors, "public_base_url", normalized.PublicBaseURL, s.isProduction)
	if normalized.SupportEmail != "" && !validEmail(normalized.SupportEmail) {
		fieldErrors["support_email"] = "Enter a valid support email."
	}

	if len(fieldErrors) > 0 {
		return normalized, SetupValidationError{FieldErrors: fieldErrors}
	}

	return normalized, nil
}

func requireText(fieldErrors map[string]string, field, value, message string) {
	if value == "" {
		fieldErrors[field] = message
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && strings.EqualFold(address.Address, email)
}

func validateRequiredURL(fieldErrors map[string]string, field, value string, requireHTTPS bool) {
	if value == "" {
		fieldErrors[field] = "Enter a destination URL."
		return
	}
	validateOptionalURL(fieldErrors, field, value, requireHTTPS)
}

func validateOptionalURL(fieldErrors map[string]string, field, value string, requireHTTPS bool) {
	if value == "" {
		return
	}

	err := validation.ValidateHTTPURL(value, requireHTTPS)
	switch {
	case errors.Is(err, validation.ErrInvalidURL):
		fieldErrors[field] = "Enter a valid URL."
	case errors.Is(err, validation.ErrUnsafeURLScheme):
		fieldErrors[field] = "Only http and https URLs are allowed."
	case errors.Is(err, validation.ErrHTTPSRequired):
		fieldErrors[field] = "Use an HTTPS URL in production."
	}
}
