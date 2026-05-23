package core

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/ai"
	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

var (
	QuarantineEngine *quarantine.Engine
	AIService        *ai.Service
)

// InitSuperApp wires AI RCA and quarantine engines (call after InitDB).
func InitSuperApp() {
	if DB == nil {
		return
	}
	QuarantineEngine = quarantine.NewEngine(
		quarantine.NewSQLiteRepository(DB),
		quarantine.DefaultPolicy(),
	)
	AIService = ai.NewService(ai.NewSQLiteRepository(DB), ai.HTTPAnalyzer{})
	slog.Info("super-app modules initialized", "ai", true, "quarantine", true)
}

// PostIncidentHooks runs async quarantine scoring and RCA enqueue.
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

	if AIService != nil && !flaky && isFailureStatus(status) {
		AIService.EnqueueForIncident(incidentID)
	}
}

func isFailureStatus(status string) bool {
	u := strings.ToUpper(strings.TrimSpace(status))
	return u != "PASSED" && u != "PASS" && u != ""
}
