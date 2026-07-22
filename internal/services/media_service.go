package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxLogoBytes    = 2 << 20
	maxFaviconBytes = 512 << 10
	maxSocialBytes  = 5 << 20
)

var (
	// ErrMediaInvalidType is returned when uploaded bytes are not an allowed image.
	ErrMediaInvalidType = errors.New("invalid media type")
	// ErrMediaTooLarge is returned when the uploaded file exceeds the configured limit.
	ErrMediaTooLarge = errors.New("media file too large")
	// ErrMediaMissing is returned when a requested media object is not present.
	ErrMediaMissing = errors.New("media not found")
)

type mediaRule struct {
	maxBytes int64
	prefix   string
}

var mediaRules = map[string]mediaRule{
	"logo":              {maxBytes: maxLogoBytes, prefix: "logo"},
	"dark_logo":         {maxBytes: maxLogoBytes, prefix: "dark-logo"},
	"organization_logo": {maxBytes: maxLogoBytes, prefix: "organization-logo"},
	"avatar":            {maxBytes: maxLogoBytes, prefix: "avatar"},
	"favicon":           {maxBytes: maxFaviconBytes, prefix: "favicon"},
	"apple_icon":        {maxBytes: maxFaviconBytes, prefix: "apple-icon"},
	"social_image":      {maxBytes: maxSocialBytes, prefix: "social"},
	"open_graph_image":  {maxBytes: maxSocialBytes, prefix: "open-graph"},
	"twitter_image":     {maxBytes: maxSocialBytes, prefix: "twitter"},
}

// MediaService validates, stores, and serves administrator-uploaded media.
type MediaService struct {
	dir string
}

// NewMediaService constructs a MediaService.
func NewMediaService(dir string) *MediaService {
	return &MediaService{dir: strings.TrimSpace(dir)}
}

// Save stores a validated upload and returns its public media URL.
func (s *MediaService) Save(kind string, header *multipart.FileHeader) (string, error) {
	rule, ok := mediaRules[kind]
	if !ok {
		return "", fmt.Errorf("unsupported media kind %q", kind)
	}
	if header == nil || header.Size == 0 {
		return "", nil
	}
	if header.Size > rule.maxBytes {
		return "", fmt.Errorf("%s file is too large: %w", mediaLabel(kind), ErrMediaTooLarge)
	}

	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("open uploaded %s: %w", mediaLabel(kind), err)
	}
	defer file.Close()

	limit := rule.maxBytes + 1
	body, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return "", fmt.Errorf("read uploaded %s: %w", mediaLabel(kind), err)
	}
	if int64(len(body)) > rule.maxBytes {
		return "", fmt.Errorf("%s file is too large: %w", mediaLabel(kind), ErrMediaTooLarge)
	}

	mimeType := http.DetectContentType(body)
	ext, ok := mediaExtension(mimeType)
	if !ok {
		return "", ErrMediaInvalidType
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", fmt.Errorf("create media directory: %w", err)
	}

	name, err := randomMediaName(rule.prefix, ext)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return "", fmt.Errorf("write media file: %w", err)
	}
	return "/media/" + name, nil
}

// Open returns a stored media object, its MIME type, and a cleanup function.
func (s *MediaService) Open(name string) (*os.File, string, func(), error) {
	if !safeMediaName(name) {
		return nil, "", nil, ErrMediaMissing
	}
	path := filepath.Join(s.dir, name)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", nil, ErrMediaMissing
		}
		return nil, "", nil, err
	}
	head := make([]byte, 512)
	n, _ := file.Read(head)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, "", nil, err
	}
	mimeType := http.DetectContentType(head[:n])
	if _, ok := mediaExtension(mimeType); !ok {
		_ = file.Close()
		return nil, "", nil, ErrMediaMissing
	}
	return file, mimeType, func() { _ = file.Close() }, nil
}

func mediaExtension(mimeType string) (string, bool) {
	switch mimeType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/webp":
		return ".webp", true
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico", true
	default:
		return "", false
	}
}

func randomMediaName(prefix, ext string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate media filename: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(b[:]) + ext, nil
}

func safeMediaName(name string) bool {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func mediaLabel(kind string) string {
	return strings.ReplaceAll(kind, "_", " ")
}
