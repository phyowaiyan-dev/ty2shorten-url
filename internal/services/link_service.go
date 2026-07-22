package services

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
)

var ErrLinkNotFound = errors.New("short link not found")

// LinkForm contains editable short-link fields.
type LinkForm struct {
	ID          uint
	Title       string
	Slug        string
	Destination string
	IsActive    bool
}

// LinkView contains administrator list display data.
type LinkView struct {
	models.ShortLink
	PublicURL string
}

// LinkValidationError contains friendly field errors.
type LinkValidationError struct {
	FieldErrors map[string]string
}

func (e LinkValidationError) Error() string {
	return "invalid short link form"
}

// LinkService owns short-link CRUD validation and persistence.
type LinkService struct {
	links        *repositories.ShortLinkRepository
	settings     *repositories.SettingsRepository
	isProduction bool
	logger       *slog.Logger
}

// NewLinkService constructs a LinkService.
func NewLinkService(links *repositories.ShortLinkRepository, settings *repositories.SettingsRepository, isProduction bool, logger *slog.Logger) *LinkService {
	return &LinkService{links: links, settings: settings, isProduction: isProduction, logger: logger}
}

// List returns administrator link rows.
func (s *LinkService) List() ([]LinkView, error) {
	links, err := s.links.List()
	if err != nil {
		return nil, err
	}
	baseURL, err := s.publicBaseURL()
	if err != nil {
		return nil, err
	}
	views := make([]LinkView, 0, len(links))
	for _, link := range links {
		views = append(views, LinkView{ShortLink: link, PublicURL: validation.JoinURLPath(baseURL, "/r/"+link.Slug)})
	}
	return views, nil
}

// FormForID returns a link as editable form data.
func (s *LinkService) FormForID(id uint) (LinkForm, error) {
	link, err := s.links.FindByID(id)
	if err != nil {
		return LinkForm{}, err
	}
	if link == nil {
		return LinkForm{}, ErrLinkNotFound
	}
	return LinkForm{
		ID:          link.ID,
		Title:       link.Title,
		Slug:        link.Slug,
		Destination: link.Destination,
		IsActive:    link.IsActive,
	}, nil
}

// Create validates and stores a short link.
func (s *LinkService) Create(form LinkForm) error {
	normalized, err := s.validate(form, 0)
	if err != nil {
		return err
	}
	link := &models.ShortLink{
		Title:       normalized.Title,
		Slug:        normalized.Slug,
		Destination: normalized.Destination,
		IsActive:    normalized.IsActive,
	}
	if err := s.links.Create(link); err != nil {
		return err
	}
	s.log("link created", slog.String("slug", normalized.Slug))
	return nil
}

// Update validates and stores a short link.
func (s *LinkService) Update(id uint, form LinkForm) error {
	existing, err := s.links.FindByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrLinkNotFound
	}
	normalized, err := s.validate(form, id)
	if err != nil {
		return err
	}
	wasActive := existing.IsActive
	existing.Title = normalized.Title
	existing.Slug = normalized.Slug
	existing.Destination = normalized.Destination
	existing.IsActive = normalized.IsActive
	if err := s.links.Update(existing); err != nil {
		return err
	}
	event := "link updated"
	if wasActive && !normalized.IsActive {
		event = "link disabled"
	}
	s.log(event, slog.String("slug", normalized.Slug))
	return nil
}

// Delete removes a short link.
func (s *LinkService) Delete(id uint) error {
	link, err := s.links.FindByID(id)
	if err != nil {
		return err
	}
	if link == nil {
		return ErrLinkNotFound
	}
	if err := s.links.Delete(id); err != nil {
		return err
	}
	s.log("link deleted", slog.String("slug", link.Slug))
	return nil
}

func (s *LinkService) validate(form LinkForm, currentID uint) (LinkForm, error) {
	normalized := LinkForm{
		ID:          currentID,
		Title:       strings.TrimSpace(form.Title),
		Slug:        validation.NormalizeSlug(form.Slug),
		Destination: strings.TrimSpace(form.Destination),
		IsActive:    form.IsActive,
	}
	fieldErrors := map[string]string{}
	requireText(fieldErrors, "title", normalized.Title, "Enter a title.")
	if err := validation.ValidateSlug(normalized.Slug); err != nil {
		fieldErrors["slug"] = "Use 2-50 lowercase letters, numbers, hyphens, or underscores. Reserved route names are not allowed."
	}
	if err := validation.ValidateHTTPURL(normalized.Destination, s.isProduction); err != nil {
		fieldErrors["destination"] = urlValidationMessage(err)
	}
	if normalized.Slug != "" {
		existing, err := s.links.FindBySlug(normalized.Slug)
		if err != nil {
			return normalized, err
		}
		if existing != nil && existing.ID != currentID {
			fieldErrors["slug"] = "That slug is already in use."
		}
	}
	if len(fieldErrors) > 0 {
		return normalized, LinkValidationError{FieldErrors: fieldErrors}
	}
	return normalized, nil
}

func (s *LinkService) publicBaseURL() (string, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return "", err
	}
	if settings == nil || strings.TrimSpace(settings.PublicBaseURL) == "" {
		return "", fmt.Errorf("public base URL is not configured")
	}
	return settings.PublicBaseURL, nil
}

func (s *LinkService) log(message string, attrs ...slog.Attr) {
	if s.logger == nil {
		return
	}
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	s.logger.Info(message, args...)
}

// ParseID parses a route ID parameter.
func ParseID(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 0)
	if err != nil || id == 0 {
		return 0, ErrLinkNotFound
	}
	return uint(id), nil
}
