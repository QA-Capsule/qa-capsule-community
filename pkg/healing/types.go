package healing

import "time"

// InsightRow is one healable failure shown in the Self-Healing Hub.
type InsightRow struct {
	IncidentID    int64     `json:"incident_id"`
	ProjectName   string    `json:"project_name"`
	TestName      string    `json:"test_name"`
	Status        string    `json:"status"`
	ErrorCategory string    `json:"error_category"`
	Summary       string    `json:"summary"`
	MCPHealed     bool      `json:"mcp_healed"`
	CreatedAt     time.Time `json:"created_at"`
}

// Context bundles framework-agnostic telemetry with rule-based hints for MCP agents.
type Context struct {
	IncidentID         int64             `json:"incident_id"`
	ProjectName        string            `json:"project_name"`
	TestName           string            `json:"test_name"`
	Status             string            `json:"status"`
	ErrorMessage       string            `json:"error_message"`
	StackTrace         string            `json:"stack_trace"`
	ConsoleLogs        string            `json:"console_logs,omitempty"`
	Fingerprint        string            `json:"fingerprint"`
	IdentitySHA256     string            `json:"identity_fingerprint_sha256"`
	PipelineRunID      string            `json:"pipeline_run_id,omitempty"`
	ExecutionTimeMs    int64             `json:"execution_time_ms,omitempty"`
	Browser            string            `json:"browser,omitempty"`
	OS                 string            `json:"os,omitempty"`
	Viewport           string            `json:"viewport,omitempty"`
	CITags             map[string]string `json:"ci_tags"`
	CreatedAt          string            `json:"created_at,omitempty"`
	ErrorCategory      string            `json:"error_category"`
	SelectorHint       string            `json:"selector_hint,omitempty"`
	SuggestedActions   []string          `json:"suggested_actions"`
	MCPPrompt          string            `json:"mcp_prompt"`
}

// Proposal is the response for self-healing propose (no internal LLM).
type Proposal struct {
	Code             string   `json:"code"`
	Explanation      string   `json:"explanation"`
	ErrorCategory    string   `json:"error_category"`
	SelectorHint     string   `json:"selector_hint,omitempty"`
	SuggestedActions []string `json:"suggested_actions"`
	MCPPrompt        string   `json:"mcp_prompt"`
	Confidence       float64  `json:"confidence"`
}

// PatchSubmission stores one MCP-proposed code patch before PR creation.
type PatchSubmission struct {
	ID          int64     `json:"id"`
	IncidentID  int64     `json:"incident_id"`
	Repo        string    `json:"repo"`
	FilePath    string    `json:"file_path"`
	CodeSHA256  string    `json:"code_sha256"`
	CodeSize    int       `json:"code_size"`
	Explanation string    `json:"explanation,omitempty"`
	AgentSource string    `json:"agent_source,omitempty"`
	Status      string    `json:"status"`
	PRURL       string    `json:"pr_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
