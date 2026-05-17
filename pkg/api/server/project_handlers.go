package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"qacapsule/internal/core"

	"github.com/golang-jwt/jwt/v5"
)

// registerProjectRoutes binds the CI/CD gateway settings endpoints
func registerProjectRoutes(config *core.Config) {

	// List projects based on user permissions
	http.HandleFunc("/api/my-projects", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		authHeader := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })

		var rows *sql.Rows
		var err error

		if claims.Role == "admin" {
			rows, err = core.DB.Query(`
				SELECT id, name, ci_system, repo_path, team_id, api_key, slack_channel, jira_project_key, teams_webhook 
				FROM projects`)
		} else {
			query := `
				SELECT p.id, p.name, p.ci_system, p.repo_path, p.team_id, p.api_key, p.slack_channel, p.jira_project_key, p.teams_webhook
				FROM projects p
				JOIN user_teams ut ON p.team_id = ut.team_id
				JOIN users u ON u.id = ut.user_id
				WHERE u.username = ?
			`
			rows, err = core.DB.Query(query, claims.Username)
		}

		if err != nil {
			log.Println("[DB ERROR] Failed to fetch projects:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var projects []map[string]interface{}
		for rows.Next() {
			var id, name, ciSystem, apiKey string
			var repoPath, slackChan, jiraKey, teamsHook sql.NullString
			var teamId sql.NullInt64

			err := rows.Scan(&id, &name, &ciSystem, &repoPath, &teamId, &apiKey, &slackChan, &jiraKey, &teamsHook)
			if err != nil {
				log.Println("[SCAN ERROR] Projects:", err)
				continue
			}

			projects = append(projects, map[string]interface{}{
				"id":        id,
				"name":      name,
				"ci_system": ciSystem,
				"repo_path": repoPath.String,
				"team_id":   teamId.Int64,
				"api_key":   apiKey,
				"routing": map[string]string{
					"slack_channel":    slackChan.String,
					"jira_project_key": jiraKey.String,
					"teams_webhook":    teamsHook.String,
				},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}))

	// Manage Project Configuration (CRUD)
	http.HandleFunc("/api/config/projects", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var newProject struct {
				core.ProjectConfig
				RepoPath string `json:"repo_path"`
				TeamID   int    `json:"team_id"`
			}
			json.NewDecoder(r.Body).Decode(&newProject)

			if newProject.TeamID == 0 {
				http.Error(w, "Team ID is required", http.StatusBadRequest)
				return
			}

			_, err := core.DB.Exec(`INSERT INTO projects (id, name, team_id, ci_system, repo_path, api_key, slack_channel, jira_project_key, teams_webhook) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				newProject.ID, newProject.Name, newProject.TeamID, newProject.CISystem, newProject.RepoPath, newProject.APIKey, newProject.Routing.SlackChannel, newProject.Routing.JiraProjectKey, newProject.Routing.TeamsWebhook)

			if err != nil {
				log.Println("[DB ERROR] Failed to provision project:", err)
				http.Error(w, "Database error.", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)

		} else if r.Method == http.MethodPut {
			var updateProject struct {
				core.ProjectConfig
				RepoPath string `json:"repo_path"`
				TeamID   int    `json:"team_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&updateProject); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			_, err := core.DB.Exec(`UPDATE projects SET name = ?, team_id = ?, ci_system = ?, repo_path = ?, slack_channel = ?, jira_project_key = ?, teams_webhook = ? WHERE id = ?`,
				updateProject.Name, updateProject.TeamID, updateProject.CISystem, updateProject.RepoPath, updateProject.Routing.SlackChannel, updateProject.Routing.JiraProjectKey, updateProject.Routing.TeamsWebhook, updateProject.ID)

			if err != nil {
				log.Println("[DB ERROR] Failed to update project:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)

		} else if r.Method == http.MethodDelete {
			projectID := r.URL.Query().Get("id")
			_, err := core.DB.Exec("DELETE FROM projects WHERE id = ?", projectID)
			if err != nil {
				http.Error(w, "Failed to delete project", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
}