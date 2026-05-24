package core

import (
	"fmt"
	"log/slog"
	"strings"
)

// IngestResult summarizes one processed alert.
type IngestResult struct {
	IncidentID  int64
	Skipped     bool
	Flaky       bool
	PerfAlert   bool
	Quarantined bool
}

// ProcessAlert handles dedup, flaky detection, perf regression, DB insert, and plugins.
func ProcessAlert(cfg Config, projectName, runID string, alert UnifiedAlert, routing map[string]string, allowedPluginPaths map[string]bool) IngestResult {
	if DB == nil {
		slog.Error("database not initialized")
		return IngestResult{}
	}

	if IsTestQuarantined(projectName, alert.Name) {
		fp := IncidentFingerprint(alert.Name, alert.Error)
		// Keep execution metrics flowing so DORA / flaky oscillation logic can recover when the test goes green.
		RecordExecutionMetric(projectName, alert.Name, fp, alert.ExecutionTimeMs, alert.Status)
		recordQuarantinedIngest(projectName, runID, alert)
		slog.Info("ingest suppressed: test quarantined", "project", projectName, "test", alert.Name)
		return IngestResult{Skipped: true, Quarantined: true}
	}

	fp := IncidentFingerprint(alert.Name, alert.Error)

	var recentCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM incidents WHERE fingerprint = ? AND project_name = ? AND pipeline_run_id = ?`,
		fp, projectName, runID).Scan(&recentCount); err != nil {
		slog.Error("dedup check failed", "error", err, "test", alert.Name)
		return IngestResult{}
	}
	if recentCount > 0 {
		slog.Info("correlation duplicate skipped", "test", alert.Name, "run", runID)
		return IngestResult{Skipped: true}
	}

	status := alert.Status
	finalName := alert.Name
	flaky := false
	perf := false

	if isPassStatus(alert.Status) {
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
	} else if isFlakyTest(projectName, fp, runID) {
		flaky = true
		finalName = "[FLAKY] " + alert.Name
		slog.Info("flaky test detected", "test", alert.Name, "project", projectName)
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
	alert.Status = status

	// Remediation: active visual workflow DAG → WorkflowEngine; otherwise legacy AUTO-RUN plugins.
	RunPostIngestRemediation(cfg, projectName, alert, routing, allowedPluginPaths)

	PostIncidentHooks(id, projectName, runID, alert.CommitSHA, alert, flaky)

	return IngestResult{IncidentID: id, Flaky: flaky, PerfAlert: perf}
}

// isFlakyTest tags failures that oscillate pass/fail or fail across multiple pipeline runs (48h window).
// Does not depend on is_resolved — human triage is not required.
func isFlakyTest(projectName, fingerprint, currentRunID string) bool {
	if fingerprint == "" || projectName == "" {
		return false
	}
	if hasFlakyStateOscillation(projectName, fingerprint) {
		return true
	}
	return hasMultipleFailureRuns(projectName, fingerprint, currentRunID)
}

func hasMultipleFailureRuns(projectName, fingerprint, currentRunID string) bool {
	const q = `
		SELECT COUNT(DISTINCT pipeline_run_id) FROM incidents
		WHERE fingerprint = ? AND project_name = ?
		AND created_at > datetime('now', '-48 hours')
		AND TRIM(COALESCE(pipeline_run_id, '')) != ''
		AND UPPER(COALESCE(status, '')) NOT IN ('PASSED', 'PASS')`

	var distinctRuns int
	if err := DB.QueryRow(q, fingerprint, projectName).Scan(&distinctRuns); err != nil {
		slog.Error("flaky multi-run check failed", "error", err)
		return false
	}
	if distinctRuns >= 2 {
		return true
	}
	if distinctRuns == 1 && strings.TrimSpace(currentRunID) != "" {
		var inSameRun int
		err := DB.QueryRow(`
			SELECT COUNT(*) FROM incidents
			WHERE fingerprint = ? AND project_name = ? AND pipeline_run_id = ?
			AND created_at > datetime('now', '-48 hours')
			AND UPPER(COALESCE(status, '')) NOT IN ('PASSED', 'PASS')`,
			fingerprint, projectName, currentRunID).Scan(&inSameRun)
		if err != nil {
			slog.Error("flaky run-id check failed", "error", err)
			return false
		}
		return inSameRun == 0
	}
	return false
}

func hasFlakyStateOscillation(projectName, fingerprint string) bool {
	rows, err := DB.Query(`
		SELECT status FROM (
			SELECT status, created_at FROM test_execution_metrics
			WHERE project_name = ? AND fingerprint = ?
			AND created_at > datetime('now', '-48 hours')
			UNION ALL
			SELECT status, created_at FROM incidents
			WHERE project_name = ? AND fingerprint = ?
			AND created_at > datetime('now', '-48 hours')
		)
		ORDER BY created_at ASC`,
		projectName, fingerprint, projectName, fingerprint)
	if err != nil {
		slog.Error("flaky oscillation check failed", "error", err)
		return false
	}
	defer rows.Close()

	var buckets []string
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			continue
		}
		b := statusBucket(status)
		if b == "" {
			continue
		}
		if len(buckets) == 0 || buckets[len(buckets)-1] != b {
			buckets = append(buckets, b)
		}
	}
	return hasPassFailPassSequence(buckets)
}

func statusBucket(status string) string {
	if isPassStatus(status) {
		return "pass"
	}
	s := strings.ToUpper(strings.TrimSpace(status))
	if s == "" || s == "PERF_DEGRADATION" {
		return ""
	}
	return "fail"
}

func hasPassFailPassSequence(buckets []string) bool {
	for i := 0; i+2 < len(buckets); i++ {
		a, b, c := buckets[i], buckets[i+1], buckets[i+2]
		if a == "pass" && b == "fail" && c == "pass" {
			return true
		}
		if a == "fail" && b == "pass" && c == "fail" {
			return true
		}
	}
	return false
}

func isPassStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PASSED", "PASS":
		return true
	default:
		return false
	}
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
