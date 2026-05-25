package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

// registerQuarantineRoutes binds CI gate and dashboard quarantine APIs.
func registerQuarantineRoutes(config *core.Config) {
	_ = config
	http.HandleFunc("/api/ci/quarantine", handleCIQuarantine)
	http.HandleFunc("/api/ci/quarantine/status", handleCIQuarantineStatus)
	http.HandleFunc("/api/quarantine/status", handleCIQuarantineStatus)
	http.HandleFunc("/api/quarantine", jwtAuthMiddleware(config, "", handleQuarantineManage))
}

func projectNameFromAPIKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey == "" {
		http.Error(w, "Missing X-API-Key", http.StatusUnauthorized)
		return "", false
	}
	var projectName string
	if err := core.DB.QueryRow("SELECT name FROM projects WHERE api_key = ?", apiKey).Scan(&projectName); err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return "", false
	}
	return projectName, true
}

// handleCIQuarantineStatus — GET /api/ci/quarantine/status?hash=… or ?test=…
// Used by GitHub Actions / GitLab before running a test.
func handleCIQuarantineStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projectName, ok := projectNameFromAPIKey(w, r)
	if !ok {
		return
	}
	if core.QuarantineEngine == nil {
		writeJSONError(w, "Quarantine not available", http.StatusServiceUnavailable)
		return
	}

	hash := strings.TrimSpace(r.URL.Query().Get("hash"))
	testName := strings.TrimSpace(r.URL.Query().Get("test"))
	if testName == "" {
		testName = strings.TrimSpace(r.URL.Query().Get("name"))
	}

	resp, err := core.QuarantineEngine.CheckCIStatus(r.Context(), projectName, hash, testName)
	if err != nil {
		if err == quarantine.ErrMissingCIIdentifier {
			writeJSONError(w, "Provide query param hash (fingerprint) or test (test name)", http.StatusBadRequest)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// handleCIQuarantine — GET lists all active quarantined tests for the project API key.
func handleCIQuarantine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projectName, ok := projectNameFromAPIKey(w, r)
	if !ok {
		return
	}
	if core.QuarantineEngine == nil {
		writeJSONError(w, "Quarantine not available", http.StatusServiceUnavailable)
		return
	}
	resp, err := core.QuarantineEngine.ListCI(r.Context(), projectName)
	if err != nil {
		writeJSONError(w, "Failed to load quarantine", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
