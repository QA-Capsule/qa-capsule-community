package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// UserPreferences holds per-user UI and dashboard defaults.
type UserPreferences struct {
	Theme                       string          `json:"theme"`
	DefaultStatusFilter         string          `json:"default_status_filter"`
	AnalyticsExpanded           bool            `json:"analytics_expanded"`
	AnalyticsLayout             json.RawMessage `json:"analytics_layout,omitempty"`
	DefaultTimeRange            string          `json:"default_time_range"`
	CompactUI                   bool            `json:"compact_ui"`
	SidebarCollapsedDefault     bool            `json:"sidebar_collapsed_default"`
	DashboardAutoRefresh        bool            `json:"dashboard_auto_refresh"`
	DashboardRefreshIntervalSec int             `json:"dashboard_refresh_interval_sec"`
	DateFormat                  string          `json:"date_format"`
	Timezone                    string          `json:"timezone"`
	DefaultLandingView          string          `json:"default_landing_view"`
	ReducedMotion               bool            `json:"reduced_motion"`
	HighContrast                bool            `json:"high_contrast"`
	BrowserNotifications        bool            `json:"browser_notifications"`
	ExpandIncidentCards         bool            `json:"expand_incident_cards"`
	DenseTables                 bool            `json:"dense_tables"`
}

func defaultUserPreferences() UserPreferences {
	return UserPreferences{
		Theme:                       "dark",
		DefaultStatusFilter:         "all",
		AnalyticsExpanded:           false,
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
	}
}

func userIDByUsername(username string) (int, error) {
	var id int
	err := core.DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&id)
	return id, err
}

func loadUserPreferences(userID int) UserPreferences {
	prefs := defaultUserPreferences()
	var raw sql.NullString
	err := core.DB.QueryRow("SELECT prefs_json FROM user_preferences WHERE user_id = ?", userID).Scan(&raw)
	if err != nil || !raw.Valid || raw.String == "" {
		return prefs
	}
	if json.Unmarshal([]byte(raw.String), &prefs) != nil {
		return defaultUserPreferences()
	}
	if prefs.Theme != "dark" && prefs.Theme != "light" {
		prefs.Theme = "dark"
	}
	if prefs.DefaultStatusFilter != "all" && prefs.DefaultStatusFilter != "active" && prefs.DefaultStatusFilter != "resolved" {
		prefs.DefaultStatusFilter = "all"
	}
	prefs.DefaultTimeRange = normalizeTimeRangePreset(prefs.DefaultTimeRange)
	prefs.DateFormat = normalizeDateFormat(prefs.DateFormat)
	prefs.Timezone = normalizeTimezonePref(prefs.Timezone)
	prefs.DefaultLandingView = normalizeLandingView(prefs.DefaultLandingView)
	prefs.DashboardRefreshIntervalSec = normalizeRefreshIntervalSec(prefs.DashboardRefreshIntervalSec)
	return prefs
}

func normalizeTimeRangePreset(v string) string {
	switch v {
	case "5m", "15m", "30m", "1h", "6h", "24h", "7d", "30d", "today", "yesterday", "all":
		return v
	default:
		return "15m"
	}
}

func normalizeDateFormat(v string) string {
	switch v {
	case "locale", "short", "iso":
		return v
	default:
		return "locale"
	}
}

func normalizeTimezonePref(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "auto" {
		return "auto"
	}
	return v
}

func normalizeLandingView(v string) string {
	switch v {
	case "dashboard", "ingestion", "finops", "dora", "plugins", "rca", "quarantine", "runbooks", "about":
		return v
	default:
		return "dashboard"
	}
}

func normalizeRefreshIntervalSec(sec int) int {
	switch sec {
	case 15, 30, 60, 120, 300:
		return sec
	}
	if sec < 15 {
		return 15
	}
	if sec > 300 {
		return 300
	}
	return 60
}

func saveUserPreferences(userID int, prefs UserPreferences) error {
	b, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	_, err = core.DB.Exec(`
		INSERT INTO user_preferences (user_id, prefs_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET prefs_json = excluded.prefs_json, updated_at = excluded.updated_at`,
		userID, string(b), time.Now().UTC().Format(time.RFC3339))
	return err
}

func mergePreferences(current UserPreferences, patch map[string]interface{}) UserPreferences {
	if v, ok := patch["theme"].(string); ok && (v == "dark" || v == "light") {
		current.Theme = v
	}
	if v, ok := patch["default_status_filter"].(string); ok {
		switch v {
		case "all", "active", "resolved":
			current.DefaultStatusFilter = v
		}
	}
	if v, ok := patch["analytics_expanded"].(bool); ok {
		current.AnalyticsExpanded = v
	}
	if v, ok := patch["analytics_layout"]; ok {
		if b, err := json.Marshal(v); err == nil {
			current.AnalyticsLayout = b
		}
	}
	if v, ok := patch["default_time_range"].(string); ok {
		current.DefaultTimeRange = normalizeTimeRangePreset(v)
	}
	if v, ok := patch["compact_ui"].(bool); ok {
		current.CompactUI = v
	}
	if v, ok := patch["sidebar_collapsed_default"].(bool); ok {
		current.SidebarCollapsedDefault = v
	}
	if v, ok := patch["dashboard_auto_refresh"].(bool); ok {
		current.DashboardAutoRefresh = v
	}
	if v, ok := patch["dashboard_refresh_interval_sec"].(float64); ok {
		current.DashboardRefreshIntervalSec = normalizeRefreshIntervalSec(int(v))
	}
	if v, ok := patch["date_format"].(string); ok {
		current.DateFormat = normalizeDateFormat(v)
	}
	if v, ok := patch["timezone"].(string); ok {
		current.Timezone = normalizeTimezonePref(v)
	}
	if v, ok := patch["default_landing_view"].(string); ok {
		current.DefaultLandingView = normalizeLandingView(v)
	}
	if v, ok := patch["reduced_motion"].(bool); ok {
		current.ReducedMotion = v
	}
	if v, ok := patch["high_contrast"].(bool); ok {
		current.HighContrast = v
	}
	if v, ok := patch["browser_notifications"].(bool); ok {
		current.BrowserNotifications = v
	}
	if v, ok := patch["expand_incident_cards"].(bool); ok {
		current.ExpandIncidentCards = v
	}
	if v, ok := patch["dense_tables"].(bool); ok {
		current.DenseTables = v
	}
	return current
}

// registerPreferencesRoutes binds personal account and preference endpoints.
func registerPreferencesRoutes(config *core.Config) {
	http.HandleFunc("/api/me", jwtAuthMiddleware(config, "", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims := claimsFromRequest(r)
		if claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := userIDByUsername(claims.Username)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			var fullname, role string
			if err := core.DB.QueryRow("SELECT fullname, role FROM users WHERE id = ?", userID).Scan(&fullname, &role); err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"username":    claims.Username,
				"fullname":    fullname,
				"role":        role,
				"preferences": loadUserPreferences(userID),
			})

		case http.MethodPut:
			var body struct {
				Fullname string `json:"fullname"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			fullname := strings.TrimSpace(body.Fullname)
			if fullname == "" {
				http.Error(w, "Full name is required", http.StatusBadRequest)
				return
			}
			if _, err := core.DB.Exec("UPDATE users SET fullname = ? WHERE id = ?", fullname, userID); err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/api/me/preferences", jwtAuthMiddleware(config, "", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims := claimsFromRequest(r)
		if claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := userIDByUsername(claims.Username)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(loadUserPreferences(userID))

		case http.MethodPut:
			var patch map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			merged := mergePreferences(loadUserPreferences(userID), patch)
			if err := saveUserPreferences(userID, merged); err != nil {
				log.Println("[ERROR] save user preferences:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(merged)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/api/me/password", jwtAuthMiddleware(config, "", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims := claimsFromRequest(r)
		if claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if len(body.NewPassword) < 8 {
			http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		var hash string
		if err := core.DB.QueryRow("SELECT password_hash FROM users WHERE username = ?", claims.Username).Scan(&hash); err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.CurrentPassword)) != nil {
			http.Error(w, "Current password is incorrect", http.StatusForbidden)
			return
		}

		hashed, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
		if _, err := core.DB.Exec("UPDATE users SET password_hash = ?, require_password_change = 0 WHERE username = ?",
			string(hashed), claims.Username); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func claimsFromRequest(r *http.Request) *Claims {
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &Claims{}
	if _, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	}); err != nil {
		return nil
	}
	return claims
}
