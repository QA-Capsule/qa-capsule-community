package core

import "time"

// IncidentArtifact is a file linked to an incident (trace zip, screenshot, video).
type IncidentArtifact struct {
	ID              int64     `json:"id"`
	IncidentID      int64     `json:"incident_id"`
	FileName        string    `json:"file_name"`
	ContentType     string    `json:"content_type"`
	SizeBytes       int64     `json:"size_bytes"`
	StorageProvider string    `json:"storage_provider"`
	StoragePath     string    `json:"storage_path"`
	CreatedAt       time.Time `json:"created_at"`
}

// TestExecutionMetric stores timing history for performance regression detection.
type TestExecutionMetric struct {
	ProjectName     string
	TestName        string
	Fingerprint     string
	ExecutionTimeMs int64
	Status          string
}

// ExecutionEnv tags the deployment target of a pipeline run.
type ExecutionEnv string

const (
	ExecutionEnvUnknown ExecutionEnv = "UNKNOWN"
	ExecutionEnvProd    ExecutionEnv = "PROD"
	ExecutionEnvStaging ExecutionEnv = "STAGING"
	ExecutionEnvCanary  ExecutionEnv = "CANARY"
	ExecutionEnvDev     ExecutionEnv = "DEV"
)

// ExecutionType tags the intent of a pipeline run.
type ExecutionType string

const (
	ExecutionTypeUnknown ExecutionType = "UNKNOWN"
	ExecutionTypeReal    ExecutionType = "REAL"
	ExecutionTypeTestRun ExecutionType = "TEST-RUN"
	ExecutionTypeNightly ExecutionType = "NIGHTLY"
	ExecutionTypeSmoke   ExecutionType = "SMOKE"
)

// ExecutionFlags is persisted on pipeline_runs.
type ExecutionFlags struct {
	Env  ExecutionEnv  `json:"execution_env"`
	Type ExecutionType `json:"execution_type"`
}

// ExecutionSummary is the aggregated rollup per pipeline run.
type ExecutionSummary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Skipped    int   `json:"skipped"`
	Flaky      int   `json:"flaky"`
	DurationMs int64 `json:"duration_ms"`
}

// TestCaseResult is one entry in the unified test matrix report.
type TestCaseResult struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Fingerprint  string `json:"fingerprint,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	ConsoleLogs  string `json:"console_logs,omitempty"`
	ErrorLogs    string `json:"error_logs,omitempty"`
	IncidentID   int64  `json:"incident_id,omitempty"`
}

// UnifiedExecutionReport is stored in pipeline_runs.report_json.
type UnifiedExecutionReport struct {
	SchemaVersion int              `json:"schema_version"`
	Flags         ExecutionFlags   `json:"flags"`
	Summary       ExecutionSummary `json:"summary"`
	Tests         []TestCaseResult `json:"tests"`
	Framework     string           `json:"framework,omitempty"`
	ParsedAt      time.Time        `json:"parsed_at"`
}

// PipelineRunRecord mirrors pipeline_runs for APIs.
type PipelineRunRecord struct {
	ProjectName   string                  `json:"project_name"`
	PipelineRunID string                  `json:"pipeline_run_id"`
	CommitSHA     string                  `json:"commit_sha,omitempty"`
	Branch        string                  `json:"branch,omitempty"`
	Outcome       string                  `json:"outcome"`
	Flags         ExecutionFlags          `json:"flags"`
	Summary       ExecutionSummary        `json:"summary"`
	Report        *UnifiedExecutionReport `json:"report,omitempty"`
	StartedAt     string                  `json:"started_at,omitempty"`
	FinishedAt    string                  `json:"finished_at,omitempty"`
}

// PatchExecutionFlagsRequest is the PATCH /api/executions/{id}/flag body.
type PatchExecutionFlagsRequest struct {
	Env  string `json:"env"`
	Type string `json:"type"`
}

// IngestExecutionContext accompanies a webhook batch.
type IngestExecutionContext struct {
	ProjectName   string
	PipelineRunID string
	CommitSHA     string
	Branch        string
	Flags         ExecutionFlags
}

// IngestedCase links an ingest result to matrix report rows.
type IngestedCase struct {
	Fingerprint string
	FinalName   string
	IncidentID  int64
	Flaky       bool
}
