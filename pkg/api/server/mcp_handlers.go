package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

const mcpProtocolVersion = "2024-11-05"

func registerMCPRoutes(config *core.Config) {
	http.HandleFunc("/mcp", mcpAuthMiddleware(config, handleMCP))
}

func mcpAuthMiddleware(config *core.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := strings.TrimSpace(os.Getenv("QACAPSULE_MCP_TOKEN"))
		if expected == "" && config != nil {
			expected = strings.TrimSpace(config.Telemetry.WebhookToken)
		}
		if expected != "" {
			got := strings.TrimSpace(r.Header.Get("Authorization"))
			got = strings.TrimPrefix(got, "Bearer ")
			if got == "" {
				got = r.Header.Get("X-MCP-Token")
			}
			if got != expected {
				writeJSONRPCError(w, nil, -32001, "Unauthorized MCP request")
				return
			}
		}
		next(w, r)
	}
}

func handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "MCP requires POST JSON-RPC", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error")
		return
	}
	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error")
		return
	}
	switch req.Method {
	case "initialize":
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"protocolVersion": mcpProtocolVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "qa-capsule",
				"version": "1.0.0",
			},
		})
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"tools": mcpToolDefinitions(),
		})
	case "tools/call":
		result, callErr := dispatchMCPTool(req.Params)
		if callErr != nil {
			writeJSONRPCError(w, req.ID, -32000, callErr.Error())
			return
		}
		writeJSONRPCResult(w, req.ID, result)
	default:
		writeJSONRPCError(w, req.ID, -32601, "Method not found: "+req.Method)
	}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func mcpToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "list_failed_incidents",
			"description": "List open failed incidents with framework-agnostic healing categories.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project": map[string]string{"type": "string", "description": "Optional project filter"},
					"limit":   map[string]interface{}{"type": "integer", "description": "Max rows (default 100)"},
				},
			},
		},
		{
			"name":        "get_flaky_tests",
			"description": "List tests tagged [FLAKY] with SHA-256 identity fingerprint and failure rate (framework-agnostic).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project": map[string]string{"type": "string", "description": "Optional project filter"},
					"limit":   map[string]interface{}{"type": "integer", "description": "Max rows (default 100)"},
				},
			},
		},
		{
			"name":        "get_incident_context",
			"description": "Return standardized self-healing context for one incident (telemetry + category + actionable hints).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id": map[string]interface{}{"type": "integer", "description": "Incident primary key"},
				},
				"required": []string{"incident_id"},
			},
		},
		{
			"name":        "propose_healing",
			"description": "Generate framework-agnostic healing guidance for one incident (no internal LLM required).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id":  map[string]interface{}{"type": "integer", "description": "Incident primary key"},
					"file_content": map[string]interface{}{"type": "string", "description": "Optional source file content"},
				},
				"required": []string{"incident_id"},
			},
		},
		{
			"name":        "submit_healing_patch",
			"description": "Validate and register a patch proposal payload for an incident before PR creation.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id":  map[string]interface{}{"type": "integer", "description": "Incident primary key"},
					"repo":         map[string]interface{}{"type": "string", "description": "GitHub repo owner/name"},
					"file_path":    map[string]interface{}{"type": "string", "description": "Path to updated file"},
					"code":         map[string]interface{}{"type": "string", "description": "Full updated file content"},
					"explanation":  map[string]interface{}{"type": "string", "description": "Short rationale"},
					"agent_source": map[string]interface{}{"type": "string", "description": "Agent label (e.g. cursor_mcp)"},
				},
				"required": []string{"incident_id", "repo", "file_path", "code"},
			},
		},
		{
			"name":        "create_remediation_pr",
			"description": "Create a GitHub PR for a self-healing code proposal.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo":      map[string]interface{}{"type": "string", "description": "GitHub repo owner/name"},
					"file_path": map[string]interface{}{"type": "string", "description": "Path to file in repository"},
					"code":      map[string]interface{}{"type": "string", "description": "Full updated file content"},
				},
				"required": []string{"repo", "file_path", "code"},
			},
		},
		{
			"name":        "resolve_incident",
			"description": "Mark one incident as resolved after successful validation/rerun.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id": map[string]interface{}{"type": "integer", "description": "Incident primary key"},
					"resolved_by": map[string]interface{}{"type": "string", "description": "Optional resolver identity"},
				},
				"required": []string{"incident_id"},
			},
		},
		{
			"name":        "detect_locator_failures",
			"description": "Scan all failed incidents for a pipeline run and return those caused by missing/broken selectors (framework-agnostic: Playwright, Cypress, Selenium, Robot, Pytest-Selenium).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"run_id":  map[string]interface{}{"type": "string", "description": "CI pipeline run ID (e.g. GitHub Actions run_id)"},
					"project": map[string]interface{}{"type": "string", "description": "Optional project name filter"},
				},
				"required": []string{"run_id"},
			},
		},
		{
			"name":        "record_healing_intervention",
			"description": "Record that the MCP agent healed a locator for a specific incident and optionally propose an alternative selector. Also triggers a notification (Slack/Teams if configured).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id":      map[string]interface{}{"type": "integer", "description": "Incident primary key"},
					"run_id":           map[string]interface{}{"type": "string", "description": "CI pipeline run ID"},
					"framework":        map[string]interface{}{"type": "string", "description": "Test framework (e.g. Playwright, Selenium, Cypress)"},
					"original_locator": map[string]interface{}{"type": "string", "description": "The selector that failed"},
					"healed_locator":   map[string]interface{}{"type": "string", "description": "The replacement selector proposed by the agent"},
					"explanation":      map[string]interface{}{"type": "string", "description": "Short rationale for the change"},
					"confidence":       map[string]interface{}{"type": "number", "description": "Confidence score 0–1"},
					"agent_source":     map[string]interface{}{"type": "string", "description": "Agent identifier (e.g. cursor_mcp, github_copilot)"},
				},
				"required": []string{"incident_id", "original_locator"},
			},
		},
	}
}

func dispatchMCPTool(params json.RawMessage) (map[string]interface{}, error) {
	var call struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &call)
	}
	switch call.Name {
	case "list_failed_incidents":
		if core.HealingService == nil {
			return nil, fmt.Errorf("healing service not initialized")
		}
		project := parseStringArg(call.Arguments, "project")
		limit := parseLimitArg(call.Arguments, 100)
		rows, err := core.HealingService.ListInsights(context.Background(), project, limit)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(rows), nil
	case "get_flaky_tests":
		project := parseStringArg(call.Arguments, "project")
		limit := parseLimitArg(call.Arguments, 100)
		rows, err := core.ListFlakyTests(project, limit)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(rows), nil
	case "get_incident_context":
		id, err := parseIncidentIDArg(call.Arguments)
		if err != nil {
			return nil, err
		}
		if core.HealingService != nil {
			ctx, ctxErr := core.HealingService.BuildContext(id)
			if ctxErr == nil {
				return mcpTextResult(ctx), nil
			}
		}
		tel, telErr := core.LoadIncidentTelemetry(id)
		if telErr != nil {
			return nil, telErr
		}
		return mcpTextResult(tel), nil
	case "propose_healing":
		if core.HealingService == nil {
			return nil, fmt.Errorf("healing service not initialized")
		}
		id, err := parseIncidentIDArg(call.Arguments)
		if err != nil {
			return nil, err
		}
		fileContent := parseStringArg(call.Arguments, "file_content")
		prop, err := core.HealingService.ProposeFix(id, fileContent)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(prop), nil
	case "submit_healing_patch":
		if core.HealingService == nil {
			return nil, fmt.Errorf("healing service not initialized")
		}
		id, err := parseIncidentIDArg(call.Arguments)
		if err != nil {
			return nil, err
		}
		repo := parseStringArg(call.Arguments, "repo")
		filePath := parseStringArg(call.Arguments, "file_path")
		code := parseStringArg(call.Arguments, "code")
		if repo == "" || filePath == "" || code == "" {
			return nil, fmt.Errorf("repo, file_path, and code are required")
		}
		explanation := parseStringArg(call.Arguments, "explanation")
		if explanation == "" {
			explanation = "Submitted via MCP self-healing workflow."
		}
		agentSource := parseStringArg(call.Arguments, "agent_source")
		if agentSource == "" {
			agentSource = "mcp_agent"
		}
		sub, err := core.HealingService.RegisterPatchSubmission(id, repo, filePath, code, explanation, agentSource)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(map[string]interface{}{
			"status":          sub.Status,
			"submission_id":   sub.ID,
			"incident_id":     sub.IncidentID,
			"repo":            sub.Repo,
			"file_path":       sub.FilePath,
			"code_sha256":     sub.CodeSHA256,
			"code_size_bytes": sub.CodeSize,
			"explanation":     sub.Explanation,
			"agent_source":    sub.AgentSource,
			"submitted_at":    time.Now().UTC().Format(time.RFC3339),
			"next_step":       "call create_remediation_pr with repo/file_path/code",
		}), nil
	case "create_remediation_pr":
		repo := parseStringArg(call.Arguments, "repo")
		filePath := parseStringArg(call.Arguments, "file_path")
		code := parseStringArg(call.Arguments, "code")
		if repo == "" || filePath == "" || code == "" {
			return nil, fmt.Errorf("repo, file_path, and code are required")
		}
		prURL, err := core.CreateRemediationPR(repo, filePath, code)
		if err != nil {
			return nil, err
		}
		if core.HealingService != nil {
			incidentID, _ := parseIncidentIDArg(call.Arguments)
			if incidentID > 0 {
				_ = core.HealingService.MarkPatchPR(incidentID, repo, filePath, code, prURL)
			}
		}
		return mcpTextResult(map[string]interface{}{
			"status": "created",
			"pr_url": prURL,
		}), nil
	case "resolve_incident":
		id, err := parseIncidentIDArg(call.Arguments)
		if err != nil {
			return nil, err
		}
		resolvedBy := parseStringArg(call.Arguments, "resolved_by")
		if resolvedBy == "" {
			resolvedBy = "mcp_agent"
		}
		if err := resolveIncidentByID(id, resolvedBy); err != nil {
			return nil, err
		}
		return mcpTextResult(map[string]interface{}{
			"status":      "resolved",
			"incident_id": id,
			"resolved_by": resolvedBy,
		}), nil

	case "detect_locator_failures":
		runID := parseStringArg(call.Arguments, "run_id")
		if runID == "" {
			return nil, fmt.Errorf("run_id is required")
		}
		project := parseStringArg(call.Arguments, "project")
		failures, err := core.DetectLocatorFailuresForRun(runID, project)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(map[string]interface{}{
			"run_id":          runID,
			"locator_failures": failures,
			"count":           len(failures),
			"next_step":       "call record_healing_intervention for each failure, providing healed_locator",
		}), nil

	case "record_healing_intervention":
		id, err := parseIncidentIDArg(call.Arguments)
		if err != nil {
			return nil, err
		}
		runID := parseStringArg(call.Arguments, "run_id")
		framework := parseStringArg(call.Arguments, "framework")
		originalLocator := parseStringArg(call.Arguments, "original_locator")
		if originalLocator == "" {
			return nil, fmt.Errorf("original_locator is required")
		}
		healedLocator := parseStringArg(call.Arguments, "healed_locator")
		explanation := parseStringArg(call.Arguments, "explanation")
		agentSource := parseStringArg(call.Arguments, "agent_source")
		var confidence float64
		if v, ok := call.Arguments["confidence"].(float64); ok {
			confidence = v
		}
		if healedLocator == "" {
			healedLocator, confidence, explanation = core.SuggestHealedLocator(originalLocator, framework)
		}
		healing, err := core.RecordLocatorHealing(id, runID, framework, originalLocator, healedLocator, explanation, agentSource, confidence)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(map[string]interface{}{
			"status":           "recorded",
			"healing_id":       healing.ID,
			"incident_id":      healing.IncidentID,
			"original_locator": healing.OriginalLocator,
			"healed_locator":   healing.HealedLocator,
			"confidence":       healing.Confidence,
			"explanation":      healing.Explanation,
			"agent_source":     healing.AgentSource,
			"next_step":        "apply healed_locator in the test file, then call create_remediation_pr",
		}), nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func parseStringArg(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

func parseLimitArg(args map[string]interface{}, def int) int {
	if args == nil {
		return def
	}
	if v, ok := args["limit"].(float64); ok && v > 0 {
		return int(v)
	}
	return def
}

func parseIncidentIDArg(args map[string]interface{}) (int64, error) {
	if args == nil {
		return 0, fmt.Errorf("incident_id required")
	}
	switch v := args["incident_id"].(type) {
	case float64:
		if v <= 0 {
			return 0, fmt.Errorf("invalid incident_id")
		}
		return int64(v), nil
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid incident_id")
		}
		return id, nil
	default:
		return 0, fmt.Errorf("incident_id required")
	}
}

func resolveIncidentByID(incidentID int64, resolvedBy string) error {
	if core.HealingService != nil {
		return core.HealingService.ResolveIncident(incidentID, resolvedBy)
	}
	if core.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	resolvedBy = strings.TrimSpace(resolvedBy)
	if resolvedBy == "" {
		resolvedBy = "mcp_agent"
	}
	res, err := core.DB.Exec(
		"UPDATE incidents SET is_resolved = 1, status = 'resolved', resolved_by = ?, resolved_at = CURRENT_TIMESTAMP WHERE id = ?",
		resolvedBy, incidentID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("incident not found")
	}
	return nil
}

func mcpTextResult(payload interface{}) map[string]interface{} {
	data, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": string(data)},
		},
		"isError": false,
	}
}

func writeJSONRPCResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}
