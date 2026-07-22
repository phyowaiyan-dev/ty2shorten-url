package services

import (
	"fmt"
	"strings"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
)

// DashboardData contains initial administrator dashboard metrics.
type DashboardData struct {
	AndroidConfigured bool
	AppleConfigured   bool
	ActiveShortLinks  int64
	TotalClicks       uint64
	AppEnv            string
	SetupScore        int
	SetupCompleted    int
	SetupTotal        int
	SetupTasks        []SetupTask
	MissingSetup      []SetupTask
	ChartBars         []DashboardChartBar
	ReadinessLabel    string
}

// SetupTask describes one launch-readiness requirement.
type SetupTask struct {
	Label       string
	Description string
	URL         string
	Complete    bool
}

// DashboardChartBar describes one simple dashboard graph row.
type DashboardChartBar struct {
	Label        string
	Value        string
	WidthPercent int
}

// DashboardService loads administrator dashboard metrics.
type DashboardService struct {
	settings   *repositories.SettingsRepository
	shortLinks *repositories.ShortLinkRepository
	appEnv     string
}

// NewDashboardService constructs a DashboardService.
func NewDashboardService(settings *repositories.SettingsRepository, shortLinks *repositories.ShortLinkRepository, appEnv string) *DashboardService {
	return &DashboardService{settings: settings, shortLinks: shortLinks, appEnv: appEnv}
}

// Data returns current dashboard metrics.
func (s *DashboardService) Data() (DashboardData, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return DashboardData{}, err
	}

	activeCount, err := s.shortLinks.ActiveCount()
	if err != nil {
		return DashboardData{}, err
	}

	totalClicks, err := s.shortLinks.TotalClicks()
	if err != nil {
		return DashboardData{}, err
	}

	if settings == nil {
		return DashboardData{}, fmt.Errorf("app settings are not available")
	}

	tasks := setupTasks(settings, activeCount)
	score := setupScore(tasks)

	return DashboardData{
		AndroidConfigured: settings.AndroidURL != "",
		AppleConfigured:   settings.AppleURL != "",
		ActiveShortLinks:  activeCount,
		TotalClicks:       totalClicks,
		AppEnv:            s.appEnv,
		SetupScore:        score,
		SetupCompleted:    completedTasks(tasks),
		SetupTotal:        len(tasks),
		SetupTasks:        tasks,
		MissingSetup:      missingTasks(tasks),
		ChartBars:         chartBars(settings.AndroidURL != "", settings.AppleURL != "", activeCount, totalClicks),
		ReadinessLabel:    readinessLabel(score),
	}, nil
}

func setupTasks(settings *models.AppSetting, activeShortLinks int64) []SetupTask {
	return []SetupTask{
		{
			Label:       "Android app destination",
			Description: "Add the Google Play URL used by Android redirects.",
			URL:         "/admin/settings",
			Complete:    present(settings.AndroidURL),
		},
		{
			Label:       "Apple app destination",
			Description: "Add the App Store URL used by iOS redirects.",
			URL:         "/admin/settings",
			Complete:    present(settings.AppleURL),
		},
		{
			Label:       "Fallback destination",
			Description: "Set a default URL for desktop users and unknown devices.",
			URL:         "/admin/settings",
			Complete:    present(settings.DefaultURL),
		},
		{
			Label:       "Public base URL",
			Description: "Configure the public domain used in QR codes and shared links.",
			URL:         "/admin/settings",
			Complete:    present(settings.PublicBaseURL),
		},
		{
			Label:       "Support contact",
			Description: "Add the help email shown on public-facing pages.",
			URL:         "/admin/settings",
			Complete:    present(settings.SupportEmail),
		},
		{
			Label:       "Branding assets",
			Description: "Upload a logo or icon so the landing page feels branded.",
			URL:         "/admin/settings/branding",
			Complete:    present(settings.LogoURL) || present(settings.FaviconURL) || present(settings.AppleTouchIconURL),
		},
		{
			Label:       "Brand message",
			Description: "Add a tagline and description for the public hero.",
			URL:         "/admin/settings/branding",
			Complete:    present(settings.SiteTagline) && present(settings.SiteDescription),
		},
		{
			Label:       "SEO metadata",
			Description: "Set title, description, canonical URL, and social defaults.",
			URL:         "/admin/settings/seo",
			Complete:    present(settings.DefaultMetaTitle) && present(settings.DefaultMetaDescription) && present(settings.CanonicalBaseURL),
		},
		{
			Label:       "Public footer",
			Description: "Add footer content, copyright text, and support notes.",
			URL:         "/admin/settings/footer",
			Complete:    settings.FooterEnabled && present(settings.FooterBrandText) && present(settings.FooterDescription) && present(settings.CopyrightText),
		},
		{
			Label:       "Legal content",
			Description: "Add Privacy Policy and Terms of Use Markdown content.",
			URL:         "/admin/settings/legal",
			Complete:    present(settings.PrivacyPolicyMarkdown) && present(settings.TermsMarkdown),
		},
		{
			Label:       "Short link inventory",
			Description: "Create at least one active short link for campaign routing.",
			URL:         "/admin/links/new",
			Complete:    activeShortLinks > 0,
		},
	}
}

func completedTasks(tasks []SetupTask) int {
	var completed int
	for _, task := range tasks {
		if task.Complete {
			completed++
		}
	}
	return completed
}

func missingTasks(tasks []SetupTask) []SetupTask {
	missing := make([]SetupTask, 0, len(tasks))
	for _, task := range tasks {
		if !task.Complete {
			missing = append(missing, task)
		}
	}
	return missing
}

func setupScore(tasks []SetupTask) int {
	if len(tasks) == 0 {
		return 0
	}
	return (completedTasks(tasks)*100 + len(tasks)/2) / len(tasks)
}

func readinessLabel(score int) string {
	switch {
	case score >= 90:
		return "Production ready"
	case score >= 70:
		return "Nearly ready"
	case score >= 40:
		return "Needs attention"
	default:
		return "Setup required"
	}
}

func chartBars(androidConfigured, appleConfigured bool, activeShortLinks int64, totalClicks uint64) []DashboardChartBar {
	appTargets := int64(0)
	if androidConfigured {
		appTargets++
	}
	if appleConfigured {
		appTargets++
	}

	values := []struct {
		label string
		value uint64
	}{
		{label: "App targets", value: uint64(appTargets)},
		{label: "Active links", value: uint64(activeShortLinks)},
		{label: "Total clicks", value: totalClicks},
	}

	var max uint64
	for _, item := range values {
		if item.value > max {
			max = item.value
		}
	}
	if max == 0 {
		max = 1
	}

	bars := make([]DashboardChartBar, 0, len(values))
	for _, item := range values {
		width := int(item.value * 100 / max)
		if item.value > 0 && width < 12 {
			width = 12
		}
		if item.value == 0 {
			width = 4
		}
		bars = append(bars, DashboardChartBar{
			Label:        item.label,
			Value:        fmt.Sprintf("%d", item.value),
			WidthPercent: width,
		})
	}
	return bars
}

func present(value string) bool {
	return strings.TrimSpace(value) != ""
}
