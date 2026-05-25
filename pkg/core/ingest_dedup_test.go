package core

import (
	"testing"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

func TestIsDuplicateIngest_sameRunOpen_blocks(t *testing.T) {
	db := setupFlakyTestDB(t)
	defer db.Close()
	prev := DB
	DB = db
	defer func() { DB = prev }()

	project := "gw"
	run := "run-100"
	name := "[RobotFramework] Api Health > Demo Failure"
	_, err := db.Exec(`INSERT INTO incidents (project_name, name, status, pipeline_run_id, is_resolved)
		VALUES (?, ?, 'CRITICAL', ?, 0)`, project, name, run)
	if err != nil {
		t.Fatal(err)
	}
	if !isDuplicateIngest(project, run, name) {
		t.Fatal("expected duplicate for same run + open incident")
	}
}

func TestIsDuplicateIngest_newRun_allows(t *testing.T) {
	db := setupFlakyTestDB(t)
	defer db.Close()
	prev := DB
	DB = db
	defer func() { DB = prev }()

	project := "gw"
	_, err := db.Exec(`INSERT INTO incidents (project_name, name, status, pipeline_run_id, is_resolved)
		VALUES (?, '[RobotFramework] login', 'CRITICAL', 'run-1', 0)`, project)
	if err != nil {
		t.Fatal(err)
	}
	if isDuplicateIngest(project, "run-2", "login") {
		t.Fatal("new pipeline run should not be deduplicated")
	}
}

func TestIsDuplicateIngest_resolved_allows_reopen(t *testing.T) {
	db := setupFlakyTestDB(t)
	defer db.Close()
	prev := DB
	DB = db
	defer func() { DB = prev }()

	project := "gw"
	run := "run-50"
	norm := quarantine.NormalizeTestName("checkout")
	_, err := db.Exec(`INSERT INTO incidents (project_name, name, status, pipeline_run_id, is_resolved)
		VALUES (?, ?, 'CRITICAL', ?, 1)`, project, norm, run)
	if err != nil {
		t.Fatal(err)
	}
	if isDuplicateIngest(project, run, "checkout") {
		t.Fatal("resolved incident should not block a new failure on same run")
	}
}
