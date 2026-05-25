package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StorageReady verifies the artifact/report storage directory exists and is writable.
func StorageReady(localPath string) error {
	path := strings.TrimSpace(localPath)
	if path == "" {
		path = "./reports"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("storage mkdir: %w", err)
	}
	probe := filepath.Join(abs, ".readyz_probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("storage not writable: %w", err)
	}
	_ = os.Remove(probe)
	return nil
}
