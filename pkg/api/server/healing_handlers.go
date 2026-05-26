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
	if core.HealingService == nil {
		writeJSONError(w, "Healing service not initialized", http.StatusServiceUnavailable)
		return
	}

	switch action {
	case "context":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims := parseClaims(r)
		if !core.CanViewHealing(claims.Role) {
			writeJSONError(w, "Access denied", http.StatusForbidden)
			return
		}
		ctx, err := core.HealingService.BuildContext(id)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ctx)
	case "propose":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FileContent string `json:"file_content"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		prop, err := core.HealingService.ProposeFix(id, req.FileContent)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prop)
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
