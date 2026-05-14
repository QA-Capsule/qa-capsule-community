package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"qacapsule/internal/core"
)

// registerWebhookRoutes binds the endpoints responsible for CI/CD telemetry ingestion
func registerWebhookRoutes(config *core.Config) {

	// ==========================================
	// DIRECT XML FILE UPLOAD (The Modern API Approach)
	// ==========================================
	http.HandleFunc("/api/webhooks/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		framework := r.URL.Query().Get("framework")

		if apiKey == "" {
			http.Error(w, "Missing X-API-Key Header", http.StatusUnauthorized)
			return
		}

		// Verify API Key and fetch Project routing details
		var projectName, slackChan, jiraKey, teamsHook string
		err := core.DB.QueryRow("SELECT name, slack_channel, jira_project_key, teams_webhook FROM projects WHERE api_key = ?", apiKey).Scan(&projectName, &slackChan, &jiraKey, &teamsHook)

		if err != nil {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		// Parse Multipart Form to extract the XML file
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "File upload too large", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to retrieve the file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}

		// Delegate parsing to Core Parser
		alerts := core.ParseJUnitXML(fileBytes, framework)

		if len(alerts) == 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "No failed tests detected."})
			return
		}

		// Process each extracted failure
		for _, alert := range alerts {
			rawString := fmt.Sprintf("%s|%s", alert.Name, alert.Error)
			fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(rawString)))

			var existingID int
			err = core.DB.QueryRow("SELECT id FROM incidents WHERE fingerprint = ? AND project_name = ? AND is_resolved = 0",
				fingerprint, projectName).Scan(&existingID)

			// FIX: Duplicate suppression
			if err == nil {
				// DO NOT overwrite 'created_at'. The MTTR clock must start from the very first failure.
				// We simply skip creating a new incident to prevent spam.
				log.Printf("[CORRELATION] Incident %d is already open. Skipping duplicate.", existingID)
				continue
			}

			// Flakiness Detection
			var previousCount int
			core.DB.QueryRow("SELECT COUNT(*) FROM incidents WHERE fingerprint = ? AND project_name = ? AND is_resolved = 1 AND created_at > datetime('now', '-48 hours')",
				fingerprint, projectName).Scan(&previousCount)

			finalName := alert.Name
			if previousCount > 0 {
				finalName = "[FLAKY] " + alert.Name
			}

			// Explicitly inject CURRENT_TIMESTAMP into created_at
			_, _ = core.DB.Exec(`INSERT INTO incidents (project_name, name, status, error_message, console_logs, fingerprint, is_resolved, created_at) 
				VALUES (?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)`,
				projectName, finalName, alert.Status, alert.Error, alert.ConsoleLogs, fingerprint)

			// Trigger plugins asynchronously
			alertContext := map[string]string{
				"SLACK_CHANNEL":     slackChan,
				"JIRA_PROJECT_KEY":  jiraKey,
				"TEAMS_WEBHOOK_URL": teamsHook,
			}
			alert.Name = finalName
			go core.EvaluateAlertRules(*config, alert, alertContext)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "success",
			"failures_processed": len(alerts),
			"project":            projectName,
		})
	})
}
