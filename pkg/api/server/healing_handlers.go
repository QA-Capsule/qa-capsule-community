package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func handleIncidentHealingAction(w http.ResponseWriter, r *http.Request, idStr, action string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid incident id", http.StatusBadRequest)
		return
	}
	if core.AIService == nil {
		writeJSONError(w, "AI service not initialized", http.StatusServiceUnavailable)
		return
	}
	switch action {
	case "propose":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FileContent string `json:"file_content"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		code, explanation, err := core.AIService.ProposeFixFromIncidentID(r.Context(), id, req.FileContent)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"code": code, "explanation": explanation})
	case "pr":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Repo     string `json:"repo"`
			FilePath string `json:"file_path"`
			Code     string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "Invalid body", http.StatusBadRequest)
			return
		}
		prURL, err := core.CreateRemediationPR(req.Repo, req.FilePath, req.Code)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"pr_url": prURL})
	default:
		http.NotFound(w, r)
	}
}
