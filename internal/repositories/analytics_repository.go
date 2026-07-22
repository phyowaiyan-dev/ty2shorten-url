package repositories

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AnalyticsRepository stores and summarizes privacy-aware analytics events.
type AnalyticsRepository struct {
	db *gorm.DB
}

// AnalyticsRangeFilter constrains analytics report queries.
type AnalyticsRangeFilter struct {
	From        time.Time
	To          time.Time
	ExcludeBots bool
	Limit       int
}

// AnalyticsSummary contains dashboard totals.
type AnalyticsSummary struct {
	PageViews        int64
	RedirectEvents   int64
	Sessions         int64
	UniqueVisitors   int64
	BotSessions      int64
	AndroidRedirects int64
	AppleRedirects   int64
}

// AnalyticsBreakdownRow describes grouped dashboard data.
type AnalyticsBreakdownRow struct {
	Label string
	Count int64
}

// AnalyticsSessionTimelineRow describes one session timeline event.
type AnalyticsSessionTimelineRow struct {
	OccurredAt time.Time
	Kind       string
	Label      string
	Path       string
}

// AnalyticsCleanupSummary reports retention cleanup effects.
type AnalyticsCleanupSummary struct {
	PageViews      int64
	RedirectEvents int64
	Sessions       int64
}

// NewAnalyticsRepository constructs an AnalyticsRepository.
func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// UpsertSession creates or refreshes a visitor session.
func (r *AnalyticsRepository) UpsertSession(session *models.VisitorSession) error {
	if err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"last_seen_at",
			"is_bot",
			"bot_name",
			"ip_hash",
			"ip_network",
			"device_type",
			"device_brand",
			"device_model",
			"operating_system",
			"operating_system_version",
			"browser",
			"browser_version",
			"engine",
			"language",
			"timezone",
			"updated_at",
		}),
	}).Create(session).Error; err != nil {
		return fmt.Errorf("upsert analytics session: %w", err)
	}
	return nil
}

// IncrementSessionPageViews increments a session page-view counter.
func (r *AnalyticsRepository) IncrementSessionPageViews(sessionID string) error {
	return r.incrementSessionCounter(sessionID, "page_view_count")
}

// IncrementSessionRedirects increments a session redirect counter.
func (r *AnalyticsRepository) IncrementSessionRedirects(sessionID string) error {
	return r.incrementSessionCounter(sessionID, "redirect_count")
}

// CreatePageView stores one public page-view event.
func (r *AnalyticsRepository) CreatePageView(view *models.PageView) error {
	if err := r.db.Create(view).Error; err != nil {
		return fmt.Errorf("create page view: %w", err)
	}
	return nil
}

// CreateRedirect stores one public redirect event.
func (r *AnalyticsRepository) CreateRedirect(event *models.RedirectEvent) error {
	if err := r.db.Create(event).Error; err != nil {
		return fmt.Errorf("create redirect event: %w", err)
	}
	return nil
}

// Summary returns top-level analytics totals.
func (r *AnalyticsRepository) Summary(filter AnalyticsRangeFilter) (AnalyticsSummary, error) {
	var summary AnalyticsSummary
	pageViews := r.db.Model(&models.PageView{}).Where("occurred_at >= ? AND occurred_at < ?", filter.From, filter.To)
	redirects := r.db.Model(&models.RedirectEvent{}).Where("occurred_at >= ? AND occurred_at < ?", filter.From, filter.To)
	sessions := r.db.Model(&models.VisitorSession{}).Where("first_seen_at < ? AND last_seen_at >= ?", filter.To, filter.From)
	if filter.ExcludeBots {
		pageViews = pageViews.Where("is_bot = ?", false)
		redirects = redirects.Where("is_bot = ?", false)
		sessions = sessions.Where("is_bot = ?", false)
	}
	if err := pageViews.Count(&summary.PageViews).Error; err != nil {
		return summary, fmt.Errorf("count page views: %w", err)
	}
	if err := redirects.Count(&summary.RedirectEvents).Error; err != nil {
		return summary, fmt.Errorf("count redirect events: %w", err)
	}
	if err := sessions.Count(&summary.Sessions).Error; err != nil {
		return summary, fmt.Errorf("count sessions: %w", err)
	}
	if err := sessions.Distinct("visitor_id_hash").Where("visitor_id_hash <> ?", "").Count(&summary.UniqueVisitors).Error; err != nil {
		return summary, fmt.Errorf("count unique visitors: %w", err)
	}
	if err := r.db.Model(&models.VisitorSession{}).Where("first_seen_at < ? AND last_seen_at >= ? AND is_bot = ?", filter.To, filter.From, true).Count(&summary.BotSessions).Error; err != nil {
		return summary, fmt.Errorf("count bot sessions: %w", err)
	}
	if err := redirects.Where("destination_platform = ?", "android").Count(&summary.AndroidRedirects).Error; err != nil {
		return summary, fmt.Errorf("count android redirects: %w", err)
	}
	if err := r.db.Model(&models.RedirectEvent{}).Where("occurred_at >= ? AND occurred_at < ? AND destination_platform = ?", filter.From, filter.To, "apple").Count(&summary.AppleRedirects).Error; err != nil {
		return summary, fmt.Errorf("count apple redirects: %w", err)
	}
	return summary, nil
}

// Breakdown groups redirect events by a safe column.
func (r *AnalyticsRepository) Breakdown(table, column string, filter AnalyticsRangeFilter) ([]AnalyticsBreakdownRow, error) {
	if table != "redirect_events" && table != "page_views" && table != "visitor_sessions" {
		return nil, fmt.Errorf("unsupported analytics table")
	}
	allowed := map[string]bool{
		"device_type": true, "operating_system": true, "browser": true, "referrer_host": true,
		"route_type": true, "destination_platform": true,
	}
	if !allowed[column] {
		return nil, fmt.Errorf("unsupported analytics column")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows []AnalyticsBreakdownRow
	query := r.db.Table(table).
		Select(fmt.Sprintf("COALESCE(NULLIF(%s, ''), 'unknown') AS label, COUNT(*) AS count", column)).
		Where("created_at >= ? AND created_at < ?", filter.From, filter.To)
	if filter.ExcludeBots && table != "visitor_sessions" {
		query = query.Where("is_bot = ?", false)
	}
	if filter.ExcludeBots && table == "visitor_sessions" {
		query = query.Where("is_bot = ?", false)
	}
	if err := query.Group(column).Order("count DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("analytics breakdown: %w", err)
	}
	return rows, nil
}

// RedirectEvents returns redirect rows for CSV export.
func (r *AnalyticsRepository) RedirectEvents(filter AnalyticsRangeFilter) ([]models.RedirectEvent, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	var events []models.RedirectEvent
	query := r.db.Where("occurred_at >= ? AND occurred_at < ?", filter.From, filter.To).Order("occurred_at DESC").Limit(limit)
	if filter.ExcludeBots {
		query = query.Where("is_bot = ?", false)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list redirect events: %w", err)
	}
	return events, nil
}

// RecentRedirectEvents returns recent redirect rows for the admin dashboard.
func (r *AnalyticsRepository) RecentRedirectEvents(filter AnalyticsRangeFilter) ([]models.RedirectEvent, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var events []models.RedirectEvent
	query := r.db.Where("occurred_at >= ? AND occurred_at < ?", filter.From, filter.To).Order("occurred_at DESC").Limit(limit)
	if filter.ExcludeBots {
		query = query.Where("is_bot = ?", false)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list recent redirect events: %w", err)
	}
	return events, nil
}

// FindSession returns one analytics session by opaque session ID.
func (r *AnalyticsRepository) FindSession(sessionID string) (*models.VisitorSession, error) {
	var session models.VisitorSession
	if err := r.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find analytics session: %w", err)
	}
	return &session, nil
}

// SessionTimeline returns recent page-view and redirect events for one session.
func (r *AnalyticsRepository) SessionTimeline(sessionID string, limit int) ([]AnalyticsSessionTimelineRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var pageViews []models.PageView
	if err := r.db.Where("session_id = ?", sessionID).Order("occurred_at ASC").Limit(limit).Find(&pageViews).Error; err != nil {
		return nil, fmt.Errorf("list session page views: %w", err)
	}
	var redirects []models.RedirectEvent
	if err := r.db.Where("session_id = ?", sessionID).Order("occurred_at ASC").Limit(limit).Find(&redirects).Error; err != nil {
		return nil, fmt.Errorf("list session redirects: %w", err)
	}
	rows := make([]AnalyticsSessionTimelineRow, 0, len(pageViews)+len(redirects))
	for _, view := range pageViews {
		rows = append(rows, AnalyticsSessionTimelineRow{OccurredAt: view.OccurredAt, Kind: "Page view", Label: view.PageTitle, Path: view.Path})
	}
	for _, event := range redirects {
		label := event.RouteType
		if event.ShortLinkSlug != "" {
			label = "Short link: " + event.ShortLinkSlug
		}
		rows = append(rows, AnalyticsSessionTimelineRow{OccurredAt: event.OccurredAt, Kind: "Redirect", Label: label, Path: event.SourcePath})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].OccurredAt.Before(rows[j].OccurredAt)
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

// CleanupBefore deletes analytics data older than cutoff.
func (r *AnalyticsRepository) CleanupBefore(cutoff time.Time, dryRun bool) (AnalyticsCleanupSummary, error) {
	var summary AnalyticsCleanupSummary
	if err := r.db.Model(&models.PageView{}).Where("occurred_at < ?", cutoff).Count(&summary.PageViews).Error; err != nil {
		return summary, fmt.Errorf("count expired page views: %w", err)
	}
	if err := r.db.Model(&models.RedirectEvent{}).Where("occurred_at < ?", cutoff).Count(&summary.RedirectEvents).Error; err != nil {
		return summary, fmt.Errorf("count expired redirect events: %w", err)
	}
	if err := r.db.Model(&models.VisitorSession{}).Where("last_seen_at < ?", cutoff).Count(&summary.Sessions).Error; err != nil {
		return summary, fmt.Errorf("count expired sessions: %w", err)
	}
	if dryRun {
		return summary, nil
	}
	if err := r.db.Where("occurred_at < ?", cutoff).Delete(&models.PageView{}).Error; err != nil {
		return summary, fmt.Errorf("delete expired page views: %w", err)
	}
	if err := r.db.Where("occurred_at < ?", cutoff).Delete(&models.RedirectEvent{}).Error; err != nil {
		return summary, fmt.Errorf("delete expired redirect events: %w", err)
	}
	if err := r.db.Where("last_seen_at < ?", cutoff).Delete(&models.VisitorSession{}).Error; err != nil {
		return summary, fmt.Errorf("delete expired sessions: %w", err)
	}
	return summary, nil
}

func (r *AnalyticsRepository) incrementSessionCounter(sessionID, column string) error {
	if sessionID == "" {
		return nil
	}
	if err := r.db.Model(&models.VisitorSession{}).Where("session_id = ?", sessionID).UpdateColumn(column, gorm.Expr(column+" + ?", 1)).Error; err != nil {
		return fmt.Errorf("increment analytics session counter: %w", err)
	}
	return nil
}
