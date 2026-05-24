package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const executionReportSchemaVersion = 1

// BuildReportFromPayload builds a unified report from webhook JSON (tests[] + optional summary).
func BuildReportFromPayload(raw map[string]interface{}, flags ExecutionFlags, framework string) UnifiedExecutionReport {
	report := UnifiedExecutionReport{
		SchemaVersion: executionReportSchemaVersion,
		Flags:         flags,
		Framework:     framework,
		ParsedAt:      time.Now().UTC(),
	}

	if raw != nil {
		if sm, ok := raw["summary"].(map[string]interface{}); ok {
			report.Summary = summaryFromMap(sm)
		}
		if tests, ok := raw["tests"].([]interface{}); ok {
			for _, item := range tests {
				if m, ok := item.(map[string]interface{}); ok {
					report.Tests = append(report.Tests, testCaseFromPayloadMap(m))
				}
			}
		} else {
			single := testCaseFromAlert(NormalizePayload(raw))
			if single.Name != "" || single.Status != "" {
				report.Tests = append(report.Tests, single)
			}
		}
	}

	if report.Summary.Total == 0 && len(report.Tests) > 0 {
		report.Summary = summarizeTests(report.Tests)
	} else if len(report.Tests) > 0 {
		computed := summarizeTests(report.Tests)
		if report.Summary.Total == 0 {
			report.Summary.Total = computed.Total
		}
		if report.Summary.Passed == 0 && report.Summary.Failed == 0 && report.Summary.Skipped == 0 {
			report.Summary.Passed = computed.Passed
			report.Summary.Failed = computed.Failed
			report.Summary.Skipped = computed.Skipped
			report.Summary.Flaky = computed.Flaky
		}
		if report.Summary.DurationMs == 0 {
			report.Summary.DurationMs = computed.DurationMs
		}
	}
	return report
}

func summaryFromMap(m map[string]interface{}) ExecutionSummary {
	return ExecutionSummary{
		Total:      int(int64Field(m, "total")),
		Passed:     int(int64Field(m, "passed")),
		Failed:     int(int64Field(m, "failed")),
		Skipped:    int(int64Field(m, "skipped")),
		Flaky:      int(int64Field(m, "flaky")),
		DurationMs: int64Field(m, "duration_ms"),
	}
}

func testCaseFromPayloadMap(m map[string]interface{}) TestCaseResult {
	name, _ := m["name"].(string)
	if name == "" {
		name = stringField(m, "nodeid", "")
	}
	if name == "" {
		name = stringField(m, "title", "")
	}
	suite := stringField(m, "suite", "")
	className := stringField(m, "classname", "")
	if className == "" {
		className = stringField(m, "class_name", "")
	}
	status, _ := m["status"].(string)
	if status == "" {
		status = stringField(m, "outcome", "")
	}
	if status == "" {
		status = stringField(m, "state", "")
	}
	if status == "" {
		status = "failed"
	}
	errStr, _ := m["error"].(string)
	if errStr == "" {
		errStr, _ = m["error_message"].(string)
	}
	if errStr == "" {
		errStr = stringField(m, "longrepr", "")
	}
	if errStr == "" {
		errStr = stringField(m, "failure_reason", "")
	}
	durationMs := int64Field(m, "execution_time_ms")
	if durationMs == 0 {
		if sec := floatField(m, "duration"); sec > 0 {
			durationMs = int64(sec * 1000)
		}
	}
	tc := TestCaseResult{
		Name:         name,
		Suite:        suite,
		ClassName:    className,
		Status:       normalizeTestMatrixStatus(status, name),
		DurationMs:   durationMs,
		ErrorMessage: errStr,
	}
	if logs, ok := m["console_logs"].(string); ok {
		tc.ConsoleLogs = logs
	}
	if logs, ok := m["error_logs"].(string); ok {
		tc.ErrorLogs = logs
	}
	if tc.Name != "" {
		tc.Fingerprint = IncidentFingerprint(tc.Name, tc.ErrorMessage)
	}
	return tc
}

func testCaseFromAlert(a UnifiedAlert) TestCaseResult {
	status := normalizeTestMatrixStatus(a.Status, a.Name)
	fp := IncidentFingerprint(a.Name, a.Error)
	return TestCaseResult{
		Name:         a.Name,
		Status:       status,
		Fingerprint:  fp,
		DurationMs:   a.ExecutionTimeMs,
		ErrorMessage: a.Error,
		ConsoleLogs:  a.ConsoleLogs,
		ErrorLogs:    a.ErrorLogs,
	}
}

func normalizeTestMatrixStatus(status, name string) string {
	if strings.Contains(strings.ToUpper(name), "[FLAKY]") {
		return "flaky"
	}
	if isPassStatus(status) {
		return "pass"
	}
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "skip" || s == "skipped" {
		return "skip"
	}
	if s == "flaky" {
		return "flaky"
	}
	if s == "pass" || s == "passed" {
		return "pass"
	}
	return "fail"
}

func summarizeTests(tests []TestCaseResult) ExecutionSummary {
	var s ExecutionSummary
	for _, t := range tests {
		s.Total++
		s.DurationMs += t.DurationMs
		switch t.Status {
		case "pass":
			s.Passed++
		case "skip":
			s.Skipped++
		case "flaky":
			s.Flaky++
			s.Failed++
		default:
			s.Failed++
		}
	}
	return s
}

// AttachIngestedIncidents sets incident_id on failed/flaky matrix cells.
func AttachIngestedIncidents(report *UnifiedExecutionReport, ingested []IngestedCase) {
	if report == nil || len(ingested) == 0 {
		return
	}
	byFP := make(map[string]IngestedCase, len(ingested))
	byName := make(map[string]IngestedCase, len(ingested))
	for _, c := range ingested {
		if c.IncidentID <= 0 {
			continue
		}
		if c.Fingerprint != "" {
			byFP[c.Fingerprint] = c
		}
		if c.FinalName != "" {
			byName[c.FinalName] = c
		}
	}
	for i := range report.Tests {
		t := &report.Tests[i]
		if c, ok := byFP[t.Fingerprint]; ok {
			t.IncidentID = c.IncidentID
			if c.Flaky {
				t.Status = "flaky"
			}
			continue
		}
		if c, ok := byName[t.Name]; ok {
			t.IncidentID = c.IncidentID
		}
	}
}

// FinalizePipelineExecution upserts pipeline_runs with flags, summary, and report JSON.
func FinalizePipelineExecution(ctx IngestExecutionContext, outcome string, report UnifiedExecutionReport, ingested []IngestedCase) error {
	report.Flags = mergeExecutionFlags(ctx.Flags, report.Flags)
	AttachIngestedIncidents(&report, ingested)

	rec := PipelineRunRecord{
		ProjectName:   ctx.ProjectName,
		PipelineRunID: ctx.PipelineRunID,
		CommitSHA:     ctx.CommitSHA,
		Branch:        ctx.Branch,
		Outcome:       outcome,
		Flags:         report.Flags,
		Summary:       report.Summary,
		Report:        &report,
	}
	return UpsertPipelineExecution(rec)
}

// UpsertPipelineExecution writes pipeline_runs including execution hub fields.
func UpsertPipelineExecution(rec PipelineRunRecord) error {
	if DB == nil || rec.ProjectName == "" || rec.PipelineRunID == "" {
		return fmt.Errorf("invalid pipeline execution record")
	}
	outcome := strings.ToLower(strings.TrimSpace(rec.Outcome))
	if outcome == "" {
		outcome = "unknown"
	}
	flags := rec.Flags
	if flags.Env == ExecutionEnvUnknown {
		flags.Env = ExecutionEnvUnknown
	}
	if flags.Type == ExecutionTypeUnknown {
		flags.Type = ExecutionTypeReal
	}
	reportJSON := "{}"
	if rec.Report != nil {
		b, err := json.Marshal(rec.Report)
		if err != nil {
			return err
		}
		reportJSON = string(b)
	}
	_, err := DB.Exec(`
		INSERT INTO pipeline_runs (
			project_name, pipeline_run_id, commit_sha, branch, outcome,
			execution_env, execution_type,
			total_tests, passed_tests, failed_tests, skipped_tests, duration_ms,
			report_json, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(project_name, pipeline_run_id) DO UPDATE SET
			commit_sha = CASE WHEN excluded.commit_sha != '' THEN excluded.commit_sha ELSE pipeline_runs.commit_sha END,
			branch = CASE WHEN excluded.branch != '' THEN excluded.branch ELSE pipeline_runs.branch END,
			outcome = CASE
				WHEN excluded.outcome = 'failure' THEN 'failure'
				WHEN pipeline_runs.outcome = 'failure' THEN 'failure'
				ELSE excluded.outcome
			END,
			execution_env = excluded.execution_env,
			execution_type = excluded.execution_type,
			total_tests = excluded.total_tests,
			passed_tests = excluded.passed_tests,
			failed_tests = excluded.failed_tests,
			skipped_tests = excluded.skipped_tests,
			duration_ms = excluded.duration_ms,
			report_json = excluded.report_json,
			finished_at = CURRENT_TIMESTAMP`,
		rec.ProjectName, rec.PipelineRunID, rec.CommitSHA, rec.Branch, outcome,
		string(flags.Env), string(flags.Type),
		rec.Summary.Total, rec.Summary.Passed, rec.Summary.Failed, rec.Summary.Skipped, rec.Summary.DurationMs,
		reportJSON)
	if err != nil {
		slog.Warn("pipeline execution upsert failed", "project", rec.ProjectName, "run", rec.PipelineRunID, "error", err)
	}
	return err
}

// LoadPipelineRun loads execution metadata for a pipeline run.
func LoadPipelineRun(projectName, runID string) (*PipelineRunRecord, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var rec PipelineRunRecord
	var env, typ, reportJSON, startedAt, finishedAt sql.NullString
	var total, passed, failed, skipped int
	var durationMs int64
	err := DB.QueryRow(`
		SELECT project_name, pipeline_run_id, commit_sha, branch, outcome,
			execution_env, execution_type,
			total_tests, passed_tests, failed_tests, skipped_tests, duration_ms,
			report_json, started_at, finished_at
		FROM pipeline_runs WHERE project_name = ? AND pipeline_run_id = ?`,
		projectName, runID).Scan(
		&rec.ProjectName, &rec.PipelineRunID, &rec.CommitSHA, &rec.Branch, &rec.Outcome,
		&env, &typ,
		&total, &passed, &failed, &skipped, &durationMs,
		&reportJSON, &startedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.Flags = ExecutionFlags{
		Env:  NormalizeExecutionEnv(env.String),
		Type: NormalizeExecutionType(typ.String),
	}
	rec.Summary = ExecutionSummary{
		Total: total, Passed: passed, Failed: failed, Skipped: skipped, DurationMs: durationMs,
	}
	if reportJSON.Valid && reportJSON.String != "" {
		var report UnifiedExecutionReport
		if json.Unmarshal([]byte(reportJSON.String), &report) == nil {
			rec.Report = &report
		}
	}
	if startedAt.Valid {
		rec.StartedAt = startedAt.String
	}
	if finishedAt.Valid {
		rec.FinishedAt = finishedAt.String
	}
	return &rec, nil
}

// UpdatePipelineFlags updates execution_env and execution_type for a run.
func UpdatePipelineFlags(projectName, runID string, flags ExecutionFlags) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	flags.Env = NormalizeExecutionEnv(string(flags.Env))
	flags.Type = NormalizeExecutionType(string(flags.Type))
	if flags.Env == ExecutionEnvUnknown {
		flags.Env = ExecutionEnvUnknown
	}
	if flags.Type == ExecutionTypeUnknown {
		flags.Type = ExecutionTypeReal
	}
	res, err := DB.Exec(`
		UPDATE pipeline_runs SET execution_env = ?, execution_type = ?, finished_at = CURRENT_TIMESTAMP
		WHERE project_name = ? AND pipeline_run_id = ?`,
		string(flags.Env), string(flags.Type), projectName, runID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, err = DB.Exec(`
			INSERT INTO pipeline_runs (project_name, pipeline_run_id, execution_env, execution_type, outcome, started_at)
			VALUES (?, ?, ?, ?, 'unknown', CURRENT_TIMESTAMP)`,
			projectName, runID, string(flags.Env), string(flags.Type))
		return err
	}
	var reportJSON sql.NullString
	_ = DB.QueryRow(`SELECT report_json FROM pipeline_runs WHERE project_name = ? AND pipeline_run_id = ?`, projectName, runID).Scan(&reportJSON)
	if reportJSON.Valid && reportJSON.String != "" {
		var report UnifiedExecutionReport
		if json.Unmarshal([]byte(reportJSON.String), &report) == nil {
			report.Flags = flags
			b, _ := json.Marshal(report)
			_, _ = DB.Exec(`UPDATE pipeline_runs SET report_json = ? WHERE project_name = ? AND pipeline_run_id = ?`,
				string(b), projectName, runID)
		}
	}
	return nil
}

// LoadPipelineFlagsByRuns returns flags keyed by pipeline_run_id for a project.
func LoadPipelineFlagsByRuns(projectName string, runIDs []string) map[string]PipelineRunRecord {
	out := make(map[string]PipelineRunRecord)
	if DB == nil || projectName == "" || len(runIDs) == 0 {
		return out
	}
	for _, runID := range runIDs {
		if runID == "" {
			continue
		}
		rec, err := LoadPipelineRun(projectName, runID)
		if err == nil && rec != nil {
			out[runID] = *rec
		}
	}
	return out
}
