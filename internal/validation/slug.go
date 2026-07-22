package validation

import (
	"errors"
	"regexp"
	"strings"
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{1,49}$`)

	reservedSlugs = map[string]struct{}{
		"admin":   {},
		"setup":   {},
		"login":   {},
		"logout":  {},
		"health":  {},
		"android": {},
		"apple":   {},
		"get":     {},
		"static":  {},
		"assets":  {},
		"r":       {},
	}
)

var (
	// ErrInvalidSlug is returned when a slug fails syntax validation.
	ErrInvalidSlug = errors.New("slug must be 2-50 characters and contain only lowercase letters, numbers, hyphens, and underscores")
	// ErrReservedSlug is returned when a slug conflicts with an application route.
	ErrReservedSlug = errors.New("slug is reserved")
)

// NormalizeSlug returns the canonical form used for lookup and storage.
func NormalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

// ValidateSlug checks slug syntax and reserved route names.
func ValidateSlug(slug string) error {
	normalized := NormalizeSlug(slug)
	if !slugPattern.MatchString(normalized) {
		return ErrInvalidSlug
	}
	if _, ok := reservedSlugs[normalized]; ok {
		return ErrReservedSlug
	}

	return nil
}
