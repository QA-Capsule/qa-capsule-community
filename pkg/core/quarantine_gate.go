package core

import (
	"context"
	"log/slog"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

// IsTestQuarantined reports whether the test is on the active deny list for the project.
func IsTestQuarantined(projectName, testName string) bool {
	if QuarantineEngine == nil || projectName == "" || testName == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return QuarantineEngine.IsQuarantined(ctx, projectName, testName)
}

// recordQuarantinedIngest updates stability stats without creating an incident or firing remediation.
func recordQuarantinedIngest(projectName, runID string, alert UnifiedAlert) {
	if QuarantineEngine == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		identityFP := quarantine.TestIdentityFingerprint(projectName, alert.Name)
		ev := quarantine.TransitionEvent{
			ProjectName:             projectName,
			TestName:                alert.Name,
			TestIdentityFingerprint: identityFP,
			PipelineRunID:           runID,
			CommitSHA:               alert.CommitSHA,
			ToStatus:                alert.Status,
			IncidentFingerprint:     IncidentFingerprint(alert.Name, alert.Error),
		}
		if _, err := QuarantineEngine.RecordTransition(ctx, ev); err != nil {
			slog.Warn("quarantine ingest stats", "error", err)
		}
	}()
}
