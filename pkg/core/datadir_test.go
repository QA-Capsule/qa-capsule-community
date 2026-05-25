package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTryPrepareDataDir_Writable(t *testing.T) {
	dir := t.TempDir()
	got, ok := tryPrepareDataDir(dir)
	if !ok || got != dir {
		t.Fatalf("expected writable dir %q, got %q ok=%v", dir, got, ok)
	}
}

func TestTryPrepareDataDir_ReadonlyDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "qacapsule.db")
	if err := os.WriteFile(dbPath, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	_, ok := tryPrepareDataDir(dir)
	if ok {
		t.Fatal("expected false when db file is not writable")
	}
}

func TestResolveDataDir_ExplicitEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("QACAPSULE_DATA_DIR", dir)
	resolvedDataDir = ""
	got := DataDir()
	if got != dir {
		t.Fatalf("DataDir() = %q, want %q", got, dir)
	}
}
