package quarantine

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Engine struct {
	repo   Repository
	policy PolicyConfig
}

func NewEngine(repo Repository, policy PolicyConfig) *Engine {
	return &Engine{repo: repo, policy: policy}
}

// IsQuarantined returns true when the test is on the active deny list.
func (e *Engine) IsQuarantined(ctx context.Context, projectName, testName string) bool {
	if e == nil || e.repo == nil {
		return false
	}
	fp := TestIdentityFingerprint(projectName, testName)
	ent, err := e.repo.ActiveEntry(ctx, projectName, fp)
	return err == nil && ent != nil
}

func (e *Engine) RecordTransition(ctx context.Context, ev TransitionEvent) (*Decision, error) {
	if e == nil || e.repo == nil {
		return nil, nil
	}
	if ev.TestIdentityFingerprint == "" {
		ev.TestIdentityFingerprint = TestIdentityFingerprint(ev.ProjectName, ev.TestName)
	}
	_ = e.repo.UpsertPipelineRun(ctx, ev.ProjectName, ev.PipelineRunID, ev.CommitSHA, "")
	_ = e.repo.InsertTransition(ctx, ev)

	stats, _ := e.repo.GetStats(ctx, ev.ProjectName, ev.TestIdentityFingerprint)
	if stats == nil {
		stats = &StabilityStats{
			ProjectName:             ev.ProjectName,
			TestIdentityFingerprint: ev.TestIdentityFingerprint,
			TestName:                NormalizeTestName(ev.TestName),
		}
	}

	stats.TotalRuns++
	stats.LastPipelineRunID = ev.PipelineRunID
	stats.LastSeenAt = time.Now()
	toStatus := strings.ToUpper(strings.TrimSpace(ev.ToStatus))

	switch {
	case isPass(toStatus):
		stats.PassCount++
		stats.LastStatus = "PASSED"
		stats.ConsecutiveFailures = 0
		if ev.CommitSHA != "" {
			stats.LastCommitSHA = ev.CommitSHA
		}
		if e.maybeAutoLiftAfterPasses(ctx, ev.ProjectName, ev.TestIdentityFingerprint, stats.PassCount) {
			if err := e.repo.UpsertStats(ctx, *stats); err != nil {
				return nil, err
			}
			return &Decision{}, nil
		}
	case isFail(toStatus):
		stats.FailCount++
		stats.LastStatus = "FAILED"
		stats.ConsecutiveFailures++
		if ev.DetectedFlaky {
			stats.FlakyCount++
		}
		if e.policy.RequireSameCommit && ev.CommitSHA != "" && stats.LastCommitSHA != "" &&
			ev.CommitSHA == stats.LastCommitSHA && stats.PassCount > 0 {
			stats.FlakyCount++
		}
	default:
		stats.LastStatus = toStatus
	}

	if err := e.repo.UpsertStats(ctx, *stats); err != nil {
		return nil, err
	}

	decision := &Decision{}
	if !e.policy.AutoQuarantine || !isFail(toStatus) {
		return decision, nil
	}

	existing, _ := e.repo.ActiveEntry(ctx, ev.ProjectName, ev.TestIdentityFingerprint)
	if existing != nil {
		return decision, nil
	}

	should := ev.DetectedFlaky || stats.FlakyCount >= 1 || stats.ConsecutiveFailures >= e.policy.FailThreshold
	if !should {
		return decision, nil
	}

	entry := Entry{
		ProjectName:             ev.ProjectName,
		TestIdentityFingerprint: ev.TestIdentityFingerprint,
		TestName:                NormalizeTestName(ev.TestName),
		Reason:                  ReasonFlaky,
		Source:                  SourceAuto,
		CommitSHAAtQuarantine:   ev.CommitSHA,
		CreatedBy:               "system",
	}
	if ev.IncidentID > 0 {
		entry.IncidentID = &ev.IncidentID
	}
	if err := e.repo.CreateEntry(ctx, entry); err != nil {
		slog.Warn("quarantine entry failed", "error", err)
		return decision, err
	}
	decision.ShouldQuarantine = true
	decision.TagFlaky = true
	decision.Entry = &entry
	slog.Info("test auto-quarantined", "project", ev.ProjectName, "test", entry.TestName)
	return decision, nil
}

func (e *Engine) maybeAutoLiftAfterPasses(ctx context.Context, projectName, identityFP string, passCount int) bool {
	if e == nil || e.repo == nil || e.policy.AutoLiftAfterPasses <= 0 {
		return false
	}
	existing, err := e.repo.ActiveEntry(ctx, projectName, identityFP)
	if err != nil || existing == nil {
		return false
	}
	if passCount < e.policy.AutoLiftAfterPasses {
		return false
	}
	if err := e.repo.LiftEntry(ctx, projectName, identityFP, "auto-pass-streak"); err != nil {
		slog.Warn("auto-lift quarantine failed", "error", err)
		return false
	}
	slog.Info("test auto-lifted from quarantine after pass streak",
		"project", projectName, "fingerprint", identityFP, "passes", passCount)
	return true
}

func (e *Engine) ListCI(ctx context.Context, projectName string) (CIResponse, error) {
	entries, err := e.repo.ListActive(ctx, projectName)
	if err != nil {
		return CIResponse{}, err
	}
	out := CIResponse{ProjectName: projectName, GeneratedAt: time.Now().UTC()}
	for _, ent := range entries {
		out.Tests = append(out.Tests, CIQuarantineTest{
			TestName:                ent.TestName,
			TestIdentityFingerprint: ent.TestIdentityFingerprint,
			Reason:                  string(ent.Reason),
			Since:                   ent.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (e *Engine) ManualQuarantine(ctx context.Context, projectName, testName, createdBy string) error {
	fp := TestIdentityFingerprint(projectName, testName)
	return e.repo.CreateEntry(ctx, Entry{
		ProjectName:             projectName,
		TestIdentityFingerprint: fp,
		TestName:                NormalizeTestName(testName),
		Reason:                  ReasonManual,
		Source:                  SourceManual,
		CreatedBy:               createdBy,
	})
}

func (e *Engine) Lift(ctx context.Context, projectName, identityFP, by string) error {
	return e.repo.LiftEntry(ctx, projectName, identityFP, by)
}

func isPass(status string) bool {
	return status == "PASSED" || status == "PASS"
}

func isFail(status string) bool {
	return status == "FAILED" || status == "FAIL" || status == "CRITICAL" || status == "PERF_DEGRADATION"
}
