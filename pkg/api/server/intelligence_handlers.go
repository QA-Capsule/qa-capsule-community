package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/ai"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"

	"github.com/golang-jwt/jwt/v5"
)

func registerIntelligenceRoutes(config *core.Config) {
	http.HandleFunc("/api/rca/insights", jwtAuthMiddleware(config, "", handleRCAInsights))
	http.HandleFunc("/api/ai/config", jwtAuthMiddleware(config, core.RoleManager, handleAIConfig))
	registerQuarantineRoutes(config)
}

func handleRCAInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims := parseClaims(r)
	if !core.CanViewRCA(claims.Role) {
		writeJSONError(w, "Access denied", http.StatusForbidden)
		return
	}
	if core.AIService == nil {
		writeJSONError(w, "AI service not initialized", http.StatusServiceUnavailable)
		return
	}
	project := r.URL.Query().Get("project")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := core.AIService.ListInsights(r.Context(), project, limit)
	if err != nil {
		writeJSONError(w, "Failed to load insights", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

func handleIncidentRCA(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/incidents/")
	if !strings.HasSuffix(path, "/rca") {
		http.NotFound(w, r)
		return
	}
	idStr := strings.TrimSuffix(path, "/rca")
	idStr = strings.Trim(idStr, "/")
	incidentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || incidentID <= 0 {
		writeJSONError(w, "Invalid incident id", http.StatusBadRequest)
		return
	}
	claims := parseClaims(r)
	if !core.CanViewRCA(claims.Role) {
		writeJSONError(w, "Access denied", http.StatusForbidden)
		return
	}
	if core.AIService == nil {
		writeJSONError(w, "AI service not initialized", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rep, err := core.AIService.GetReport(r.Context(), incidentID)
		if err != nil {
			writeJSONError(w, "Database error", http.StatusInternalServerError)
			return
		}
		if rep == nil {
			writeJSONError(w, "RCA not available yet", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rep)
	case http.MethodPost:
		if !core.CanManageQuarantine(claims.Role) {
			writeJSONError(w, "Lead+ required to trigger RCA", http.StatusForbidden)
			return
		}
		core.AIService.EnqueueForIncident(incidentID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAIConfig(w http.ResponseWriter, r *http.Request) {
	if core.AIService == nil {
		writeJSONError(w, "AI service not initialized", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := core.AIService.GetConfig(r.Context())
		if err != nil {
			writeJSONError(w, "Failed to load config", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	case http.MethodPut:
		claims := parseClaims(r)
		if !core.CanConfigureAI(claims.Role) {
			writeJSONError(w, "Manager access required", http.StatusForbidden)
			return
		}
		var cfg ai.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if cfg.APIKeyEnv == "" {
			cfg.APIKeyEnv = "OPENAI_API_KEY"
		}
		if cfg.MaxTokens == 0 {
			cfg.MaxTokens = 1024
		}
		if cfg.TimeoutSeconds == 0 {
			cfg.TimeoutSeconds = 45
		}
		if err := core.AIService.SaveConfig(r.Context(), cfg); err != nil {
			writeJSONError(w, "Failed to save config", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleQuarantineManage(w http.ResponseWriter, r *http.Request) {
	claims := parseClaims(r)
	if !core.CanViewQuarantine(claims.Role) {
		writeJSONError(w, "Access denied", http.StatusForbidden)
		return
	}
	if core.QuarantineEngine == nil {
		writeJSONError(w, "Quarantine engine not initialized", http.StatusServiceUnavailable)
		return
	}
	project := r.URL.Query().Get("project")
	if project == "" {
		writeJSONError(w, "project query required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		resp, err := core.QuarantineEngine.ListCI(r.Context(), project)
		if err != nil {
			writeJSONError(w, "Failed to list quarantine", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	case http.MethodPost:
		if !core.CanManageQuarantine(claims.Role) {
			writeJSONError(w, "Lead+ required", http.StatusForbidden)
			return
		}
		var body struct {
			TestName string `json:"test_name"`
			Project  string `json:"project"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		pn := body.Project
		if pn == "" {
			pn = project
		}
		if err := core.QuarantineEngine.ManualQuarantine(r.Context(), pn, body.TestName, claims.Username); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		if !core.CanManageQuarantine(claims.Role) {
			writeJSONError(w, "Lead+ required", http.StatusForbidden)
			return
		}
		fp := r.URL.Query().Get("fingerprint")
		if fp == "" {
			writeJSONError(w, "fingerprint required", http.StatusBadRequest)
			return
		}
		if err := core.QuarantineEngine.Lift(r.Context(), project, fp, claims.Username); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseClaims(r *http.Request) *Claims {
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &Claims{}
	_, _ = jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })
	return claims
}
