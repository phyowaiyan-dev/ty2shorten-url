package models

import "time"

// VisitorSession stores a privacy-aware first-party analytics session.
type VisitorSession struct {
	ID                     uint      `gorm:"primaryKey"`
	SessionID              string    `gorm:"uniqueIndex;size:80;not null"`
	VisitorIDHash          string    `gorm:"index;size:128"`
	FirstSeenAt            time.Time `gorm:"index"`
	LastSeenAt             time.Time `gorm:"index"`
	StartedAt              time.Time `gorm:"index"`
	EndedAt                *time.Time
	PageViewCount          uint64 `gorm:"not null;default:0"`
	RedirectCount          uint64 `gorm:"not null;default:0"`
	IsBot                  bool   `gorm:"index;not null;default:false"`
	BotName                string `gorm:"size:120"`
	IPHash                 string `gorm:"size:128"`
	IPNetwork              string `gorm:"size:80"`
	CountryCode            string `gorm:"size:8"`
	Region                 string `gorm:"size:120"`
	City                   string `gorm:"size:120"`
	DeviceType             string `gorm:"index;size:40"`
	DeviceBrand            string `gorm:"size:120"`
	DeviceModel            string `gorm:"size:160"`
	OperatingSystem        string `gorm:"index;size:80"`
	OperatingSystemVersion string `gorm:"size:80"`
	Browser                string `gorm:"index;size:80"`
	BrowserVersion         string `gorm:"size:80"`
	Engine                 string `gorm:"size:80"`
	Language               string `gorm:"size:80"`
	Timezone               string `gorm:"size:120"`
	ScreenWidth            int
	ScreenHeight           int
	ReferrerType           string `gorm:"index;size:40"`
	ReferrerName           string `gorm:"size:120"`
	ReferrerHost           string `gorm:"index;size:255"`
	LandingPath            string `gorm:"size:255"`
	UTMSource              string `gorm:"index;size:120"`
	UTMMedium              string `gorm:"size:120"`
	UTMCampaign            string `gorm:"index;size:160"`
	UTMTerm                string `gorm:"size:160"`
	UTMContent             string `gorm:"size:160"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// PageView stores an eligible public page view.
type PageView struct {
	ID             uint   `gorm:"primaryKey"`
	SessionID      string `gorm:"index;size:80;not null"`
	RequestID      string `gorm:"index;size:80"`
	Path           string `gorm:"size:255"`
	NormalizedPath string `gorm:"index;size:255"`
	Method         string `gorm:"size:16"`
	StatusCode     int    `gorm:"index"`
	ReferrerType   string `gorm:"index;size:40"`
	ReferrerName   string `gorm:"size:120"`
	ReferrerHost   string `gorm:"index;size:255"`
	ReferrerPath   string `gorm:"size:255"`
	QueryMetadata  string `gorm:"type:text"`
	PageTitle      string `gorm:"size:160"`
	DurationMS     int64
	UTMSource      string    `gorm:"index;size:120"`
	UTMMedium      string    `gorm:"size:120"`
	UTMCampaign    string    `gorm:"index;size:160"`
	UTMTerm        string    `gorm:"size:160"`
	UTMContent     string    `gorm:"size:160"`
	IsBot          bool      `gorm:"index;not null;default:false"`
	OccurredAt     time.Time `gorm:"index"`
	CreatedAt      time.Time
}

// RedirectEvent stores an eligible public redirect event.
type RedirectEvent struct {
	ID                  uint      `gorm:"primaryKey"`
	SessionID           string    `gorm:"index;size:80;not null"`
	RequestID           string    `gorm:"index;size:80"`
	RouteType           string    `gorm:"index;size:80"`
	SourcePath          string    `gorm:"size:255"`
	ShortLinkID         *uint     `gorm:"index"`
	ShortLinkSlug       string    `gorm:"index;size:160"`
	DestinationHost     string    `gorm:"index;size:255"`
	DestinationPath     string    `gorm:"size:255"`
	DestinationPlatform string    `gorm:"index;size:80"`
	RedirectStatus      int       `gorm:"index"`
	IPHash              string    `gorm:"size:128"`
	IPNetwork           string    `gorm:"size:80"`
	DeviceType          string    `gorm:"index;size:40"`
	OperatingSystem     string    `gorm:"index;size:80"`
	Browser             string    `gorm:"index;size:80"`
	ReferrerType        string    `gorm:"index;size:40"`
	ReferrerName        string    `gorm:"size:120"`
	ReferrerHost        string    `gorm:"index;size:255"`
	UTMSource           string    `gorm:"index;size:120"`
	UTMMedium           string    `gorm:"size:120"`
	UTMCampaign         string    `gorm:"index;size:160"`
	UTMTerm             string    `gorm:"size:160"`
	UTMContent          string    `gorm:"size:160"`
	IsBot               bool      `gorm:"index;not null;default:false"`
	OccurredAt          time.Time `gorm:"index"`
	CreatedAt           time.Time
}
