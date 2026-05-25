package core

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

// PipelineCrashDefaultName is used when a CI webhook reports failure without a test name.
const PipelineCrashDefaultName = "[PIPELINE CRASH] Execution Failed"

// IngestResult summarizes one processed alert.
type IngestResult struct {
	IncidentID  int64
	Skipped     bool
	Flaky       bool
	PerfAlert   bool
	Quarantined bool
	Fingerprint string
	FinalName   string
}

// ProcessAlert handles dedup, flaky detection, perf regression, DB insert, and plugins.
func ProcessAlert(cfg Config, projectName, runID string, alert UnifiedAlert, routing map[string]string, allowedPluginPaths map[string]bool) IngestResult {
	if DB == nil {
		slog.Error("database not initialized")
		return IngestResult{}
	}

	alert = EnsurePipelineCrashName(alert)

	if IsTestQuarantined(projectName, alert.Name) {
		identityFP := quarantine.TestIdentityFingerprint(projectName, alert.Name)
		// Keep execution metrics flowing so DORA / flaky oscillation logic can recover when the test goes green.
		RecordExecutionMetric(projectName, quarantine.NormalizeTestName(alert.Name), identityFP, alert.ExecutionTimeMs, alert.Status)
		recordQuarantinedIngest(projectName, runID, alert)
		slog.Info("ingest suppressed: test quarantined", "project", projectName, "test", alert.Name)
		return IngestResult{Skipped: true, Quarantined: true, Fingerprint: identityFP, FinalName: alert.Name}
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
		return IngestResult{Skipped: true, Fingerprint: fp, FinalName: alert.Name}
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
				identityFP := quarantine.TestIdentityFingerprint(projectName, alert.Name)
				RecordExecutionMetric(projectName, quarantine.NormalizeTestName(alert.Name), identityFP, alert.ExecutionTimeMs, alert.Status)
			}
			return IngestResult{Skipped: true, Fingerprint: fp, FinalName: alert.Name}
		}
	} else if isFlakyTest(projectName, alert.Name) {
		flaky = true
		finalName = "[FLAKY] " + quarantine.NormalizeTestName(alert.Name)
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
		identityFP := quarantine.TestIdentityFingerprint(projectName, alert.Name)
		RecordExecutionMetric(projectName, quarantine.NormalizeTestName(alert.Name), identityFP, alert.ExecutionTimeMs, status)
	}

	// Post-insert: tag all repeated failures (covers parallel ingest workers and name variants).
	if !isPassStatus(status) {
		if n := reconcileFlakyTagsForTest(projectName, alert.Name); n >= 2 {
			flaky = true
			finalName = "[FLAKY] " + quarantine.NormalizeTestName(alert.Name)
		}
	}

	alert.Name = finalName
	alert.Status = status

	// Remediation: active visual workflow DAG → WorkflowEngine; otherwise legacy AUTO-RUN plugins.
	RunPostIngestRemediation(cfg, projectName, alert, routing, allowedPluginPaths)

	PostIncidentHooks(id, projectName, runID, alert.CommitSHA, alert, flaky)

	return IngestResult{
		IncidentID:  id,
		Flaky:       flaky,
		PerfAlert:   perf,
		Fingerprint: fp,
		FinalName:   finalName,
	}
}

// isFlakyTest tags failures that oscillate pass/fail or repeat for the same test identity (48h window).
// Uses normalized test name — not incident fingerprint (name+error), so CI stack traces do not break detection.
func isFlakyTest(projectName, testName string) bool {
	if projectName == "" || strings.TrimSpace(testName) == "" {
		return false
	}
	if hasFlakyStateOscillation(projectName, testName) {
		return true
	}
	return hasPriorFailureForTest(projectName, testName)
}

// flakyNameVariants returns incident name values that refer to the same logical test.
func flakyNameVariants(testName string) (string, string, string) {
	norm := quarantine.NormalizeTestName(testName)
	return norm, "[FLAKY] " + norm, "[PERF] " + norm
}

// failureIncidentIDsForTest returns recent failure incident ids for the same logical test (normalized name).
func failureIncidentIDsForTest(projectName, testName string) []int64 {
	target := quarantine.NormalizeTestName(testName)
	if target == "" || projectName == "" || DB == nil {
		return nil
	}
	rows, err := DB.Query(`
		SELECT id, name FROM incidents
		WHERE project_name = ?
		AND created_at > datetime('now', '-48 hours')
		AND UPPER(COALESCE(status, '')) NOT IN ('PASSED', 'PASS')`,
		projectName)
	if err != nil {
		slog.Error("flaky failure scan failed", "error", err)
		return nil
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		if quarantine.NormalizeTestName(name) == target {
			ids = append(ids, id)
		}
	}
	return ids
}

// hasPriorFailureForTest is true when the test already failed once in the last 48h (this ingest is the 2nd+ failure).
func hasPriorFailureForTest(projectName, testName string) bool {
	return len(failureIncidentIDsForTest(projectName, testName)) >= 1
}

// reconcileFlakyTagsForTest marks all failure rows for this test as [FLAKY] when count >= 2 (fixes async worker races).
func reconcileFlakyTagsForTest(projectName, testName string) int {
	ids := failureIncidentIDsForTest(projectName, testName)
	if len(ids) < 2 {
		return 0
	}
	flakyName := "[FLAKY] " + quarantine.NormalizeTestName(testName)
	updated := 0
	for _, id := range ids {
		res, err := DB.Exec(`UPDATE incidents SET name = ? WHERE id = ? AND name NOT LIKE '[FLAKY]%'`,
			flakyName, id)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			updated++
		}
	}
	if updated > 0 {
		slog.Info("flaky tags reconciled", "project", projectName, "test", quarantine.NormalizeTestName(testName), "incidents", len(ids))
	}
	return len(ids)
}

func hasFlakyStateOscillation(projectName, testName string) bool {
	target := quarantine.NormalizeTestName(testName)
	if target == "" {
		return false
	}
	rows, err := DB.Query(`
		SELECT status, name FROM (
			SELECT status, test_name AS name, created_at FROM test_execution_metrics
			WHERE project_name = ?
			AND created_at > datetime('now', '-48 hours')
			UNION ALL
			SELECT status, name, created_at FROM incidents
			WHERE project_name = ?
			AND created_at > datetime('now', '-48 hours')
		)
		ORDER BY created_at ASC`,
		projectName, projectName)
	if err != nil {
		slog.Error("flaky oscillation check failed", "error", err)
		return false
	}
	defer rows.Close()

	var buckets []string
	for rows.Next() {
		var status, name string
		if err := rows.Scan(&status, &name); err != nil {
			continue
		}
		if quarantine.NormalizeTestName(name) != target {
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

// EnsurePipelineCrashName assigns a stable incident title when CI fails without test-level data.
func EnsurePipelineCrashName(a UnifiedAlert) UnifiedAlert {
	if strings.TrimSpace(a.Name) != "" {
		return a
	}
	if !isFailureStatus(a.Status) {
		return a
	}
	a.Name = PipelineCrashDefaultName
	if strings.TrimSpace(a.Error) == "" {
		a.Error = "Pipeline execution failed (no test name or failure details in webhook payload)."
	}
	return a
}

func enrichAlert(a UnifiedAlert) UnifiedAlert {
	a = EnsurePipelineCrashName(a)
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
