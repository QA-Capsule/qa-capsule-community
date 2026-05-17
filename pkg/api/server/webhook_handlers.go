package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

// registerWebhookRoutes binds the endpoints responsible for CI/CD telemetry ingestion
func registerWebhookRoutes(config *core.Config) {

	// ==========================================
	// UNIVERSAL WEBHOOK GATEWAY (XML & JSON)
	// ==========================================
	http.HandleFunc("/api/webhooks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
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

		var alerts []core.UnifiedAlert

		// 1. DYNAMIC INGESTION ROUTING
		if strings.HasSuffix(r.URL.Path, "/upload") {
			// Handle Multipart XML Uploads (Playwright, Cypress, PyTest, etc.)
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

			framework := r.URL.Query().Get("framework")
			alerts = core.ParseJUnitXML(fileBytes, framework)

		} else {
			// Handle Generic JSON Webhooks (GitHub Actions, GitLab CI scripts)
			var rawPayload map[string]interface{}
			err = json.NewDecoder(r.Body).Decode(&rawPayload)
			if err != nil {
				http.Error(w, "Invalid JSON payload format", http.StatusBadRequest)
				return
			}
			alerts = append(alerts, core.NormalizePayload(rawPayload))
		}

		if len(alerts) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "No failed tests detected."})
			return
		}

		// 2. PROCESS EXTRACTED FAILURES
		for _, alert := range alerts {
			rawString := fmt.Sprintf("%s|%s", alert.Name, alert.Error)
			fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(rawString)))

			// --- SRE CRITICAL FIX: SMART CORRELATION & FLAKY DETECTION ---
			// We no longer suppress new executions. Every pipeline run is recorded so the UI can group them.

			// A. Anti-Spam Check: Prevent exact duplicate uploads within 2 minutes
			var recentCount int
			core.DB.QueryRow("SELECT COUNT(*) FROM incidents WHERE fingerprint = ? AND project_name = ? AND created_at > datetime('now', '-2 minutes')",
				fingerprint, projectName).Scan(&recentCount)

			if recentCount > 0 {
				log.Printf("[CORRELATION] Incident fingerprint %s uploaded twice within 2 mins. Skipping spam.", fingerprint)
				continue
			}

			// B. Flakiness Detection: Did this exact test fail, get resolved, and fail again within 48 hours?
			var flakyCount int
			core.DB.QueryRow("SELECT COUNT(*) FROM incidents WHERE fingerprint = ? AND project_name = ? AND is_resolved = 1 AND created_at > datetime('now', '-48 hours')",
				fingerprint, projectName).Scan(&flakyCount)

			finalName := alert.Name
			if flakyCount > 0 {
				finalName = "[FLAKY] " + alert.Name
				log.Printf("[FLAKY DETECTED] %s failed again within 48h of resolution.", alert.Name)
			}

			// C. Insert new incident into DB
			// FIX: Added the missing 'error_logs' column to persist stacktraces correctly!
			_, err = core.DB.Exec(`INSERT INTO incidents (project_name, name, status, error_message, console_logs, error_logs, fingerprint, is_resolved, created_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)`,
				projectName, finalName, alert.Status, alert.Error, alert.ConsoleLogs, alert.ErrorLogs, fingerprint)

			if err != nil {
				log.Printf("[DB ERROR] Failed to insert incident: %v", err)
			}

			// 3. TRIGGER AUTOMATED PLAYBOOKS / PLUGINS
			alertContext := map[string]string{
				"SLACK_CHANNEL":     slackChan,
				"JIRA_PROJECT_KEY":  jiraKey,
				"TEAMS_WEBHOOK_URL": teamsHook,
			}
			alert.Name = finalName
			go core.EvaluateAlertRules(*config, alert, alertContext)
		}

		// Return success response to CI runner
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "success",
			"failures_processed": len(alerts),
			"project":            projectName,
		})
	})
}
