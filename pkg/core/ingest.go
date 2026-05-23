package core

import (
	"fmt"
	"log/slog"
)

// IngestResult summarizes one processed alert.
type IngestResult struct {
	IncidentID int64
	Skipped    bool
	Flaky      bool
	PerfAlert  bool
}

// ProcessAlert handles dedup, flaky detection, perf regression, DB insert, and plugins.
func ProcessAlert(cfg Config, projectName, runID string, alert UnifiedAlert, routing map[string]string, allowedPluginPaths map[string]bool) IngestResult {
	fp := IncidentFingerprint(alert.Name, alert.Error)

	var recentCount int
	DB.QueryRow(`SELECT COUNT(*) FROM incidents WHERE fingerprint = ? AND project_name = ? AND pipeline_run_id = ?`,
		fp, projectName, runID).Scan(&recentCount)
	if recentCount > 0 {
		slog.Info("correlation duplicate skipped", "test", alert.Name, "run", runID)
		return IngestResult{Skipped: true}
	}

	status := alert.Status
	finalName := alert.Name
	flaky := false
	perf := false

	if alert.Status == "PASSED" || alert.Status == "passed" {
		if alert.ExecutionTimeMs > 0 && IsPerformanceDegraded(projectName, fp, alert.ExecutionTimeMs) {
			perf = true
			finalName = "[PERF] " + alert.Name
			status = "PERF_DEGRADATION"
			alert.Error = FormatPerfMessage(alert.Name, alert.ExecutionTimeMs)
			slog.Warn("performance degradation detected", "test", alert.Name, "ms", alert.ExecutionTimeMs)
		} else {
			if alert.ExecutionTimeMs > 0 {
				RecordExecutionMetric(projectName, alert.Name, fp, alert.ExecutionTimeMs, alert.Status)
			}
			return IngestResult{Skipped: true}
		}
	} else {
		var flakyCount int
		DB.QueryRow(`SELECT COUNT(*) FROM incidents WHERE fingerprint = ? AND project_name = ? AND is_resolved = 1 AND created_at > datetime('now', '-48 hours')`,
			fp, projectName).Scan(&flakyCount)
		if flakyCount > 0 {
			flaky = true
			finalName = "[FLAKY] " + alert.Name
			slog.Info("flaky test detected", "test", alert.Name)
		}
	}

	jiraKey := alert.JiraIssueKey
	if jiraKey == "" {
		jiraKey = ExtractJiraIssueKey(alert.Name)
	}
	if jiraKey != "" && routing != nil {
		routing["JIRA_ISSUE_KEY"] = jiraKey
		if routing["JIRA_PROJECT_KEY"] == "" {
			if idx := indexDash(jiraKey); idx > 0 {
				routing["JIRA_PROJECT_KEY"] = jiraKey[:idx]
			}
		}
	}

	res, err := DB.Exec(`INSERT INTO incidents (
		project_name, name, status, error_message, console_logs, error_logs, fingerprint, pipeline_run_id,
		browser, os, viewport, execution_time_ms, jira_issue_key, is_resolved, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)`,
		projectName, finalName, status, alert.Error, alert.ConsoleLogs, alert.ErrorLogs, fp, runID,
		alert.Browser, alert.OS, alert.Viewport, nullIfZero(alert.ExecutionTimeMs), nullString(jiraKey))
	if err != nil {
		slog.Error("incident insert failed", "error", err)
		return IngestResult{}
	}
	id, _ := res.LastInsertId()

	if alert.ExecutionTimeMs > 0 {
		RecordExecutionMetric(projectName, finalName, fp, alert.ExecutionTimeMs, status)
	}

	alert.Name = finalName
	go EvaluateAlertRules(cfg, projectName, alert, routing, allowedPluginPaths)
<<<<<<< HEAD
	PostIncidentHooks(id, projectName, runID, alert.CommitSHA, alert, flaky)
=======
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb

	return IngestResult{IncidentID: id, Flaky: flaky, PerfAlert: perf}
}

func nullIfZero(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func indexDash(s string) int {
	for i, c := range s {
		if c == '-' {
			return i
		}
	}
	return -1
}

// NormalizePayloads supports single object or batch "tests" array.
func NormalizePayloads(raw map[string]interface{}) []UnifiedAlert {
	if tests, ok := raw["tests"].([]interface{}); ok {
		var out []UnifiedAlert
		for _, t := range tests {
			if m, ok := t.(map[string]interface{}); ok {
				out = append(out, enrichAlert(NormalizePayload(m)))
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []UnifiedAlert{enrichAlert(NormalizePayload(raw))}
}

func enrichAlert(a UnifiedAlert) UnifiedAlert {
	if a.JiraIssueKey == "" {
		a.JiraIssueKey = ExtractJiraIssueKey(a.Name)
	}
	return a
}

// ParseAlertsFromRaw is used by webhooks after JSON decode.
func ParseAlertsFromRaw(raw map[string]interface{}) []UnifiedAlert {
	return NormalizePayloads(raw)
}

// FormatPerfMessage builds a human-readable perf alert body.
func FormatPerfMessage(name string, ms int64) string {
	return fmt.Sprintf("Test %q passed but execution time %dms exceeds 150%% of its 30-day average.", name, ms)
}
