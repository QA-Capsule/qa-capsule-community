package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
	"github.com/QA-Capsule/qa-capsule-community/pkg/integrations"

	"github.com/golang-jwt/jwt/v5"
)

func registerWorkflowRoutes(config *core.Config) {
	http.HandleFunc("/api/projects/", jwtAuthMiddleware(config, "", handleProjectWorkflow(config)))
}

func handleProjectWorkflow(config *core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		path = strings.TrimSuffix(path, "/")
		if !strings.HasSuffix(path, "/workflow") {
			http.NotFound(w, r)
			return
		}
		projectID := strings.TrimSuffix(path, "/workflow")
		projectID = strings.Trim(projectID, "/")
		if projectID == "" {
			writeJSONError(w, "project id required", http.StatusBadRequest)
			return
		}

		authHeader := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		_, _ = jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })

		if !core.UserCanAccessProject(claims.Username, claims.Role, projectID) {
			writeJSONError(w, "Access denied", http.StatusForbidden)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleGetWorkflow(w, r, projectID, claims)
		case http.MethodPut:
			if !core.CanManageWorkflow(claims.Role) {
				writeJSONError(w, "Workflow edit requires Lead, Manager, or Platform Admin", http.StatusForbidden)
				return
			}
			handlePutWorkflow(w, r, projectID)
		case http.MethodDelete:
			if !core.CanManageWorkflow(claims.Role) {
				writeJSONError(w, "Workflow edit requires Lead, Manager, or Platform Admin", http.StatusForbidden)
				return
			}
			handleDeleteWorkflow(w, projectID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetWorkflow(w http.ResponseWriter, r *http.Request, projectID string, claims *Claims) {
	doc, err := core.LoadProjectWorkflowByID(projectID)
	if err != nil {
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	_ = projectID

	routingEntries := loadRoutingEntriesForProject(projectID)
	catalog := workflowCatalog()
	wfSummary := core.WorkflowSummary{}
	if doc != nil {
		wfSummary = core.WorkflowSummary{
			HasWorkflow: doc.Entry != "" || len(doc.Nodes) > 0,
			Enabled:     integrations.IsWorkflowActive(doc),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id":       projectID,
		"workflow":         doc,
		"can_edit":         core.CanManageWorkflow(claims.Role),
		"has_workflow":     wfSummary.HasWorkflow,
		"workflow_enabled": wfSummary.Enabled,
		"routing_entries":  routingEntries,
		"catalog":          catalog,
	})
}

func handlePutWorkflow(w http.ResponseWriter, r *http.Request, projectID string) {
	var doc integrations.WorkflowDocument
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		writeJSONError(w, "Invalid workflow JSON", http.StatusBadRequest)
		return
	}
	if doc.Version == 0 {
		doc.Version = integrations.WorkflowSchemaVersion
	}
	if doc.Meta.UpdatedAt == "" {
		doc.Meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	reg := core.RemediationRegistry()
	_, allowed := projectAllowedPathsByID(projectID)
	if doc.Enabled {
		if err := integrations.ValidateWorkflow(&doc, reg, allowed); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if err := core.SaveProjectWorkflow(projectID, &doc); err != nil {
		writeJSONError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleDeleteWorkflow(w http.ResponseWriter, projectID string) {
	if err := core.ClearProjectWorkflow(projectID); err != nil {
		writeJSONError(w, "Failed to clear workflow", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func loadRoutingEntriesForProject(projectID string) []core.SRERoutingEntry {
	var raw string
	if err := core.DB.QueryRow(`SELECT sre_routing_json FROM projects WHERE id = ?`, projectID).Scan(&raw); err != nil {
		return nil
	}
	return core.ParseSRERoutingJSON(raw)
}

func projectAllowedPathsByID(projectID string) (string, map[string]bool) {
	var name, routingJSON string
	if err := core.DB.QueryRow(`SELECT name, sre_routing_json FROM projects WHERE id = ?`, projectID).Scan(&name, &routingJSON); err != nil {
		return "", nil
	}
	_, allowed := core.ProjectAlertContext(name)
	return name, allowed
}

func workflowCatalog() []map[string]interface{} {
	reg := core.RemediationRegistry()
	if reg == nil {
		return nil
	}
	out := make([]map[string]interface{}, 0)
	for _, m := range reg.List() {
		if !integrations.IsActiveForRouting(m.Status, m.RoutingEnabled, m.AutoRun) {
			continue
		}
		out = append(out, map[string]interface{}{
			"file_path":      m.FilePath,
			"integration":    m.Integration,
			"name":           m.Name,
			"routing_fields": integrations.RoutingFieldsForIntegration(m.Integration),
		})
	}
	return out
}
