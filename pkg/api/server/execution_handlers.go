package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func registerExecutionRoutes(config *core.Config) {
	http.HandleFunc("/api/executions/", jwtAuthMiddleware(config, "", handleExecutionSubroutes))
}

func handleExecutionSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/executions/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	runID := parts[0]
	action := parts[1]
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if project == "" {
		writeJSONError(w, "project query parameter required", http.StatusBadRequest)
		return
	}

	switch action {
	case "flag":
		handleExecutionFlagPatch(w, r, project, runID)
	case "report":
		handleExecutionReportGet(w, r, project, runID)
	default:
		http.NotFound(w, r)
	}
}

func handleExecutionFlagPatch(w http.ResponseWriter, r *http.Request, project, runID string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims := parseClaims(r)
	if !core.CanPatchExecutionFlags(claims.Role) {
		writeJSONError(w, "Lead or higher role required to update execution flags", http.StatusForbidden)
		return
	}
	var req core.PatchExecutionFlagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	existing, err := core.LoadPipelineRun(project, runID)
	if err != nil {
		existing = &core.PipelineRunRecord{Flags: core.ExecutionFlags{Env: core.ExecutionEnvUnknown, Type: core.ExecutionTypeReal}}
	}
	flags := existing.Flags
	if strings.TrimSpace(req.Env) != "" {
		flags.Env = core.NormalizeExecutionEnv(req.Env)
	}
	if strings.TrimSpace(req.Type) != "" {
		flags.Type = core.NormalizeExecutionType(req.Type)
	}
	if flags.Type == core.ExecutionTypeUnknown {
		flags.Type = core.ExecutionTypeReal
	}
	if !flags.Valid() {
		writeJSONError(w, "Invalid execution env or type", http.StatusBadRequest)
		return
	}
	if err := core.UpdatePipelineFlags(project, runID, flags); err != nil {
		writeJSONError(w, "Failed to update execution flags", http.StatusInternalServerError)
		return
	}
	rec, err := core.LoadPipelineRun(project, runID)
	if err != nil {
		writeJSONError(w, "Pipeline run not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

func handleExecutionReportGet(w http.ResponseWriter, r *http.Request, project, runID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rec, err := core.LoadPipelineRun(project, runID)
	if err != nil {
		writeJSONError(w, "Pipeline run not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if rec.Report != nil {
		json.NewEncoder(w).Encode(rec.Report)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schema_version": 1,
		"flags":          rec.Flags,
		"summary":        rec.Summary,
		"tests":          []core.TestCaseResult{},
	})
}
