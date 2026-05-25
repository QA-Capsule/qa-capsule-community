package core

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

// IncidentTelemetry is the canonical incident view for MCP and self-healing (framework-agnostic).
type IncidentTelemetry struct {
	ID              int64             `json:"incident_id"`
	ProjectName     string            `json:"project_name"`
	TestName        string            `json:"test_name"`
	Status          string            `json:"status"`
	ErrorMessage    string            `json:"error_message"`
	StackTrace      string            `json:"stack_trace"`
	ConsoleLogs     string            `json:"console_logs,omitempty"`
	Fingerprint     string            `json:"fingerprint"`
	IdentitySHA256  string            `json:"identity_fingerprint_sha256"`
	PipelineRunID   string            `json:"pipeline_run_id,omitempty"`
	ExecutionTimeMs int64             `json:"execution_time_ms,omitempty"`
	Browser         string            `json:"browser,omitempty"`
	OS              string            `json:"os,omitempty"`
	Viewport        string            `json:"viewport,omitempty"`
	CITags          map[string]string `json:"ci_tags"`
	CreatedAt       string            `json:"created_at,omitempty"`
}

// FlakyTestTelemetry describes unstable tests for MCP clients (no framework coupling).
type FlakyTestTelemetry struct {
	ProjectName        string  `json:"project_name"`
	TestName           string  `json:"test_name"`
	IdentitySHA256     string  `json:"identity_fingerprint_sha256"`
	IncidentFingerprint string `json:"incident_fingerprint,omitempty"`
	FailureRate        float64 `json:"failure_rate"`
	TotalRuns          int     `json:"total_runs"`
	FailCount          int     `json:"fail_count"`
	PassCount          int     `json:"pass_count"`
	LastSeen           string  `json:"last_seen,omitempty"`
}

// LoadIncidentTelemetry loads everything QA Capsule knows about one incident.
func LoadIncidentTelemetry(incidentID int64) (*IncidentTelemetry, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	row := DB.QueryRow(`
		SELECT i.id, i.project_name, i.name, i.status, i.error_message, i.console_logs, i.error_logs,
			COALESCE(i.fingerprint,''), COALESCE(i.pipeline_run_id,''), COALESCE(i.execution_time_ms,0),
			COALESCE(i.browser,''), COALESCE(i.os,''), COALESCE(i.viewport,''), i.created_at
		FROM incidents i WHERE i.id = ?`, incidentID)

	var t IncidentTelemetry
	var created string
	err := row.Scan(&t.ID, &t.ProjectName, &t.TestName, &t.Status, &t.ErrorMessage, &t.ConsoleLogs, &t.StackTrace,
		&t.Fingerprint, &t.PipelineRunID, &t.ExecutionTimeMs, &t.Browser, &t.OS, &t.Viewport, &created)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("incident not found")
	}
	if err != nil {
		return nil, err
	}
	t.CreatedAt = created
	t.IdentitySHA256 = quarantine.TestIdentityFingerprint(t.ProjectName, t.TestName)
	t.StackTrace = strings.TrimSpace(t.StackTrace)
	if t.StackTrace == "" {
		t.StackTrace = strings.TrimSpace(t.ErrorMessage)
	}
	t.CITags = buildCITags(t.ProjectName, t.PipelineRunID, t.Browser, t.OS, t.Viewport)
	return &t, nil
}

func buildCITags(projectName, runID, browser, osName, viewport string) map[string]string {
	tags := map[string]string{
		"project": projectName,
	}
	if runID != "" {
		tags["pipeline_run_id"] = runID
	}
	if browser != "" {
		tags["browser"] = browser
	}
	if osName != "" {
		tags["os"] = osName
	}
	if viewport != "" {
		tags["viewport"] = viewport
	}
	if runID == "" || DB == nil {
		return tags
	}
	var env, typ, commit, branch sql.NullString
	err := DB.QueryRow(`
		SELECT execution_env, execution_type, commit_sha, branch
		FROM pipeline_runs WHERE project_name = ? AND pipeline_run_id = ?`,
		projectName, runID).Scan(&env, &typ, &commit, &branch)
	if err != nil {
		return tags
	}
	if env.Valid && env.String != "" {
		tags["execution_env"] = env.String
	}
	if typ.Valid && typ.String != "" {
		tags["execution_type"] = typ.String
	}
	if commit.Valid && commit.String != "" {
		tags["commit_sha"] = commit.String
	}
	if branch.Valid && branch.String != "" {
		tags["branch"] = branch.String
	}
	return tags
}

// ListFlakyTests returns tests tagged [FLAKY] with SHA-256 identity fingerprint and failure rate.
func ListFlakyTests(projectName string, limit int) ([]FlakyTestTelemetry, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	q := `
		SELECT project_name, name,
			COALESCE(fingerprint,'') as inc_fp,
			COUNT(*) as total,
			SUM(CASE WHEN UPPER(status) NOT IN ('PASSED','PASS') THEN 1 ELSE 0 END) as fails,
			MAX(created_at) as last_seen
		FROM incidents
		WHERE name LIKE '[FLAKY]%'
	`
	args := []interface{}{}
	if projectName != "" {
		q += ` AND project_name = ?`
		args = append(args, projectName)
	}
	q += ` GROUP BY project_name, name, inc_fp ORDER BY fails DESC, total DESC LIMIT ?`
	args = append(args, limit)

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FlakyTestTelemetry
	for rows.Next() {
		var row FlakyTestTelemetry
		var total, fails int
		var lastSeen string
		if err := rows.Scan(&row.ProjectName, &row.TestName, &row.IncidentFingerprint, &total, &fails, &lastSeen); err != nil {
			continue
		}
		row.IdentitySHA256 = quarantine.TestIdentityFingerprint(row.ProjectName, row.TestName)
		row.TotalRuns = total
		row.FailCount = fails
		row.PassCount = total - fails
		if total > 0 {
			row.FailureRate = float64(fails) / float64(total)
		}
		row.LastSeen = lastSeen
		mergeStabilityStats(&row)
		out = append(out, row)
	}
	return out, nil
}

func mergeStabilityStats(row *FlakyTestTelemetry) {
	if DB == nil {
		return
	}
	var total, fail, pass int
	err := DB.QueryRow(`
		SELECT total_runs, fail_count, pass_count FROM test_stability_stats
		WHERE project_name = ? AND test_identity_fingerprint = ?`,
		row.ProjectName, row.IdentitySHA256).Scan(&total, &fail, &pass)
	if err != nil || total <= 0 {
		return
	}
	row.TotalRuns = total
	row.FailCount = fail
	row.PassCount = pass
	if total > 0 {
		row.FailureRate = float64(fail) / float64(total)
	}
}
