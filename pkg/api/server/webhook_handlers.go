package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func lastID(ids []int64) int64 {
	if len(ids) == 0 {
		return 0
	}
	return ids[len(ids)-1]
}

// registerWebhookRoutes binds the endpoints responsible for CI/CD telemetry ingestion
func registerWebhookRoutes(config *core.Config) {

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

		var projectName, slackChan, jiraKey, teamsHook string
		err := core.DB.QueryRow("SELECT name, slack_channel, jira_project_key, teams_webhook FROM projects WHERE api_key = ?", apiKey).Scan(&projectName, &slackChan, &jiraKey, &teamsHook)
		if err != nil {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		commitSHA := r.Header.Get("X-Commit-Sha")
		if commitSHA == "" {
			commitSHA = r.Header.Get("X-Git-Commit")
		}

		var alerts []core.UnifiedAlert

		if strings.HasSuffix(r.URL.Path, "/upload") {
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
			for i := range alerts {
				alerts[i].CommitSHA = commitSHA
			}
		} else {
			var rawPayload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
				http.Error(w, "Invalid JSON payload format", http.StatusBadRequest)
				return
			}
			if commitSHA == "" {
				if v, ok := rawPayload["commit_sha"].(string); ok {
					commitSHA = v
				}
			}
			alerts = core.ParseAlertsFromRaw(rawPayload)
			for i := range alerts {
				if alerts[i].CommitSHA == "" {
					alerts[i].CommitSHA = commitSHA
				}
			}
		}

		nonSkipped := 0
		for _, a := range alerts {
			if a.Status == "PASSED" || a.Status == "passed" {
				if a.ExecutionTimeMs > 0 {
					nonSkipped++
				}
			} else {
				nonSkipped++
			}
		}
		if nonSkipped == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "No actionable events detected."})
			return
		}

		runID := r.Header.Get("X-Run-Id")
		if runID == "" {
			runID = r.Header.Get("X-Pipeline-Run-Id")
		}
		if runID == "" {
			runID = fmt.Sprintf("run-%d", time.Now().UnixNano())
		}
		branch := r.Header.Get("X-Branch")
		if branch == "" {
			branch = r.Header.Get("X-Git-Branch")
		}

		alertContext, allowedPlugins := core.ProjectAlertContext(projectName)
		if alertContext == nil {
			alertContext = make(map[string]string)
		}
		if slackChan != "" {
			alertContext["SLACK_CHANNEL"] = slackChan
		}
		if jiraKey != "" {
			alertContext["JIRA_PROJECT_KEY"] = jiraKey
		}
		if teamsHook != "" {
			alertContext["TEAMS_WEBHOOK_URL"] = teamsHook
		}

		processed := 0
		var incidentIDs []int64
		for _, alert := range alerts {
			res := core.ProcessAlert(*config, projectName, runID, alert, alertContext, allowedPlugins)
			if !res.Skipped {
				processed++
				if res.IncidentID > 0 {
					incidentIDs = append(incidentIDs, res.IncidentID)
				}
			}
		}

		outcome := "success"
		if processed > 0 {
			outcome = "failure"
		}
		core.RecordPipelineRun(projectName, runID, commitSHA, branch, outcome)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "success",
			"failures_processed": processed,
			"incident_ids":       incidentIDs,
			"last_incident_id":   lastID(incidentIDs),
			"project":            projectName,
			"pipeline_run_id":    runID,
		})
	})
}
