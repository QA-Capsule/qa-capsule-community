package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

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
			"description": "Return standardized QA Capsule telemetry for one incident (test name, error, stack trace, duration, CI tags).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id": map[string]interface{}{"type": "integer", "description": "Incident primary key"},
				},
				"required": []string{"incident_id"},
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
	case "get_flaky_tests":
		project, _ := call.Arguments["project"].(string)
		limit := 100
		if v, ok := call.Arguments["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
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
		tel, err := core.LoadIncidentTelemetry(id)
		if err != nil {
			return nil, err
		}
		return mcpTextResult(tel), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
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
