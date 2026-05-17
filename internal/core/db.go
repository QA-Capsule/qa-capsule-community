package core

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite" // SQLite driver
)

// Global database connection pool
var DB *sql.DB

// InitDB initializes the SQLite database, forcing it into the ./data/ directory
func InitDB(ignoredPath string) {
	// 1. Force the database path to ./data/qacapsule.db
	const dbFolder = "./data"
	const dbName = "qacapsule.db"
	dbPath := filepath.Join(dbFolder, dbName)

	// 2. Ensure the directory for the database exists
	if _, err := os.Stat(dbFolder); os.IsNotExist(err) {
		log.Printf("[INFO] Creating storage directory: %s", dbFolder)
		err = os.MkdirAll(dbFolder, 0755)
		if err != nil {
			log.Fatalf("[FATAL] Failed to create database directory: %v", err)
		}
	}

	// 3. Print the exact absolute path to the terminal for verification
	absPath, _ := filepath.Abs(dbPath)
	log.Printf("[INFO] Initializing SQLite database at: %s", absPath)

	var err error
	// 4. Open the SQLite database
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open SQLite database: %v", err)
	}

	// ====================================================================
	// SRE CRITICAL FIX: SÉRIALISATION STRICTE ET PROTECTION ANTI-CONTENTION
	// ====================================================================
	// SQLite ne gère pas nativement les accès concurrents multi-connexions en écriture.
	// Limiter à 1 connexion active élimine à 100% l'isolement périmé des lectures/écritures.
	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(time.Hour)

	_, err = DB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Printf("[WARNING] Could not enable WAL mode: %v", err)
	}
	DB.Exec("PRAGMA busy_timeout=5000;")

	// 5. Force a connection to ensure the physical file is actually created right now
	if err = DB.Ping(); err != nil {
		log.Fatalf("[FATAL] Failed to ping database: %v", err)
	}

	// 1. Create global users table (IAM)
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		fullname TEXT,
		password_hash TEXT NOT NULL,
		role TEXT DEFAULT 'viewer',
		is_active INTEGER DEFAULT 1,
		require_password_change INTEGER DEFAULT 1
	);`
	_, err = DB.Exec(createUsersTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create users table: %v", err)
	}

	// 2. Create hierarchical teams table (Organizations)
	createTeamsTable := `
	CREATE TABLE IF NOT EXISTS teams (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		parent_id INTEGER,
		FOREIGN KEY(parent_id) REFERENCES teams(id) ON DELETE CASCADE
	);`
	_, err = DB.Exec(createTeamsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create teams table: %v", err)
	}

	// 3. Create mapping table for Granular RBAC (Users <-> Teams)
	createUserTeamsTable := `
	CREATE TABLE IF NOT EXISTS user_teams (
		user_id INTEGER,
		team_id INTEGER,
		team_role TEXT DEFAULT 'team_viewer',
		PRIMARY KEY (user_id, team_id),
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
	);`
	_, err = DB.Exec(createUserTeamsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create user_teams table: %v", err)
	}

	// 4. Create projects table (CI/CD Gateways mapped to Teams)
	createProjectsTable := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		team_id INTEGER NOT NULL,
		ci_system TEXT,
		repo_path TEXT,
		api_key TEXT UNIQUE NOT NULL,
		slack_channel TEXT,
		jira_project_key TEXT,
		teams_webhook TEXT,
		FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
	);`
	_, err = DB.Exec(createProjectsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create projects table: %v", err)
	}

	// 5. Create incidents table
	createIncidentsTable := `
	CREATE TABLE IF NOT EXISTS incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_name TEXT,
		name TEXT,
		status TEXT,
		error_message TEXT,
		console_logs TEXT,
		error_logs TEXT, 
		fingerprint TEXT, 
		is_resolved INTEGER DEFAULT 0,
		resolved_by TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME
	);`
	_, err = DB.Exec(createIncidentsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create incidents table: %v", err)
	}

	// 6. Create index for Smart Correlation
	createIndex := `CREATE INDEX IF NOT EXISTS idx_incidents_fingerprint ON incidents(fingerprint, is_resolved);`
	_, err = DB.Exec(createIndex)
	if err != nil {
		log.Printf("[WARNING] Failed to create index for fingerprint: %v", err)
	}

	// Create FinOps Settings Table
	createFinOpsTable := `
	CREATE TABLE IF NOT EXISTS finops_settings (
		id INTEGER PRIMARY KEY,
		dev_hourly_rate REAL,
		ci_minute_cost REAL,
		avg_pipeline_duration REAL,
		avg_investigation_time REAL
	);`
	_, err = DB.Exec(createFinOpsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create finops_settings table: %v", err)
	}

	// Create Enterprise License Config Table
	createEnterpriseTable := `
	CREATE TABLE IF NOT EXISTS enterprise_config (
		id INTEGER PRIMARY KEY,
		license_key TEXT
	);`
	_, err = DB.Exec(createEnterpriseTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create enterprise_config table: %v", err)
	}

	log.Println("[INFO] Database initialized successfully with Smart Correlation schema.")
	seedInitialData()
}

func seedInitialData() {
	var teamCount int
	DB.QueryRow("SELECT COUNT(*) FROM teams").Scan(&teamCount)
	if teamCount == 0 {
		_, err := DB.Exec("INSERT INTO teams (id, name, parent_id) VALUES (1, 'Root Organization', NULL)")
		if err != nil {
			log.Printf("[WARNING] Could not seed root team: %v", err)
		} else {
			log.Println("[INFO] Default 'Root Organization' created.")
		}
	}

	var userCount int
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		_, err := DB.Exec(`INSERT INTO users (username, fullname, password_hash, role, is_active, require_password_change) 
			VALUES ('admin', 'System Administrator', ?, 'admin', 1, 1)`, string(hashedPassword))
		if err != nil {
			log.Printf("[WARNING] Could not seed admin user: %v", err)
		} else {
			log.Println("[INFO] Default 'admin' user created (password: admin). Please change upon first login.")
		}
	}

	var finopsCount int
	DB.QueryRow("SELECT COUNT(*) FROM finops_settings").Scan(&finopsCount)
	if finopsCount == 0 {
		_, err := DB.Exec(`INSERT INTO finops_settings (id, dev_hourly_rate, ci_minute_cost, avg_pipeline_duration, avg_investigation_time) 
			VALUES (1, 50.0, 0.008, 15.0, 30.0)`)
		if err != nil {
			log.Printf("[WARNING] Could not seed default FinOps settings: %v", err)
		} else {
			log.Println("[INFO] Default FinOps baseline metrics initialized.")
		}
	}
}

func GetProjectByAPIKey(apiKey string) (ProjectConfig, error) {
	var p ProjectConfig
	err := DB.QueryRow(`
		SELECT id, name, ci_system, api_key, slack_channel, jira_project_key, teams_webhook 
		FROM projects WHERE api_key = ?`, apiKey).Scan(
		&p.ID, &p.Name, &p.CISystem, &p.APIKey, &p.Routing.SlackChannel, &p.Routing.JiraProjectKey, &p.Routing.TeamsWebhook)
	return p, err
}
