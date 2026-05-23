package ai

import "time"

type ProviderKind string

const (
	ProviderDisabled ProviderKind = "disabled"
	ProviderOpenAI   ProviderKind = "openai"
	ProviderOllama   ProviderKind = "ollama"
)

type ProviderConfig struct {
	Provider       ProviderKind `json:"provider"`
	Model          string       `json:"model"`
	BaseURL        string       `json:"base_url"`
	APIKeyEnv      string       `json:"api_key_env"`
	MaxTokens      int          `json:"max_tokens"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	Enabled        bool         `json:"enabled"`
}

type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobSkipped   JobStatus = "skipped"
)

type AnalysisInput struct {
	IncidentID   int64
	ProjectName  string
	TestName     string
	Status       string
	ErrorMessage string
	ConsoleLogs  string
	Browser      string
	OS           string
	Viewport     string
}

type AnalysisResult struct {
	Summary      string
	RootCause    string
	SuggestedFix string
	SelectorHint string
	Confidence   float64
	RawJSON      string
	TokensIn     int
	TokensOut    int
	LatencyMs    int64
}

type RCAReport struct {
	ID           int64     `json:"id"`
	IncidentID   int64     `json:"incident_id"`
	ProjectName  string    `json:"project_name"`
	TestName     string    `json:"test_name"`
	Summary      string    `json:"summary"`
	RootCause    string    `json:"root_cause"`
	SuggestedFix string    `json:"suggested_fix"`
	SelectorHint string    `json:"selector_hint"`
	Confidence   float64   `json:"confidence"`
	CreatedAt    time.Time `json:"created_at"`
}

type InsightRow struct {
	IncidentID  int64     `json:"incident_id"`
	ProjectName string    `json:"project_name"`
	TestName    string    `json:"test_name"`
	Status      string    `json:"status"`
	RCAStatus   string    `json:"rca_status"`
	Summary     string    `json:"summary"`
	CreatedAt   time.Time `json:"created_at"`
}

const PromptVersion = "rca-v1"
