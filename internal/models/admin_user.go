package models

import "time"

// AdminUser represents an administrator account.
type AdminUser struct {
	ID           uint `gorm:"primaryKey"`
	Name         string
	Email        string `gorm:"uniqueIndex;not null"`
	AvatarURL    string
	PasswordHash string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
