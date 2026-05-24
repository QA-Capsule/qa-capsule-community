package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func registerReportRoutes(config *core.Config) {
	http.HandleFunc("/api/reports/", jwtAuthMiddleware(config, "", handleUnifiedReport))
}

// GET /api/reports/{pipeline_run_id}?project={project_name}
func handleUnifiedReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pipelineRunID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/reports/"), "/")
	if pipelineRunID == "" {
		writeJSONError(w, "pipeline_run_id required", http.StatusBadRequest)
		return
	}

	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if project == "" {
		writeJSONError(w, "project query parameter required", http.StatusBadRequest)
		return
	}

	claims := parseClaims(r)
	if claims == nil {
		writeJSONError(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	if !userCanAccessProjectName(claims.Username, claims.Role, project) {
		writeJSONError(w, "Access denied for this project", http.StatusForbidden)
		return
	}

	report, err := core.BuildUnifiedPipelineReport(project, pipelineRunID)
	if err != nil {
		writeJSONError(w, "Pipeline report not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// userCanAccessProjectName resolves projects.name or projects.id then checks RBAC.
func userCanAccessProjectName(username, role, projectName string) bool {
	if core.DB == nil {
		return false
	}
	var projectID string
	err := core.DB.QueryRow(`
		SELECT id FROM projects WHERE name = ? OR id = ? LIMIT 1`,
		projectName, projectName).Scan(&projectID)
	if err != nil {
		return false
	}
	return core.UserCanAccessProject(username, role, projectID)
}
