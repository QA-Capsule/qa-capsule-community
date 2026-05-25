package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func callAnthropic(ctx context.Context, cfg ProviderConfig, prompt string) (AnalysisResult, error) {
	key := apiKeyFromEnv(cfg)
	if key == "" {
		return AnalysisResult{}, fmt.Errorf("missing API key env %s", cfg.APIKeyEnv)
	}
	base := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	url := base
	if !strings.Contains(url, "/v1/messages") {
		url = base + "/v1/messages"
	}
	maxTok := cfg.MaxTokens
	if maxTok <= 0 {
		maxTok = 1024
	}
	body := map[string]interface{}{
		"model":      cfg.Model,
		"max_tokens": maxTok,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	raw, err := postRawHeaders(ctx, url, map[string]string{
		"x-api-key":         key,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return AnalysisResult{}, err
	}
	var resp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || len(resp.Content) == 0 {
		return parseFreeform(string(raw)), nil
	}
	return parseFreeform(resp.Content[0].Text), nil
}

func callGemini(ctx context.Context, cfg ProviderConfig, prompt string) (AnalysisResult, error) {
	key := apiKeyFromEnv(cfg)
	if key == "" {
		return AnalysisResult{}, fmt.Errorf("missing API key env %s", cfg.APIKeyEnv)
	}
	model := cfg.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	base := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	var url string
	if base != "" && strings.Contains(base, "generateContent") {
		url = base
		if strings.Contains(url, "?") {
			url += "&key=" + key
		} else {
			url += "?key=" + key
		}
	} else {
		if base == "" {
			base = "https://generativelanguage.googleapis.com/v1beta"
		}
		url = fmt.Sprintf("%s/models/%s:generateContent?key=%s", base, model, key)
	}
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}
	raw, err := postRaw(ctx, url, "", body)
	if err != nil {
		return AnalysisResult{}, err
	}
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return parseFreeform(string(raw)), nil
	}
	return parseFreeform(resp.Candidates[0].Content.Parts[0].Text), nil
}
