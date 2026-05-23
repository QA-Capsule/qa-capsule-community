package core

import (
	"crypto/sha256"
	"fmt"
)

// IncidentFingerprint hashes test name + error for dedup and flaky CLI checks.
func IncidentFingerprint(name, err string) string {
	raw := fmt.Sprintf("%s|%s", name, err)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
}
