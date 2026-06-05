package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// NoOpAnalyzer tests
// ---------------------------------------------------------------------------

func TestNoOpAnalyzer_Analyze_ReturnsDeterministicSummary(t *testing.T) {
	a := NoOpAnalyzer{}
	in := AnalysisInput{
		TestName:     "Login_Test",
		Status:       "FAILED",
		ErrorMessage: "Element '#submit-btn' not found",
	}
	res, err := a.Analyze(context.Background(), ProviderConfig{}, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Summary == "" {
		t.Error("expected a non-empty summary")
	}
	// SelectorHint must be populated because error mentions a selector keyword.
	if res.SelectorHint == "" {
		t.Error("expected SelectorHint to be populated for a locator error")
	}
	if res.Confidence <= 0 {
		t.Error("expected confidence > 0")
	}
}

func TestNoOpAnalyzer_Analyze_TimeoutHint(t *testing.T) {
	a := NoOpAnalyzer{}
	in := AnalysisInput{
		TestName:     "Checkout_Test",
		Status:       "FAILED",
		ErrorMessage: "Timeout waiting for element",
	}
	res, err := a.Analyze(context.Background(), ProviderConfig{}, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The NoOpAnalyzer applies a special hint for timeout errors.
	if !strings.Contains(strings.ToLower(res.Summary), "timeout") {
		t.Errorf("expected timeout hint in summary, got: %s", res.Summary)
	}
}

func TestNoOpAnalyzer_GenerateFixProposal_WithFileContent(t *testing.T) {
	a := NoOpAnalyzer{}
	inc := Incident{
		TestName:     "Payment_Test",
		Status:       "FAILED",
		ErrorMessage: "No such element",
	}
	prop, err := a.GenerateFixProposal(context.Background(), ProviderConfig{}, inc, "existing code here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When file content is provided it must be returned verbatim.
	if prop.Code != "existing code here" {
		t.Errorf("expected code to be preserved, got: %s", prop.Code)
	}
	if prop.Confidence <= 0 {
		t.Error("expected confidence > 0")
	}
}

func TestNoOpAnalyzer_GenerateFixProposal_NoFileContent(t *testing.T) {
	a := NoOpAnalyzer{}
	inc := Incident{
		TestName:     "Payment_Test",
		Status:       "FAILED",
		ErrorMessage: "Element not found",
	}
	prop, err := a.GenerateFixProposal(context.Background(), ProviderConfig{}, inc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When no file content, a synthetic comment block must be returned.
	if strings.TrimSpace(prop.Code) == "" {
		t.Error("expected non-empty code even without file content")
	}
}

// ---------------------------------------------------------------------------
// parseFixProposal tests
// ---------------------------------------------------------------------------

func TestParseFixProposal_ValidJSON(t *testing.T) {
	payload := `{"code":"def test(): pass","explanation":"Updated selector","confidence":0.9}`
	got := parseFixProposal(payload)
	if got.Code != "def test(): pass" {
		t.Errorf("unexpected code: %q", got.Code)
	}
	if got.Explanation != "Updated selector" {
		t.Errorf("unexpected explanation: %q", got.Explanation)
	}
	if got.Confidence != 0.9 {
		t.Errorf("unexpected confidence: %v", got.Confidence)
	}
}

func TestParseFixProposal_JSONEmbeddedInText(t *testing.T) {
	// The LLM sometimes adds prose before the JSON object.
	payload := `Here is the fix:\n{"code":"pass","explanation":"fixed","confidence":0.8}`
	got := parseFixProposal(payload)
	if got.Code != "pass" {
		t.Errorf("expected code 'pass', got: %q", got.Code)
	}
}

func TestParseFixProposal_NonJSON_FallsBackToRawCode(t *testing.T) {
	payload := "replace the selector with data-testid"
	got := parseFixProposal(payload)
	// No JSON found — raw text becomes the Code field.
	if got.Code != payload {
		t.Errorf("expected raw text as code, got: %q", got.Code)
	}
	if got.Confidence != 0.4 {
		t.Errorf("expected fallback confidence 0.4, got: %v", got.Confidence)
	}
}

func TestParseFixProposal_EmptyJSON_FallsBackToRaw(t *testing.T) {
	// Valid JSON but missing meaningful fields — should fall through.
	payload := `{}`
	got := parseFixProposal(payload)
	if got.Code != payload {
		t.Errorf("expected raw payload as code for empty JSON, got: %q", got.Code)
	}
}

// ---------------------------------------------------------------------------
// parseFreeform tests
// ---------------------------------------------------------------------------

func TestParseFreeform_ValidJSON(t *testing.T) {
	raw := `{"summary":"Test failed","root_cause":"missing element","suggested_fix":"update selector","selector_hint":"#btn","confidence":0.75}`
	res := parseFreeform(raw)
	if res.Summary != "Test failed" {
		t.Errorf("unexpected summary: %q", res.Summary)
	}
	if res.SelectorHint != "#btn" {
		t.Errorf("unexpected selector_hint: %q", res.SelectorHint)
	}
	if res.Confidence != 0.75 {
		t.Errorf("unexpected confidence: %v", res.Confidence)
	}
}

func TestParseFreeform_NonJSON_UsesRawSummary(t *testing.T) {
	raw := "the test failed because the element was missing"
	res := parseFreeform(raw)
	if res.Summary != raw {
		t.Errorf("expected raw text as summary, got: %q", res.Summary)
	}
	if res.Confidence != 0.6 {
		t.Errorf("expected fallback confidence 0.6, got: %v", res.Confidence)
	}
}

func TestParseFreeform_LongTextIsTruncated(t *testing.T) {
	raw := strings.Repeat("x", 600)
	res := parseFreeform(raw)
	if len(res.Summary) > 410 { // 400 chars + ellipsis
		t.Errorf("expected truncated summary, got length %d", len(res.Summary))
	}
}

// ---------------------------------------------------------------------------
// extractSelectorHint tests
// ---------------------------------------------------------------------------

func TestExtractSelectorHint_LocatorKeyword(t *testing.T) {
	hint := extractSelectorHint("locator('#pay-button').click() failed")
	if hint == "" {
		t.Error("expected a selector hint, got empty string")
	}
}

func TestExtractSelectorHint_NoKeyword_ReturnsEmpty(t *testing.T) {
	hint := extractSelectorHint("network request timed out")
	if hint != "" {
		t.Errorf("expected empty hint, got: %q", hint)
	}
}

// ---------------------------------------------------------------------------
// chatCompletionsURL tests
// ---------------------------------------------------------------------------

func TestChatCompletionsURL_Groq(t *testing.T) {
	cfg := ProviderConfig{Provider: ProviderGroq}
	url := chatCompletionsURL(cfg)
	if !strings.Contains(url, "groq.com") {
		t.Errorf("expected Groq URL, got: %s", url)
	}
	if !strings.HasSuffix(url, "/chat/completions") {
		t.Errorf("expected URL to end with /chat/completions, got: %s", url)
	}
}

func TestChatCompletionsURL_OpenAI_Default(t *testing.T) {
	cfg := ProviderConfig{Provider: ProviderOpenAI}
	url := chatCompletionsURL(cfg)
	if !strings.Contains(url, "openai.com") {
		t.Errorf("expected openai.com URL, got: %s", url)
	}
}

func TestChatCompletionsURL_CustomBaseURL_NoDuplication(t *testing.T) {
	cfg := ProviderConfig{
		Provider: ProviderOpenAI,
		BaseURL:  "https://custom.proxy.com/v1/chat/completions",
	}
	url := chatCompletionsURL(cfg)
	// The function must not append /chat/completions again.
	count := strings.Count(url, "/chat/completions")
	if count != 1 {
		t.Errorf("expected exactly one /chat/completions segment, got %d in %s", count, url)
	}
}

// ---------------------------------------------------------------------------
// apiKeyFromEnv tests
// ---------------------------------------------------------------------------

func TestAPIKeyFromEnv_ExplicitEnvVar(t *testing.T) {
	const envName = "TEST_QAC_API_KEY_ENV"
	t.Setenv(envName, "my-secret-key")
	cfg := ProviderConfig{APIKeyEnv: envName, Provider: ProviderOpenAI}
	key := apiKeyFromEnv(cfg)
	if key != "my-secret-key" {
		t.Errorf("expected env key, got: %q", key)
	}
}

func TestAPIKeyFromEnv_DirectKey(t *testing.T) {
	cfg := ProviderConfig{APIKey: "direct-key", Provider: ProviderGroq}
	key := apiKeyFromEnv(cfg)
	if key != "direct-key" {
		t.Errorf("expected direct key, got: %q", key)
	}
}

func TestAPIKeyFromEnv_FallbackGroq(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "groq-fallback")
	cfg := ProviderConfig{Provider: ProviderGroq}
	key := apiKeyFromEnv(cfg)
	if key != "groq-fallback" {
		t.Errorf("expected GROQ_API_KEY fallback, got: %q", key)
	}
}

func TestAPIKeyFromEnv_EnvVarTakesPriorityOverDirectKey(t *testing.T) {
	const envName = "TEST_PRIORITY_KEY"
	t.Setenv(envName, "env-priority")
	cfg := ProviderConfig{APIKeyEnv: envName, APIKey: "direct-key", Provider: ProviderOpenAI}
	key := apiKeyFromEnv(cfg)
	// Explicit env var must win over the direct key stored in DB.
	if key != "env-priority" {
		t.Errorf("expected env var to take priority, got: %q", key)
	}
}

func TestAPIKeyFromEnv_Missing_ReturnsEmpty(t *testing.T) {
	// Ensure the fallback env vars are not set for this test.
	os.Unsetenv("OPENAI_API_KEY")
	cfg := ProviderConfig{Provider: ProviderOpenAI}
	key := apiKeyFromEnv(cfg)
	if key != "" {
		t.Errorf("expected empty key when nothing is set, got: %q", key)
	}
}

// ---------------------------------------------------------------------------
// HTTPAnalyzer integration test (uses httptest server to avoid real LLM calls)
// ---------------------------------------------------------------------------

func TestHTTPAnalyzer_Analyze_OpenAI_HappyPath(t *testing.T) {
	// Simulate a minimal OpenAI-compatible /v1/chat/completions response.
	llmResponse := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]string{
					"content": `{"summary":"Element not found","root_cause":"broken selector","suggested_fix":"use data-testid","selector_hint":"#old-btn","confidence":0.85}`,
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(llmResponse)
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		Provider:  ProviderOpenAI,
		BaseURL:   srv.URL + "/v1/chat/completions",
		APIKey:    "test-key",
		Model:     "gpt-4o-mini",
		MaxTokens: 512,
	}
	in := AnalysisInput{
		TestName:     "Checkout_Test",
		Status:       "FAILED",
		ErrorMessage: "Element '#old-btn' not found",
	}

	a := HTTPAnalyzer{}
	res, err := a.Analyze(context.Background(), cfg, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Summary != "Element not found" {
		t.Errorf("unexpected summary: %q", res.Summary)
	}
	if res.Confidence != 0.85 {
		t.Errorf("unexpected confidence: %v", res.Confidence)
	}
}

func TestHTTPAnalyzer_Analyze_LLM4xx_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		Provider: ProviderOpenAI,
		BaseURL:  srv.URL + "/v1/chat/completions",
		APIKey:   "bad-key",
		Model:    "gpt-4o-mini",
	}
	a := HTTPAnalyzer{}
	_, err := a.Analyze(context.Background(), cfg, AnalysisInput{TestName: "X", Status: "FAILED"})
	if err == nil {
		t.Fatal("expected error from 4xx response")
	}
}

func TestHTTPAnalyzer_GenerateFixProposal_HappyPath(t *testing.T) {
	llmResponse := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]string{
					"content": `{"code":"cy.get('[data-testid=submit]').click()","explanation":"Replaced fragile ID selector with data-testid","confidence":0.92}`,
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(llmResponse)
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		Provider: ProviderOpenAI,
		BaseURL:  srv.URL + "/v1/chat/completions",
		APIKey:   "test-key",
		Model:    "gpt-4o-mini",
	}
	inc := Incident{
		TestName:     "Checkout_Submit",
		Status:       "FAILED",
		ErrorMessage: "Element '#submit' not found",
	}

	a := HTTPAnalyzer{}
	prop, err := a.GenerateFixProposal(context.Background(), cfg, inc, "cy.get('#submit').click()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prop.Code, "data-testid") {
		t.Errorf("expected data-testid in code, got: %q", prop.Code)
	}
	if prop.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got: %v", prop.Confidence)
	}
}

// ---------------------------------------------------------------------------
// truncate helper tests
// ---------------------------------------------------------------------------

func TestTruncate_ShortString_Unchanged(t *testing.T) {
	s := "hello"
	got := truncate(s, 100)
	if got != s {
		t.Errorf("expected unchanged string, got: %q", got)
	}
}

func TestTruncate_LongString_IsCut(t *testing.T) {
	s := strings.Repeat("a", 200)
	got := truncate(s, 50)
	if len(got) > 55 { // 50 chars + multi-byte ellipsis
		t.Errorf("expected truncated string, got length %d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Error("expected ellipsis at end of truncated string")
	}
}

func TestTruncate_ExactBoundary_Unchanged(t *testing.T) {
	s := strings.Repeat("b", 10)
	got := truncate(s, 10)
	if got != s {
		t.Errorf("exact-length string should be unchanged, got: %q", got)
	}
}
