package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

		// Load the incident to determine the failure type before choosing a strategy.
		var errMsg, consoleLogs, errorLogs, framework, testName string
		if core.DB != nil {
			core.DB.QueryRow(
				`SELECT COALESCE(error_message,''), COALESCE(console_logs,''),
				        COALESCE(error_logs,''), COALESCE(framework,''), COALESCE(name,'')
				 FROM incidents WHERE id = ?`, id,
			).Scan(&errMsg, &consoleLogs, &errorLogs, &framework, &testName)
		}

		// Diagnostic logs — visible in the server console to help triage issues.
		domSnapshot := core.ExtractDOMSnapshotFromLogs(consoleLogs, errorLogs)

		testSource := fileContent
		if testSource == "" {
			testSource = autoReadTestFile(id)
		}
		combinedErr := errMsg + "\n" + errorLogs
		isLocatorFail := core.IsLocatorFailure(combinedErr, errorLogs, testSource, testName)

		slog.Info("propose-fix: incident loaded",
			"incident_id", id,
			"framework", framework,
			"is_locator_error", isLocatorFail,
			"test_source_bytes", len(testSource),
			"dom_captured", domSnapshot != "",
			"dom_bytes", len(domSnapshot),
			"error_msg_preview", truncateStr(errMsg, 120),
		)

		// For locator failures, route to the DOM-aware healer (works with or without AI).
		if isLocatorFail {
			originalLocator := core.ResolveOriginalLocator(combinedErr, testSource, framework)
			domUsed := domSnapshot != ""

			// When the DOM was not captured during the test run, attempt to
			// fetch the page HTML live.  This works for public or intranet pages
			// that are still accessible and serve static HTML.
			if !domUsed {
				// Primary: scan logs/error message for URLs.
				liveHTML := fetchLivePageHTML(errMsg, errorLogs, consoleLogs)

				// Secondary: if no URL found in logs, read the test source file
				// and extract URLs from it (e.g. ${BASE_URL} assignments).
				if liveHTML == "" {
					if testSource == "" {
						testSource = autoReadTestFile(id)
					}
					if testSource != "" {
						liveHTML = fetchLivePageHTML("", "", testSource)
						if liveHTML != "" {
							slog.Info("propose-fix: URL found in test source file",
								"incident_id", id, "html_bytes", len(liveHTML))
						}
					}
				}

				if liveHTML != "" {
					domSnapshot = liveHTML
					domUsed = true
					slog.Info("propose-fix: using live-fetched page HTML",
						"incident_id", id, "html_bytes", len(liveHTML))
				} else {
					slog.Warn("propose-fix: no DOM and live fetch failed — AI will guess from error message only",
						"incident_id", id, "original_locator", originalLocator)
				}
			}

			f := core.LocatorError{
				TestName:        testName,
				Framework:       framework,
				OriginalLocator: originalLocator,
				ErrorMessage:    combinedErr,
				IncidentID:      id,
			}
			var healed string
			var confidence float64
			var explanation string
			if core.AIService != nil {
				healed, confidence, explanation = aiLocatorHeal(f, framework)
			} else if domSnapshot != "" && originalLocator != "" {
				var ok bool
				healed, confidence, explanation, ok = core.HealLocatorFromHTML(originalLocator, domSnapshot)
				if !ok {
					healed, confidence, explanation = core.SuggestHealedLocator(originalLocator, framework)
				}
			} else {
				healed, confidence, explanation = core.SuggestHealedLocator(originalLocator, framework)
			}

			codeSnippet := buildHealedLocatorSnippet(originalLocator, healed, framework)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":              codeSnippet,
				"explanation":       explanation,
				"confidence":        confidence,
				"healed_locator":    healed,
				"original_locator":  originalLocator,
				"dom_used":          domUsed,
			})
			return
		}

		// For non-locator failures (script errors, assertion failures, etc.),
		// auto-read the test file and ask the AI for a code-level fix.
		if fileContent == "" {
			fileContent = autoReadTestFile(id)
		}
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
// It first checks console_logs for a captured DOM snapshot, then falls back to
// a live HTTP fetch.  Rule-based heuristics are used when AI is not available.
func aiLocatorHeal(f core.LocatorError, framework string) (healed string, confidence float64, explanation string) {
	if core.AIService == nil {
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	// 1. Try the DOM snapshot captured by the test listener.
	var consoleLogs, errorLogs string
	if core.DB != nil {
		core.DB.QueryRow(
			`SELECT COALESCE(console_logs,''), COALESCE(error_logs,'') FROM incidents WHERE id = ?`,
			f.IncidentID,
		).Scan(&consoleLogs, &errorLogs)
	}

	// Resolve broken locator from logs + test source (Robot timeouts often omit selector).
	if f.OriginalLocator == "" {
		testSource := autoReadTestFile(f.IncidentID)
		f.OriginalLocator = core.ResolveOriginalLocator(f.ErrorMessage+"\n"+errorLogs, testSource, framework)
		slog.Info("aiLocatorHeal: resolved locator from test source",
			"incident_id", f.IncidentID, "locator", f.OriginalLocator)
	}

	pageHTML := core.ExtractDOMSnapshotFromLogs(consoleLogs, errorLogs)

	// 2. No snapshot? Fetch the page live so the AI always has real HTML.
	if pageHTML == "" {
		pageHTML = fetchLivePageHTML(f.ErrorMessage, errorLogs, consoleLogs)
		if pageHTML != "" {
			slog.Info("aiLocatorHeal: used live-fetched page HTML (from logs)", "incident_id", f.IncidentID, "bytes", len(pageHTML))
		}
	}
	// 3. Still no HTML? Try extracting the URL from the test source file.
	if pageHTML == "" {
		testSource := autoReadTestFile(f.IncidentID)
		if testSource != "" {
			pageHTML = fetchLivePageHTML("", "", testSource)
			if pageHTML != "" {
				slog.Info("aiLocatorHeal: used live-fetched page HTML (from test file)", "incident_id", f.IncidentID, "bytes", len(pageHTML))
			}
		}
	}

	// 3. HTML scan (fast, deterministic) before calling the LLM.
	if pageHTML != "" && f.OriginalLocator != "" {
		if healed, conf, expl, ok := core.HealLocatorFromHTML(f.OriginalLocator, pageHTML); ok {
			slog.Info("aiLocatorHeal: healed from HTML scan",
				"incident_id", f.IncidentID,
				"original", f.OriginalLocator,
				"healed", healed,
				"confidence", conf,
			)
			return healed, conf, expl
		}
	}

	// 4. No HTML at all and no locator extracted → nothing useful to offer.
	if f.OriginalLocator == "" && pageHTML == "" {
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	// Build a locator-healing prompt and call the AI.
	cfg, err := core.AIService.GetConfig(context.Background())
	if err != nil || !cfg.Enabled || cfg.Provider == aiPkg.ProviderDisabled {
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	prompt := buildLocatorPrompt(f.OriginalLocator, framework, pageHTML, f.ErrorMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// AnalyzeRaw sends the prompt verbatim — no internal re-wrapping or
	// truncation — so the full page HTML always reaches the LLM.
	rawText, callErr := aiPkg.AnalyzeRaw(ctx, cfg, prompt)
	if callErr != nil {
		slog.Warn("aiLocatorHeal: LLM call failed", "incident_id", f.IncidentID, "err", callErr)
		return core.SuggestHealedLocator(f.OriginalLocator, framework)
	}

	slog.Info("aiLocatorHeal: LLM responded",
		"incident_id", f.IncidentID,
		"dom_bytes_sent", len(pageHTML),
		"response_preview", truncateStr(rawText, 200),
	)

	// Parse JSON response: {"healed_locator":"...","confidence":0.9,"explanation":"..."}
	parsed := parseLocatorResponse(rawText)
	if parsed.healed != "" {
		// Reject LLM answers that repeat the broken selector or xpath variants of it.
		if parsed.healed == f.OriginalLocator || core.IsEquivalentBrokenLocator(f.OriginalLocator, parsed.healed) {
			slog.Warn("aiLocatorHeal: LLM returned equivalent broken selector",
				"incident_id", f.IncidentID, "healed", parsed.healed)
		} else if pageHTML != "" && !core.LocatorExistsInHTML(pageHTML, parsed.healed) {
			slog.Warn("aiLocatorHeal: LLM selector not found in HTML, trying HTML scan",
				"incident_id", f.IncidentID, "healed", parsed.healed)
			if healed, conf, expl, ok := core.HealLocatorFromHTML(f.OriginalLocator, pageHTML); ok {
				return healed, conf, expl
			}
		} else {
			return parsed.healed, parsed.confidence, parsed.explanation
		}
	}
	if pageHTML != "" && f.OriginalLocator != "" {
		if healed, conf, expl, ok := core.HealLocatorFromHTML(f.OriginalLocator, pageHTML); ok {
			return healed, conf, expl
		}
	}
	return core.SuggestHealedLocator(f.OriginalLocator, framework)
}

func buildLocatorPrompt(original, framework, pageHTML, errorMsg string) string {
	var b strings.Builder
	b.WriteString("You are an expert test automation engineer specializing in self-healing locators.\n\n")
	b.WriteString("A UI test failed because the selector no longer exists on the page.\n")
	b.WriteString("Your ONLY task: look at the page HTML below and find the correct replacement selector.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Inspect the HTML carefully. Find the element that matches the intent of the broken selector.\n")
	b.WriteString("- Prefer: id attributes (#id), then data-testid, then ARIA role, then stable class.\n")
	b.WriteString("- NEVER return the same broken selector as the fix.\n")
	b.WriteString("- NEVER return an xpath/css variant of the same missing attribute (e.g. do NOT change data-qa='X' to xpath with data-qa='X').\n")
	b.WriteString("- The healed_locator MUST exist in the page HTML below.\n")
	b.WriteString("- Return ONLY a raw JSON object — no markdown, no explanation outside the JSON.\n\n")
	b.WriteString(`{"healed_locator":"<selector found in HTML>","confidence":0.95,"explanation":"<one sentence: what element you found and why>"}`)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Framework: %s\n", framework)
	fmt.Fprintf(&b, "Broken locator: %s\n", original)
	fmt.Fprintf(&b, "Error message: %s\n\n", errorMsg)
	if pageHTML != "" {
		if len(pageHTML) > 20000 {
			pageHTML = pageHTML[:20000]
		}
		if candidates := core.FormatInteractiveElements(pageHTML, 40); candidates != "" {
			b.WriteString("=== INTERACTIVE ELEMENTS FOUND IN PAGE HTML ===\n")
			b.WriteString("Pick the selector for the element that matches the broken locator's intent:\n")
			b.WriteString(candidates)
			b.WriteString("=== END INTERACTIVE ELEMENTS ===\n\n")
		}
		b.WriteString("=== PAGE HTML AT TIME OF FAILURE ===\n")
		b.WriteString(pageHTML)
		b.WriteString("\n=== END PAGE HTML ===\n")
		b.WriteString("\nIMPORTANT: Your healed_locator MUST appear verbatim in the HTML or interactive elements list above.\n")
	} else {
		b.WriteString("WARNING: No page HTML was captured. ")
		b.WriteString("Suggest the most likely fix based on the broken selector name alone, with low confidence.\n")
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
	if filePath != "" {
		if content := core.ReadTestFileAtPaths(filePath, repoPath); content != "" {
			return content
		}
	}

	// Fallback: locate test file by incident test name (JUnit often omits file paths).
	var testName string
	_ = core.DB.QueryRow(`SELECT COALESCE(name,'') FROM incidents WHERE id = ?`, incidentID).Scan(&testName)
	return core.FindTestSourceInRepo(repoPath, testName)
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

// truncateStr returns the first n characters of s for safe log previews.
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// fetchLivePageHTML tries to retrieve the HTML of the page under test so the
// AI has real DOM context even when the test listener did not capture a snapshot.
// It extracts candidate URLs from the error message, error logs, and console
// logs, then performs an HTTP GET for each until one succeeds.
// Returns empty string when no URL is reachable or no URL is found.
func fetchLivePageHTML(errorMsg, errorLogs, consoleLogs string) string {
	urls := extractPageURLs(errorMsg + "\n" + errorLogs + "\n" + consoleLogs)
	for _, u := range urls {
		if html := doHTTPFetchPage(u); html != "" {
			return html
		}
	}
	return ""
}

// urlRe matches http/https URLs in free text.
var urlRe = regexp.MustCompile(`https?://[^\s"'<>\)]+`)

// extractPageURLs returns deduplicated, likely-page URLs found in text.
// It skips JS, CSS, image, and API paths that are unlikely to be test targets.
// It also resolves variable assignments like:
//
//	${BASE_URL}    https://example.com/path
//	const BASE_URL = "https://example.com";
//	BASE_URL       = "https://example.com/path"
func extractPageURLs(text string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimRight(raw, ".,;)/\"'")
		if raw == "" || seen[raw] {
			return
		}
		lower := strings.ToLower(raw)
		for _, suffix := range []string{".js", ".css", ".png", ".jpg", ".svg", ".ico", ".woff", "/api/", "/static/"} {
			if strings.Contains(lower, suffix) {
				return
			}
		}
		seen[raw] = true
		out = append(out, raw)
	}

	// Direct URLs anywhere in the text.
	for _, raw := range urlRe.FindAllString(text, -1) {
		add(raw)
	}

	// Variable assignment patterns:
	//   ${BASE_URL}    https://...        (Robot Framework)
	//   BASE_URL     = "https://..."     (Python / .env)
	//   const X      = 'https://...';    (JS)
	//   baseURL:       "https://..."     (YAML / config)
	varURLRe := regexp.MustCompile(`(?i)(?:base.?url|page.?url|site.?url|app.?url)[^h]+(https?://[^\s"'<>\)]+)`)
	for _, m := range varURLRe.FindAllStringSubmatch(text, -1) {
		if len(m) >= 2 {
			add(m[1])
		}
	}
	return out
}

// doHTTPFetchPage performs an HTTP GET and returns the response body as a
// string (capped at 512 KB).  Returns empty string on any error.
func doHTTPFetchPage(url string) string {
	client := &http.Client{
		Timeout: 12 * time.Second,
		// Do not follow redirects to unrelated domains.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	// Mimic a real browser so the server does not block the request.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; QACapsule-Healer/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()

	// Verify the response is HTML before reading the full body.
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "html") {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return ""
	}
	slog.Info("fetchLivePageHTML: fetched page", "url", url, "bytes", len(body))
	return string(body)
}

// buildHealedLocatorSnippet formats the healed locator as a ready-to-paste
// code line using the framework's native selector syntax.
func buildHealedLocatorSnippet(original, healed, framework string) string {
	if healed == "" {
		healed = original
	}
	fw := strings.ToLower(framework)
	switch {
	case strings.Contains(fw, "robot") || strings.Contains(fw, "robotframework"):
		return fmt.Sprintf("# Replace the broken locator with:\nClick    %s\n\n# Original (broken):\n# Click    %s", healed, original)
	case strings.Contains(fw, "playwright"):
		return fmt.Sprintf("// Replace the broken locator with:\nawait page.locator('%s').click();\n\n// Original (broken):\n// await page.locator('%s').click();", healed, original)
	case strings.Contains(fw, "cypress"):
		return fmt.Sprintf("// Replace the broken locator with:\ncy.get('%s').click();\n\n// Original (broken):\n// cy.get('%s').click();", healed, original)
	case strings.Contains(fw, "selenium"):
		return fmt.Sprintf("# Replace the broken locator with:\ndriver.find_element(By.CSS_SELECTOR, '%s')\n\n# Original (broken):\n# driver.find_element(By.CSS_SELECTOR, '%s')", healed, original)
	default:
		return fmt.Sprintf("Healed locator: %s\n\nOriginal (broken): %s", healed, original)
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
