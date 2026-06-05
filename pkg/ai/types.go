// Package ai provides the LLM integration layer for QA Capsule. It covers
// provider configuration (Groq, Gemini, OpenAI, Ollama, Anthropic, Mistral,
// Azure OpenAI, OpenRouter), the Repository interface for persisting settings
// and analysis results, and the Analyzer interface that drives Root Cause
// Analysis (RCA) and code fix proposals from incident telemetry.
package ai

import "time"

type ProviderKind string

const (
	ProviderDisabled   ProviderKind = "disabled"
	ProviderOpenAI     ProviderKind = "openai"
	ProviderOllama     ProviderKind = "ollama"
	ProviderAnthropic  ProviderKind = "anthropic"
	ProviderGemini     ProviderKind = "gemini"
	ProviderMistral    ProviderKind = "mistral"
	ProviderGroq       ProviderKind = "groq"
	ProviderOpenRouter ProviderKind = "openrouter"
	ProviderAzure      ProviderKind = "azure"
)

type ProviderConfig struct {
	Provider       ProviderKind `json:"provider"`
	Model          string       `json:"model"`
	BaseURL        string       `json:"base_url"`
	APIKeyEnv      string       `json:"api_key_env"`
	// APIKey holds the key entered directly in the UI and stored in the local DB.
	// It is never returned to the client — only its presence is reported via APIKeyStored.
	APIKey         string       `json:"-"`
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

// Incident is framework-agnostic telemetry used for RCA and self-healing prompts.
type Incident struct {
	ID              int64  `json:"id"`
	ProjectName     string `json:"project_name"`
	TestName        string `json:"test_name"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message"`
	StackTrace      string `json:"stack_trace"`
	ConsoleLogs     string `json:"console_logs,omitempty"`
	Fingerprint     string `json:"fingerprint,omitempty"`
	PipelineRunID   string `json:"pipeline_run_id,omitempty"`
	ExecutionTimeMs int64  `json:"execution_time_ms,omitempty"`
	Browser         string `json:"browser,omitempty"`
	OS              string `json:"os,omitempty"`
	Viewport        string `json:"viewport,omitempty"`
}

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

type FixProposal struct {
	Code        string  `json:"code"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence,omitempty"`
	RawJSON     string  `json:"-"`
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
const FixPromptVersion = "self-heal-v1"
