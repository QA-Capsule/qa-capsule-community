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
}

type StubAnalyzer struct{}

func (StubAnalyzer) Analyze(ctx context.Context, cfg ProviderConfig, in AnalysisInput) (AnalysisResult, error) {
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
		RawJSON:      `{"source":"stub"}`,
	}, nil
}

type HTTPAnalyzer struct{}

func (HTTPAnalyzer) Analyze(ctx context.Context, cfg ProviderConfig, in AnalysisInput) (AnalysisResult, error) {
	prompt := buildPrompt(in)
	switch cfg.Provider {
	case ProviderOllama:
		return callOllama(ctx, cfg, prompt)
	case ProviderOpenAI:
		return callOpenAI(ctx, cfg, prompt)
	default:
		return StubAnalyzer{}.Analyze(ctx, cfg, in)
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
	key := os.Getenv(cfg.APIKeyEnv)
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if key == "" {
		return AnalysisResult{}, fmt.Errorf("missing API key env %s", cfg.APIKeyEnv)
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.openai.com"
	}
	body := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": cfg.MaxTokens,
	}
	return postChat(ctx, base+"/v1/chat/completions", key, body)
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
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
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
	for _, kw := range []string{"locator", "selector", "getbyrole", "getbytext", "xpath", "css"} {
		if strings.Contains(lower, kw) {
			idx := strings.Index(lower, kw)
			return truncate(err[idx:], 120)
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
