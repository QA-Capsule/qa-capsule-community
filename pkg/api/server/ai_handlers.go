package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	aiPkg "github.com/QA-Capsule/qa-capsule-community/pkg/ai"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func registerAIRoutes(config *core.Config) {
	// Admin: read and write the AI provider configuration.
	http.HandleFunc("/api/ai/config", jwtAuthMiddleware(config, core.RoleAdmin, handleAIConfig))
	// Observer+: lightweight status (enabled, provider, MCP token set).
	http.HandleFunc("/api/ai/status", jwtAuthMiddleware(config, "", handleAIStatus))
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"provider":        string(cfg.Provider),
			"model":           cfg.Model,
			"base_url":        cfg.BaseURL,
			"api_key_env":     cfg.APIKeyEnv,
			// api_key_set tells the UI whether the key env var is currently set on the server.
			"api_key_set":     os.Getenv(cfg.APIKeyEnv) != "",
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
