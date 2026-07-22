package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidPasswordChange = errors.New("invalid password change form")

// ErrInvalidProfileUpdate is returned when account profile input is invalid.
var ErrInvalidProfileUpdate = errors.New("invalid account profile form")

// PasswordForm contains administrator password change input.
type PasswordForm struct {
	CurrentPassword string
	NewPassword     string
	ConfirmPassword string
}

// ProfileForm contains editable administrator profile input.
type ProfileForm struct {
	Name      string
	Email     string
	AvatarURL string
}

// AccountProfile contains display-safe administrator account details.
type AccountProfile struct {
	ID        uint
	Name      string
	Email     string
	AvatarURL string
	Initials  string
	Role      string
	CreatedAt string
	UpdatedAt string
}

// PasswordValidationError contains friendly field errors.
type PasswordValidationError struct {
	FieldErrors map[string]string
}

func (e PasswordValidationError) Error() string {
	return ErrInvalidPasswordChange.Error()
}

// ProfileValidationError contains friendly profile field errors.
type ProfileValidationError struct {
	FieldErrors map[string]string
}

func (e ProfileValidationError) Error() string {
	return ErrInvalidProfileUpdate.Error()
}

// AccountService owns account security changes.
type AccountService struct {
	admins *repositories.AdminRepository
}

// NewAccountService constructs an AccountService.
func NewAccountService(admins *repositories.AdminRepository) *AccountService {
	return &AccountService{admins: admins}
}

// Profile returns the currently authenticated administrator profile.
func (s *AccountService) Profile(adminID uint) (AccountProfile, error) {
	admin, err := s.admins.FindByID(adminID)
	if err != nil {
		return AccountProfile{}, err
	}
	if admin == nil {
		return AccountProfile{}, ErrInvalidCredentials
	}

	return accountProfileFromAdmin(admin), nil
}

// UpdateProfile validates and stores editable administrator profile details.
func (s *AccountService) UpdateProfile(adminID uint, form ProfileForm) error {
	admin, err := s.admins.FindByID(adminID)
	if err != nil {
		return err
	}
	if admin == nil {
		return ErrInvalidCredentials
	}

	normalized := ProfileForm{
		Name:      strings.TrimSpace(form.Name),
		Email:     normalizeEmail(form.Email),
		AvatarURL: strings.TrimSpace(form.AvatarURL),
	}

	fieldErrors := map[string]string{}
	if normalized.Name == "" {
		fieldErrors["name"] = "Enter your display name."
	}
	if normalized.Email == "" {
		fieldErrors["email"] = "Enter your email address."
	} else if !validEmail(normalized.Email) {
		fieldErrors["email"] = "Enter a valid email address."
	}
	if normalized.AvatarURL != "" && !validMediaURL(normalized.AvatarURL) {
		fieldErrors["avatar_url"] = "Use an uploaded profile image."
	}

	if normalized.Email != "" && normalized.Email != admin.Email {
		existing, err := s.admins.FindByEmail(normalized.Email)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != admin.ID {
			fieldErrors["email"] = "That email is already used by another administrator."
		}
	}

	if len(fieldErrors) > 0 {
		return ProfileValidationError{FieldErrors: fieldErrors}
	}

	admin.Name = normalized.Name
	admin.Email = normalized.Email
	admin.AvatarURL = normalized.AvatarURL
	if err := s.admins.UpdateProfile(admin); err != nil {
		return err
	}
	return nil
}

// ChangePassword validates and stores a new administrator password hash.
func (s *AccountService) ChangePassword(adminID uint, form PasswordForm) error {
	admin, err := s.admins.FindByID(adminID)
	if err != nil {
		return err
	}
	if admin == nil {
		return ErrInvalidCredentials
	}

	fieldErrors := map[string]string{}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(form.CurrentPassword)) != nil {
		fieldErrors["current_password"] = "Current password is incorrect."
	}
	if len(form.NewPassword) < 10 {
		fieldErrors["new_password"] = "Use a password with at least 10 characters."
	}
	if form.NewPassword != form.ConfirmPassword {
		fieldErrors["confirm_password"] = "Passwords must match."
	}
	if len(fieldErrors) > 0 {
		return PasswordValidationError{FieldErrors: fieldErrors}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(form.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new admin password: %w", err)
	}
	if err := s.admins.UpdatePasswordHash(admin.ID, string(hash)); err != nil {
		return err
	}
	return nil
}

func accountProfileFromAdmin(admin *models.AdminUser) AccountProfile {
	name := strings.TrimSpace(admin.Name)
	if name == "" {
		name = "Administrator"
	}

	return AccountProfile{
		ID:        admin.ID,
		Name:      name,
		Email:     admin.Email,
		AvatarURL: admin.AvatarURL,
		Initials:  initials(name, admin.Email),
		Role:      "Administrator",
		CreatedAt: formatAccountTime(admin.CreatedAt),
		UpdatedAt: formatAccountTime(admin.UpdatedAt),
	}
}

func initials(name, email string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		parts = strings.Fields(strings.ReplaceAll(email, "@", " "))
	}
	if len(parts) == 0 {
		return "A"
	}

	first := []rune(parts[0])
	if len(parts) == 1 {
		return strings.ToUpper(string(first[0]))
	}

	last := []rune(parts[len(parts)-1])
	return strings.ToUpper(string(first[0]) + string(last[0]))
}

func formatAccountTime(value time.Time) string {
	if value.IsZero() {
		return "Not available"
	}
	return value.Format("Jan 2, 2006 15:04")
}

func validMediaURL(value string) bool {
	return strings.HasPrefix(value, "/media/") && !strings.Contains(value, "..")
}
