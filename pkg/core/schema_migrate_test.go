package core

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTableHasColumn(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, label TEXT)`); err != nil {
		t.Fatal(err)
	}
	has, err := tableHasColumn(db, "sample", "label")
	if err != nil || !has {
		t.Fatalf("label: has=%v err=%v", has, err)
	}
	has, err = tableHasColumn(db, "sample", "missing")
	if err != nil || has {
		t.Fatalf("missing: has=%v err=%v", has, err)
	}
}

func TestAddColumnIfNotExists_idempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE legacy (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	prev := DB
	DB = db
	defer func() { DB = prev }()

	alter := `ALTER TABLE legacy ADD COLUMN extra TEXT DEFAULT ''`
	addColumnIfNotExists("legacy", "extra", alter)
	addColumnIfNotExists("legacy", "extra", alter)

	has, err := tableHasColumn(db, "legacy", "extra")
	if err != nil || !has {
		t.Fatalf("extra column: has=%v err=%v", has, err)
	}
}

func TestIsDuplicateColumnErr(t *testing.T) {
	if isDuplicateColumnErr(nil) {
		t.Fatal("nil should not be duplicate column")
	}
	if isDuplicateColumnErr(sql.ErrConnDone) {
		t.Fatal("unrelated error should not match")
	}
	err := &fakeSQLiteErr{msg: "SQL logic error: duplicate column name: foo"}
	if !isDuplicateColumnErr(err) {
		t.Fatal("expected duplicate column detection")
	}
}

type fakeSQLiteErr struct{ msg string }

func (e *fakeSQLiteErr) Error() string { return e.msg }
