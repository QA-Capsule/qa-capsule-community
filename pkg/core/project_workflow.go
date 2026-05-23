package core

import (
	"database/sql"

	"github.com/QA-Capsule/qa-capsule-community/pkg/integrations"
)

// LoadProjectWorkflow reads the visual workflow DAG for a project by name.
func LoadProjectWorkflow(projectName string) *integrations.WorkflowDocument {
	var raw sql.NullString
	err := DB.QueryRow(`SELECT sre_workflow_json FROM projects WHERE name = ?`, projectName).Scan(&raw)
	if err != nil || !raw.Valid || raw.String == "" {
		return nil
	}
	doc, err := integrations.ParseWorkflowJSON(raw.String)
	if err != nil || doc == nil {
		return nil
	}
	return doc
}

// LoadProjectWorkflowByID reads workflow by project id.
func LoadProjectWorkflowByID(projectID string) (*integrations.WorkflowDocument, error) {
	var raw sql.NullString
	err := DB.QueryRow(`SELECT sre_workflow_json FROM projects WHERE id = ?`, projectID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	return integrations.ParseWorkflowJSON(raw.String)
}

// SaveProjectWorkflow persists workflow JSON for a project.
func SaveProjectWorkflow(projectID string, doc *integrations.WorkflowDocument) error {
	payload := ""
	if doc != nil {
		payload = integrations.MarshalWorkflowJSON(doc)
	}
	_, err := DB.Exec(`UPDATE projects SET sre_workflow_json = ? WHERE id = ?`, payload, projectID)
	return err
}

// ClearProjectWorkflow removes the DAG (reverts to legacy auto-trigger).
func ClearProjectWorkflow(projectID string) error {
	_, err := DB.Exec(`UPDATE projects SET sre_workflow_json = '' WHERE id = ?`, projectID)
	return err
}

// WorkflowSummary describes persisted workflow state for API / UI badges.
type WorkflowSummary struct {
	HasWorkflow bool `json:"has_workflow"`
	Enabled     bool `json:"workflow_enabled"`
}

// WorkflowSummaryFromJSON parses stored workflow JSON for list endpoints.
func WorkflowSummaryFromJSON(raw string) WorkflowSummary {
	doc, err := integrations.ParseWorkflowJSON(raw)
	if err != nil || doc == nil {
		return WorkflowSummary{}
	}
	has := doc.Entry != "" || len(doc.Nodes) > 0
	return WorkflowSummary{
		HasWorkflow: has,
		Enabled:     integrations.IsWorkflowActive(doc),
	}
}

// UserCanAccessProject reports whether the user may read/write project-scoped resources.
func UserCanAccessProject(username, role, projectID string) bool {
	r := NormalizeRole(role)
	if r == RoleAdmin || r == RoleManager {
		return true
	}
	var teamID int
	if err := DB.QueryRow(`SELECT team_id FROM projects WHERE id = ?`, projectID).Scan(&teamID); err != nil {
		return false
	}
	var userID int
	if err := DB.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&userID); err != nil {
		return false
	}
	var count int
	_ = DB.QueryRow(`
		SELECT COUNT(*) FROM user_teams WHERE user_id = ? AND team_id = ?`,
		userID, teamID).Scan(&count)
	return count > 0
}
