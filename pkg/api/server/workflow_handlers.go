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
<<<<<<< HEAD
		projectID := ""
		isSimulate := strings.HasSuffix(path, "/workflow/simulate")
		switch {
		case isSimulate:
			projectID = strings.TrimSuffix(path, "/workflow/simulate")
		case strings.HasSuffix(path, "/workflow"):
			projectID = strings.TrimSuffix(path, "/workflow")
		default:
			http.NotFound(w, r)
			return
		}
=======
		if !strings.HasSuffix(path, "/workflow") {
			http.NotFound(w, r)
			return
		}
		projectID := strings.TrimSuffix(path, "/workflow")
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
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

<<<<<<< HEAD
		if isSimulate {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if !core.CanManageWorkflow(claims.Role) {
				writeJSONError(w, "Workflow simulate requires Lead, Manager, or Platform Admin", http.StatusForbidden)
				return
			}
			handleSimulateWorkflow(w, r, projectID)
			return
		}

=======
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
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

<<<<<<< HEAD
func handleSimulateWorkflow(w http.ResponseWriter, r *http.Request, projectID string) {
	var payload struct {
		Name     string                          `json:"name"`
		Status   string                          `json:"status"`
		Error    string                          `json:"error"`
		Workflow *integrations.WorkflowDocument `json:"workflow,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.Name == "" {
		payload.Name = "[FLAKY] checkout payment"
	}
	if payload.Status == "" {
		payload.Status = "CRITICAL"
	}
	if payload.Error == "" {
		payload.Error = "timeout waiting for upstream"
	}

	var doc *integrations.WorkflowDocument
	if payload.Workflow != nil && payload.Workflow.Entry != "" && len(payload.Workflow.Nodes) > 0 {
		doc = payload.Workflow
	} else {
		var err error
		doc, err = core.LoadProjectWorkflowByID(projectID)
		if err != nil {
			writeJSONError(w, "Database error", http.StatusInternalServerError)
			return
		}
	}
	if doc == nil || doc.Entry == "" || len(doc.Nodes) == 0 {
		writeJSONError(w, "No workflow on canvas or saved for this gateway", http.StatusBadRequest)
		return
	}
	reg := core.RemediationRegistry()
	if err := integrations.ValidateWorkflowStructure(doc, reg); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	engine := core.RemediationEngine()
	if engine == nil {
		writeJSONError(w, "Remediation engine not initialized", http.StatusServiceUnavailable)
		return
	}
	_, allowed := projectAllowedPathsByID(projectID)
	wctx := integrations.WorkflowContext{
		Incident: integrations.IncidentContext{
			Name:   payload.Name,
			Error:  payload.Error,
			Status: payload.Status,
			Action: "AUTO_EVENT:" + payload.Name,
		},
		Tags:    integrations.DeriveTags(payload.Name),
		Allowed: allowed,
	}
	plan := integrations.NewWorkflowEngine(engine).PlanExecution(doc, wctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sample": map[string]string{
			"name":   payload.Name,
			"status": payload.Status,
			"error":  payload.Error,
		},
		"plan": plan,
	})
}

=======
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
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
