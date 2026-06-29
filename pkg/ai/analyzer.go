package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Analyzer interface {
	Analyze(ctx context.Context, cfg ProviderConfig, in AnalysisInput) (AnalysisResult, error)
	GenerateFixProposal(ctx context.Context, cfg ProviderConfig, incident Incident, fileContent string) (FixProposal, error)
}

// NoOpAnalyzer satisfies the Analyzer interface and is used when no AI
// provider is configured (ProviderDisabled) or when the provider encounters
// an error. It returns deterministic, rule-based summaries derived from the
// incident data alone, without making any network calls.
type NoOpAnalyzer struct{}

func (NoOpAnalyzer) GenerateFixProposal(ctx context.Context, cfg ProviderConfig, incident Incident, fileContent string) (FixProposal, error) {
	_ = ctx
	_ = cfg
	explanation := fmt.Sprintf(
		"AI provider not configured. Based on the error %q, review test %q and apply the minimal fix.",
		truncate(incident.ErrorMessage, 120), incident.TestName,
	)
	code := fileContent
	if strings.TrimSpace(code) == "" {
		code = fmt.Sprintf("# Test: %s\n# Error: %s\n# Fix: update the test logic to match the actual application behaviour.",
			incident.TestName, truncate(incident.ErrorMessage, 200))
	}
	return FixProposal{
		Code:        code,
		Explanation: explanation,
		Confidence:  0.3,
		RawJSON:     `{"source":"no_op"}`,
	}, nil
}

func (NoOpAnalyzer) Analyze(ctx context.Context, cfg ProviderConfig, in AnalysisInput) (AnalysisResult, error) {
	summary := fmt.Sprintf("Test %q failed with status %s. Review the error and console output for the failing step.", in.TestName, in.Status)
	if strings.Contains(strings.ToLower(in.ErrorMessage), "timeout") {
		summary = fmt.Sprintf("Likely timeout on %q — increase wait or stabilize the target element before assertion.", in.TestName)
	}
	return AnalysisResult{
		Summary:      summary,
		RootCause:    truncate(in.ErrorMessage, 500),
		SuggestedFix: "Verify selectors, network stability, and test data for this step.",
		SelectorHint: extractSelectorHint(in.ErrorMessage),
		Confidence:   0.5,
		RawJSON:      `{"source":"no_op"}`,
	}, nil
}

type HTTPAnalyzer struct{}

func (HTTPAnalyzer) Analyze(ctx context.Context, cfg ProviderConfig, in AnalysisInput) (AnalysisResult, error) {
	prompt := buildPrompt(in)
	switch cfg.Provider {
	case ProviderOllama:
		return callOllama(ctx, cfg, prompt)
	case ProviderAnthropic:
		return callAnthropic(ctx, cfg, prompt)
	case ProviderGemini:
		return callGemini(ctx, cfg, prompt)
	case ProviderOpenAI, ProviderMistral, ProviderGroq, ProviderOpenRouter, ProviderAzure:
		return callOpenAI(ctx, cfg, prompt)
	default:
		return NoOpAnalyzer{}.Analyze(ctx, cfg, in)
	}
}

func (HTTPAnalyzer) GenerateFixProposal(ctx context.Context, cfg ProviderConfig, incident Incident, fileContent string) (FixProposal, error) {
	prompt := buildFixProposalPrompt(incident, fileContent)
	var text string
	switch cfg.Provider {
	case ProviderOllama:
		res, e := callOllama(ctx, cfg, prompt)
		if e != nil {
			return FixProposal{}, e
		}
		text = llmTextFromAnalysis(res)
	case ProviderAnthropic:
		res, e := callAnthropic(ctx, cfg, prompt)
		if e != nil {
			return FixProposal{}, e
		}
		text = llmTextFromAnalysis(res)
	case ProviderGemini:
		res, e := callGemini(ctx, cfg, prompt)
		if e != nil {
			return FixProposal{}, e
		}
		text = llmTextFromAnalysis(res)
	case ProviderOpenAI, ProviderMistral, ProviderGroq, ProviderOpenRouter, ProviderAzure:
		res, e := callOpenAI(ctx, cfg, prompt)
		if e != nil {
			return FixProposal{}, e
		}
		text = llmTextFromAnalysis(res)
	default:
		return NoOpAnalyzer{}.GenerateFixProposal(ctx, cfg, incident, fileContent)
	}
	return parseFixProposal(text), nil
}

func llmTextFromAnalysis(res AnalysisResult) string {
	if strings.TrimSpace(res.RawJSON) != "" {
		return res.RawJSON
	}
	return res.Summary
}

// GenerateFixProposal asks the configured LLM for a minimal source patch (language-agnostic).
func GenerateFixProposal(ctx context.Context, cfg ProviderConfig, incident Incident, fileContent string) (code string, explanation string, err error) {
	var analyzer Analyzer = HTTPAnalyzer{}
	if cfg.Provider == ProviderDisabled || !cfg.Enabled {
		analyzer = NoOpAnalyzer{}
	}
	prop, err := analyzer.GenerateFixProposal(ctx, cfg, incident, fileContent)
	if err != nil {
		return "", "", err
	}
	return prop.Code, prop.Explanation, nil
}

func buildFixProposalPrompt(incident Incident, fileContent string) string {
	var b strings.Builder
	b.WriteString(`You are an expert software engineer performing test-driven self-healing for a CI failure.
Infer the test framework and language from the test name, stack trace, and error message.
Respond with a single JSON object (no markdown fences) with keys:
  "code" (string: the corrected test code),
  "explanation" (string: concise rationale — what was wrong and what was changed),
  "confidence" (number 0-1).

Rules:
- When source file content is provided, return the full corrected file.
- When source file content is ABSENT, generate a minimal corrected version of the failing test
  based solely on the error message, stack trace, and test name. Do NOT ask for source code.
- Apply the smallest correct fix. Do not rewrite unrelated logic.
- Never output markdown fences, only raw JSON.

`)
	fmt.Fprintf(&b, "TestName: %s\nStatus: %s\nProject: %s\n", incident.TestName, incident.Status, incident.ProjectName)
	b.WriteString("\nErrorMessage:\n")
	b.WriteString(truncate(incident.ErrorMessage, 6000))
	b.WriteString("\nStackTrace:\n")
	b.WriteString(truncate(incident.StackTrace, 8000))
	if incident.ConsoleLogs != "" {
		// ConsoleLogs may contain the full page DOM snapshot (up to ~24 KB).
		// Passing it to the LLM lets the AI read the real page structure
		// instead of guessing from the error message alone.
		b.WriteString("\nConsoleLogs (may contain page DOM snapshot):\n")
		b.WriteString(truncate(incident.ConsoleLogs, 20000))
	}
	if strings.TrimSpace(fileContent) != "" {
		b.WriteString("\n--- Source file (raw) ---\n")
		b.WriteString(truncate(fileContent, 12000))
	}
	return b.String()
}

func parseFixProposal(text string) FixProposal {
	text = strings.TrimSpace(text)
	var parsed struct {
		Code        string  `json:"code"`
		Explanation string  `json:"explanation"`
		Confidence  float64 `json:"confidence"`
	}
	if i := strings.Index(text, "{"); i >= 0 {
		if json.Unmarshal([]byte(text[i:]), &parsed) == nil && (parsed.Code != "" || parsed.Explanation != "") {
			return FixProposal{
				Code:        parsed.Code,
				Explanation: parsed.Explanation,
				Confidence:  parsed.Confidence,
				RawJSON:     text[i:],
			}
		}
	}
	return FixProposal{
		Code:        text,
		Explanation: "LLM returned non-JSON; raw output used as code.",
		Confidence:  0.4,
		RawJSON:     text,
	}
}

func buildPrompt(in AnalysisInput) string {
	var b strings.Builder
	b.WriteString("You are an SRE assistant. Analyze this CI test failure and respond in JSON with keys: summary, root_cause, suggested_fix, selector_hint, confidence (0-1).\n")
	fmt.Fprintf(&b, "Project: %s\nTest: %s\nStatus: %s\nBrowser: %s OS: %s Viewport: %s\n",
		in.ProjectName, in.TestName, in.Status, in.Browser, in.OS, in.Viewport)
	b.WriteString("\nError:\n")
	b.WriteString(truncate(in.ErrorMessage, 4000))
	b.WriteString("\nConsole:\n")
	b.WriteString(truncate(in.ConsoleLogs, 2000))
	return b.String()
}

func callOpenAI(ctx context.Context, cfg ProviderConfig, prompt string) (AnalysisResult, error) {
	key := apiKeyFromEnv(cfg)
	if key == "" {
		return AnalysisResult{}, fmt.Errorf("missing API key env %s", cfg.APIKeyEnv)
	}
	url := chatCompletionsURL(cfg)
	body := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": cfg.MaxTokens,
	}
	return postChat(ctx, url, key, body)
}

func chatCompletionsURL(cfg ProviderConfig) string {
	base := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		switch cfg.Provider {
		case ProviderMistral:
			base = "https://api.mistral.ai"
		case ProviderGroq:
			base = "https://api.groq.com/openai"
		case ProviderOpenRouter:
			base = "https://openrouter.ai/api"
		case ProviderAzure:
			base = "https://YOUR-RESOURCE.openai.azure.com/openai/deployments/YOUR-DEPLOYMENT"
		default:
			base = "https://api.openai.com"
		}
	}
	if strings.Contains(base, "chat/completions") {
		return base
	}
	if cfg.Provider == ProviderAzure {
		return base + "/chat/completions?api-version=2024-08-01-preview"
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	if cfg.Provider == ProviderGroq {
		return base + "/v1/chat/completions"
	}
	if cfg.Provider == ProviderOpenRouter {
		return base + "/v1/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func apiKeyFromEnv(cfg ProviderConfig) string {
	// 1. Env var takes priority (set at server level — most secure).
	if cfg.APIKeyEnv != "" {
		if k := os.Getenv(cfg.APIKeyEnv); k != "" {
			return k
		}
	}
	// 2. Key entered directly from the UI and stored in the local DB.
	if strings.TrimSpace(cfg.APIKey) != "" {
		return strings.TrimSpace(cfg.APIKey)
	}
	// 3. Well-known fallback env var names per provider.
	switch cfg.Provider {
	case ProviderOpenAI, ProviderAzure:
		return os.Getenv("OPENAI_API_KEY")
	case ProviderAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case ProviderGemini:
		if k := os.Getenv("GEMINI_API_KEY"); k != "" {
			return k
		}
		return os.Getenv("GOOGLE_API_KEY")
	case ProviderMistral:
		return os.Getenv("MISTRAL_API_KEY")
	case ProviderGroq:
		return os.Getenv("GROQ_API_KEY")
	case ProviderOpenRouter:
		return os.Getenv("OPENROUTER_API_KEY")
	default:
		return ""
	}
}

func callOllama(ctx context.Context, cfg ProviderConfig, prompt string) (AnalysisResult, error) {
	base := cfg.BaseURL
	if base == "" {
		base = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}
	body := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	raw, err := postRaw(ctx, base+"/api/generate", "", body)
	if err != nil {
		return AnalysisResult{}, err
	}
	var resp struct {
		Response string `json:"response"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return parseFreeform(string(raw)), nil
	}
	return parseFreeform(resp.Response), nil
}

func postChat(ctx context.Context, url, apiKey string, body map[string]interface{}) (AnalysisResult, error) {
	raw, err := postRaw(ctx, url, apiKey, body)
	if err != nil {
		return AnalysisResult{}, err
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || len(resp.Choices) == 0 {
		return parseFreeform(string(raw)), nil
	}
	return parseFreeform(resp.Choices[0].Message.Content), nil
}

func postRaw(ctx context.Context, url, apiKey string, body map[string]interface{}) ([]byte, error) {
	headers := map[string]string{}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	return postRawHeaders(ctx, url, headers, body)
}

func postRawHeaders(ctx context.Context, url string, headers map[string]string, body map[string]interface{}) ([]byte, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("LLM HTTP %d: %s", res.StatusCode, truncate(string(raw), 200))
	}
	return raw, nil
}

func parseFreeform(text string) AnalysisResult {
	text = strings.TrimSpace(text)
	var parsed struct {
		Summary      string  `json:"summary"`
		RootCause    string  `json:"root_cause"`
		SuggestedFix string  `json:"suggested_fix"`
		SelectorHint string  `json:"selector_hint"`
		Confidence   float64 `json:"confidence"`
	}
	if i := strings.Index(text, "{"); i >= 0 {
		if json.Unmarshal([]byte(text[i:]), &parsed) == nil && parsed.Summary != "" {
			return AnalysisResult{
				Summary: parsed.Summary, RootCause: parsed.RootCause,
				SuggestedFix: parsed.SuggestedFix, SelectorHint: parsed.SelectorHint,
				Confidence: parsed.Confidence, RawJSON: text[i:],
			}
		}
	}
	return AnalysisResult{
		Summary:      truncate(text, 400),
		RootCause:    "",
		SuggestedFix: "",
		SelectorHint: extractSelectorHint(text),
		Confidence:   0.6,
		RawJSON:      text,
	}
}

func extractSelectorHint(err string) string {
	lower := strings.ToLower(err)
	for _, kw := range []string{"locator", "selector", "element", "getbyrole", "getbytext", "xpath", "css"} {
		if strings.Contains(lower, kw) {
			idx := strings.Index(lower, kw)
			return truncate(err[idx:], 120)
		}
	}
	if idx := strings.Index(err, "#"); idx >= 0 {
		return truncate(err[idx:], 120)
	}
	return ""
}

// AnalyzeRaw sends a pre-built prompt directly to the configured LLM without
// any re-wrapping or truncation.  Use this when the caller has already
// constructed the full prompt (e.g. locator healing with embedded page HTML).
// Returns the raw text response from the LLM.
func AnalyzeRaw(ctx context.Context, cfg ProviderConfig, prompt string) (string, error) {
	var res AnalysisResult
	var err error
	switch cfg.Provider {
	case ProviderOpenAI, ProviderMistral, ProviderGroq, ProviderOpenRouter, ProviderAzure:
		res, err = callOpenAI(ctx, cfg, prompt)
	case ProviderGemini:
		res, err = callGemini(ctx, cfg, prompt)
	case ProviderAnthropic:
		res, err = callAnthropic(ctx, cfg, prompt)
	case ProviderOllama:
		res, err = callOllama(ctx, cfg, prompt)
	default:
		return "", fmt.Errorf("no AI provider configured")
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.RawJSON) != "" {
		return res.RawJSON, nil
	}
	return res.Summary, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
