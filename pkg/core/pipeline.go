package core

import (
	"log/slog"
	"strings"
)

// RecordPipelineRun upserts deployment metadata for DORA metrics.
func RecordPipelineRun(projectName, runID, commitSHA, branch, outcome string) {
	if DB == nil || projectName == "" || runID == "" {
		return
	}
	outcome = strings.ToLower(strings.TrimSpace(outcome))
	if outcome == "" {
		outcome = "unknown"
	}
	_, err := DB.Exec(`
		INSERT INTO pipeline_runs (project_name, pipeline_run_id, commit_sha, branch, outcome, started_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(project_name, pipeline_run_id) DO UPDATE SET
			commit_sha = excluded.commit_sha,
			branch = CASE WHEN excluded.branch != '' THEN excluded.branch ELSE pipeline_runs.branch END,
			outcome = CASE
				WHEN excluded.outcome = 'failure' THEN 'failure'
				WHEN pipeline_runs.outcome = 'failure' THEN 'failure'
				ELSE excluded.outcome
			END,
			finished_at = CURRENT_TIMESTAMP`,
		projectName, runID, commitSHA, branch, outcome)
	if err != nil {
		slog.Warn("pipeline run record failed", "project", projectName, "run", runID, "error", err)
	}
}
