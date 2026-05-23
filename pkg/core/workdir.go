package core

import (
	"log"
	"os"
	"path/filepath"
)

// EnsureProjectRoot changes the process working directory to the folder that contains config.yaml.
// Avoids an empty ./data/qacapsule.db when the server is started from another directory (common on Windows).
func EnsureProjectRoot() {
	if _, err := os.Stat("config.yaml"); err == nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(candidate); err == nil {
			if err := os.Chdir(dir); err != nil {
				log.Printf("[WARNING] Could not chdir to project root %s: %v", dir, err)
				return
			}
			log.Printf("[INFO] Working directory set to: %s", dir)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}
