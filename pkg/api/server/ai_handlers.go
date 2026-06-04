package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	aiPkg "github.com/QA-Capsule/qa-capsule-community/pkg/ai"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

// aiManagerOnly wraps a handler so that only users with the exact "manager" role
// can call it. Admin accounts are intentionally excluded — AI agent configuration
// is a manager responsibility.
func aiManagerOnly(config *core.Config, next http.HandlerFunc) http.HandlerFunc {
	return jwtAuthMiddleware(config, "", func(w http.ResponseWriter, r *http.Request) {
		claims := parseClaims(r)
		if claims.Role != core.RoleManager {
			writeJSONError(w, "AI configuration requires the manager role", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func registerAIRoutes(config *core.Config) {
	// Manager only: read and write the AI provider configuration.
	http.HandleFunc("/api/ai/config", aiManagerOnly(config, handleAIConfig))
	// Observer+: lightweight status (enabled, provider, MCP token set).
	http.HandleFunc("/api/ai/status", jwtAuthMiddleware(config, "", handleAIStatus))
	// Manager only: fire a test prompt to verify the configured LLM is reachable.
	http.HandleFunc("/api/ai/test", aiManagerOnly(config, handleAITest))
}

// handleAIConfig handles GET (load) and PUT (save) of the AI provider config.
// The API key itself is never stored in QA Capsule — only the env var name.
func handleAIConfig(w http.ResponseWriter, r *http.Request) {
	if core.AIService == nil {
		writeJSONError(w, "AI service not initialized", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := core.AIService.GetConfig(context.Background())
		if err != nil {
			writeJSONError(w, "Failed to load AI config", http.StatusInternalServerError)
			return
		}
		envKeySet := cfg.APIKeyEnv != "" && os.Getenv(cfg.APIKeyEnv) != ""
		dbKeySet := strings.TrimSpace(cfg.APIKey) != ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"provider":        string(cfg.Provider),
			"model":           cfg.Model,
			"base_url":        cfg.BaseURL,
			"api_key_env":     cfg.APIKeyEnv,
			"api_key_set":     envKeySet || dbKeySet,
			// api_key_stored lets the UI show a masked indicator when a key is saved in the DB.
			"api_key_stored":  dbKeySet,
			"max_tokens":      cfg.MaxTokens,
			"timeout_seconds": cfg.TimeoutSeconds,
			"enabled":         cfg.Enabled,
		})

	case http.MethodPut:
		var req struct {
			Provider       string `json:"provider"`
			Model          string `json:"model"`
			BaseURL        string `json:"base_url"`
			APIKeyEnv      string `json:"api_key_env"`
			APIKey         string `json:"api_key"`
			MaxTokens      int    `json:"max_tokens"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Enabled        bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.MaxTokens <= 0 {
			req.MaxTokens = 1024
		}
		if req.TimeoutSeconds <= 0 {
			req.TimeoutSeconds = 45
		}
		cfg := aiPkg.ProviderConfig{
			Provider:       aiPkg.ProviderKind(strings.TrimSpace(req.Provider)),
			Model:          strings.TrimSpace(req.Model),
			BaseURL:        strings.TrimSpace(req.BaseURL),
			APIKeyEnv:      strings.TrimSpace(req.APIKeyEnv),
			APIKey:         strings.TrimSpace(req.APIKey),
			MaxTokens:      req.MaxTokens,
			TimeoutSeconds: req.TimeoutSeconds,
			Enabled:        req.Enabled,
		}
		if cfg.Provider == "" {
			cfg.Provider = aiPkg.ProviderDisabled
		}
		if err := core.AIService.SaveConfig(context.Background(), cfg); err != nil {
			writeJSONError(w, "Failed to save AI config", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAITest fires a minimal test prompt to the configured LLM and returns latency + response.
func handleAITest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if core.AIService == nil {
		writeJSONError(w, "AI service not initialized", http.StatusServiceUnavailable)
		return
	}
	cfg, err := core.AIService.GetConfig(r.Context())
	if err != nil {
		writeJSONError(w, "Failed to load AI config", http.StatusInternalServerError)
		return
	}
	if !cfg.Enabled || cfg.Provider == aiPkg.ProviderDisabled || cfg.Provider == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "AI provider is disabled. Enable it and save configuration first.",
		})
		return
	}
	envKeyMissing := cfg.APIKeyEnv == "" || os.Getenv(cfg.APIKeyEnv) == ""
	dbKeyMissing := strings.TrimSpace(cfg.APIKey) == ""
	if envKeyMissing && dbKeyMissing && cfg.Provider != aiPkg.ProviderOllama {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "No API key found. Enter the key directly in the AI configuration form, or set the environment variable " + cfg.APIKeyEnv + " on the server.",
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	start := time.Now()
	result, callErr := aiPkg.HTTPAnalyzer{}.Analyze(ctx, cfg, aiPkg.AnalysisInput{
		ProjectName:  "qa-capsule",
		TestName:     "connection-test",
		Status:       "FAILED",
		ErrorMessage: `This is a connectivity probe. Respond with exactly this JSON: {"summary":"Connection OK","root_cause":"probe","suggested_fix":"none","confidence":1.0}`,
	})
	elapsed := time.Since(start).Milliseconds()
	w.Header().Set("Content-Type", "application/json")
	if callErr != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":         false,
			"error":      callErr.Error(),
			"provider":   string(cfg.Provider),
			"model":      cfg.Model,
			"elapsed_ms": elapsed,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":         true,
		"response":   result.Summary,
		"provider":   string(cfg.Provider),
		"model":      cfg.Model,
		"elapsed_ms": elapsed,
	})
}

// handleAIStatus returns a lightweight status for the header/dashboard (no admin required).
func handleAIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mcpTokenSet := strings.TrimSpace(os.Getenv("QACAPSULE_MCP_TOKEN")) != ""
	if core.AIService == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":       false,
			"provider":      "disabled",
			"model":         "",
			"mcp_token_set": mcpTokenSet,
		})
		return
	}
	cfg, _ := core.AIService.GetConfig(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":       cfg.Enabled,
		"provider":      string(cfg.Provider),
		"model":         cfg.Model,
		"api_key_set":   os.Getenv(cfg.APIKeyEnv) != "",
		"mcp_token_set": mcpTokenSet,
	})
}
