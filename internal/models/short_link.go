package models

import "time"

// ShortLink stores a public short URL destination.
type ShortLink struct {
	ID          uint   `gorm:"primaryKey"`
	Slug        string `gorm:"uniqueIndex;not null"`
	Title       string
	Destination string `gorm:"not null"`
	IsActive    bool   `gorm:"not null"`
	ClickCount  uint64 `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
