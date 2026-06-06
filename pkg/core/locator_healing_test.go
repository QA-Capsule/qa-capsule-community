package core

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// IsLocatorError tests
// ---------------------------------------------------------------------------

func TestIsLocatorError_PlaywrightTimeout(t *testing.T) {
	err := "locator('#submit').click() — Timeout 5000ms exceeded"
	if !IsLocatorError(err) {
		t.Errorf("expected locator error for: %q", err)
	}
}

func TestIsLocatorError_SeleniumNoSuchElement(t *testing.T) {
	err := "NoSuchElementException: no such element; CSS selector: #login-btn"
	if !IsLocatorError(err) {
		t.Errorf("expected locator error for: %q", err)
	}
}

func TestIsLocatorError_CypressUnableToFind(t *testing.T) {
	err := "Unable to find element: '[data-testid=\"submit-payment\"]'"
	if !IsLocatorError(err) {
		t.Errorf("expected locator error for: %q", err)
	}
}

func TestIsLocatorError_RobotElementNotFound(t *testing.T) {
	err := "Element '#confirm-btn' not found"
	if !IsLocatorError(err) {
		t.Errorf("expected locator error for: %q", err)
	}
}

func TestIsLocatorError_NetworkError_IsFalse(t *testing.T) {
	err := "ConnectionRefusedError: dial tcp :8080 connection refused"
	if IsLocatorError(err) {
		t.Errorf("should not flag network error as locator error: %q", err)
	}
}

func TestIsLocatorError_AssertionError_IsFalse(t *testing.T) {
	err := "AssertionError: expected 200 but got 404"
	if IsLocatorError(err) {
		t.Errorf("should not flag assertion error as locator error: %q", err)
	}
}

func TestIsLocatorError_Empty_IsFalse(t *testing.T) {
	if IsLocatorError("") {
		t.Error("empty error should not be classified as a locator error")
	}
}

// ---------------------------------------------------------------------------
// ExtractLocator tests
// ---------------------------------------------------------------------------

func TestExtractLocator_Playwright_SingleQuotes(t *testing.T) {
	msg := "locator('#pay-button').click() — Timeout exceeded"
	got := ExtractLocator(msg, "Playwright")
	if got != "#pay-button" {
		t.Errorf("expected '#pay-button', got: %q", got)
	}
}

func TestExtractLocator_Playwright_DoubleQuotes(t *testing.T) {
	msg := `locator("#submit-btn").click()`
	got := ExtractLocator(msg, "Playwright")
	if got != "#submit-btn" {
		t.Errorf("expected '#submit-btn', got: %q", got)
	}
}

func TestExtractLocator_Selenium_SelectorColon(t *testing.T) {
	msg := `NoSuchElementException: selector: "#login-form"`
	got := ExtractLocator(msg, "Selenium")
	if got == "" {
		t.Error("expected a selector to be extracted")
	}
}

func TestExtractLocator_Cypress_DataTestID(t *testing.T) {
	msg := `Expected to find element: '[data-testid="checkout-btn"]'`
	got := ExtractLocator(msg, "Cypress")
	if got == "" {
		t.Errorf("expected a locator to be extracted from: %q", msg)
	}
}

func TestExtractLocator_Generic_CSSHash(t *testing.T) {
	msg := "element #header not found"
	got := ExtractLocator(msg, "")
	if !strings.HasPrefix(got, "#") {
		t.Errorf("expected CSS ID selector, got: %q", got)
	}
}

func TestExtractLocator_NoLocatorInMessage_ReturnsEmpty(t *testing.T) {
	msg := "network timeout after 30s"
	got := ExtractLocator(msg, "")
	if got != "" {
		t.Errorf("expected empty result, got: %q", got)
	}
}

func TestExtractLocator_FrameworkHint_FiltersByFramework(t *testing.T) {
	// Message contains only a Cypress-style locator.
	msg := `Expected to find element: '#checkout'`
	gotPlaywright := ExtractLocator(msg, "Playwright")
	gotCypress := ExtractLocator(msg, "Cypress")
	// Cypress should match; Playwright patterns should not match this message.
	if gotCypress == "" {
		t.Errorf("Cypress hint should extract locator from: %q", msg)
	}
	// Playwright may or may not match the generic fallback — just ensure
	// the function does not panic or return a nonsensical result.
	_ = gotPlaywright
}

// ---------------------------------------------------------------------------
// SuggestHealedLocator tests
// ---------------------------------------------------------------------------

func TestSuggestHealedLocator_IDSelector_SuggestsDataTestID(t *testing.T) {
	healed, conf, explanation := SuggestHealedLocator("#submit-btn", "Playwright")
	if !strings.Contains(healed, "data-testid") {
		t.Errorf("expected data-testid suggestion, got: %q", healed)
	}
	if conf <= 0 {
		t.Error("expected confidence > 0")
	}
	if explanation == "" {
		t.Error("expected a non-empty explanation")
	}
}

func TestSuggestHealedLocator_ClassSelector_SuggestsPartialMatch(t *testing.T) {
	healed, conf, explanation := SuggestHealedLocator(".primary-cta", "")
	if !strings.Contains(healed, "class*=") {
		t.Errorf("expected partial class match, got: %q", healed)
	}
	if conf <= 0 {
		t.Error("expected confidence > 0")
	}
	if explanation == "" {
		t.Error("expected a non-empty explanation")
	}
}

func TestSuggestHealedLocator_DataTestID_SuggestsRole(t *testing.T) {
	healed, conf, explanation := SuggestHealedLocator(`[data-testid="pay-button"]`, "Playwright")
	if !strings.Contains(healed, "getByRole") {
		t.Errorf("expected getByRole suggestion, got: %q", healed)
	}
	if conf <= 0 {
		t.Error("expected confidence > 0")
	}
	if explanation == "" {
		t.Error("expected a non-empty explanation")
	}
}

func TestSuggestHealedLocator_DataTestIDWithInputRole(t *testing.T) {
	healed, _, _ := SuggestHealedLocator(`[data-testid="email-input"]`, "Playwright")
	// "input" keyword in the locator → role should be textbox.
	if !strings.Contains(healed, "textbox") {
		t.Errorf("expected textbox role for input locator, got: %q", healed)
	}
}

func TestSuggestHealedLocator_DataTestIDWithLinkRole(t *testing.T) {
	healed, _, _ := SuggestHealedLocator(`[data-testid="nav-link"]`, "")
	if !strings.Contains(healed, "link") {
		t.Errorf("expected link role, got: %q", healed)
	}
}

func TestSuggestHealedLocator_UnrecognizedPattern_LowConfidence(t *testing.T) {
	healed, conf, explanation := SuggestHealedLocator("xpath://div[@id='foo']", "")
	if conf >= 0.5 {
		t.Errorf("expected low confidence for unrecognized pattern, got: %v", conf)
	}
	if healed == "" || explanation == "" {
		t.Error("expected non-empty healed and explanation even for unknown pattern")
	}
}

func TestSuggestHealedLocator_EmptyLocator_ZeroConfidence(t *testing.T) {
	_, conf, _ := SuggestHealedLocator("", "")
	if conf != 0 {
		t.Errorf("expected 0 confidence for empty locator, got: %v", conf)
	}
}

// ---------------------------------------------------------------------------
// ExtractDOMSnapshot tests
// ---------------------------------------------------------------------------

func TestExtractDOMSnapshot_ValidMarkers(t *testing.T) {
	logs := "some log line\n[QA_CAPSULE_DOM_SNAPSHOT_START]<html><body>page</body></html>[QA_CAPSULE_DOM_SNAPSHOT_END]\nafter"
	got := ExtractDOMSnapshot(logs)
	if got != "<html><body>page</body></html>" {
		t.Errorf("unexpected DOM snapshot: %q", got)
	}
}

func TestExtractDOMSnapshot_NoStartMarker_ReturnsEmpty(t *testing.T) {
	logs := "no markers here"
	got := ExtractDOMSnapshot(logs)
	if got != "" {
		t.Errorf("expected empty snapshot, got: %q", got)
	}
}

func TestExtractDOMSnapshot_NoEndMarker_ReturnsEverythingAfterStart(t *testing.T) {
	logs := "[QA_CAPSULE_DOM_SNAPSHOT_START]<html>truncated"
	got := ExtractDOMSnapshot(logs)
	if got != "<html>truncated" {
		t.Errorf("expected content after start marker, got: %q", got)
	}
}

func TestExtractDOMSnapshot_MultilineHTML(t *testing.T) {
	logs := "[QA_CAPSULE_DOM_SNAPSHOT_START]\n<html>\n  <body>\n    <div>content</div>\n  </body>\n</html>\n[QA_CAPSULE_DOM_SNAPSHOT_END]"
	got := ExtractDOMSnapshot(logs)
	if !strings.Contains(got, "<div>content</div>") {
		t.Errorf("expected full multi-line HTML, got: %q", got)
	}
}

func TestExtractDOMSnapshot_EmptyInput_ReturnsEmpty(t *testing.T) {
	got := ExtractDOMSnapshot("")
	if got != "" {
		t.Errorf("expected empty for empty input, got: %q", got)
	}
}

func TestExtractDOMSnapshotFromLogs_PrefersConsoleLogs(t *testing.T) {
	console := "[QA_CAPSULE_DOM_SNAPSHOT_START]<html>console</html>[QA_CAPSULE_DOM_SNAPSHOT_END]"
	errorLogs := "[QA_CAPSULE_DOM_SNAPSHOT_START]<html>stderr</html>[QA_CAPSULE_DOM_SNAPSHOT_END]"
	got := ExtractDOMSnapshotFromLogs(console, errorLogs)
	if got != "<html>console</html>" {
		t.Errorf("expected console snapshot, got: %q", got)
	}
}

func TestExtractDOMSnapshotFromLogs_FallsBackToErrorLogs(t *testing.T) {
	errorLogs := "[QA_CAPSULE_DOM_SNAPSHOT_START]<html>playwright</html>[QA_CAPSULE_DOM_SNAPSHOT_END]"
	got := ExtractDOMSnapshotFromLogs("", errorLogs)
	if got != "<html>playwright</html>" {
		t.Errorf("expected error_logs snapshot, got: %q", got)
	}
}

func TestHealLocatorFromHTML_BrokenDataQA_FindsSubmitID(t *testing.T) {
	html := `<html><body><button id="submit" type="submit">Submit</button></body></html>`
	healed, conf, _, ok := HealLocatorFromHTML(`button[data-qa="submit-v1"]`, html)
	if !ok {
		t.Fatal("expected HTML heal to succeed")
	}
	if healed != "#submit" {
		t.Errorf("expected #submit, got: %q", healed)
	}
	if conf < 0.7 {
		t.Errorf("expected high confidence, got: %v", conf)
	}
}

func TestLocatorExistsInHTML_IDSelector(t *testing.T) {
	html := `<button id="submit" type="submit">Go</button>`
	if !LocatorExistsInHTML(html, "#submit") {
		t.Error("expected #submit to exist in HTML")
	}
	if LocatorExistsInHTML(html, "#missing") {
		t.Error("expected #missing to be absent")
	}
}

func TestExtractLocatorFromTestSource_SeleniumPython(t *testing.T) {
	source := `def test_login(driver):
    driver.find_element(By.CSS_SELECTOR, 'button[data-qa="submit-v1"]').click()
`
	got := ExtractLocatorFromTestSource(source, "Selenium")
	if got != `button[data-qa="submit-v1"]` {
		t.Errorf("expected selenium selector, got: %q", got)
	}
}

func TestExtractLocatorFromTestSource_PlaywrightJS(t *testing.T) {
	source := `test('broken', async ({ page }) => {
  await page.locator('button[data-qa="checkout-v2"]').click();
});`
	got := ExtractLocatorFromTestSource(source, "Playwright")
	if got != `button[data-qa="checkout-v2"]` {
		t.Errorf("expected playwright selector, got: %q", got)
	}
}

func TestExtractLocatorFromTestSource_RobotBrokenVariable(t *testing.T) {
	source := `*** Variables ***
${BROKEN_SUBMIT}    button[data-qa="submit-v1"]
*** Test Cases ***
Demo
    Click    ${BROKEN_SUBMIT}
`
	got := ExtractLocatorFromTestSource(source, "RobotFramework")
	if got != `button[data-qa="submit-v1"]` {
		t.Errorf("expected broken selector from robot file, got: %q", got)
	}
}

func TestIsLocatorFailure_RobotTimeoutWithTestSource(t *testing.T) {
	source := `${BROKEN_SUBMIT}    button[data-qa="submit-v1"]`
	errMsg := "TimeoutError: timeout 10000ms exceeded"
	if !IsLocatorFailure(errMsg, "", source, "TC-04 Broken Locator") {
		t.Error("expected timeout + broken selector in test source to be locator failure")
	}
}

func TestIsEquivalentBrokenLocator_XPathVariant(t *testing.T) {
	orig := `button[data-qa="submit-v1"]`
	healed := `xpath=(//button[@data-qa='submit-v1'])[1]`
	if !IsEquivalentBrokenLocator(orig, healed) {
		t.Error("expected xpath variant of same data-qa to be rejected")
	}
}
