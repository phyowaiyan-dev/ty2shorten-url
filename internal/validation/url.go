package validation

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var (
	// ErrInvalidURL is returned when a destination URL is malformed.
	ErrInvalidURL = errors.New("enter a valid URL")
	// ErrUnsafeURLScheme is returned when a URL uses a non-http scheme.
	ErrUnsafeURLScheme = errors.New("only http and https URLs are allowed")
	// ErrHTTPSRequired is returned when production settings require HTTPS.
	ErrHTTPSRequired = errors.New("use an HTTPS URL in production")
)

// ValidateHTTPURL validates a public destination URL.
func ValidateHTTPURL(value string, requireHTTPS bool) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrUnsafeURLScheme
	}
	if parsed.Host == "" {
		return ErrInvalidURL
	}
	if requireHTTPS && parsed.Scheme != "https" && !isLocalHost(parsed.Hostname()) {
		return ErrHTTPSRequired
	}

	return nil
}

// JoinURLPath appends a path to a base URL without trusting user-controlled redirects.
func JoinURLPath(baseURL, path string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return trimmedBase + path
}

func isLocalHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}
