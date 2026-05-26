package healing

import "strings"

const (
	CategoryTimeout      = "timeout"
	CategoryLocator      = "locator"
	CategoryAssertion    = "assertion"
	CategoryNetwork      = "network"
	CategoryStaleElement = "stale_element"
	CategoryUnknown      = "unknown"
)

// ClassifyError maps failure text to a framework-agnostic category.
func ClassifyError(err string) string {
	lower := strings.ToLower(err)
	switch {
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "timed out"), strings.Contains(lower, "waiting for"):
		return CategoryTimeout
	case strings.Contains(lower, "stale element"), strings.Contains(lower, "staleelementreference"):
		return CategoryStaleElement
	case strings.Contains(lower, "locator"), strings.Contains(lower, "selector"), strings.Contains(lower, "getbyrole"),
		strings.Contains(lower, "getbytext"), strings.Contains(lower, "element not found"), strings.Contains(lower, "no such element"):
		return CategoryLocator
	case strings.Contains(lower, "assert"), strings.Contains(lower, " should be"),
		strings.Contains(lower, "should equal"), strings.Contains(lower, "does not match"),
		strings.Contains(lower, " expected "), strings.HasPrefix(lower, "expected "):
		return CategoryAssertion
	case strings.Contains(lower, "econnrefused"), strings.Contains(lower, "network"), strings.Contains(lower, "fetch failed"),
		strings.Contains(lower, "connection refused"), strings.Contains(lower, "502"), strings.Contains(lower, "503"):
		return CategoryNetwork
	default:
		return CategoryUnknown
	}
}

// SuggestedActions returns actionable hints for MCP agents (any test framework).
func SuggestedActions(category string) []string {
	switch category {
	case CategoryTimeout:
		return []string{
			"Increase wait timeout or use explicit wait-for-stable-state before assertion.",
			"Verify the target service responds within SLA in CI (not just locally).",
			"Check for race conditions: element visible but not yet interactive.",
		}
	case CategoryLocator:
		return []string{
			"Replace brittle CSS/XPath with role- or text-based selectors where possible.",
			"Confirm the element exists in the DOM at assertion time (screenshot/trace if available).",
			"Check for dynamic IDs, iframe nesting, or shadow DOM boundaries.",
		}
	case CategoryStaleElement:
		return []string{
			"Re-query the element after navigation or DOM refresh before interacting.",
			"Avoid holding element references across page transitions or AJAX updates.",
		}
	case CategoryAssertion:
		return []string{
			"Compare expected vs actual from the failure message; fix test data or application bug.",
			"Ensure test isolation: no dependency on execution order or shared state.",
		}
	case CategoryNetwork:
		return []string{
			"Verify API/base URL env vars match the CI environment.",
			"Add retry for transient upstream failures or mock external dependencies in CI.",
		}
	default:
		return []string{
			"Use MCP get_incident_context with this incident_id for full telemetry.",
			"Inspect stack trace and console logs to locate the failing step in source.",
			"Apply the smallest fix; re-run the single failing test in CI before full suite.",
		}
	}
}

func extractSelectorHint(err string) string {
	lower := strings.ToLower(err)
	for _, kw := range []string{"locator", "selector", "getbyrole", "getbytext", "xpath", "css", "data-testid"} {
		if idx := strings.Index(lower, kw); idx >= 0 {
			return truncate(err[idx:], 160)
		}
	}
	return ""
}

func buildSummary(category, testName string) string {
	switch category {
	case CategoryTimeout:
		return "Timeout failure on \"" + testName + "\" — stabilize waits or target service."
	case CategoryLocator:
		return "Locator/element failure on \"" + testName + "\" — review selectors and DOM timing."
	case CategoryStaleElement:
		return "Stale element on \"" + testName + "\" — re-query after DOM updates."
	case CategoryAssertion:
		return "Assertion failure on \"" + testName + "\" — verify expected vs actual."
	case CategoryNetwork:
		return "Network/upstream failure on \"" + testName + "\" — check CI env and dependencies."
	default:
		return "Failure on \"" + testName + "\" — use MCP context to propose a minimal fix."
	}
}

func buildMCPPrompt(incidentID int64, category string) string {
	return strings.TrimSpace(strings.Join([]string{
		"Self-heal this CI test failure using QA Capsule MCP tools.",
		"1. Call get_incident_context with incident_id=" + formatInt(incidentID) + ".",
		"2. Read the test source file from the repo (infer path from stack trace).",
		"3. Apply the smallest correct fix for category \"" + category + "\" (framework-agnostic).",
		"4. Optionally call create_remediation_pr after review.",
	}, "\n"))
}

func buildProposalExplanation(category, testName string, fileContent string) string {
	actions := SuggestedActions(category)
	base := "Self-healing context for \"" + testName + "\" (" + category + "). "
	if len(actions) > 0 {
		base += actions[0]
	}
	if strings.TrimSpace(fileContent) == "" {
		base += " Paste source file content in propose request or open the test file in your IDE via MCP."
	}
	return base
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func formatInt(n int64) string {
	if n <= 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
