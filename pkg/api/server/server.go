package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// jwtKey is the secret used to sign JWT tokens
var jwtKey = []byte("sre-super-secret-jwt-key")

// Claims defines the JWT payload structure
type Claims struct {
	Username              string `json:"username"`
	Role                  string `json:"role"`
	RequirePasswordChange bool   `json:"require_password_change"`
	jwt.RegisteredClaims
}

// upgrader configures WebSocket connection upgrades
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// generateRandomPassword creates a secure temporary password
func generateRandomPassword() string {
	b := make([]byte, 12)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:12]
}

// sendEmail handles outbound SMTP communication for user provisioning and resets
func sendEmail(config *core.Config, to string, subject string, htmlBody string) {
	if config.SMTP.Host == "" {
		log.Println("[SMTP ERROR] SMTP server is not configured.")
		return
	}
	auth := smtp.PlainAuth("", config.SMTP.User, config.SMTP.Password, config.SMTP.Host)

	msg := []byte("From: " + config.SMTP.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		htmlBody + "\r\n")

	addr := fmt.Sprintf("%s:%d", config.SMTP.Host, config.SMTP.Port)
	err := smtp.SendMail(addr, auth, config.SMTP.From, []string{to}, msg)
	if err != nil {
		log.Printf("[SMTP ERROR] Failed to send to %s: %v\n", to, err)
	} else {
		log.Printf("[SMTP SUCCESS] HTML Email delivered to %s\n", to)
	}
}

// jwtAuthMiddleware intercepts requests to verify identity and RBAC permissions
func jwtAuthMiddleware(config *core.Config, requireAdmin bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !config.Security.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if claims.RequirePasswordChange && r.URL.Path != "/api/users/change-password" {
			http.Error(w, "Password change required", http.StatusForbidden)
			return
		}

		if requireAdmin && claims.Role != "admin" {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

var IsEnterpriseActive = func() bool {
	return false
}

// SSOLoginHandler est une variable globale modifiable par l'Enterprise Edition.
// Par défaut, elle bloque la connexion SSO.
var SSOLoginHandler = func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "QA Capsule PRO License Required for SSO",
	})
}

// enterpriseMiddleware blocks access to premium features if no valid license is found
func enterpriseMiddleware(config *core.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// On utilise la variable exportée (le Hook)
		if !IsEnterpriseActive() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			json.NewEncoder(w).Encode(map[string]string{"error": "Enterprise license required"})
			return
		}
		next.ServeHTTP(w, r)
	}
}

// Start boots up the HTTP server and registers all routes
func Start(initialConfig core.Config) {
	config := &initialConfig

	// Initialize FinOps settings table
	core.DB.Exec(`CREATE TABLE IF NOT EXISTS finops_settings (
		id INTEGER PRIMARY KEY,
		dev_hourly_rate REAL,
		ci_minute_cost REAL,
		avg_pipeline_duration REAL,
		avg_investigation_time REAL
	)`)
	core.DB.Exec(`INSERT INTO finops_settings (id, dev_hourly_rate, ci_minute_cost, avg_pipeline_duration, avg_investigation_time) 
				  SELECT 1, 50.0, 0.008, 15.0, 30.0 
				  WHERE NOT EXISTS (SELECT 1 FROM finops_settings WHERE id = 1)`)

	// Create Enterprise Configuration Table
	core.DB.Exec(`CREATE TABLE IF NOT EXISTS enterprise_config (
		id INTEGER PRIMARY KEY,
		license_key TEXT
	)`)
	core.DB.Exec(`INSERT INTO enterprise_config (id, license_key) SELECT 1, '' WHERE NOT EXISTS (SELECT 1 FROM enterprise_config WHERE id = 1)`)

	// Register isolated route handlers
	registerAuthRoutes(config)
	registerTeamRoutes(config)
	registerProjectRoutes(config)
	registerWebhookRoutes(config)
	registerIncidentRoutes(config)
	registerSystemRoutes(config)

	// Serve static frontend files
	http.Handle("/", http.FileServer(http.Dir("./web")))

	log.Printf("[SERVER] Started on port %s", config.Server.Port)
	log.Fatal(http.ListenAndServe(":"+config.Server.Port, nil))
}
