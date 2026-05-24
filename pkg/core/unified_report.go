package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// UnifiedPipelineReport is the full HTML-report payload for one pipeline run.
type UnifiedPipelineReport struct {
	ProjectName   string           `json:"project_name"`
	PipelineRunID string           `json:"pipeline_run_id"`
	CommitSHA     string           `json:"commit_sha,omitempty"`
	Branch        string           `json:"branch,omitempty"`
	StartedAt     string           `json:"started_at,omitempty"`
	FinishedAt    string           `json:"finished_at,omitempty"`
	Outcome       string           `json:"outcome"`
	Status        string           `json:"status"`
	ExecutionEnv  ExecutionEnv     `json:"execution_env"`
	ExecutionType ExecutionType    `json:"execution_type"`
	Framework     string           `json:"framework,omitempty"`
	DurationMs    int64            `json:"duration_ms"`
	Summary       ExecutionSummary `json:"summary"`
	Tests         []TestCaseResult `json:"tests"`
}

// BuildUnifiedPipelineReport loads pipeline_runs + report_json and enriches from incidents when needed.
func BuildUnifiedPipelineReport(projectName, pipelineRunID string) (*UnifiedPipelineReport, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	projectName = strings.TrimSpace(projectName)
	pipelineRunID = strings.TrimSpace(pipelineRunID)
	if projectName == "" || pipelineRunID == "" {
		return nil, fmt.Errorf("project and pipeline_run_id required")
	}

	rec, err := LoadPipelineRun(projectName, pipelineRunID)
	if err == sql.ErrNoRows {
		return buildReportFromIncidentsOnly(projectName, pipelineRunID)
	}
	if err != nil {
		return nil, err
	}

	out := &UnifiedPipelineReport{
		ProjectName:   rec.ProjectName,
		PipelineRunID: rec.PipelineRunID,
		CommitSHA:     rec.CommitSHA,
		Branch:        rec.Branch,
		StartedAt:     rec.StartedAt,
		FinishedAt:    rec.FinishedAt,
		Outcome:       rec.Outcome,
		ExecutionEnv:  rec.Flags.Env,
		ExecutionType: rec.Flags.Type,
		Summary:       rec.Summary,
		DurationMs:    rec.Summary.DurationMs,
	}

	if rec.Report != nil {
		out.Framework = rec.Report.Framework
		if len(rec.Report.Tests) > 0 {
			out.Tests = append([]TestCaseResult(nil), rec.Report.Tests...)
		}
		if rec.Report.Summary.Total > 0 {
			out.Summary = rec.Report.Summary
			out.DurationMs = rec.Report.Summary.DurationMs
		}
	}

	incidentTests, err := loadReportTestsFromIncidents(projectName, pipelineRunID)
	if err != nil {
		return nil, err
	}
	out.Tests = mergeReportTests(out.Tests, incidentTests)

	if out.Summary.Total == 0 && len(out.Tests) > 0 {
		out.Summary = summarizeTests(out.Tests)
		out.DurationMs = out.Summary.DurationMs
	}

	out.Status = globalReportStatus(out.Summary, out.Outcome)
	return out, nil
}

func buildReportFromIncidentsOnly(projectName, pipelineRunID string) (*UnifiedPipelineReport, error) {
	tests, err := loadReportTestsFromIncidents(projectName, pipelineRunID)
	if err != nil {
		return nil, err
	}
	if len(tests) == 0 {
		return nil, sql.ErrNoRows
	}
	summary := summarizeTests(tests)
	outcome := "success"
	if summary.Failed > 0 {
		outcome = "failure"
	}
	return &UnifiedPipelineReport{
		ProjectName:   projectName,
		PipelineRunID: pipelineRunID,
		Outcome:       outcome,
		Status:        globalReportStatus(summary, outcome),
		ExecutionEnv:  ExecutionEnvUnknown,
		ExecutionType: ExecutionTypeReal,
		Summary:       summary,
		DurationMs:    summary.DurationMs,
		Tests:         tests,
	}, nil
}

// loadReportTestsFromIncidents builds test rows from incidents linked to this pipeline run.
func loadReportTestsFromIncidents(projectName, pipelineRunID string) ([]TestCaseResult, error) {
	rows, err := DB.Query(`
		SELECT id, name, status, error_message, console_logs, error_logs, execution_time_ms
		FROM incidents
		WHERE project_name = ? AND pipeline_run_id = ?
		ORDER BY id ASC`,
		projectName, pipelineRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []TestCaseResult
	for rows.Next() {
		var id int64
		var name, status, errMsg, cLogs, eLogs string
		var execMs sql.NullInt64
		if err := rows.Scan(&id, &name, &status, &errMsg, &cLogs, &eLogs, &execMs); err != nil {
			return nil, err
		}
		tc := TestCaseResult{
			Name:         name,
			Status:       normalizeTestMatrixStatus(status, name),
			ErrorMessage: errMsg,
			ConsoleLogs:  cLogs,
			ErrorLogs:    eLogs,
			IncidentID:   id,
			Fingerprint:  IncidentFingerprint(name, errMsg),
		}
		if execMs.Valid {
			tc.DurationMs = execMs.Int64
		}
		tests = append(tests, tc)
	}
	return tests, rows.Err()
}

func mergeReportTests(primary, fromIncidents []TestCaseResult) []TestCaseResult {
	if len(fromIncidents) == 0 {
		return primary
	}
	if len(primary) == 0 {
		return fromIncidents
	}
	byKey := make(map[string]TestCaseResult, len(primary)+len(fromIncidents))
	order := make([]string, 0, len(primary)+len(fromIncidents))

	add := func(tc TestCaseResult) {
		key := reportTestKey(tc)
		if key == "" {
			key = fmt.Sprintf("idx:%d", len(order))
		}
		if _, seen := byKey[key]; !seen {
			order = append(order, key)
		}
		existing, ok := byKey[key]
		if !ok {
			byKey[key] = tc
			return
		}
		if tc.IncidentID > 0 {
			existing.IncidentID = tc.IncidentID
		}
		if tc.ErrorMessage != "" {
			existing.ErrorMessage = tc.ErrorMessage
		}
		if tc.ErrorLogs != "" {
			existing.ErrorLogs = tc.ErrorLogs
		}
		if tc.ConsoleLogs != "" {
			existing.ConsoleLogs = tc.ConsoleLogs
		}
		if tc.DurationMs > 0 {
			existing.DurationMs = tc.DurationMs
		}
		if tc.Status == "fail" || tc.Status == "flaky" {
			existing.Status = tc.Status
		}
		byKey[key] = existing
	}

	for _, tc := range primary {
		add(tc)
	}
	for _, tc := range fromIncidents {
		add(tc)
	}

	out := make([]TestCaseResult, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out
}

func reportTestKey(tc TestCaseResult) string {
	if tc.Fingerprint != "" {
		return "fp:" + tc.Fingerprint
	}
	return "name:" + strings.TrimSpace(tc.Name)
}

func globalReportStatus(summary ExecutionSummary, outcome string) string {
	if summary.Failed > 0 || summary.Flaky > 0 {
		return "failed"
	}
	if strings.EqualFold(strings.TrimSpace(outcome), "failure") {
		return "failed"
	}
	if summary.Total > 0 && summary.Passed+summary.Skipped >= summary.Total {
		return "passed"
	}
	if summary.Total == 0 {
		return "unknown"
	}
	return "passed"
}

// ReportJSONSnapshot returns raw report_json for debugging (optional).
func ReportJSONSnapshot(projectName, pipelineRunID string) (json.RawMessage, error) {
	var raw sql.NullString
	err := DB.QueryRow(`
		SELECT report_json FROM pipeline_runs
		WHERE project_name = ? AND pipeline_run_id = ?`,
		projectName, pipelineRunID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return json.RawMessage("{}"), nil
	}
	return json.RawMessage(raw.String), nil
}
