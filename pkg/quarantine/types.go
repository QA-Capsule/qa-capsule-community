// Package quarantine implements the optional test-quarantine subsystem. A
// quarantined test is excluded from blocking the CI pipeline even if it
// fails, and its failures are still recorded as incidents for later triage.
// The engine is disabled in the community build (QuarantineEngine = nil in
// core.InitSuperApp), so all public functions are no-ops at runtime.
package quarantine

import "time"

type Reason string

const (
	ReasonFlaky  Reason = "flaky"
	ReasonManual Reason = "manual"
	ReasonPolicy Reason = "policy"
)

type Source string

const (
	SourceAuto   Source = "auto"
	SourceManual Source = "lead"
)

type PolicyConfig struct {
	FailThreshold       int
	FlakyWindowHours    int
	RequireSameCommit   bool
	AutoQuarantine      bool
	AutoLiftAfterPasses int
}

func DefaultPolicy() PolicyConfig {
	return PolicyConfig{
		FailThreshold:       2,
		FlakyWindowHours:    48,
		RequireSameCommit:   true,
		AutoQuarantine:      true,
		AutoLiftAfterPasses: 5,
	}
}

type StabilityStats struct {
	ID                      int64
	ProjectName             string
	TestIdentityFingerprint string
	TestName                string
	TotalRuns               int
	FailCount               int
	PassCount               int
	FlakyCount              int
	LastStatus              string
	LastCommitSHA           string
	LastPipelineRunID       string
	ConsecutiveFailures     int
	LastSeenAt              time.Time
}

type Entry struct {
	ID                      int64
	ProjectName             string
	TestIdentityFingerprint string
	TestName                string
	Reason                  Reason
	Source                  Source
	IncidentID              *int64
	CommitSHAAtQuarantine   string
	ExpiresAt               *time.Time
	IsActive                bool
	CreatedBy               string
	CreatedAt               time.Time
}

type TransitionEvent struct {
	ProjectName             string
	TestName                string
	TestIdentityFingerprint string
	PipelineRunID           string
	CommitSHA               string
	FromStatus              string
	ToStatus                string
	IncidentFingerprint     string
	IncidentID              int64
	DetectedFlaky           bool
}

type Decision struct {
	ShouldQuarantine bool
	TagFlaky         bool
	Entry            *Entry
}

type CIResponse struct {
	ProjectName string          `json:"project_name"`
	GeneratedAt time.Time       `json:"generated_at"`
	Tests       []CIQuarantineTest `json:"tests"`
}

type CIQuarantineTest struct {
	TestName                string `json:"test_name"`
	TestIdentityFingerprint string `json:"fingerprint"`
	Reason                  string `json:"reason"`
	Since                   string `json:"since"`
}
