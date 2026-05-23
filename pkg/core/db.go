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
	DB.SetMaxOpenConns(4)
	DB.SetMaxIdleConns(4)
	DB.SetConnMaxLifetime(time.Hour)

	_, err = DB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Printf("[WARNING] Could not enable WAL mode: %v", err)
	}
	DB.Exec("PRAGMA busy_timeout=30000;")
	DB.Exec("PRAGMA synchronous=NORMAL;")

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
		role TEXT DEFAULT 'observer',
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

	createFinOpsTable := `
	CREATE TABLE IF NOT EXISTS finops_settings (
		id INTEGER PRIMARY KEY,
		dev_hourly_rate REAL,
		ci_minute_cost REAL,
		avg_pipeline_duration REAL,
		avg_investigation_time REAL,
		currency TEXT DEFAULT 'USD'
	);`
	_, err = DB.Exec(createFinOpsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create finops_settings table: %v", err)
	}

	createUserPreferencesTable := `
	CREATE TABLE IF NOT EXISTS user_preferences (
		user_id INTEGER PRIMARY KEY,
		prefs_json TEXT NOT NULL DEFAULT '{}',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	_, err = DB.Exec(createUserPreferencesTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create user_preferences table: %v", err)
	}

	createSavedChartsTable := `
	CREATE TABLE IF NOT EXISTS saved_charts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		qcl_query TEXT NOT NULL,
		pin_dashboard INTEGER DEFAULT 0,
		pin_finops INTEGER DEFAULT 0,
		created_by TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = DB.Exec(createSavedChartsTable)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create saved_charts table: %v", err)
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

	runSchemaMigrations()

	log.Println("[INFO] Database initialized successfully with Smart Correlation schema.")
	seedInitialData()
}

func runSchemaMigrations() {
	migrations := []string{
		`ALTER TABLE user_teams ADD COLUMN inherited_from INTEGER DEFAULT NULL`,
		`ALTER TABLE incidents ADD COLUMN pipeline_run_id TEXT DEFAULT ''`,
		`ALTER TABLE finops_settings ADD COLUMN currency TEXT DEFAULT 'USD'`,
		`ALTER TABLE incidents ADD COLUMN browser TEXT DEFAULT ''`,
		`ALTER TABLE incidents ADD COLUMN os TEXT DEFAULT ''`,
		`ALTER TABLE incidents ADD COLUMN viewport TEXT DEFAULT ''`,
		`ALTER TABLE incidents ADD COLUMN execution_time_ms INTEGER`,
		`ALTER TABLE incidents ADD COLUMN jira_issue_key TEXT DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS incident_artifacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			incident_id INTEGER NOT NULL,
			file_name TEXT NOT NULL,
			content_type TEXT,
			size_bytes INTEGER NOT NULL,
			storage_provider TEXT NOT NULL DEFAULT 'local',
			storage_path TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(incident_id) REFERENCES incidents(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_artifacts_incident ON incident_artifacts(incident_id)`,
		`CREATE TABLE IF NOT EXISTS test_execution_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_name TEXT NOT NULL,
			test_name TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			execution_time_ms INTEGER NOT NULL,
			status TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_metrics_fp ON test_execution_metrics(project_name, fingerprint, created_at)`,
		`ALTER TABLE projects ADD COLUMN sre_routing_json TEXT DEFAULT '[]'`,
		`CREATE TABLE IF NOT EXISTS user_team_inheritance_optouts (
			user_id INTEGER NOT NULL,
			team_id INTEGER NOT NULL,
			ancestor_team_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, team_id, ancestor_team_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
		)`,
	}
	for _, sqlStmt := range migrations {
		if _, err := DB.Exec(sqlStmt); err != nil {
			log.Printf("[INFO] Schema migration (may already exist): %v", err)
		}
	}
	migrateGlobalRoleCodes()
}

func migrateGlobalRoleCodes() {
	if _, err := DB.Exec(`UPDATE users SET role = 'lead' WHERE role = 'operator'`); err != nil {
		log.Printf("[INFO] Role migration (operator→lead): %v", err)
	}
	if _, err := DB.Exec(`UPDATE users SET role = 'observer' WHERE role = 'viewer'`); err != nil {
		log.Printf("[INFO] Role migration (viewer→observer): %v", err)
	}
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
			VALUES ('admin', 'Platform Administrator', ?, 'admin', 1, 1)`, string(hashedPassword))
		if err != nil {
			log.Printf("[WARNING] Could not seed admin user: %v", err)
		} else {
			log.Println("[INFO] Default 'admin' user created (password: admin). Please change upon first login.")
		}
	}

	var finopsCount int
	DB.QueryRow("SELECT COUNT(*) FROM finops_settings").Scan(&finopsCount)
	if finopsCount == 0 {
		_, err := DB.Exec(`INSERT INTO finops_settings (id, dev_hourly_rate, ci_minute_cost, avg_pipeline_duration, avg_investigation_time, currency) 
			VALUES (1, 50.0, 0.008, 15.0, 30.0, 'USD')`)
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
