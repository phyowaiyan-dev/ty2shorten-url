package models

import "time"

// AuditLog stores an append-only application audit event.
type AuditLog struct {
	ID           uint  `gorm:"primaryKey"`
	AdminUserID  *uint `gorm:"index"`
	ActorName    string
	ActorEmail   string `gorm:"index"`
	Action       string `gorm:"index;not null"`
	ResourceType string `gorm:"index"`
	ResourceID   string
	Summary      string
	MetadataJSON string
	IPAddress    string
	UserAgent    string
	RequestID    string    `gorm:"index"`
	CreatedAt    time.Time `gorm:"index"`
}
