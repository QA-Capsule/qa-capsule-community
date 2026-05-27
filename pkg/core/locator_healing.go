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
var locatorPatterns = []struct {
	framework string
	re        *regexp.Regexp
}{
	// Playwright: locator('#stripe-pay-button').click: Timeout exceeded
	{"Playwright", regexp.MustCompile(`locator\('([^']+)'\)`)},
	{"Playwright", regexp.MustCompile(`locator\("([^"]+)"\)`)},
	{"Playwright", regexp.MustCompile(`waiting for locator\('([^']+)'\)`)},
	// Selenium/Python: NoSuchElementException: no such element … selector":"#foo"
	{"Selenium", regexp.MustCompile(`selector["']?\s*[:\s]+["']([^"']+)["']`)},
	{"Selenium", regexp.MustCompile(`Unable to locate element[^"]*"([^"]+)"`)},
	{"Selenium", regexp.MustCompile(`NoSuchElement[^:]*:\s*(.+)`)},
	// Cypress: Expected to find element: '#foo'  or  Timed out retrying … '#foo'
	{"Cypress", regexp.MustCompile(`Expected to find element:\s*'([^']+)'`)},
	{"Cypress", regexp.MustCompile(`Unable to find element:\s*([^\s,]+)`)},
	{"Cypress", regexp.MustCompile(`\[data-testid="([^"]+)"\]`)},
	// Robot Framework: Element '#foo' not found  or  ElementNotFound: #foo
	{"RobotFramework", regexp.MustCompile(`Element\s+'([^']+)'\s+not found`)},
	{"RobotFramework", regexp.MustCompile(`ElementNotFound[:\s]+(.+)`)},
	// Generic CSS / XPath fallback
	{"generic", regexp.MustCompile(`(#[\w][\w-]*)`)},
	{"generic", regexp.MustCompile(`(\[data-testid=['"][\w-]+['"]\])`)},
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
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
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

// GetLocatorHealingsForRun returns all locator healings for a pipeline run.
func GetLocatorHealingsForRun(runID string) ([]LocatorHealing, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := DB.Query(`
		SELECT id, incident_id, run_id, framework, original_locator,
		       healed_locator, confidence, explanation, agent_source, created_at
		FROM locator_healings WHERE run_id = ? ORDER BY id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLocatorHealings(rows)
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
// those whose error message matches a locator/selector pattern.
func DetectLocatorFailuresForRun(runID, projectName string) ([]LocatorError, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := DB.Query(`
		SELECT id, name, error_message
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
		var name, errMsg string
		if err := rows.Scan(&id, &name, &errMsg); err != nil {
			continue
		}
		if !IsLocatorError(errMsg) {
			continue
		}
		locator := ExtractLocator(errMsg, "")
		out = append(out, LocatorError{
			TestName:        name,
			OriginalLocator: locator,
			ErrorMessage:    errMsg,
			IncidentID:      id,
		})
	}
	return out, rows.Err()
}

// ── Healing suggestion (heuristic, no LLM required) ──────────────────────────

// SuggestHealedLocator returns an alternative locator based on simple heuristics.
// When an LLM is wired, replace this with an LLM call using the page HTML as context.
func SuggestHealedLocator(original, framework string) (healed string, confidence float64, explanation string) {
	original = strings.TrimSpace(original)
	if original == "" {
		return "", 0, "no locator detected"
	}

	// ID → data-testid equivalent suggestion
	if strings.HasPrefix(original, "#") {
		name := strings.TrimPrefix(original, "#")
		name = strings.ReplaceAll(name, "-", " ")
		return fmt.Sprintf("[data-testid=%q]", strings.TrimPrefix(original, "#")),
			0.60,
			fmt.Sprintf("ID selectors are fragile; prefer data-testid or ARIA role. Original: %s", original)
	}

	// data-testid already good but element was missing → suggest aria role
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

	// Class selector → more stable alternative
	if strings.HasPrefix(original, ".") {
		return fmt.Sprintf("[class*=%q]", strings.TrimPrefix(original, ".")),
			0.50,
			fmt.Sprintf("Class selectors break on CSS refactors; partial match is more resilient. Original: %s", original)
	}

	return original, 0.30, "locator pattern not recognized; manual inspection recommended"
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
