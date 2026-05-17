package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"qacapsule/internal/core"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func isEnterpriseActive() bool {
	var key string
	err := core.DB.QueryRow("SELECT license_key FROM enterprise_config WHERE id = 1").Scan(&key)
	if err != nil || key == "" {
		return false
	}
	return true
}

// registerAuthRoutes binds authentication and user management endpoints
func registerAuthRoutes(config *core.Config) {

	// Enterprise License Management
	http.HandleFunc("/api/license", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			var key string
			core.DB.QueryRow("SELECT license_key FROM enterprise_config WHERE id = 1").Scan(&key)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"license_key": key,
				"is_active":   isEnterpriseActive(),
			})
		} else if r.Method == http.MethodPost {
			var req struct {
				LicenseKey string `json:"license_key"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			
			// Crée la table si elle n'existe pas
			core.DB.Exec("CREATE TABLE IF NOT EXISTS enterprise_config (id INTEGER PRIMARY KEY, license_key TEXT)")
			core.DB.Exec("INSERT OR REPLACE INTO enterprise_config (id, license_key) VALUES (1, ?)", req.LicenseKey)
			
			w.WriteHeader(http.StatusOK)
		}
	}))

	// Public endpoint to check SSO status
	http.HandleFunc("/api/sso/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"enterprise_active": isEnterpriseActive()})
	})

	// Protected SSO Login Route
	http.HandleFunc("/api/sso/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if !isEnterpriseActive() {
			w.WriteHeader(http.StatusPaymentRequired) // Code HTTP 402
			json.NewEncoder(w).Encode(map[string]string{
				"message": "QA Capsule PRO License is missing or expired.",
			})
			return
		}

		// Mock implementation for Identity Provider redirect
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Enterprise License Verified. Redirecting to Identity Provider...",
			"sso_url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=...",
		})
	})

	// Standard Login
	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)

		var hash, role string
		var requireChange int
		err := core.DB.QueryRow("SELECT password_hash, role, require_password_change FROM users WHERE username = ?", creds.Username).Scan(&hash, &role, &requireChange)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password)) != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &Claims{
			Username:              creds.Username,
			Role:                  role,
			RequirePasswordChange: requireChange == 1,
			RegisteredClaims:      jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expirationTime)},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtKey)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":                   tokenString,
			"require_password_change": requireChange == 1,
		})
	})

	// Force password change handler
	http.HandleFunc("/api/users/change-password", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			NewPassword string `json:"new_password"`
		}
		json.NewDecoder(r.Body).Decode(&payload)

		authHeader := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })

		hashed, _ := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), 14)
		core.DB.Exec("UPDATE users SET password_hash = ?, require_password_change = 0 WHERE username = ?", string(hashed), claims.Username)
		w.WriteHeader(http.StatusOK)
	}))

	// Manage Users (CRUD)
	http.HandleFunc("/api/users", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rows, err := core.DB.Query("SELECT username, fullname, role, is_active FROM users")
			if err != nil {
				log.Println("[ERROR] DB GET users:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var users []map[string]interface{}
			for rows.Next() {
				var active int
				var u, fn, ro string
				rows.Scan(&u, &fn, &ro, &active)
				users = append(users, map[string]interface{}{"username": u, "fullname": fn, "role": ro, "is_active": active == 1})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(users)

		} else if r.Method == http.MethodPost {
			var newUser struct{ Username, Fullname, Role string }
			json.NewDecoder(r.Body).Decode(&newUser)

			// Domain restriction check
			if config.Security.AllowedDomain != "" && !strings.HasSuffix(newUser.Username, config.Security.AllowedDomain) {
				http.Error(w, "Domain not allowed", http.StatusBadRequest)
				return
			}

			tempPwd := generateRandomPassword()
			hashed, _ := bcrypt.GenerateFromPassword([]byte(tempPwd), 14)

			_, err := core.DB.Exec(`INSERT INTO users (username, fullname, password_hash, role, is_active, require_password_change) VALUES (?, ?, ?, ?, 1, 1)`,
				newUser.Username, newUser.Fullname, string(hashed), newUser.Role)

			if err == nil {
				// Send temporary credentials via email
				htmlBody := fmt.Sprintf(`
				<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0d1117; color: #c9d1d9; padding: 30px; border-radius: 8px; border: 1px solid #30363d; max-width: 600px; margin: 0 auto;">
					<h2 style="color: #58a6ff;">QA Flight Recorder</h2>
					<p>Hello <strong>%s</strong>,</p>
					<p>Your SRE Control Plane identity has been successfully provisioned. Please find your temporary access credentials below:</p>
					<div style="background-color: #161b22; padding: 20px; border-radius: 6px; border: 1px solid #30363d; margin: 25px 0;">
						<p><strong>Username:</strong> <code>%s</code></p>
						<p><strong>Password:</strong> <code>%s</code></p>
					</div>
					<p style="color: #ff7b72;">⚠️ You will be required to change this temporary password upon your first login.</p>
				</div>`, newUser.Fullname, newUser.Username, tempPwd)

				go sendEmail(config, newUser.Username, "QA Flight Recorder - Your Access Credentials", htmlBody)
				w.WriteHeader(http.StatusCreated)
			} else {
				http.Error(w, "Database error", http.StatusInternalServerError)
			}
		}
	}))

	// Admin manual password reset
	http.HandleFunc("/api/users/reset-password", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		tempPwd := generateRandomPassword()
		hashed, _ := bcrypt.GenerateFromPassword([]byte(tempPwd), 14)

		_, err := core.DB.Exec("UPDATE users SET password_hash = ?, require_password_change = 1 WHERE username = ?", string(hashed), req.Username)

		if err == nil {
			htmlBody := fmt.Sprintf(`
			<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0d1117; color: #c9d1d9; padding: 30px; border-radius: 8px; border: 1px solid #30363d; max-width: 600px; margin: 0 auto;">
				<h2 style="color: #58a6ff;">QA Flight Recorder</h2>
				<p>Password Reset Requested.</p>
				<p>Your SRE Control Plane identity has been reset. Please use your new temporary passphrase:</p>
				<div style="background-color: #161b22; padding: 20px; border-radius: 6px; border: 1px solid #30363d; margin: 25px 0;">
					<p><strong>Username:</strong> <code>%s</code></p>
					<p><strong>New Password:</strong> <code>%s</code></p>
				</div>
				<p style="color: #ff7b72;">⚠️ You will be required to change this temporary password upon your next login.</p>
			</div>`, req.Username, tempPwd)

			go sendEmail(config, req.Username, "QA Flight Recorder - Password Reset", htmlBody)
			w.WriteHeader(http.StatusOK)
		} else {
			log.Println("[DB ERROR] Failed to reset password:", err)
			http.Error(w, "Failed to reset password", http.StatusInternalServerError)
		}
	}))

	// Activate/Deactivate users
	http.HandleFunc("/api/users/status", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			IsActive bool   `json:"is_active"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		activeInt := 0
		if req.IsActive {
			activeInt = 1
		}
		core.DB.Exec("UPDATE users SET is_active = ? WHERE username = ?", activeInt, req.Username)
		w.WriteHeader(http.StatusOK)
	}))

	// Delete users
	http.HandleFunc("/api/users/delete", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		core.DB.Exec("DELETE FROM users WHERE username = ?", r.URL.Query().Get("username"))
		w.WriteHeader(http.StatusOK)
	}))
}