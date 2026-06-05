package server

import (
	"testing"
)

// ---------------------------------------------------------------------------
// detectFramework tests
// ---------------------------------------------------------------------------

func TestDetectFramework_PlaywrightFromTestName(t *testing.T) {
	fw := detectFramework("[Playwright] Login > Submit button", "", "")
	if fw != "Playwright" {
		t.Errorf("expected Playwright, got %q", fw)
	}
}

func TestDetectFramework_CypressFromTestName(t *testing.T) {
	fw := detectFramework("[Cypress] Checkout > Payment", "", "")
	if fw != "Cypress" {
		t.Errorf("expected Cypress, got %q", fw)
	}
}

func TestDetectFramework_RobotFromStackTrace(t *testing.T) {
	fw := detectFramework("Login test", "tests/login.robot:27", "")
	if fw != "RobotFramework" {
		t.Errorf("expected RobotFramework, got %q", fw)
	}
}

func TestDetectFramework_PlaywrightFromStackTrace(t *testing.T) {
	fw := detectFramework("Test", "  at checkout.spec.ts:42:15", "")
	if fw != "Playwright" {
		t.Errorf("expected Playwright, got %q", fw)
	}
}

func TestDetectFramework_SeleniumFromErrorMsg(t *testing.T) {
	fw := detectFramework("Test", "", "NoSuchElementException: selenium driver error")
	if fw != "Selenium" {
		t.Errorf("expected Selenium, got %q", fw)
	}
}

func TestDetectFramework_Unknown(t *testing.T) {
	fw := detectFramework("", "", "network error 503")
	if fw != "unknown" {
		t.Errorf("expected unknown, got %q", fw)
	}
}

// ---------------------------------------------------------------------------
// canonicalFramework tests
// ---------------------------------------------------------------------------

func TestCanonicalFramework_KnownValues(t *testing.T) {
	cases := map[string]string{
		"playwright": "Playwright",
		"cypress":    "Cypress",
		"selenium":   "Selenium",
		"robot":      "RobotFramework",
		"pytest":     "Pytest",
		"jest":       "Jest",
		"vitest":     "Jest",
	}
	for input, want := range cases {
		got := canonicalFramework(input)
		if got != want {
			t.Errorf("canonicalFramework(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCanonicalFramework_UnknownPassthrough(t *testing.T) {
	got := canonicalFramework("mocha")
	if got != "mocha" {
		t.Errorf("expected passthrough for unknown framework, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// buildLocatorWhatChanged tests
// ---------------------------------------------------------------------------

func TestBuildLocatorWhatChanged_NormalChange(t *testing.T) {
	msg := buildLocatorWhatChanged("#submit-btn", "[data-testid='submit']", "Playwright")
	if msg == "" {
		t.Error("expected a non-empty what-changed message")
	}
	// Must mention both the original and healed locators.
	if !contains(msg, "#submit-btn") {
		t.Errorf("expected original locator in message, got: %q", msg)
	}
	if !contains(msg, "[data-testid='submit']") {
		t.Errorf("expected healed locator in message, got: %q", msg)
	}
}

func TestBuildLocatorWhatChanged_EmptyHealed(t *testing.T) {
	msg := buildLocatorWhatChanged("#broken", "", "")
	if msg == "" {
		t.Error("expected a fallback message for empty healed locator")
	}
}

func TestBuildLocatorWhatChanged_SameLocator(t *testing.T) {
	msg := buildLocatorWhatChanged("#same", "#same", "Cypress")
	if msg == "" {
		t.Error("expected a message even when locator is unchanged")
	}
}

// ---------------------------------------------------------------------------
// buildScriptWhatChanged tests
// ---------------------------------------------------------------------------

func TestBuildScriptWhatChanged_TruncatesLongExplanation(t *testing.T) {
	long := "This is a very long explanation that goes on and on and should be truncated to the first sentence. Everything after this point should be omitted."
	msg := buildScriptWhatChanged(long)
	if len(msg) > 210 {
		t.Errorf("expected truncated message, got length %d", len(msg))
	}
}

func TestBuildScriptWhatChanged_FirstSentenceOnly(t *testing.T) {
	explanation := "Replaced fragile ID selector. Updated timeout to 10 seconds."
	msg := buildScriptWhatChanged(explanation)
	if msg != "Replaced fragile ID selector." {
		t.Errorf("expected first sentence only, got: %q", msg)
	}
}

func TestBuildScriptWhatChanged_EmptyFallback(t *testing.T) {
	msg := buildScriptWhatChanged("")
	if msg == "" {
		t.Error("expected a non-empty fallback message")
	}
}

// ---------------------------------------------------------------------------
// buildLocatorNextSteps tests
// ---------------------------------------------------------------------------

func TestBuildLocatorNextSteps_WithFilePath(t *testing.T) {
	steps := buildLocatorNextSteps(42, "tests/login.robot", "#old", "[data-testid='submit']")
	if len(steps) < 3 {
		t.Errorf("expected at least 3 steps, got %d", len(steps))
	}
	// First step must mention the file.
	if !contains(steps[0], "tests/login.robot") {
		t.Errorf("expected file path in first step, got: %q", steps[0])
	}
}

func TestBuildLocatorNextSteps_WithoutFilePath(t *testing.T) {
	steps := buildLocatorNextSteps(42, "", "#old", "[data-testid='submit']")
	if len(steps) == 0 {
		t.Error("expected at least one step even without file path")
	}
}

// ---------------------------------------------------------------------------
// buildScriptNextSteps tests
// ---------------------------------------------------------------------------

func TestBuildScriptNextSteps_WithFilePath(t *testing.T) {
	steps := buildScriptNextSteps(7, "tests/checkout.spec.ts")
	if len(steps) < 2 {
		t.Errorf("expected at least 2 steps, got %d", len(steps))
	}
	if !contains(steps[0], "checkout.spec.ts") {
		t.Errorf("expected file path in first step, got: %q", steps[0])
	}
}

func TestBuildScriptNextSteps_WithoutFilePath(t *testing.T) {
	steps := buildScriptNextSteps(7, "")
	if len(steps) == 0 {
		t.Error("expected at least one step without file path")
	}
}

// ---------------------------------------------------------------------------
// helper
// ---------------------------------------------------------------------------

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
