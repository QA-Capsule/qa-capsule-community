package quarantine

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// NormalizeTestName strips flaky/perf prefixes for stable CI identity.
func NormalizeTestName(name string) string {
	n := strings.TrimSpace(name)
	for _, p := range []string{"[FLAKY] ", "[PERF] "} {
		if strings.HasPrefix(n, p) {
			n = strings.TrimPrefix(n, p)
		}
	}
	return strings.TrimSpace(n)
}

// TestIdentityFingerprint hashes project + normalized test name (no error text).
func TestIdentityFingerprint(projectName, testName string) string {
	raw := fmt.Sprintf("%s|%s", strings.TrimSpace(projectName), NormalizeTestName(testName))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
}
