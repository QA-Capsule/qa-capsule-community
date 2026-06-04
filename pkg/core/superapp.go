package core

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/ai"
	"github.com/QA-Capsule/qa-capsule-community/pkg/healing"
	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

var (
	QuarantineEngine *quarantine.Engine
	HealingService   *healing.Service
	// AIService handles AI provider config, RCA jobs, and locator fix proposals.
	// Nil when DB is not initialized. Check Enabled before calling.
	AIService *ai.Service
)

// InitSuperApp wires self-healing and AI engines (call after InitDB).
// Quarantine engine is intentionally disabled — all tests are ingested regardless of flakiness history.
func InitSuperApp() {
	if DB == nil {
		return
	}
	QuarantineEngine = nil
	// Clear any existing quarantine entries so no test is blocked on restart.
	if _, err := DB.Exec(`UPDATE test_quarantine_entries SET is_active = 0`); err != nil {
		slog.Warn("could not clear quarantine entries", "error", err)
	}
	HealingService = healing.NewService(DB)
	AIService = ai.NewService(ai.NewSQLiteRepository(DB), nil)
	slog.Info("super-app modules initialized", "healing", true, "quarantine", false, "ai", true)
}

// PostIncidentHooks runs async quarantine scoring after ingest.
func PostIncidentHooks(incidentID int64, projectName, runID, commitSHA string, alert UnifiedAlert, flaky bool) {
	if incidentID <= 0 {
		return
	}
	status := alert.Status
	testName := alert.Name
	identityFP := quarantine.TestIdentityFingerprint(projectName, testName)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if QuarantineEngine != nil {
			ev := quarantine.TransitionEvent{
				ProjectName:             projectName,
				TestName:                testName,
				TestIdentityFingerprint: identityFP,
				PipelineRunID:           runID,
				CommitSHA:               commitSHA,
				FromStatus:              "",
				ToStatus:                status,
				IncidentFingerprint:     IncidentFingerprint(alert.Name, alert.Error),
				IncidentID:              incidentID,
				DetectedFlaky:           flaky,
			}
			if _, err := QuarantineEngine.RecordTransition(ctx, ev); err != nil {
				slog.Warn("quarantine transition", "error", err)
			}
		}
	}()
}

func isFailureStatus(status string) bool {
	u := strings.ToUpper(strings.TrimSpace(status))
	return u != "PASSED" && u != "PASS" && u != ""
}
