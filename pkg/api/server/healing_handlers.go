package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func handleIncidentHealingAction(w http.ResponseWriter, r *http.Request, idStr, action string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid incident id", http.StatusBadRequest)
		return
	}
	if core.HealingService == nil {
		writeJSONError(w, "Healing service not initialized", http.StatusServiceUnavailable)
		return
	}

	switch action {
	case "context":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims := parseClaims(r)
		if !core.CanViewHealing(claims.Role) {
			writeJSONError(w, "Access denied", http.StatusForbidden)
			return
		}
		ctx, err := core.HealingService.BuildContext(id)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ctx)
	case "propose":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FileContent string `json:"file_content"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		prop, err := core.HealingService.ProposeFix(id, req.FileContent)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prop)
	case "pr":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Repo     string `json:"repo"`
			FilePath string `json:"file_path"`
			Code     string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "Invalid body", http.StatusBadRequest)
			return
		}
		prURL, err := core.CreateRemediationPR(req.Repo, req.FilePath, req.Code)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"pr_url": prURL})
	default:
		http.NotFound(w, r)
	}
}

// ── /api/healing/gate — CI-triggered, framework-agnostic healing analysis ─────

// handleHealingGate is called by CI pipelines immediately after uploading test
// results.  It detects locator-based failures in the run's incidents, records
// each intervention in locator_healings, and fires a notification so operators
// know the MCP intervened without having to open the UI.
//
// Method : POST
// Auth   : X-API-Key (project API key — same key used for the JUnit upload)
// Headers: X-Run-Id   (required — GitHub Actions run_id or equivalent)
//          X-Framework (optional hint for better locator extraction)
func handleHealingGate(w http.ResponseWriter, r *http.Request, cfg *core.Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey == "" {
		writeJSONError(w, "Missing X-API-Key header", http.StatusUnauthorized)
		return
	}

	var projectName, slackChan, jiraKey, teamsHook string
	err := core.DB.QueryRow(
		"SELECT name, slack_channel, jira_project_key, teams_webhook FROM projects WHERE api_key = ?",
		apiKey,
	).Scan(&projectName, &slackChan, &jiraKey, &teamsHook)
	if err != nil {
		writeJSONError(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	runID := strings.TrimSpace(r.Header.Get("X-Run-Id"))
	if runID == "" {
		writeJSONError(w, "Missing X-Run-Id header", http.StatusBadRequest)
		return
	}
	framework := strings.TrimSpace(r.Header.Get("X-Framework"))

	// Detect locator failures already ingested for this run.
	failures, err := core.DetectLocatorFailuresForRun(runID, projectName)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("Detection error: %v", err), http.StatusInternalServerError)
		return
	}

	type HealingEntry struct {
		core.LocatorHealing
		TestName string `json:"test_name"`
	}

	var healings []HealingEntry
	for _, f := range failures {
		fw := framework
		if fw == "" {
			fw = f.Framework
		}
		healed, confidence, explanation := core.SuggestHealedLocator(f.OriginalLocator, fw)
		h, recErr := core.RecordLocatorHealing(
			f.IncidentID, runID, fw,
			f.OriginalLocator, healed, explanation, "mcp_gate", confidence,
		)
		if recErr != nil {
			continue
		}
		healings = append(healings, HealingEntry{LocatorHealing: *h, TestName: f.TestName})
	}

	// Fire notification through the existing remediation/plugin engine so
	// Slack/Teams/Jira receive a "[MCP] locator intervention" message.
	if len(healings) > 0 {
		routing := map[string]string{
			"SLACK_CHANNEL":     slackChan,
			"JIRA_PROJECT_KEY":  jiraKey,
			"TEAMS_WEBHOOK_URL": teamsHook,
		}
		firstName := healings[0].TestName
		detail := fmt.Sprintf("[MCP Healing Gate] %d locator intervention(s) detected in run %s.", len(healings), runID)
		alert := core.UnifiedAlert{
			Name:        fmt.Sprintf("[MCP] Locator healing — %s", firstName),
			Error:       detail,
			ConsoleLogs: detail,
			Status:      "MCP_HEALED",
		}
		core.RunPostIngestRemediation(*cfg, projectName, alert, routing, nil)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project":          projectName,
		"run_id":           runID,
		"locator_failures": len(failures),
		"interventions":    len(healings),
		"healings":         healings,
		"status":           "ok",
	})
}
