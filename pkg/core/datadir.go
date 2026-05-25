package core

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

var resolvedDataDir string

// DataDir returns the writable directory used for qacapsule.db and artifacts.
// Order: QACAPSULE_DATA_DIR → ./data → ~/.qa-capsule/data (fallback when ./data is not writable).
func DataDir() string {
	if resolvedDataDir == "" {
		resolvedDataDir = resolveDataDir()
	}
	return resolvedDataDir
}

// DBFilePath is the SQLite database file path under DataDir().
func DBFilePath() string {
	return filepath.Join(DataDir(), "qacapsule.db")
}

func resolveDataDir() string {
	if v := strings.TrimSpace(os.Getenv("QACAPSULE_DATA_DIR")); v != "" {
		return mustPrepareDataDir(v, "QACAPSULE_DATA_DIR")
	}
	if dir, ok := tryPrepareDataDir("./data"); ok {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[FATAL] ./data is not writable and HOME is unavailable; set QACAPSULE_DATA_DIR to a writable path")
	}
	fallback := filepath.Join(home, ".qa-capsule", "data")
	absFallback, _ := filepath.Abs(fallback)
	log.Printf("[WARNING] ./data is not writable (common after Docker created root-owned files). Using %s instead. To use ./data: chmod -R u+w data  OR  rm -f data/qacapsule.db*  OR  export QACAPSULE_DATA_DIR=./data after fixing permissions.", absFallback)
	return mustPrepareDataDir(fallback, "fallback data directory")
}

func tryPrepareDataDir(dir string) (string, bool) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false
	}
	if !isDirWritable(dir) {
		return "", false
	}
	dbPath := filepath.Join(dir, "qacapsule.db")
	if _, err := os.Stat(dbPath); err == nil && !isFileWritable(dbPath) {
		return "", false
	}
	return dir, true
}

func mustPrepareDataDir(dir, label string) string {
	if d, ok := tryPrepareDataDir(dir); ok {
		return d
	}
	abs, _ := filepath.Abs(dir)
	log.Fatalf("[FATAL] Data directory not writable (%s): %s — mkdir -p and chmod u+w, or choose another QACAPSULE_DATA_DIR", label, abs)
	return ""
}

func isDirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".write-check-*")
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return true
}

func isFileWritable(path string) bool {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
