package healing

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupHealingTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE healing_patch_submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		incident_id INTEGER NOT NULL,
		repo TEXT NOT NULL,
		file_path TEXT NOT NULL,
		code_sha256 TEXT NOT NULL,
		code_size INTEGER NOT NULL DEFAULT 0,
		explanation TEXT DEFAULT '',
		agent_source TEXT DEFAULT 'mcp_agent',
		status TEXT NOT NULL DEFAULT 'accepted',
		pr_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_name TEXT,
		name TEXT,
		status TEXT,
		error_message TEXT,
		console_logs TEXT,
		error_logs TEXT,
		fingerprint TEXT,
		pipeline_run_id TEXT,
		is_resolved INTEGER DEFAULT 0,
		resolved_by TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRegisterPatchSubmission(t *testing.T) {
	db := setupHealingTestDB(t)
	defer db.Close()
	svc := NewService(db)

	sub, err := svc.RegisterPatchSubmission(7, "acme/repo", "tests/e2e/checkout.spec.ts", "console.log('fix')", "reason", "cursor_mcp")
	if err != nil {
		t.Fatal(err)
	}
	if sub.ID == 0 || sub.CodeSHA256 == "" || sub.CodeSize == 0 {
		t.Fatalf("unexpected submission payload: %+v", sub)
	}
}

func TestResolveIncident(t *testing.T) {
	db := setupHealingTestDB(t)
	defer db.Close()
	svc := NewService(db)
	_, err := db.Exec(`INSERT INTO incidents (id, project_name, name, status, is_resolved) VALUES (1, 'p', 't', 'CRITICAL', 0)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveIncident(1, "agent"); err != nil {
		t.Fatal(err)
	}
	var resolved int
	var status, by string
	err = db.QueryRow(`SELECT is_resolved, status, COALESCE(resolved_by,'') FROM incidents WHERE id = 1`).Scan(&resolved, &status, &by)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != 1 || status != "resolved" || by != "agent" {
		t.Fatalf("unexpected resolved row: resolved=%d status=%s by=%s", resolved, status, by)
	}
}
