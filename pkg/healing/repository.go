package healing

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

type incidentRow struct {
	ID              int64
	ProjectName     string
	TestName        string
	Status          string
	ErrorMessage    string
	ConsoleLogs     string
	StackTrace      string
	Fingerprint     string
	PipelineRunID   string
	ExecutionTimeMs int64
	Browser         string
	OS              string
	Viewport        string
	CreatedAt       string
}

func (s *Service) loadIncident(incidentID int64) (*incidentRow, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("healing service not initialized")
	}
	row := s.db.QueryRow(`
		SELECT i.id, i.project_name, i.name, i.status, i.error_message, i.console_logs, i.error_logs,
			COALESCE(i.fingerprint,''), COALESCE(i.pipeline_run_id,''), COALESCE(i.execution_time_ms,0),
			COALESCE(i.browser,''), COALESCE(i.os,''), COALESCE(i.viewport,''), i.created_at
		FROM incidents i WHERE i.id = ?`, incidentID)

	var inc incidentRow
	err := row.Scan(&inc.ID, &inc.ProjectName, &inc.TestName, &inc.Status, &inc.ErrorMessage, &inc.ConsoleLogs,
		&inc.StackTrace, &inc.Fingerprint, &inc.PipelineRunID, &inc.ExecutionTimeMs, &inc.Browser, &inc.OS, &inc.Viewport, &inc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("incident not found")
	}
	if err != nil {
		return nil, err
	}
	inc.StackTrace = strings.TrimSpace(inc.StackTrace)
	if inc.StackTrace == "" {
		inc.StackTrace = strings.TrimSpace(inc.ErrorMessage)
	}
	return &inc, nil
}

func (s *Service) buildCITags(projectName, runID, browser, osName, viewport string) map[string]string {
	tags := map[string]string{"project": projectName}
	if runID != "" {
		tags["pipeline_run_id"] = runID
	}
	if browser != "" {
		tags["browser"] = browser
	}
	if osName != "" {
		tags["os"] = osName
	}
	if viewport != "" {
		tags["viewport"] = viewport
	}
	if runID == "" || s.db == nil {
		return tags
	}
	var env, typ, commit, branch sql.NullString
	err := s.db.QueryRow(`
		SELECT execution_env, execution_type, commit_sha, branch
		FROM pipeline_runs WHERE project_name = ? AND pipeline_run_id = ?`,
		projectName, runID).Scan(&env, &typ, &commit, &branch)
	if err != nil {
		return tags
	}
	if env.Valid && env.String != "" {
		tags["execution_env"] = env.String
	}
	if typ.Valid && typ.String != "" {
		tags["execution_type"] = typ.String
	}
	if commit.Valid && commit.String != "" {
		tags["commit_sha"] = commit.String
	}
	if branch.Valid && branch.String != "" {
		tags["branch"] = branch.String
	}
	return tags
}

func identityFingerprint(projectName, testName string) string {
	return quarantine.TestIdentityFingerprint(projectName, testName)
}
