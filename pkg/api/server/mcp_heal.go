package server

// mcp_heal.go implements the heal_incident MCP tool, which is the single
// entry-point for AI-powered self-healing from an IDE agent (Cursor, Copilot,
// etc.).  Given an incident ID it:
//
//  1. Loads all telemetry stored for that incident (error message, stack trace,
//     captured page HTML, framework tag).
//  2. Classifies the failure as either a "locator" error (broken CSS/XPath
//     selector) or a "script_failure" (assertion, timeout, network, code bug).
//  3a. Locator path: extracts the captured DOM snapshot from console_logs,
//      builds a targeted prompt, and calls the configured AI provider to find
//      the correct replacement selector on the live page HTML.
//  3b. Script-failure path: reads the test source file from disk (using the
//      repo_path stored in the project settings plus the stack trace), then
//      calls the AI to generate a minimal, corrected version of the file.
//  4. Returns a single structured JSON result with the fix, a plain-English
//     explanation of what changed and why, a confidence score, and a list of
//     concrete next steps for the agent.

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	aiPkg "github.com/QA-Capsule/qa-capsule-community/pkg/ai"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

// HealResult is the structured payload returned by the heal_incident tool.
type HealResult struct {
	IncidentID int64  `json:"incident_id"`
	TestName   string `json:"test_name"`
	Project    string `json:"project"`
	Status     string `json:"status"`

	// ErrorType classifies the failure: "locator", "script_failure", or "unknown".
	ErrorType string `json:"error_type"`
	// Framework is the detected test framework (Playwright, Cypress, Robot, etc.).
	Framework string `json:"framework"`

	// AIPowered is true when the AI provider was used for the fix.
	AIPowered bool `json:"ai_powered"`
	// DOMCaptured is true when a page HTML snapshot was available and used.
	DOMCaptured bool `json:"dom_captured"`
	// FileRead is true when the test source file was read from disk.
	FileRead bool `json:"file_read"`

	// --- Locator fields (populated when ErrorType == "locator") ---

	// OriginalLocator is the selector that could not be found on the page.
	OriginalLocator string `json:"original_locator,omitempty"`
	// HealedLocator is the AI-proposed replacement selector.
	HealedLocator string `json:"healed_locator,omitempty"`

	// --- Script-failure fields (populated when ErrorType == "script_failure") ---

	// FilePath is the test source file that needs to be updated.
	FilePath string `json:"file_path,omitempty"`
	// CorrectedCode is the full corrected content of the test file.
	CorrectedCode string `json:"corrected_code,omitempty"`

	// --- Common fields ---

	// Explanation describes what was wrong and what the AI changed.
	Explanation string `json:"explanation"`
	// WhatChanged is a short, bullet-friendly summary of the specific change(s).
	WhatChanged string `json:"what_changed"`
	// Confidence is a 0–1 score for the AI's certainty.
	Confidence float64 `json:"confidence"`
	// NextSteps lists actionable instructions for the agent.
	NextSteps []string `json:"next_steps"`
}

// mcpHealIncident is the implementation of the heal_incident MCP tool.
// It orchestrates the full self-healing pipeline for a single incident.
func mcpHealIncident(incidentID int64) (map[string]interface{}, error) {
	if core.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// ── 1. Load all incident telemetry from the database ─────────────────────
	var (
		testName    string
		projectName string
		status      string
		errorMsg    string
		consoleLogs string
		stackTrace  string
	)
	err := core.DB.QueryRow(`
		SELECT
			COALESCE(name,''),
			COALESCE(project_name,''),
			COALESCE(status,''),
			COALESCE(error_message,''),
			COALESCE(console_logs,''),
			COALESCE(error_logs,'')
		FROM incidents WHERE id = ?`, incidentID,
	).Scan(&testName, &projectName, &status, &errorMsg, &consoleLogs, &stackTrace)
	if err != nil {
		return nil, fmt.Errorf("incident %d not found: %w", incidentID, err)
	}

	// ── 2. Classify the failure and detect the framework ─────────────────────
	framework := detectFramework(testName, stackTrace, errorMsg)
	testSource := autoReadTestFile(incidentID)
	combinedErr := errorMsg + "\n" + stackTrace
	isLocator := core.IsLocatorFailure(combinedErr, stackTrace, testSource, testName)

	errorType := "script_failure"
	if isLocator {
		errorType = "locator"
	}

	result := HealResult{
		IncidentID: incidentID,
		TestName:   testName,
		Project:    projectName,
		Status:     status,
		ErrorType:  errorType,
		Framework:  framework,
	}

	// ── 3a. Locator path ──────────────────────────────────────────────────────
	if isLocator {
		return mcpHealLocator(result, errorMsg, consoleLogs, stackTrace)
	}

	// ── 3b. Script-failure path ───────────────────────────────────────────────
	return mcpHealScriptFailure(result, stackTrace)
}

// mcpHealLocator handles the locator-healing branch of heal_incident.
// It extracts the captured DOM snapshot from console_logs and asks the
// AI to find the correct replacement selector on the page.
func mcpHealLocator(base HealResult, errorMsg, consoleLogs, stackTrace string) (map[string]interface{}, error) {
	testSource := autoReadTestFile(base.IncidentID)
	combinedErr := errorMsg + "\n" + stackTrace

	// Extract the broken locator from error logs and/or the test source file.
	originalLocator := core.ResolveOriginalLocator(combinedErr, testSource, base.Framework)
	base.OriginalLocator = originalLocator

	// Extract the page HTML that the test listener captured (stdout or stderr).
	pageHTML := core.ExtractDOMSnapshotFromLogs(consoleLogs, stackTrace)
	base.DOMCaptured = pageHTML != ""

	// Build the locator error struct needed by aiLocatorHeal.
	f := core.LocatorError{
		TestName:        base.TestName,
		Framework:       base.Framework,
		OriginalLocator: originalLocator,
		ErrorMessage:    errorMsg,
		IncidentID:      base.IncidentID,
	}

	healed, confidence, explanation := aiLocatorHeal(f, base.Framework)
	base.AIPowered = core.AIService != nil
	base.HealedLocator = healed
	base.Confidence = confidence
	base.Explanation = explanation
	base.WhatChanged = buildLocatorWhatChanged(originalLocator, healed, base.Framework)

	// Determine the test file path for the next-step guidance.
	filePath := extractFileFromStackTrace(stackTrace)
	base.NextSteps = buildLocatorNextSteps(base.IncidentID, filePath, originalLocator, healed)

	return mcpTextResult(base), nil
}

// mcpHealScriptFailure handles the script-failure branch of heal_incident.
// It reads the test source file from disk then asks the AI for a minimal
// corrected version.
func mcpHealScriptFailure(base HealResult, stackTrace string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Auto-read the test file referenced in the stack trace.
	fileContent := autoReadTestFile(base.IncidentID)
	base.FileRead = fileContent != ""
	base.FilePath = extractFileFromStackTrace(stackTrace)

	// Resolve the absolute path for display (best-effort).
	if base.FilePath != "" {
		if repoPath := getProjectRepoPath(base.Project); repoPath != "" {
			candidate := filepath.Join(repoPath, base.FilePath)
			base.FilePath = candidate
		}
	}

	// Call the AI service when available; fall back to rule-based otherwise.
	if core.AIService != nil {
		code, explanation, err := core.AIService.ProposeFixFromIncidentID(ctx, base.IncidentID, fileContent)
		if err == nil {
			base.AIPowered = true
			base.CorrectedCode = code
			base.Explanation = explanation
			base.WhatChanged = buildScriptWhatChanged(explanation)
			base.Confidence = 0.80
			base.NextSteps = buildScriptNextSteps(base.IncidentID, base.FilePath)
			return mcpTextResult(base), nil
		}
	}

	// Rule-based fallback.
	if core.HealingService == nil {
		return nil, fmt.Errorf("healing service not initialized")
	}
	prop, err := core.HealingService.ProposeFix(base.IncidentID, fileContent)
	if err != nil {
		return nil, err
	}
	base.AIPowered = false
	base.CorrectedCode = prop.Code
	base.Explanation = prop.Explanation
	base.WhatChanged = buildScriptWhatChanged(prop.Explanation)
	base.Confidence = prop.Confidence
	base.NextSteps = buildScriptNextSteps(base.IncidentID, base.FilePath)
	return mcpTextResult(base), nil
}

// ── Framework detection ────────────────────────────────────────────────────

// detectFramework infers the test framework from the test name prefix
// (e.g. "[Playwright]"), the stack trace file extension, or error keywords.
func detectFramework(testName, stackTrace, errorMsg string) string {
	// Test name tags injected by the ingest worker (e.g. "[Robot] Login > Test").
	for _, fw := range []string{"playwright", "cypress", "selenium", "robot", "pytest", "jest", "vitest"} {
		if strings.Contains(strings.ToLower(testName), fw) {
			return canonicalFramework(fw)
		}
	}

	// Stack trace file extensions.
	lower := strings.ToLower(stackTrace + " " + errorMsg)
	switch {
	case strings.Contains(lower, ".robot") || strings.Contains(lower, "robotframework"):
		return "RobotFramework"
	case strings.Contains(lower, ".spec.ts") || strings.Contains(lower, ".spec.js") ||
		strings.Contains(lower, "playwright"):
		return "Playwright"
	case strings.Contains(lower, "cypress"):
		return "Cypress"
	case strings.Contains(lower, "selenium") || strings.Contains(lower, "nosuchelement"):
		return "Selenium"
	case strings.Contains(lower, ".feature") || strings.Contains(lower, "cucumber"):
		return "Cucumber"
	case strings.Contains(lower, "pytest") || strings.Contains(lower, ".py"):
		return "Pytest"
	}
	return "unknown"
}

// canonicalFramework maps lowercase identifiers to the display names used
// throughout the codebase (must match locator pattern labels in locator_healing.go).
func canonicalFramework(fw string) string {
	switch fw {
	case "playwright":
		return "Playwright"
	case "cypress":
		return "Cypress"
	case "selenium":
		return "Selenium"
	case "robot":
		return "RobotFramework"
	case "pytest":
		return "Pytest"
	case "jest", "vitest":
		return "Jest"
	default:
		return fw
	}
}

// ── Helper: project repo path ─────────────────────────────────────────────

// getProjectRepoPath looks up the repo_path configured for a project so
// the file path in HealResult can be presented as an absolute path.
func getProjectRepoPath(projectName string) string {
	if core.DB == nil || projectName == "" {
		return ""
	}
	var repoPath string
	core.DB.QueryRow(`SELECT COALESCE(repo_path,'') FROM projects WHERE name = ?`, projectName).Scan(&repoPath)
	return repoPath
}

// ── Next-step builders ────────────────────────────────────────────────────

// buildLocatorNextSteps returns ordered instructions for the IDE agent after a
// locator fix has been proposed.
func buildLocatorNextSteps(incidentID int64, filePath, original, healed string) []string {
	steps := []string{}
	if filePath != "" {
		steps = append(steps, fmt.Sprintf(
			"Open %q and replace %q with %q", filePath, original, healed,
		))
	} else {
		steps = append(steps, fmt.Sprintf(
			"Find the test file that uses %q and replace it with %q", original, healed,
		))
	}
	if filePath != "" {
		steps = append(steps, fmt.Sprintf(
			"Call submit_healing_patch with incident_id=%d, file_path=%q, and the corrected file content",
			incidentID, filePath,
		))
		steps = append(steps, "Call create_remediation_pr to open a GitHub PR with the fix")
	}
	steps = append(steps, fmt.Sprintf(
		"Call record_healing_intervention with incident_id=%d, original_locator=%q, healed_locator=%q to log the intervention",
		incidentID, original, healed,
	))
	steps = append(steps, fmt.Sprintf("Call resolve_incident with incident_id=%d after validating the fix", incidentID))
	return steps
}

// buildScriptNextSteps returns ordered instructions for the IDE agent after a
// script fix has been proposed.
func buildScriptNextSteps(incidentID int64, filePath string) []string {
	steps := []string{}
	if filePath != "" {
		steps = append(steps, fmt.Sprintf("Review the corrected_code and apply it to %q", filePath))
		steps = append(steps, fmt.Sprintf(
			"Call submit_healing_patch with incident_id=%d, file_path=%q, and the corrected code",
			incidentID, filePath,
		))
		steps = append(steps, "Call create_remediation_pr to open a GitHub PR with the fix")
	} else {
		steps = append(steps, "Identify the failing test file from the stack trace")
		steps = append(steps, "Apply the changes described in corrected_code and explanation")
	}
	steps = append(steps, fmt.Sprintf("Call resolve_incident with incident_id=%d after re-running the test successfully", incidentID))
	return steps
}

// ── Explanation helpers ───────────────────────────────────────────────────

// buildLocatorWhatChanged produces a concise one-liner that describes the
// selector change for the WhatChanged field.
func buildLocatorWhatChanged(original, healed, framework string) string {
	if healed == "" || healed == original {
		return fmt.Sprintf("No replacement found for %q — manual inspection required", original)
	}
	return fmt.Sprintf(
		"Replace %q with %q in the %s test file",
		original, healed, framework,
	)
}

// buildScriptWhatChanged extracts the first sentence of the AI explanation as
// a compact WhatChanged summary.
func buildScriptWhatChanged(explanation string) string {
	explanation = strings.TrimSpace(explanation)
	if explanation == "" {
		return "See corrected_code for the proposed fix"
	}
	// First sentence only — keep it short for display.
	if i := strings.IndexAny(explanation, ".!?\n"); i > 0 && i < 200 {
		return explanation[:i+1]
	}
	if len(explanation) > 200 {
		return explanation[:200] + "..."
	}
	return explanation
}

// ── MCP tool used via AI provider directly  ──────────────────────────────

// mcpAIHealLocatorDirect calls the AI directly via the configured provider
// without going through the healing gate. Used when the MCP agent has already
// detected a locator failure and passes the page HTML explicitly.
// This is exposed as a secondary entry-point in propose_healing.
func mcpAIHealLocatorDirect(incidentID int64, originalLocator, framework, pageHTML string) (map[string]interface{}, error) {
	if core.AIService == nil {
		healed, conf, expl := core.SuggestHealedLocator(originalLocator, framework)
		return mcpTextResult(map[string]interface{}{
			"healed_locator": healed,
			"confidence":     conf,
			"explanation":    expl,
			"ai_powered":     false,
		}), nil
	}

	cfg, err := core.AIService.GetConfig(context.Background())
	if err != nil || !cfg.Enabled || cfg.Provider == aiPkg.ProviderDisabled {
		healed, conf, expl := core.SuggestHealedLocator(originalLocator, framework)
		return mcpTextResult(map[string]interface{}{
			"healed_locator": healed,
			"confidence":     conf,
			"explanation":    expl,
			"ai_powered":     false,
		}), nil
	}

	f := core.LocatorError{
		IncidentID:      incidentID,
		OriginalLocator: originalLocator,
		Framework:       framework,
	}
	if core.DB != nil && incidentID > 0 {
		var errorMsg string
		core.DB.QueryRow(`SELECT COALESCE(error_message,'') FROM incidents WHERE id = ?`, incidentID).Scan(&errorMsg)
		f.ErrorMessage = errorMsg
	}

	// If no page HTML was passed in, try console_logs and error_logs.
	if pageHTML == "" && incidentID > 0 && core.DB != nil {
		var consoleLogs, errorLogs string
		core.DB.QueryRow(
			`SELECT COALESCE(console_logs,''), COALESCE(error_logs,'') FROM incidents WHERE id = ?`,
			incidentID,
		).Scan(&consoleLogs, &errorLogs)
		pageHTML = core.ExtractDOMSnapshotFromLogs(consoleLogs, errorLogs)
	}

	prompt := buildLocatorPrompt(originalLocator, framework, pageHTML, f.ErrorMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	rawText, callErr := aiPkg.AnalyzeRaw(ctx, cfg, prompt)
	if callErr != nil {
		if healed, conf, expl, ok := core.HealLocatorFromHTML(originalLocator, pageHTML); ok {
			return mcpTextResult(map[string]interface{}{
				"healed_locator": healed,
				"confidence":     conf,
				"explanation":    expl,
				"ai_powered":     false,
				"dom_used":       pageHTML != "",
			}), nil
		}
		healed, conf, expl := core.SuggestHealedLocator(originalLocator, framework)
		return mcpTextResult(map[string]interface{}{
			"healed_locator": healed,
			"confidence":     conf,
			"explanation":    expl,
			"ai_powered":     false,
			"ai_error":       callErr.Error(),
		}), nil
	}

	parsed := parseLocatorResponse(rawText)
	if parsed.healed != "" && parsed.healed != originalLocator {
		if pageHTML == "" || core.LocatorExistsInHTML(pageHTML, parsed.healed) {
			return mcpTextResult(map[string]interface{}{
				"healed_locator": parsed.healed,
				"confidence":     parsed.confidence,
				"explanation":    parsed.explanation,
				"ai_powered":     true,
				"dom_used":       pageHTML != "",
			}), nil
		}
	}
	if healed, conf, expl, ok := core.HealLocatorFromHTML(originalLocator, pageHTML); ok {
		return mcpTextResult(map[string]interface{}{
			"healed_locator": healed,
			"confidence":     conf,
			"explanation":    expl,
			"ai_powered":     false,
			"dom_used":       pageHTML != "",
		}), nil
	}

	healed, conf, expl := core.SuggestHealedLocator(originalLocator, framework)
	return mcpTextResult(map[string]interface{}{
		"healed_locator": healed,
		"confidence":     conf,
		"explanation":    expl,
		"ai_powered":     false,
	}), nil
}
