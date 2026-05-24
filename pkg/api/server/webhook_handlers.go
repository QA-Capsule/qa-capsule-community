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

func executionFlagsFromRequest(r *http.Request, raw map[string]interface{}) core.ExecutionFlags {
	flags := core.ExecutionFlagsFromPayload(raw)
	if v := strings.TrimSpace(r.Header.Get("X-Execution-Env")); v != "" {
		flags.Env = core.NormalizeExecutionEnv(v)
	}
	if v := strings.TrimSpace(r.Header.Get("X-Execution-Type")); v != "" {
		flags.Type = core.NormalizeExecutionType(v)
	}
	return flags
}

// pipelineOutcomeFromReport marks the run failed when JUnit/XML reports failures, even if incidents were deduplicated.
func pipelineOutcomeFromReport(processed int, report core.UnifiedExecutionReport) string {
	if processed > 0 {
		return "failure"
	}
	if report.Summary.Failed > 0 || report.Summary.Flaky > 0 {
		return "failure"
	}
	for _, tc := range report.Tests {
		st := strings.ToLower(strings.TrimSpace(tc.Status))
		if st == "fail" || st == "flaky" {
			return "failure"
		}
	}
	return "success"
}

func appendIngested(ingested []core.IngestedCase, res core.IngestResult) []core.IngestedCase {
	if res.IncidentID <= 0 {
		return ingested
	}
	return append(ingested, core.IngestedCase{
		Fingerprint: res.Fingerprint,
		FinalName:   res.FinalName,
		IncidentID:  res.IncidentID,
		Flaky:       res.Flaky,
	})
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
		var report core.UnifiedExecutionReport
		var rawPayload map[string]interface{}
		framework := r.URL.Query().Get("framework")
		format := core.DetectReportFormat(r.URL.Path, r.Header.Get("Content-Type"))

		switch format {
		case core.ReportFormatJUnitXML:
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
			flags := executionFlagsFromRequest(r, nil)
			norm := core.DefaultUnifiedReporter.Normalize(core.IngestPayload{
				Format:    core.ReportFormatJUnitXML,
				Framework: framework,
				Flags:     flags,
				XML:       fileBytes,
			})
			alerts = norm.Failures
			report = norm.Report
		default:
			if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
				http.Error(w, "Invalid JSON payload format", http.StatusBadRequest)
				return
			}
			if commitSHA == "" {
				if v, ok := rawPayload["commit_sha"].(string); ok {
					commitSHA = v
				}
			}
			flags := executionFlagsFromRequest(r, rawPayload)
			norm := core.DefaultUnifiedReporter.Normalize(core.IngestPayload{
				Format:    core.ReportFormatJSON,
				Framework: framework,
				Flags:     flags,
				JSON:      rawPayload,
			})
			report = norm.Report
			alerts = norm.Failures
		}

		for i := range alerts {
			if alerts[i].CommitSHA == "" {
				alerts[i].CommitSHA = commitSHA
			}
		}

		if len(alerts) == 0 && len(report.Tests) == 0 {
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

		flags := executionFlagsFromRequest(r, rawPayload)
		report.Flags = core.ExecutionFlags{
			Env:  flags.Env,
			Type: flags.Type,
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
		quarantined := 0
		var incidentIDs []int64
		var ingested []core.IngestedCase
		for _, alert := range alerts {
			res := core.ProcessAlert(*config, projectName, runID, alert, alertContext, allowedPlugins)
			ingested = appendIngested(ingested, res)
			if res.Quarantined {
				quarantined++
				continue
			}
			if !res.Skipped {
				processed++
				if res.IncidentID > 0 {
					incidentIDs = append(incidentIDs, res.IncidentID)
				}
			}
		}

		outcome := pipelineOutcomeFromReport(processed, report)

		ctx := core.IngestExecutionContext{
			ProjectName:   projectName,
			PipelineRunID: runID,
			CommitSHA:     commitSHA,
			Branch:        branch,
			Flags:         flags,
		}
		_ = core.FinalizePipelineExecution(ctx, outcome, report, ingested)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":              "success",
			"failures_processed":  processed,
			"quarantined_skipped": quarantined,
			"incident_ids":        incidentIDs,
			"last_incident_id":    lastID(incidentIDs),
			"project":             projectName,
			"pipeline_run_id":     runID,
		})
	})
}
