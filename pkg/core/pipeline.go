package core

import "log/slog"

// RecordPipelineRun upserts deployment metadata for DORA metrics (legacy entry point).
func RecordPipelineRun(projectName, runID, commitSHA, branch, outcome string) {
	flags := ExecutionFlags{Env: ExecutionEnvUnknown, Type: ExecutionTypeReal}
	rec := PipelineRunRecord{
		ProjectName:   projectName,
		PipelineRunID: runID,
		CommitSHA:     commitSHA,
		Branch:        branch,
		Outcome:       outcome,
		Flags:         flags,
	}
	if err := UpsertPipelineExecution(rec); err != nil {
		slog.Warn("pipeline run record failed", "project", projectName, "run", runID, "error", err)
	}
}
