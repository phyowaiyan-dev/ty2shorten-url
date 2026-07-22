package services

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
)

var (
	// ErrMissingDestination is returned when an app-store destination is not configured.
	ErrMissingDestination = errors.New("destination is not configured")
	// ErrShortLinkNotFound is returned when a short link cannot be resolved.
	ErrShortLinkNotFound = errors.New("short link was not found")
)

// RedirectService resolves public redirect destinations.
type RedirectService struct {
	settings   *repositories.SettingsRepository
	shortLinks *repositories.ShortLinkRepository
	logger     *slog.Logger
}

// NewRedirectService constructs a RedirectService.
func NewRedirectService(settings *repositories.SettingsRepository, shortLinks *repositories.ShortLinkRepository, logger *slog.Logger) *RedirectService {
	return &RedirectService{settings: settings, shortLinks: shortLinks, logger: logger}
}

// AndroidURL returns the configured Android destination.
func (s *RedirectService) AndroidURL() (string, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return "", err
	}
	if settings == nil || settings.AndroidURL == "" {
		s.logFailure("android", ErrMissingDestination)
		return "", ErrMissingDestination
	}

	return settings.AndroidURL, nil
}

// AppleURL returns the configured Apple destination.
func (s *RedirectService) AppleURL() (string, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return "", err
	}
	if settings == nil || settings.AppleURL == "" {
		s.logFailure("apple", ErrMissingDestination)
		return "", ErrMissingDestination
	}

	return settings.AppleURL, nil
}

// DeviceTarget returns the route that should handle a User-Agent.
func DeviceTarget(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "android"):
		return "/android"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"), strings.Contains(ua, "ipod"):
		return "/apple"
	default:
		return "/"
	}
}

// ResolveShortLink finds an active custom short link and records the click.
func (s *RedirectService) ResolveShortLink(slug string) (*models.ShortLink, error) {
	normalized := validation.NormalizeSlug(slug)
	if err := validation.ValidateSlug(normalized); err != nil {
		s.logFailure("short_link", err)
		return nil, ErrShortLinkNotFound
	}

	link, err := s.shortLinks.FindActiveBySlug(normalized)
	if err != nil {
		return nil, err
	}
	if link == nil {
		s.logFailure("short_link", ErrShortLinkNotFound)
		return nil, ErrShortLinkNotFound
	}

	if err := s.shortLinks.IncrementClickCount(link.ID); err != nil && s.logger != nil {
		s.logger.Error("increment short link click count",
			slog.String("slug", normalized),
			slog.String("error", err.Error()),
		)
	}

	return link, nil
}

func (s *RedirectService) logFailure(kind string, err error) {
	if s.logger != nil {
		s.logger.Warn("redirect resolution failure", slog.String("kind", kind), slog.String("error", err.Error()))
	}
}
