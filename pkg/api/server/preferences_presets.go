package server

import (
	"fmt"
	"strings"
)

// PresetSettings holds workspace options stored inside a named configuration.
type PresetSettings struct {
	ThemeMode                   string `json:"theme_mode"`
	ThemePalette                string `json:"theme_palette"`
	Theme                       string `json:"theme"`
	DefaultStatusFilter         string `json:"default_status_filter"`
	DefaultTimeRange            string `json:"default_time_range"`
	CompactUI                   bool   `json:"compact_ui"`
	SidebarCollapsedDefault     bool   `json:"sidebar_collapsed_default"`
	DashboardAutoRefresh        bool   `json:"dashboard_auto_refresh"`
	DashboardRefreshIntervalSec int    `json:"dashboard_refresh_interval_sec"`
	DateFormat                  string `json:"date_format"`
	Timezone                    string `json:"timezone"`
	DefaultLandingView          string `json:"default_landing_view"`
	ReducedMotion               bool   `json:"reduced_motion"`
	HighContrast                bool   `json:"high_contrast"`
	BrowserNotifications        bool   `json:"browser_notifications"`
	ExpandIncidentCards         bool   `json:"expand_incident_cards"`
	DenseTables                 bool   `json:"dense_tables"`
	AnalyticsExpanded           bool   `json:"analytics_expanded"`
	Currency                    string `json:"currency"`
	AlertSounds                 bool   `json:"alert_sounds"`
}

// ConfigurationPreset is a named workspace profile (built-in or user-created).
type ConfigurationPreset struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BuiltIn     bool           `json:"built_in"`
	Settings    PresetSettings `json:"settings"`
}

func defaultPresetSettings() PresetSettings {
	return PresetSettings{
		ThemeMode:                   "dark",
		ThemePalette:                "default",
		Theme:                       "dark",
		DefaultStatusFilter:         "all",
		DefaultTimeRange:            "15m",
		CompactUI:                   false,
		SidebarCollapsedDefault:     false,
		DashboardAutoRefresh:        true,
		DashboardRefreshIntervalSec: 60,
		DateFormat:                  "locale",
		Timezone:                    "auto",
		DefaultLandingView:          "dashboard",
		ReducedMotion:               false,
		HighContrast:                false,
		BrowserNotifications:        false,
		ExpandIncidentCards:         false,
		DenseTables:                 false,
		AnalyticsExpanded:           false,
		Currency:                    "USD",
		AlertSounds:                 false,
	}
}

func builtInConfigurationPresets() []ConfigurationPreset {
	base := defaultPresetSettings()

	onCall := base
	onCall.ThemeMode = "dark"
	onCall.ThemePalette = "ops"
	onCall.Theme = "dark"
	onCall.DefaultTimeRange = "5m"
	onCall.DashboardRefreshIntervalSec = 15
	onCall.DashboardAutoRefresh = true
	onCall.HighContrast = true
	onCall.DenseTables = true
	onCall.ExpandIncidentCards = true
	onCall.AlertSounds = true
	onCall.BrowserNotifications = true
	onCall.DefaultLandingView = "dashboard"

	executive := base
	executive.ThemeMode = "light"
	executive.ThemePalette = "graphite"
	executive.Theme = "light"
	executive.DefaultTimeRange = "24h"
	executive.AnalyticsExpanded = true
	executive.DefaultLandingView = "finops"
	executive.DashboardRefreshIntervalSec = 120

	incident := base
	incident.ThemeMode = "dark"
	incident.ThemePalette = "ops"
	incident.Theme = "dark"
	incident.DefaultTimeRange = "5m"
	incident.DefaultStatusFilter = "active"
	incident.DashboardRefreshIntervalSec = 15
	incident.ExpandIncidentCards = true
	incident.CompactUI = true
	incident.DefaultLandingView = "healing"

	audit := base
	audit.ThemeMode = "light"
	audit.ThemePalette = "solarized"
	audit.Theme = "light"
	audit.DefaultTimeRange = "7d"
	audit.DefaultStatusFilter = "resolved"
	audit.DenseTables = true
	audit.DashboardRefreshIntervalSec = 300

	terminal := base
	terminal.ThemeMode = "dark"
	terminal.ThemePalette = "terminal"
	terminal.Theme = "dark"
	terminal.CompactUI = true
	terminal.DenseTables = true
	terminal.DefaultTimeRange = "1h"

	return []ConfigurationPreset{
		{ID: "sre-default", Name: "SRE Default", Description: "Balanced control-plane layout for daily operations.", BuiltIn: true, Settings: base},
		{ID: "on-call", Name: "On-Call Shift", Description: "Fast refresh, high contrast, alerts — optimized for incident response.", BuiltIn: true, Settings: onCall},
		{ID: "incident-command", Name: "Incident Command", Description: "Active failures only, compact density, Self-Healing landing.", BuiltIn: true, Settings: incident},
		{ID: "executive-review", Name: "Executive Review", Description: "Light theme, analytics open, FinOps-oriented time window.", BuiltIn: true, Settings: executive},
		{ID: "audit-review", Name: "Audit & Review", Description: "Resolved incidents, weekly window, dense tables for post-mortems.", BuiltIn: true, Settings: audit},
		{ID: "terminal-ops", Name: "Terminal Ops", Description: "Classic terminal palette with compact ops layout.", BuiltIn: true, Settings: terminal},
	}
}

func normalizeThemeMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "light", "dark", "system":
		return strings.ToLower(v)
	default:
		return "dark"
	}
}

func normalizeThemePalette(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "default", "ocean", "graphite", "ops", "terminal", "solarized":
		return strings.ToLower(v)
	default:
		return "default"
	}
}

func normalizeCurrency(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if len(v) != 3 {
		return "USD"
	}
	return v
}

func settingsFromPreferences(p UserPreferences) PresetSettings {
	mode := p.ThemeMode
	if mode == "" {
		mode = p.Theme
	}
	return PresetSettings{
		ThemeMode:                   normalizeThemeMode(mode),
		ThemePalette:                normalizeThemePalette(p.ThemePalette),
		Theme:                       p.Theme,
		DefaultStatusFilter:         p.DefaultStatusFilter,
		DefaultTimeRange:            p.DefaultTimeRange,
		CompactUI:                   p.CompactUI,
		SidebarCollapsedDefault:     p.SidebarCollapsedDefault,
		DashboardAutoRefresh:        p.DashboardAutoRefresh,
		DashboardRefreshIntervalSec: p.DashboardRefreshIntervalSec,
		DateFormat:                  p.DateFormat,
		Timezone:                    p.Timezone,
		DefaultLandingView:          p.DefaultLandingView,
		ReducedMotion:               p.ReducedMotion,
		HighContrast:                p.HighContrast,
		BrowserNotifications:        p.BrowserNotifications,
		ExpandIncidentCards:         p.ExpandIncidentCards,
		DenseTables:                 p.DenseTables,
		AnalyticsExpanded:           p.AnalyticsExpanded,
		Currency:                    normalizeCurrency(p.Currency),
		AlertSounds:                 p.AlertSounds,
	}
}

func applyPresetSettings(p *UserPreferences, s PresetSettings) {
	mode := normalizeThemeMode(s.ThemeMode)
	if mode == "" && s.Theme != "" {
		mode = normalizeThemeMode(s.Theme)
	}
	p.ThemeMode = mode
	p.ThemePalette = normalizeThemePalette(s.ThemePalette)
	if mode == "system" {
		p.Theme = "dark"
	} else {
		p.Theme = mode
	}
	p.DefaultStatusFilter = s.DefaultStatusFilter
	if p.DefaultStatusFilter != "all" && p.DefaultStatusFilter != "active" && p.DefaultStatusFilter != "resolved" {
		p.DefaultStatusFilter = "all"
	}
	p.DefaultTimeRange = normalizeTimeRangePreset(s.DefaultTimeRange)
	p.CompactUI = s.CompactUI
	p.SidebarCollapsedDefault = s.SidebarCollapsedDefault
	p.DashboardAutoRefresh = s.DashboardAutoRefresh
	p.DashboardRefreshIntervalSec = normalizeRefreshIntervalSec(s.DashboardRefreshIntervalSec)
	p.DateFormat = normalizeDateFormat(s.DateFormat)
	p.Timezone = normalizeTimezonePref(s.Timezone)
	p.DefaultLandingView = normalizeLandingView(s.DefaultLandingView)
	p.ReducedMotion = s.ReducedMotion
	p.HighContrast = s.HighContrast
	p.BrowserNotifications = s.BrowserNotifications
	p.ExpandIncidentCards = s.ExpandIncidentCards
	p.DenseTables = s.DenseTables
	p.AnalyticsExpanded = s.AnalyticsExpanded
	p.Currency = normalizeCurrency(s.Currency)
	p.AlertSounds = s.AlertSounds
}

func ensureConfigurationPresets(p *UserPreferences) {
	builtIns := builtInConfigurationPresets()
	byID := make(map[string]ConfigurationPreset, len(builtIns))
	for _, preset := range builtIns {
		byID[preset.ID] = preset
	}
	custom := make([]ConfigurationPreset, 0)
	for _, preset := range p.ConfigurationPresets {
		if preset.BuiltIn {
			continue
		}
		if strings.TrimSpace(preset.ID) == "" || strings.TrimSpace(preset.Name) == "" {
			continue
		}
		custom = append(custom, preset)
	}
	merged := make([]ConfigurationPreset, 0, len(builtIns)+len(custom))
	for _, preset := range builtIns {
		merged = append(merged, preset)
	}
	merged = append(merged, custom...)
	p.ConfigurationPresets = merged

	if p.ActivePresetID == "" {
		p.ActivePresetID = "sre-default"
	}
	if _, ok := byID[p.ActivePresetID]; !ok {
		for _, c := range custom {
			if c.ID == p.ActivePresetID {
				return
			}
		}
		p.ActivePresetID = "sre-default"
	}
	_ = byID
}

func migrateLegacyThemeFields(p *UserPreferences) {
	if p.ThemeMode == "" {
		if p.Theme == "light" || p.Theme == "dark" {
			p.ThemeMode = p.Theme
		} else {
			p.ThemeMode = "dark"
		}
	}
	p.ThemeMode = normalizeThemeMode(p.ThemeMode)
	p.ThemePalette = normalizeThemePalette(p.ThemePalette)
	if p.ThemeMode != "system" {
		p.Theme = p.ThemeMode
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
}

func findConfigurationPreset(presets []ConfigurationPreset, id string) (*ConfigurationPreset, bool) {
	for i := range presets {
		if presets[i].ID == id {
			return &presets[i], true
		}
	}
	return nil, false
}

func activateConfigurationPreset(prefs *UserPreferences, presetID string) error {
	preset, ok := findConfigurationPreset(prefs.ConfigurationPresets, presetID)
	if !ok {
		return fmt.Errorf("configuration preset not found")
	}
	applyPresetSettings(prefs, preset.Settings)
	prefs.ActivePresetID = presetID
	return nil
}
