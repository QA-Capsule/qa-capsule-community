package core

import (
	"database/sql"
	"testing"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"

	_ "modernc.org/sqlite"
)

func setupFlakyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestReconcileFlakyTags_secondPlaywrightFailure(t *testing.T) {
	db := setupFlakyTestDB(t)
	defer db.Close()
	prev := DB
	DB = db
	defer func() { DB = prev }()

	project := "demo-gateway"
	if hasPriorFailureForTest(project, "[Playwright] checkout") {
		t.Fatal("empty DB should have no prior failure")
	}
	_, err := db.Exec(`INSERT INTO incidents (project_name, name, status, fingerprint, pipeline_run_id)
		VALUES (?, '[Playwright] checkout', 'CRITICAL', 'fp1', 'run-1')`, project)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPriorFailureForTest(project, "[Playwright] checkout") {
		t.Fatal("second ingest should see first failure as prior")
	}

	_, err = db.Exec(`INSERT INTO incidents (project_name, name, status, fingerprint, pipeline_run_id)
		VALUES (?, '[Playwright] checkout', 'CRITICAL', 'fp2', 'run-2')`, project)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPriorFailureForTest(project, "checkout") {
		t.Fatal("second failure should see prior by normalized name")
	}
	reconcileFlakyTagsForTest(project, "[Playwright] checkout")

	var flakyCount int
	db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE project_name = ? AND name LIKE '[FLAKY]%'`, project).Scan(&flakyCount)
	if flakyCount != 2 {
		t.Fatalf("expected 2 flaky-tagged rows, got %d", flakyCount)
	}
}

func TestHasPriorFailureForTest_playwrightPrefix(t *testing.T) {
	db := setupFlakyTestDB(t)
	defer db.Close()
	prev := DB
	DB = db
	defer func() { DB = prev }()

	project := "p1"
	db.Exec(`INSERT INTO incidents (project_name, name, status) VALUES (?, '[Playwright] login', 'CRITICAL')`, project)
	if !hasPriorFailureForTest(project, "login") {
		t.Fatal("expected prior failure when matching bare test title")
	}
	if quarantine.NormalizeTestName("[Playwright] login") != quarantine.NormalizeTestName("login") {
		t.Fatal("normalize should align playwright prefix")
	}
}
