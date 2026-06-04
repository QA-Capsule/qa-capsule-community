package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	aiPkg "github.com/QA-Capsule/qa-capsule-community/pkg/ai"
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

		fileContent := strings.TrimSpace(req.FileContent)

		// Auto-read the test file from the local repo when no content was supplied.
		if fileContent == "" {
			fileContent = autoReadTestFile(id)
		}

		// Use the AI service when available — it calls Groq/Gemini directly.
		if core.AIService != nil {
			ctx := r.Context()
			code, explanation, err := core.AIService.ProposeFixFromIncidentID(ctx, id, fileContent)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":        code,
					"explanation": explanation,
					"confidence":  0.8,
				})
				return
			}
		}

		// Fallback to rule-based proposal when AI is not configured.
		prop, err := core.HealingService.ProposeFix(id, fileContent)
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

// aiLocatorHeal uses Groq/Gemini to find the real locator from the page DOM.
// Falls back to rule-based heuristics when AI is not available or DOM is absent.
func aiLocatorHeal(f core.LocatorError, framework string) (healed string, confidence float64, explanation string) {
	if core.AIService == nil || f.OriginalLocator == "" {
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	// Read console_logs for this incident to get the DOM snapshot.
	var consoleLogs string
	if core.DB != nil {
		core.DB.QueryRow(
			`SELECT COALESCE(console_logs,'') FROM incidents WHERE id = ?`, f.IncidentID,
		).Scan(&consoleLogs)
	}
	pageHTML := core.ExtractDOMSnapshot(consoleLogs)

	// Build a locator-healing prompt and call the AI.
	cfg, err := core.AIService.GetConfig(context.Background())
	if err != nil || !cfg.Enabled || cfg.Provider == aiPkg.ProviderDisabled {
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	prompt := buildLocatorPrompt(f.OriginalLocator, framework, pageHTML, f.ErrorMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, callErr := aiPkg.HTTPAnalyzer{}.Analyze(ctx, cfg, aiPkg.AnalysisInput{
		ProjectName:  "locator-healing",
		TestName:     f.TestName,
		Status:       "FAILED",
		ErrorMessage: prompt,
	})
	if callErr != nil {
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	// Parse JSON response: {"healed_locator":"...","confidence":0.9,"explanation":"..."}
	parsed := parseLocatorResponse(res.RawJSON)
	if parsed.healed != "" {
		return parsed.healed, parsed.confidence, parsed.explanation
	}
	return core.SuggestHealedLocator(f.OriginalLocator, framework)
}

func buildLocatorPrompt(original, framework, pageHTML, errorMsg string) string {
	var b strings.Builder
	b.WriteString("You are an expert test automation engineer specializing in self-healing tests.\n")
	b.WriteString("A UI test failed because a locator/selector no longer exists on the page.\n")
	b.WriteString("Your task: find the correct replacement locator from the page HTML.\n\n")
	b.WriteString("Respond ONLY with a JSON object (no markdown fences):\n")
	b.WriteString(`{"healed_locator":"<new selector>","confidence":0.95,"explanation":"<why>"}`)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Framework: %s\n", framework)
	fmt.Fprintf(&b, "Broken locator: %s\n", original)
	fmt.Fprintf(&b, "Error: %s\n\n", errorMsg)
	if pageHTML != "" {
		b.WriteString("Page HTML at time of failure:\n")
		if len(pageHTML) > 18000 {
			pageHTML = pageHTML[:18000]
		}
		b.WriteString(pageHTML)
	} else {
		b.WriteString("No page HTML available — suggest the most likely replacement based on the broken selector name.\n")
	}
	return b.String()
}

type locatorResult struct {
	healed      string
	confidence  float64
	explanation string
}

func parseLocatorResponse(raw string) locatorResult {
	raw = strings.TrimSpace(raw)
	i := strings.Index(raw, "{")
	if i < 0 {
		return locatorResult{}
	}
	var parsed struct {
		HealedLocator string  `json:"healed_locator"`
		Confidence    float64 `json:"confidence"`
		Explanation   string  `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(raw[i:]), &parsed); err == nil && parsed.HealedLocator != "" {
		return locatorResult{parsed.HealedLocator, parsed.Confidence, parsed.Explanation}
	}
	return locatorResult{}
}

// autoReadTestFile tries to find and read the test source file for an incident.
// It looks up the project's repo_path from the DB, then searches for the file
// referenced in the stack trace relative to that path.
func autoReadTestFile(incidentID int64) string {
	if core.DB == nil {
		return ""
	}
	var projectName, stackTrace string
	err := core.DB.QueryRow(
		`SELECT project_name, COALESCE(error_logs,'') FROM incidents WHERE id = ?`, incidentID,
	).Scan(&projectName, &stackTrace)
	if err != nil {
		return ""
	}

	// Get the repo path configured for this project.
	var repoPath string
	core.DB.QueryRow(
		`SELECT COALESCE(repo_path,'') FROM projects WHERE name = ?`, projectName,
	).Scan(&repoPath)

	// Try to extract a file path from the stack trace (e.g. "login_broken_selector.robot:27").
	filePath := extractFileFromStackTrace(stackTrace)
	if filePath == "" {
		return ""
	}

	// Resolve the path: repo_path / filePath (both absolute and relative attempts).
	candidates := []string{filePath}
	if repoPath != "" {
		candidates = append(candidates, filepath.Join(repoPath, filePath))
		candidates = append(candidates, filepath.Join(repoPath, filepath.Base(filePath)))
	}
	// Also search relative to the current working directory.
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, filePath))
		if repoPath != "" {
			candidates = append(candidates, filepath.Join(cwd, repoPath, filePath))
		}
	}

	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}
	return ""
}

// extractFileFromStackTrace parses lines like "at path/to/file.robot:27" or
// "path/to/file.robot line 27" and returns the bare file path without line number.
var stackFileRe = regexp.MustCompile(`(?i)([\w./\\-]+\.(robot|py|js|ts|java|cs|rb|go|feature))[:\s]`)

func extractFileFromStackTrace(stack string) string {
	m := stackFileRe.FindStringSubmatch(stack)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
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

		// Try AI-powered locator healing when AIService is configured.
		// The Robot listener may have captured the page DOM in console_logs.
		healed, confidence, explanation := aiLocatorHeal(f, fw)

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
