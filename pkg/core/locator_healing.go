package core

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ── Locator error patterns (framework-agnostic) ───────────────────────────────

// locatorPatterns maps each framework's error signature to a regex that
// captures the offending selector/locator string in capture group 1.
//
// Robot Framework Browser Library uses Playwright under the hood, so its
// error messages are Playwright-style (e.g. "locator('button[...]') Timeout").
// RobotFramework entries therefore include both RF-native and PW-style patterns.
var locatorPatterns = []struct {
	framework string
	re        *regexp.Regexp
}{
	// Playwright (and Robot Framework Browser Library) errors:
	//   locator('button[data-qa="submit-v1"]') Timeout 10000ms exceeded.
	//   waiting for locator('#submit')
	{"Playwright", regexp.MustCompile(`locator\('([^']+)'\)`)},
	{"Playwright", regexp.MustCompile(`locator\("([^"]+)"\)`)},
	{"Playwright", regexp.MustCompile(`waiting for locator\('([^']+)'\)`)},
	{"Playwright", regexp.MustCompile(`waiting for locator\("([^"]+)"\)`)},
	// Robot Framework Browser Library produces Playwright-style errors.
	// Keep these under RobotFramework so they match when the framework hint is RF.
	{"RobotFramework", regexp.MustCompile(`locator\('([^']+)'\)`)},
	{"RobotFramework", regexp.MustCompile(`locator\("([^"]+)"\)`)},
	{"RobotFramework", regexp.MustCompile(`Element\s+'([^']+)'\s+not found`)},
	{"RobotFramework", regexp.MustCompile(`ElementNotFound[:\s]+(.+)`)},
	// Selenium/Python: NoSuchElementException: no such element … selector":"#foo"
	{"Selenium", regexp.MustCompile(`selector["']?\s*[:\s]+["']([^"']+)["']`)},
	{"Selenium", regexp.MustCompile(`Unable to locate element[^"]*"([^"]+)"`)},
	{"Selenium", regexp.MustCompile(`NoSuchElement[^:]*:\s*(.+)`)},
	// Cypress: Expected to find element: '#foo'  or  Timed out retrying … '#foo'
	{"Cypress", regexp.MustCompile(`Expected to find element:\s*'([^']+)'`)},
	{"Cypress", regexp.MustCompile(`Unable to find element:\s*([^\s,]+)`)},
	{"Cypress", regexp.MustCompile(`Timed out retrying.*:\s*'([^']+)'`)},
	// Generic CSS / XPath fallbacks — tried after framework-specific patterns fail.
	{"generic", regexp.MustCompile(`(#[\w][\w-]*)`)},
	{"generic", regexp.MustCompile(`(\[data-testid=['"][\w-]+['"]\])`)},
	// Attribute selectors like [data-qa="submit-v1"] or [aria-label="Sign in"].
	{"generic", regexp.MustCompile(`(\[[^\]=]+="[^"]+"\])`)},
	{"generic", regexp.MustCompile(`(\[[^\]=]+='[^']+'\])`)},
}

// LocatorError describes a detected selector failure in a test.
type LocatorError struct {
	TestName        string `json:"test_name"`
	Framework       string `json:"framework"`
	OriginalLocator string `json:"original_locator"`
	ErrorMessage    string `json:"error_message"`
	IncidentID      int64  `json:"incident_id,omitempty"`
}

// IsLocatorError returns true when the error message looks like a locator/selector failure.
func IsLocatorError(errorMsg string) bool {
	lower := strings.ToLower(errorMsg)
	keywords := []string{
		"locator", "selector", "noelement", "nosuchelement",
		"elementnotfound", "unable to find", "unable to locate",
		"timed out retrying", "waiting for", "not found",
		"expected to find element", "#", "[data-testid",
		"getbyrole", "getbytext", "getbylabel",
		"data-qa", "element not found", "could not find",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// IsLocatorFailure detects locator failures even when JUnit only stores a short
// timeout message in error_message (common with Robot Framework Browser library).
func IsLocatorFailure(errorMsg, errorLogs, testSource, testName string) bool {
	combined := strings.TrimSpace(errorMsg + "\n" + errorLogs)
	if IsLocatorError(combined) {
		return true
	}
	if ResolveOriginalLocator(combined, testSource, "") != "" {
		lower := strings.ToLower(combined)
		if strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out") ||
			strings.Contains(lower, "waiting") || strings.Contains(lower, "click") ||
			strings.Contains(lower, "element") {
			return true
		}
	}
	// Timeout + broken selector variable in test file (Robot, env vars, etc.).
	if testSource != "" {
		lower := strings.ToLower(combined)
		if strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out") ||
			strings.Contains(lower, "waiting") || testName != "" {
			for _, m := range robotVarAssignRe.FindAllStringSubmatch(testSource, -1) {
				if len(m) >= 3 && looksLikeBrokenSelector(strings.TrimSpace(m[2])) {
					return true
				}
			}
		}
	}
	return false
}

// ResolveOriginalLocator finds the broken selector from error logs and/or test source.
func ResolveOriginalLocator(errorMsg, testSource, framework string) string {
	if loc := ExtractLocator(errorMsg, framework); loc != "" {
		return loc
	}
	return ExtractLocatorFromTestSource(testSource, framework)
}

var (
	robotVarAssignRe    = regexp.MustCompile(`(?m)^\$\{(\w+)\}\s+(.+)$`)
	robotActionRe       = regexp.MustCompile(`(?m)^\s*(Click|Fill Text|Get Element|Tap|Type Text|Wait For Elements State)\s+(\S+)`)
	pwLocatorSQRe       = regexp.MustCompile(`(?:page\.)?locator\(\s*'([^']*)'\s*\)`)
	pwLocatorDQRe       = regexp.MustCompile(`(?:page\.)?locator\(\s*"([^"]*)"\s*\)`)
	pwGetByRoleRe       = regexp.MustCompile(`getByRole\s*\(\s*['"](\w+)['"]`)
	pwGetByTestIdSQRe   = regexp.MustCompile(`getByTestId\s*\(\s*'([^']*)'`)
	pwGetByTestIdDQRe   = regexp.MustCompile(`getByTestId\s*\(\s*"([^"]*)"`)
	pwGetByLabelSQRe    = regexp.MustCompile(`getByLabel\s*\(\s*'([^']*)'`)
	pwGetByLabelDQRe    = regexp.MustCompile(`getByLabel\s*\(\s*"([^"]*)"`)
	cyGetSQRe           = regexp.MustCompile(`cy\.get\(\s*'([^']*)'\s*\)`)
	cyGetDQRe           = regexp.MustCompile(`cy\.get\(\s*"([^"]*)"\s*\)`)
	cyContainsSQRe      = regexp.MustCompile(`cy\.contains\s*\(\s*'([^']*)'`)
	cyContainsDQRe      = regexp.MustCompile(`cy\.contains\s*\(\s*"([^"]*)"`)
	seleniumPyBySQRe    = regexp.MustCompile(`find_element\s*\(\s*By\.\w+\s*,\s*'([^']*)'`)
	seleniumPyByDQRe    = regexp.MustCompile(`find_element\s*\(\s*By\.\w+\s*,\s*"([^"]*)"`)
	seleniumJavaBySQRe  = regexp.MustCompile(`findElement\s*\(\s*By\.\w+\(\s*'([^']*)'\s*\)`)
	seleniumJavaByDQRe  = regexp.MustCompile(`findElement\s*\(\s*By\.\w+\(\s*"([^"]*)"\s*\)`)
	seleniumFindByRe    = regexp.MustCompile(`@FindBy\s*\([^)]*['"]([^'"]+)['"]`)
	pythonPageLocatorSQRe = regexp.MustCompile(`page\.(?:locator|get_by_\w+)\(\s*'([^']*)'`)
	pythonPageLocatorDQRe = regexp.MustCompile(`page\.(?:locator|get_by_\w+)\(\s*"([^"]*)"`)
)

// ExtractLocatorFromTestSource reads the failing selector from the test file when
// JUnit error_message only says "TimeoutError" without the selector string.
func ExtractLocatorFromTestSource(source, framework string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	fw := strings.ToLower(framework)

	// Robot Framework: resolve ${VAR} used in action keywords.
	if framework == "" || strings.Contains(fw, "robot") {
		if loc := extractRobotLocator(source); loc != "" {
			return loc
		}
	}

	if strings.Contains(fw, "playwright") || fw == "" {
		for _, re := range []*regexp.Regexp{
			pwLocatorSQRe, pwLocatorDQRe,
			pwGetByTestIdSQRe, pwGetByTestIdDQRe,
			pwGetByLabelSQRe, pwGetByLabelDQRe,
			pythonPageLocatorSQRe, pythonPageLocatorDQRe,
		} {
			if m := re.FindStringSubmatch(source); len(m) >= 2 && looksLikeBrokenSelector(m[1]) {
				return m[1]
			}
		}
		if m := pwGetByRoleRe.FindStringSubmatch(source); len(m) >= 2 {
			return "getByRole('" + m[1] + "')"
		}
	}

	if strings.Contains(fw, "cypress") || fw == "" {
		for _, re := range []*regexp.Regexp{cyGetSQRe, cyGetDQRe, cyContainsSQRe, cyContainsDQRe} {
			if m := re.FindStringSubmatch(source); len(m) >= 2 && looksLikeBrokenSelector(m[1]) {
				return m[1]
			}
		}
	}

	if strings.Contains(fw, "selenium") || strings.Contains(fw, "java") || strings.Contains(fw, "testng") || fw == "" {
		for _, re := range []*regexp.Regexp{
			seleniumPyBySQRe, seleniumPyByDQRe,
			seleniumJavaBySQRe, seleniumJavaByDQRe,
			seleniumFindByRe,
		} {
			if m := re.FindStringSubmatch(source); len(m) >= 2 && looksLikeBrokenSelector(m[1]) {
				return m[1]
			}
		}
	}

	// Last resort: any framework-agnostic pattern in unknown test files.
	for _, re := range []*regexp.Regexp{
		pwLocatorSQRe, pwLocatorDQRe,
		cyGetSQRe, cyGetDQRe,
		seleniumPyBySQRe, seleniumPyByDQRe,
		seleniumJavaBySQRe, seleniumJavaByDQRe,
	} {
		if m := re.FindStringSubmatch(source); len(m) >= 2 && looksLikeBrokenSelector(m[1]) {
			return m[1]
		}
	}
	return ""
}

func extractRobotLocator(source string) string {
	vars := map[string]string{}
	for _, m := range robotVarAssignRe.FindAllStringSubmatch(source, -1) {
		if len(m) >= 3 {
			vars[m[1]] = strings.TrimSpace(m[2])
		}
	}
	var last string
	for _, m := range robotActionRe.FindAllStringSubmatch(source, -1) {
		if len(m) < 3 {
			continue
		}
		target := strings.TrimSpace(m[2])
		if strings.HasPrefix(target, "${") && strings.HasSuffix(target, "}") {
			if v, ok := vars[target[2:len(target)-1]]; ok {
				target = v
			}
		}
		if looksLikeBrokenSelector(target) {
			last = target
		}
	}
	return last
}

func looksLikeBrokenSelector(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "id=") {
		return false
	}
	return strings.Contains(s, "[") || strings.HasPrefix(s, "#") ||
		strings.HasPrefix(s, ".") || strings.Contains(lower, "xpath")
}

// IsEquivalentBrokenLocator returns true when healed still targets the same
// missing attribute/value as the broken selector (e.g. xpath variant of data-qa).
func IsEquivalentBrokenLocator(original, healed string) bool {
	original = strings.TrimSpace(original)
	healed = strings.TrimSpace(healed)
	if original == "" || healed == "" {
		return false
	}
	if strings.EqualFold(original, healed) {
		return true
	}
	ol := strings.ToLower(original)
	hl := strings.ToLower(healed)
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`data-qa\s*=\s*["']([^"']+)["']`),
		regexp.MustCompile(`data-testid\s*=\s*["']([^"']+)["']`),
		regexp.MustCompile(`@data-qa\s*=\s*['"]([^'"]+)['"]`),
	} {
		if m := re.FindStringSubmatch(ol); len(m) >= 2 {
			val := strings.ToLower(m[1])
			if strings.Contains(hl, val) && (strings.Contains(hl, "data-qa") || strings.Contains(hl, "data-testid")) {
				return true
			}
		}
	}
	return false
}

// ExtractLocator returns the first locator/selector string detected in the error message.
// It tries framework-specific patterns first, then generic fallbacks.
func ExtractLocator(errorMsg, hintFramework string) string {
	hintFramework = strings.ToLower(strings.TrimSpace(hintFramework))

	// Framework-specific patterns first
	for _, p := range locatorPatterns {
		if hintFramework != "" && p.framework != "generic" &&
			!strings.EqualFold(p.framework, hintFramework) {
			continue
		}
		if m := p.re.FindStringSubmatch(errorMsg); len(m) >= 2 {
			loc := strings.TrimSpace(m[1])
			if loc != "" {
				return loc
			}
		}
	}

	// Generic fallback: look for any CSS-like selector
	for _, p := range locatorPatterns {
		if p.framework == "generic" {
			if m := p.re.FindStringSubmatch(errorMsg); len(m) >= 2 {
				loc := strings.TrimSpace(m[1])
				if loc != "" {
					return loc
				}
			}
		}
	}
	return ""
}

// ── Database model ────────────────────────────────────────────────────────────

// LocatorHealing records a single MCP locator-healing intervention.
type LocatorHealing struct {
	ID              int64     `json:"id"`
	IncidentID      int64     `json:"incident_id"`
	RunID           string    `json:"run_id"`
	Framework       string    `json:"framework"`
	OriginalLocator string    `json:"original_locator"`
	HealedLocator   string    `json:"healed_locator,omitempty"`
	Confidence      float64   `json:"confidence"`
	Explanation     string    `json:"explanation"`
	AgentSource     string    `json:"agent_source"`
	CreatedAt       time.Time `json:"created_at"`
}

// RecordLocatorHealing persists a healing intervention and marks the incident
// with mcp_healed = 1 so the UI and notifications can surface it.
func RecordLocatorHealing(incidentID int64, runID, framework, originalLocator, healedLocator, explanation, agentSource string, confidence float64) (*LocatorHealing, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if agentSource == "" {
		agentSource = "mcp_gate"
	}
	res, err := DB.Exec(`
		INSERT INTO locator_healings
			(incident_id, run_id, framework, original_locator, healed_locator, confidence, explanation, agent_source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		incidentID, runID, framework, originalLocator, healedLocator, confidence, explanation, agentSource)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	// Mark the incident so reports distinguish healed vs raw failures.
	_, _ = DB.Exec(`UPDATE incidents SET mcp_healed = 1 WHERE id = ?`, incidentID)

	return &LocatorHealing{
		ID:              id,
		IncidentID:      incidentID,
		RunID:           runID,
		Framework:       framework,
		OriginalLocator: originalLocator,
		HealedLocator:   healedLocator,
		Confidence:      confidence,
		Explanation:     explanation,
		AgentSource:     agentSource,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

// ListLocatorHealings returns recent locator healings for a project (via incidents join).
func ListLocatorHealings(projectName string, limit int) ([]LocatorHealing, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := DB.Query(`
		SELECT lh.id, lh.incident_id, lh.run_id, lh.framework,
		       lh.original_locator, lh.healed_locator, lh.confidence,
		       lh.explanation, lh.agent_source, lh.created_at
		FROM locator_healings lh
		JOIN incidents i ON i.id = lh.incident_id
		WHERE (? = '' OR i.project_name = ?)
		ORDER BY lh.created_at DESC LIMIT ?`, projectName, projectName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLocatorHealings(rows)
}

// DetectLocatorFailuresForRun scans all failed incidents for a run and returns
// those whose error message or test source indicate a locator/selector failure.
func DetectLocatorFailuresForRun(runID, projectName string) ([]LocatorError, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var repoPath string
	_ = DB.QueryRow(`SELECT COALESCE(repo_path,'') FROM projects WHERE name = ?`, projectName).Scan(&repoPath)

	rows, err := DB.Query(`
		SELECT id, name, error_message, COALESCE(error_logs,''), COALESCE(framework,'')
		FROM incidents
		WHERE pipeline_run_id = ?
		  AND (? = '' OR project_name = ?)
		  AND is_resolved = 0
		ORDER BY id ASC`, runID, projectName, projectName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LocatorError
	for rows.Next() {
		var id int64
		var name, errMsg, errorLogs, framework string
		if err := rows.Scan(&id, &name, &errMsg, &errorLogs, &framework); err != nil {
			continue
		}
		testSource := FindTestSourceInRepo(repoPath, name)
		if !IsLocatorFailure(errMsg, errorLogs, testSource, name) {
			continue
		}
		combined := strings.TrimSpace(errMsg + "\n" + errorLogs)
		locator := ResolveOriginalLocator(combined, testSource, framework)
		out = append(out, LocatorError{
			TestName:        name,
			Framework:       framework,
			OriginalLocator: locator,
			ErrorMessage:    errMsg,
			IncidentID:      id,
		})
	}
	return out, rows.Err()
}

// ── Healing suggestion (heuristic, no LLM required) ──────────────────────────

// SuggestHealedLocator returns an alternative locator based on heuristics.
// For AI-powered healing with page DOM, use SuggestHealedLocatorWithContext
// which is called from the healing gate handler when AIService is available.
func SuggestHealedLocator(original, framework string) (healed string, confidence float64, explanation string) {
	original = strings.TrimSpace(original)
	if original == "" {
		return "", 0, "no locator detected"
	}
	if strings.HasPrefix(original, "#") {
		return fmt.Sprintf("[data-testid=%q]", strings.TrimPrefix(original, "#")),
			0.60,
			fmt.Sprintf("ID selectors are fragile; prefer data-testid or ARIA role. Original: %s", original)
	}
	if strings.Contains(original, "data-testid") {
		role := "button"
		if strings.Contains(strings.ToLower(original), "input") {
			role = "textbox"
		} else if strings.Contains(strings.ToLower(original), "link") {
			role = "link"
		}
		return fmt.Sprintf("getByRole('%s')", role),
			0.45,
			fmt.Sprintf("data-testid not found; element may have changed. Try ARIA role: %s", role)
	}
	if strings.HasPrefix(original, ".") {
		return fmt.Sprintf("[class*=%q]", strings.TrimPrefix(original, ".")),
			0.50,
			fmt.Sprintf("Class selectors break on CSS refactors; partial match is more resilient. Original: %s", original)
	}
	return original, 0.30, "locator pattern not recognized; manual inspection recommended"
}

// ExtractDOMSnapshot extracts the page HTML captured by a test listener
// (between QA_CAPSULE_DOM_SNAPSHOT_START/END markers) from a single log blob.
func ExtractDOMSnapshot(consoleLogs string) string {
	const start = "[QA_CAPSULE_DOM_SNAPSHOT_START]"
	const end = "[QA_CAPSULE_DOM_SNAPSHOT_END]"
	i := strings.Index(consoleLogs, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	j := strings.Index(consoleLogs[i:], end)
	if j < 0 {
		return strings.TrimSpace(consoleLogs[i:])
	}
	return strings.TrimSpace(consoleLogs[i : i+j])
}

// ExtractDOMSnapshotFromLogs searches both console_logs and error_logs.
// Robot/Cypress write DOM snapshots to stdout → console_logs.
// Playwright writes to stderr → error_logs (see qa-capsule-reporter.js).
func ExtractDOMSnapshotFromLogs(consoleLogs, errorLogs string) string {
	if s := ExtractDOMSnapshot(consoleLogs); s != "" {
		return s
	}
	return ExtractDOMSnapshot(errorLogs)
}

// ── HTML-aware locator healing (no LLM) ───────────────────────────────────────

var (
	interactiveVoidTagRe  = regexp.MustCompile(`(?i)<(input|textarea|select)\b([^>]*)/?>`)
	interactiveButtonRe   = regexp.MustCompile(`(?i)<button\b([^>]*)>([^<]*)</button\s*>`)
	interactiveAnchorRe   = regexp.MustCompile(`(?i)<a\b([^>]*)>([^<]*)</a\s*>`)
	attrIDRe             = regexp.MustCompile(`(?i)\bid\s*=\s*["']([^"']+)["']`)
	attrNameRe           = regexp.MustCompile(`(?i)\bname\s*=\s*["']([^"']+)["']`)
	attrTypeRe           = regexp.MustCompile(`(?i)\btype\s*=\s*["']([^"']+)["']`)
	attrTestIDRe         = regexp.MustCompile(`(?i)\bdata-testid\s*=\s*["']([^"']+)["']`)
	attrDataQARe         = regexp.MustCompile(`(?i)\bdata-qa\s*=\s*["']([^"']+)["']`)
	attrAriaLabelRe      = regexp.MustCompile(`(?i)\baria-label\s*=\s*["']([^"']+)["']`)
)

type domElement struct {
	tag       string
	id        string
	name      string
	typ       string
	testID    string
	dataQA    string
	ariaLabel string
	text      string
}

func attrFirst(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// ParseInteractiveElements extracts buttons, inputs, links and other controls
// from raw HTML so the healer can match intent without relying on the LLM alone.
func ParseInteractiveElements(html string) []domElement {
	if html == "" {
		return nil
	}
	var out []domElement
	seen := map[string]bool{}
	add := func(tag, attrs, text string) {
		tag = strings.ToLower(tag)
		el := domElement{
			tag:       tag,
			id:        attrFirst(attrIDRe, attrs),
			name:      attrFirst(attrNameRe, attrs),
			typ:       attrFirst(attrTypeRe, attrs),
			testID:    attrFirst(attrTestIDRe, attrs),
			dataQA:    attrFirst(attrDataQARe, attrs),
			ariaLabel: attrFirst(attrAriaLabelRe, attrs),
			text:      strings.TrimSpace(text),
		}
		key := el.tag + "|" + el.id + "|" + el.testID + "|" + el.dataQA + "|" + el.name
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, el)
	}
	for _, m := range interactiveVoidTagRe.FindAllStringSubmatch(html, -1) {
		if len(m) >= 3 {
			add(m[1], m[2], "")
		}
	}
	for _, m := range interactiveButtonRe.FindAllStringSubmatch(html, -1) {
		if len(m) >= 3 {
			add("button", m[1], m[2])
		}
	}
	for _, m := range interactiveAnchorRe.FindAllStringSubmatch(html, -1) {
		if len(m) >= 3 {
			add("a", m[1], m[2])
		}
	}
	return out
}

// FormatInteractiveElements returns a compact, human-readable list for LLM prompts.
func FormatInteractiveElements(html string, max int) string {
	els := ParseInteractiveElements(html)
	if len(els) == 0 {
		return ""
	}
	if max <= 0 {
		max = 40
	}
	var b strings.Builder
	for i, el := range els {
		if i >= max {
			fmt.Fprintf(&b, "... (%d more elements)\n", len(els)-max)
			break
		}
		parts := []string{el.tag}
		if el.id != "" {
			parts = append(parts, "id="+el.id, "selector=#"+el.id)
		}
		if el.testID != "" {
			parts = append(parts, "data-testid="+el.testID)
		}
		if el.dataQA != "" {
			parts = append(parts, "data-qa="+el.dataQA)
		}
		if el.name != "" {
			parts = append(parts, "name="+el.name)
		}
		if el.typ != "" {
			parts = append(parts, "type="+el.typ)
		}
		if el.ariaLabel != "" {
			parts = append(parts, "aria-label="+el.ariaLabel)
		}
		if el.text != "" {
			parts = append(parts, "text="+truncateDOMText(el.text, 40))
		}
		b.WriteString("- ")
		b.WriteString(strings.Join(parts, " | "))
		b.WriteByte('\n')
	}
	return b.String()
}

func truncateDOMText(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// LocatorExistsInHTML returns true when key parts of a CSS selector appear in the HTML.
func LocatorExistsInHTML(html, selector string) bool {
	selector = strings.TrimSpace(selector)
	if html == "" || selector == "" {
		return false
	}
	lower := strings.ToLower(html)

	// #id
	if strings.HasPrefix(selector, "#") {
		id := selector[1:]
		if id == "" {
			return false
		}
		return strings.Contains(lower, `id="`+strings.ToLower(id)+`"`) ||
			strings.Contains(lower, `id='`+strings.ToLower(id)+`'`)
	}

	// [attr="value"] or [attr='value']
	if strings.HasPrefix(selector, "[") && strings.Contains(selector, "=") {
		re := regexp.MustCompile(`\[([^=\]]+)=["']([^"']+)["']\]`)
		if m := re.FindStringSubmatch(selector); len(m) >= 3 {
			attr := strings.ToLower(strings.TrimSpace(m[1]))
			val := strings.ToLower(strings.TrimSpace(m[2]))
			return strings.Contains(lower, attr+`="`+val+`"`) ||
				strings.Contains(lower, attr+`='`+val+`'`)
		}
	}

	// tag[attr="value"] e.g. button[data-qa="submit"]
	if idx := strings.Index(selector, "["); idx > 0 {
		return LocatorExistsInHTML(html, selector[idx:])
	}

	return strings.Contains(lower, strings.ToLower(selector))
}

func selectorKeywords(original string) []string {
	lower := strings.ToLower(original)
	stop := map[string]bool{
		"button": true, "input": true, "click": true, "locator": true,
		"data": true, "testid": true, "type": true, "text": true,
		"element": true, "select": true, "option": true, "form": true,
		"div": true, "span": true, "href": true, "class": true,
	}
	seen := map[string]bool{}
	var kws []string
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if len(s) >= 3 && !stop[s] && !seen[s] {
			seen[s] = true
			kws = append(kws, s)
		}
	}
	for _, token := range regexp.MustCompile(`[a-z][a-z0-9_-]{2,}`).FindAllString(lower, -1) {
		add(token)
	}
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`data-qa\s*=\s*["']([^"']+)["']`),
		regexp.MustCompile(`data-testid\s*=\s*["']([^"']+)["']`),
		regexp.MustCompile(`name\s*=\s*["']([^"']+)["']`),
		regexp.MustCompile(`aria-label\s*=\s*["']([^"']+)["']`),
	} {
		if m := re.FindStringSubmatch(original); len(m) >= 2 {
			for _, part := range strings.FieldsFunc(m[1], func(r rune) bool {
				return r == '-' || r == '_' || r == '.' || r == ' '
			}) {
				add(part)
			}
		}
	}
	return kws
}

func selectorTagHint(original string) string {
	lower := strings.ToLower(original)
	for _, tag := range []string{"button", "input", "a", "select", "textarea"} {
		if strings.HasPrefix(lower, tag) {
			return tag
		}
	}
	return ""
}

func elementToSelector(el domElement) string {
	if el.id != "" {
		return "#" + el.id
	}
	if el.testID != "" {
		return `[data-testid="` + el.testID + `"]`
	}
	if el.dataQA != "" {
		return `[data-qa="` + el.dataQA + `"]`
	}
	if el.name != "" {
		return `[name="` + el.name + `"]`
	}
	if el.ariaLabel != "" {
		return `[aria-label="` + el.ariaLabel + `"]`
	}
	return el.tag
}

func scoreElement(el domElement, tagHint string, keywords []string) int {
	score := 0
	if tagHint != "" && el.tag == tagHint {
		score += 3
	}
	blob := strings.ToLower(el.id + " " + el.name + " " + el.typ + " " + el.testID + " " + el.dataQA + " " + el.ariaLabel + " " + el.text)
	for _, kw := range keywords {
		if kw != "" && strings.Contains(blob, kw) {
			score += 2
		}
	}
	if el.id != "" {
		score++
	}
	if el.typ == "submit" {
		score += 2
	}
	return score
}

// HealLocatorFromHTML scans captured page HTML and picks the best matching
// interactive element for the broken selector. Returns ok=false when no
// confident match is found.
func HealLocatorFromHTML(originalLocator, html string) (healed string, confidence float64, explanation string, ok bool) {
	originalLocator = strings.TrimSpace(originalLocator)
	if originalLocator == "" || html == "" {
		return "", 0, "", false
	}
	if LocatorExistsInHTML(html, originalLocator) {
		return originalLocator, 0.95, "Broken selector still present in captured page HTML — verify timing or visibility.", true
	}

	els := ParseInteractiveElements(html)
	if len(els) == 0 {
		return "", 0, "", false
	}

	tagHint := selectorTagHint(originalLocator)
	keywords := selectorKeywords(originalLocator)

	bestScore := 0
	var best domElement
	for _, el := range els {
		s := scoreElement(el, tagHint, keywords)
		if s > bestScore {
			bestScore = s
			best = el
		}
	}
	if bestScore < 4 {
		return "", 0, "", false
	}

	healed = elementToSelector(best)
	if healed == "" || healed == originalLocator {
		return "", 0, "", false
	}
	if !LocatorExistsInHTML(html, healed) {
		return "", 0, "", false
	}

	confidence = 0.75
	if bestScore >= 6 {
		confidence = 0.88
	}
	if best.id != "" {
		confidence = 0.92
	}
	explanation = fmt.Sprintf(
		"Scanned page HTML and matched %q to element <%s id=%q type=%q> — replace the broken selector.",
		healed, best.tag, best.id, best.typ,
	)
	return healed, confidence, explanation, true
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func scanLocatorHealings(rows *sql.Rows) ([]LocatorHealing, error) {
	var out []LocatorHealing
	for rows.Next() {
		var h LocatorHealing
		var createdAt string
		if err := rows.Scan(&h.ID, &h.IncidentID, &h.RunID, &h.Framework,
			&h.OriginalLocator, &h.HealedLocator, &h.Confidence,
			&h.Explanation, &h.AgentSource, &createdAt); err != nil {
			continue
		}
		h.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		out = append(out, h)
	}
	return out, rows.Err()
}
