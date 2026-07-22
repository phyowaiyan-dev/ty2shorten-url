package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
)

// ErrAuditLogNotFound is returned when an audit log detail cannot be found.
var ErrAuditLogNotFound = errors.New("audit log not found")

// AuditListFilter constrains audit log list queries.
type AuditListFilter struct {
	DateFrom string
	DateTo   string
	Limit    int
	Offset   int
}

// AuditEvent contains data for an audit entry.
type AuditEvent struct {
	AdminUserID  *uint
	ActorName    string
	ActorEmail   string
	Action       string
	ResourceType string
	ResourceID   string
	Summary      string
	Metadata     map[string]any
	IPAddress    string
	UserAgent    string
	RequestID    string
}

// AuditService writes and reads audit logs.
type AuditService struct {
	audits *repositories.AuditRepository
}

// NewAuditService constructs an AuditService.
func NewAuditService(audits *repositories.AuditRepository) *AuditService {
	return &AuditService{audits: audits}
}

// Record appends a redacted audit log.
func (s *AuditService) Record(event AuditEvent) error {
	metadata, _ := json.Marshal(redactMetadata(event.Metadata))
	userAgent := event.UserAgent
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	return s.audits.Create(&models.AuditLog{
		AdminUserID:  event.AdminUserID,
		ActorName:    event.ActorName,
		ActorEmail:   event.ActorEmail,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		Summary:      event.Summary,
		MetadataJSON: string(metadata),
		IPAddress:    strings.TrimSpace(event.IPAddress),
		UserAgent:    userAgent,
		RequestID:    event.RequestID,
	})
}

// List returns recent audit logs.
func (s *AuditService) List() ([]models.AuditLog, error) {
	return s.audits.List(50)
}

// Search returns audit logs matching read-only admin filters.
func (s *AuditService) Search(filter AuditListFilter) ([]models.AuditLog, error) {
	createdFrom := parseAuditDate(filter.DateFrom)
	createdTo := parseAuditDate(filter.DateTo)
	if createdTo != nil {
		end := createdTo.AddDate(0, 0, 1)
		createdTo = &end
	}
	return s.audits.Search(repositories.AuditListFilter{
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// Detail returns one audit log entry.
func (s *AuditService) Detail(id uint) (*models.AuditLog, error) {
	log, err := s.audits.FindByID(id)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, ErrAuditLogNotFound
	}
	return log, nil
}

func parseAuditDate(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", trimmed, time.Local)
	if err != nil {
		return nil
	}
	return &parsed
}

func redactMetadata(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, val := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "cookie") || strings.Contains(lower, "dsn") {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactMetadata(val)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, val := range typed {
			out = append(out, redactMetadata(val))
		}
		return out
	default:
		return typed
	}
}
