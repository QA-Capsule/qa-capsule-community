package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"

	"qacapsule/internal/core"

	"github.com/golang-jwt/jwt/v5"
)

// registerIncidentRoutes binds endpoints for tracking alerts and calculating FinOps
func registerIncidentRoutes(config *core.Config) {

	// ==========================================
	// INCIDENTS DASHBOARD API
	// ==========================================
	http.HandleFunc("/api/incidents", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })

		if r.Method == http.MethodGet {
			projectFilter := r.URL.Query().Get("project")

			var query string
			var args []interface{}

			// Tri chronologique STRICT (created_at DESC LIMIT 200) pour garantir la séparation des pipelines
			if claims.Role == "admin" {
				if projectFilter != "" && projectFilter != "all" {
					query = "SELECT id, project_name, name, status, error_message, console_logs, error_logs, is_resolved, resolved_by, created_at, resolved_at FROM incidents WHERE project_name = ? ORDER BY created_at DESC LIMIT 200"
					args = []interface{}{projectFilter}
				} else {
					query = "SELECT id, project_name, name, status, error_message, console_logs, error_logs, is_resolved, resolved_by, created_at, resolved_at FROM incidents ORDER BY created_at DESC LIMIT 200"
				}
			} else {
				if projectFilter != "" && projectFilter != "all" {
					query = `
						SELECT DISTINCT i.id, i.project_name, i.name, i.status, i.error_message, i.console_logs, i.error_logs, i.is_resolved, i.resolved_by, i.created_at, i.resolved_at 
						FROM incidents i
						JOIN projects p ON i.project_name = p.name
						JOIN user_teams ut ON p.team_id = ut.team_id
						JOIN users u ON u.id = ut.user_id
						WHERE u.username = ? AND i.project_name = ?
						ORDER BY i.created_at DESC LIMIT 200
					`
					args = []interface{}{claims.Username, projectFilter}
				} else {
					query = `
						SELECT DISTINCT i.id, i.project_name, i.name, i.status, i.error_message, i.console_logs, i.error_logs, i.is_resolved, i.resolved_by, i.created_at, i.resolved_at 
						FROM incidents i
						JOIN projects p ON i.project_name = p.name
						JOIN user_teams ut ON p.team_id = ut.team_id
						JOIN users u ON u.id = ut.user_id
						WHERE u.username = ?
						ORDER BY i.created_at DESC LIMIT 200
					`
					args = []interface{}{claims.Username}
				}
			}

			rows, err := core.DB.Query(query, args...)
			if err != nil {
				log.Println("[DB ERROR] Error fetching incidents:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var incidents []map[string]interface{}
			for rows.Next() {
				var id, isResolved int
				var pName, name, status, errMsg, cLogs, eLogs string
				var resolvedBy, createdAt, resolvedAt sql.NullString

				rows.Scan(&id, &pName, &name, &status, &errMsg, &cLogs, &eLogs, &isResolved, &resolvedBy, &createdAt, &resolvedAt)

				incidents = append(incidents, map[string]interface{}{
					"id": id, "project_name": pName, "name": name, "status": status,
					"error_message": errMsg, "console_logs": cLogs, "error_logs": eLogs, "is_resolved": isResolved == 1,
					"resolved_by": resolvedBy.String, "created_at": createdAt.String, "resolved_at": resolvedAt.String,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(incidents)

		} else if r.Method == http.MethodPut {
			if claims.Role == "viewer" {
				http.Error(w, "Viewers are not allowed to resolve incidents", http.StatusForbidden)
				return
			}

			// SUPPORT POUR LES MISES A JOUR DE MASSE (BULK UPDATES)
			var req struct {
				ID  int   `json:"id"`
				IDs []int `json:"ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			idsToResolve := req.IDs
			if req.ID != 0 {
				idsToResolve = append(idsToResolve, req.ID)
			}

			if len(idsToResolve) > 0 {
				placeholders := make([]string, len(idsToResolve))
				args := make([]interface{}, len(idsToResolve)+1)
				args[0] = claims.Username
				for i, id := range idsToResolve {
					placeholders[i] = "?"
					args[i+1] = id
				}

				query := "UPDATE incidents SET is_resolved = 1, resolved_by = ?, resolved_at = CURRENT_TIMESTAMP WHERE id IN (" + strings.Join(placeholders, ",") + ")"
				_, err := core.DB.Exec(query, args...)
				if err != nil {
					log.Println("[DB ERROR] Bulk Resolve failed:", err)
					http.Error(w, "Failed to resolve incidents", http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusOK)

		} else if r.Method == http.MethodDelete {
			if claims.Role != "admin" {
				http.Error(w, "Only administrators can delete incident logs", http.StatusForbidden)
				return
			}

			// SUPPORT POUR LA SUPPRESSION DE MASSE (BULK DELETION)
			incidentID := r.URL.Query().Get("id")
			incidentIDs := r.URL.Query().Get("ids")

			if incidentIDs != "" {
				idList := strings.Split(incidentIDs, ",")
				placeholders := make([]string, len(idList))
				args := make([]interface{}, len(idList))
				for i, id := range idList {
					placeholders[i] = "?"
					args[i] = id
				}
				query := "DELETE FROM incidents WHERE id IN (" + strings.Join(placeholders, ",") + ")"
				_, err := core.DB.Exec(query, args...)
				if err != nil {
					log.Println("[DB ERROR] Failed to delete incidents:", err)
					http.Error(w, "Failed to delete incidents", http.StatusInternalServerError)
					return
				}
			} else if incidentID != "" {
				_, err := core.DB.Exec("DELETE FROM incidents WHERE id = ?", incidentID)
				if err != nil {
					log.Println("[DB ERROR] Failed to delete incident:", err)
					http.Error(w, "Failed to delete incident", http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
		}
	}))

	// ==========================================
	// WEEKLY REPORT API
	// ==========================================
	http.HandleFunc("/api/reports/weekly", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		projectFilter := r.URL.Query().Get("project")
		var query string
		var args []interface{}

		// Filtrage dynamique selon le choix de l'utilisateur dans l'interface
		if projectFilter != "" && projectFilter != "all" {
			query = `
				SELECT 
					project_name, 
					COUNT(*) as total_failures, 
					SUM(case when is_resolved = 1 then 1 else 0 end) as total_resolved,
					SUM(case when name LIKE '[FLAKY]%' then 1 else 0 end) as flaky_count
				FROM incidents 
				WHERE created_at >= datetime('now', '-7 days') AND project_name = ?
				GROUP BY project_name
				ORDER BY total_failures DESC
			`
			args = []interface{}{projectFilter}
		} else {
			query = `
				SELECT 
					project_name, 
					COUNT(*) as total_failures, 
					SUM(case when is_resolved = 1 then 1 else 0 end) as total_resolved,
					SUM(case when name LIKE '[FLAKY]%' then 1 else 0 end) as flaky_count
				FROM incidents 
				WHERE created_at >= datetime('now', '-7 days')
				GROUP BY project_name
				ORDER BY total_failures DESC
			`
		}

		rows, err := core.DB.Query(query, args...)
		if err != nil {
			http.Error(w, "Failed to generate report", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var report []map[string]interface{}
		for rows.Next() {
			var projName string
			var total, resolved, flaky int
			rows.Scan(&projName, &total, &resolved, &flaky)

			healthScore := 100
			if total > 0 {
				healthScore = (resolved * 100) / total
			}

			report = append(report, map[string]interface{}{
				"pipeline":        projName,
				"total_alerts":    total,
				"resolved_alerts": resolved,
				"flaky_tests":     flaky,
				"health_score":    healthScore,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}))

	// ==========================================
	// ANALYTICS & FINOPS METRICS DASHBOARD
	// ==========================================
	http.HandleFunc("/api/metrics", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		var total, resolved, flaky int
		core.DB.QueryRow("SELECT COUNT(*) FROM incidents").Scan(&total)
		core.DB.QueryRow("SELECT COUNT(*) FROM incidents WHERE is_resolved = 1").Scan(&resolved)
		core.DB.QueryRow("SELECT COUNT(*) FROM incidents WHERE name LIKE '[FLAKY]%'").Scan(&flaky)

		var mttrMinutes sql.NullFloat64

		core.DB.QueryRow(`
			SELECT AVG((julianday(resolved_at) - julianday(created_at)) * 24 * 60) 
			FROM incidents 
			WHERE is_resolved = 1 
			  AND resolved_at IS NOT NULL 
			  AND created_at IS NOT NULL
			  AND ((julianday(resolved_at) - julianday(created_at)) * 24 * 60) >= 1.0
		`).Scan(&mttrMinutes)

		mttrValue := 0.0
		if mttrMinutes.Valid {
			mttrValue = mttrMinutes.Float64
		}

		mttrDisplay := int(math.Round(mttrValue))
		if mttrValue == 0 && resolved > 0 {
			mttrDisplay = 1
		}

		var devRate, ciCost, avgDuration, avgInvestigation float64
		core.DB.QueryRow("SELECT dev_hourly_rate, ci_minute_cost, avg_pipeline_duration, avg_investigation_time FROM finops_settings WHERE id = 1").Scan(&devRate, &ciCost, &avgDuration, &avgInvestigation)

		costPerInvestigation := (devRate / 60.0) * avgInvestigation
		totalMinutesLost := float64(total) * avgDuration
		flakyMinutesLost := float64(flaky) * avgDuration

		totalFinancialImpact := (totalMinutesLost * ciCost) + (float64(total) * costPerInvestigation)
		flakyFinancialImpact := (flakyMinutesLost * ciCost) + (float64(flaky) * costPerInvestigation)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_incidents":    total,
			"resolved_incidents": resolved,
			"flaky_tests":        flaky,
			"stable_failures":    total - flaky,
			"mttr_minutes":       mttrDisplay,
			"sre_impact": map[string]interface{}{
				"ci_minutes_lost":      int(totalMinutesLost),
				"flaky_minutes_lost":   int(flakyMinutesLost),
				"estimated_cost_usd":   int(totalFinancialImpact),
				"flaky_waste_cost_usd": int(flakyFinancialImpact),
			},
		})
	}))

	// ==========================================
	// FINOPS SETTINGS CONFIGURATION
	// ==========================================
	http.HandleFunc("/api/finops", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			var devRate, ciCost, avgDuration, avgInvestigation float64
			core.DB.QueryRow("SELECT dev_hourly_rate, ci_minute_cost, avg_pipeline_duration, avg_investigation_time FROM finops_settings WHERE id = 1").Scan(&devRate, &ciCost, &avgDuration, &avgInvestigation)

			json.NewEncoder(w).Encode(map[string]float64{
				"dev_hourly_rate":        devRate,
				"ci_minute_cost":         ciCost,
				"avg_pipeline_duration":  avgDuration,
				"avg_investigation_time": avgInvestigation,
			})
		} else if r.Method == http.MethodPut {
			var req map[string]float64
			json.NewDecoder(r.Body).Decode(&req)

			core.DB.Exec(`UPDATE finops_settings SET dev_hourly_rate = ?, ci_minute_cost = ?, avg_pipeline_duration = ?, avg_investigation_time = ? WHERE id = 1`,
				req["dev_hourly_rate"], req["ci_minute_cost"], req["avg_pipeline_duration"], req["avg_investigation_time"])
			w.WriteHeader(http.StatusOK)
		}
	}))
}
