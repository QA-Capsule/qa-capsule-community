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

// registerTeamRoutes binds organization and team management endpoints
func registerTeamRoutes(config *core.Config) {

	// CRUD for Teams (GET is public for authenticated users, Mutations are Admin only)
	http.HandleFunc("/api/teams", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })

		if r.Method == http.MethodGet {
			rows, err := core.DB.Query("SELECT id, name, parent_id FROM teams")
			if err != nil {
				log.Println("[DB ERROR] Failed to fetch teams:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var teams []map[string]interface{}
			for rows.Next() {
				var id int
				var name string
				var parentId sql.NullInt64
				rows.Scan(&id, &name, &parentId)

				team := map[string]interface{}{"id": id, "name": name, "parent_id": nil}
				if parentId.Valid {
					team["parent_id"] = parentId.Int64
				}
				teams = append(teams, team)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(teams)

		} else {
			// SECURITY: Block non-admins from mutating teams
			if claims.Role != "admin" {
				http.Error(w, "Admin access required", http.StatusForbidden)
				return
			}

			if r.Method == http.MethodPost {
				var req struct {
					Name     string `json:"name"`
					ParentID *int   `json:"parent_id"`
				}
				json.NewDecoder(r.Body).Decode(&req)

				if req.Name == "" {
					http.Error(w, "Group name required", http.StatusBadRequest)
					return
				}

				_, err := core.DB.Exec("INSERT INTO teams (name, parent_id) VALUES (?, ?)", req.Name, req.ParentID)
				if err != nil {
					http.Error(w, "Failed to create group", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)

			} else if r.Method == http.MethodPut {
				var req struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				}
				json.NewDecoder(r.Body).Decode(&req)

				if req.Name == "" || req.ID == 0 {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}

				_, err := core.DB.Exec("UPDATE teams SET name = ? WHERE id = ?", req.Name, req.ID)
				if err != nil {
					http.Error(w, "Failed to rename group", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)

			} else if r.Method == http.MethodDelete {
				teamID := r.URL.Query().Get("id")
				if teamID == "1" {
					http.Error(w, "Cannot delete Root Organization", http.StatusForbidden)
					return
				}
				_, err := core.DB.Exec("DELETE FROM teams WHERE id = ?", teamID)
				if err != nil {
					http.Error(w, "Failed to delete group", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
			}
		}
	}))

	// Associate/Dissociate users with teams (Admin only -> true)
	http.HandleFunc("/api/user-teams", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var payload struct {
				UserEmail string `json:"user_email"`
				TeamID    int    `json:"team_id"`
				Role      string `json:"role"`
			}

			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "Invalid payload", http.StatusBadRequest)
				return
			}

			var userID int
			err := core.DB.QueryRow("SELECT id FROM users WHERE username = ?", payload.UserEmail).Scan(&userID)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			_, err = core.DB.Exec(`
				INSERT INTO user_teams (user_id, team_id, team_role) 
				VALUES (?, ?, ?) 
				ON CONFLICT(user_id, team_id) DO UPDATE SET team_role = ?`,
				userID, payload.TeamID, payload.Role, payload.Role)

			if err != nil {
				log.Println("[DB ERROR] Failed to assign user to team:", err)
				http.Error(w, "Failed to assign group", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodDelete {
			userID := r.URL.Query().Get("user_id")
			teamID := r.URL.Query().Get("team_id")
			core.DB.Exec("DELETE FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))

	// Fetch teams for a specific user (Read Only - Open to all -> false)
	http.HandleFunc("/api/users/teams", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			username := r.URL.Query().Get("username")
			query := `
				SELECT t.id, t.name, ut.team_role 
				FROM teams t
				JOIN user_teams ut ON t.id = ut.team_id
				JOIN users u ON u.id = ut.user_id
				WHERE u.username = ?`

			rows, err := core.DB.Query(query, username)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var teams []map[string]interface{}
			for rows.Next() {
				var id int
				var name, teamRole string
				rows.Scan(&id, &name, &teamRole)
				teams = append(teams, map[string]interface{}{"id": id, "name": name, "role": teamRole})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(teams)
		}
	}))

	// Manage team members (GET is open, POST/DELETE are reserved for Admins)
	http.HandleFunc("/api/teams/members", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })

		if r.Method == http.MethodGet {
			teamID := r.URL.Query().Get("team_id")
			query := `
				SELECT u.username, u.fullname, u.role, ut.team_role 
				FROM users u
				JOIN user_teams ut ON u.id = ut.user_id
				WHERE ut.team_id = ?`

			rows, err := core.DB.Query(query, teamID)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var members []map[string]interface{}
			for rows.Next() {
				var u, fn, globalRole, teamRole string
				rows.Scan(&u, &fn, &globalRole, &teamRole)
				members = append(members, map[string]interface{}{
					"username":    u,
					"fullname":    fn,
					"global_role": globalRole,
					"team_role":   teamRole,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(members)

		} else {
			// SECURITY: Block non-admins from modifying members
			if claims.Role != "admin" {
				http.Error(w, "Admin access required", http.StatusForbidden)
				return
			}

			if r.Method == http.MethodPost {
				var req struct {
					Username string `json:"username"`
					TeamID   int    `json:"team_id"`
					TeamRole string `json:"team_role"`
				}
				json.NewDecoder(r.Body).Decode(&req)

				if req.TeamRole == "" {
					req.TeamRole = "team_viewer"
				}

				var userID int
				err := core.DB.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&userID)
				if err != nil {
					http.Error(w, "User not found", http.StatusNotFound)
					return
				}

				_, err = core.DB.Exec("INSERT OR REPLACE INTO user_teams (user_id, team_id, team_role) VALUES (?, ?, ?)", userID, req.TeamID, req.TeamRole)
				if err != nil {
					http.Error(w, "Failed to assign user", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)

			} else if r.Method == http.MethodDelete {
				username := r.URL.Query().Get("username")
				teamID := r.URL.Query().Get("team_id")

				var userID int
				err := core.DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
				if err != nil {
					http.Error(w, "User not found", http.StatusNotFound)
					return
				}

				_, err = core.DB.Exec("DELETE FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID)
				if err != nil {
					http.Error(w, "Failed to remove user", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
}
