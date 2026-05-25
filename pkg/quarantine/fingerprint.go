package quarantine

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// frameworkNamePrefixes are stripped so Playwright/Robot/JUnit titles match across CI payloads.
var frameworkNamePrefixes = []string{
	"[Playwright] ", "[RobotFW] ", "[Robot] ", "[Cypress] ",
	"[JUnit] ", "[Pytest] ", "[Pipeline] ", "[PIPELINE CRASH] ",
}

// NormalizeTestName strips flaky/perf and framework reporter prefixes for stable CI identity.
func NormalizeTestName(name string) string {
	n := strings.TrimSpace(name)
	for {
		changed := false
		for _, p := range []string{"[FLAKY] ", "[PERF] "} {
			if strings.HasPrefix(n, p) {
				n = strings.TrimPrefix(n, p)
				changed = true
			}
		}
		for _, p := range frameworkNamePrefixes {
			if strings.HasPrefix(n, p) {
				n = strings.TrimPrefix(n, p)
				changed = true
			}
		}
		n = strings.TrimSpace(n)
		if !changed {
			break
		}
	}
	return n
}

// TestIdentityFingerprint hashes project + normalized test name (no error text).
func TestIdentityFingerprint(projectName, testName string) string {
	raw := fmt.Sprintf("%s|%s", strings.TrimSpace(projectName), NormalizeTestName(testName))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
}
