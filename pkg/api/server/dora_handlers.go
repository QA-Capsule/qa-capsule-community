package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func registerDORARoutes(config *core.Config) {
	http.HandleFunc("/api/dora/metrics", jwtAuthMiddleware(config, "", managerOnlyHandler(config, handleDORAMetrics)))
	http.HandleFunc("/api/dora/signals", jwtAuthMiddleware(config, "", managerOnlyHandler(config, handleDORASignals)))
	http.HandleFunc("/api/webhooks/prometheus", handlePrometheusWebhook(config))
}

func handleDORAMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tw := incidentTimeWindowFromRequest(r)
	fromT, _ := time.Parse("2006-01-02 15:04:05", tw.From)
	toT, _ := time.Parse("2006-01-02 15:04:05", tw.To)
	if fromT.IsZero() || toT.IsZero() {
		fromT = time.Now().UTC().Add(-30 * 24 * time.Hour)
		toT = time.Now().UTC()
	}
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	metrics := core.ComputeDORAMetrics(project, fromT, toT)
	correlations := core.ListSignalCorrelations(project, 25)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics":      metrics,
		"correlations": correlations,
		"from":         tw.From,
		"to":           tw.To,
	})
}

func handleDORASignals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	signals, err := core.ListExternalSignals(r.Context(), project, 50)
	if err != nil {
		http.Error(w, "Failed to load signals", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"signals": signals})
}

func handlePrometheusWebhook(config *core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "Missing X-API-Key", http.StatusUnauthorized)
			return
		}
		project, err := core.GetProjectByAPIKey(apiKey)
		if err != nil {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		queryProject := strings.TrimSpace(r.URL.Query().Get("project"))
		if queryProject == "" {
			queryProject = project.Name
		}
		signalID, correlated, err := core.IngestPrometheusWebhook(queryProject, body)
		if err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		_ = config
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                "accepted",
			"signal_id":             signalID,
			"correlated_incidents":  correlated,
			"project":               queryProject,
		})
	}
}
