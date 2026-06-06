package core

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultTestSearchRoots are common directories where client test files live.
var DefaultTestSearchRoots = []string{
	"tests", "test", "e2e", "spec", "specs", "integration",
	"src/test", "src/test/java", "src/test/resources",
	"cypress", "cypress/e2e", "playwright", "robotframework",
}

// testSourceExtensions maps file extensions searched during repo walk.
var testSourceExtensions = map[string]bool{
	".robot": true, ".py": true, ".js": true, ".ts": true, ".tsx": true,
	".java": true, ".cs": true, ".rb": true, ".go": true, ".feature": true,
	".cy.js": true, ".spec.js": true, ".spec.ts": true,
}

var tcTokenRe = regexp.MustCompile(`(?i)\bTC-\d+\b`)

// TestNameSearchTokens extracts meaningful fragments from a JUnit/Robot test name
// so we can locate the source file in any client repo layout.
func TestNameSearchTokens(testName string) []string {
	name := strings.TrimSpace(testName)
	if name == "" {
		return nil
	}
	// Strip [Framework] prefix injected by ingest.
	if i := strings.Index(name, "]"); i >= 0 {
		name = strings.TrimSpace(name[i+1:])
	}
	// Robot nested suites: "Suite A > Suite B > TC-01 My test"
	if i := strings.LastIndex(name, " > "); i >= 0 {
		name = strings.TrimSpace(name[i+2:])
	}
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if len(s) < 4 || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(name)
	if m := tcTokenRe.FindString(name); m != "" {
		add(m)
	}
	// Individual significant words (>=5 chars) for fuzzy file match.
	for _, w := range strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '(' || r == ')'
	}) {
		if len(w) >= 5 {
			add(w)
		}
	}
	return out
}

// SearchRoots returns directories to scan for a client repo.
func SearchRoots(repoPath string) []string {
	seen := map[string]bool{}
	var roots []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			roots = append(roots, p)
		}
	}
	if repoPath != "" {
		add(repoPath)
		for _, sub := range DefaultTestSearchRoots {
			add(filepath.Join(repoPath, sub))
		}
	}
	add(".")
	if cwd, err := os.Getwd(); err == nil {
		add(cwd)
		for _, sub := range DefaultTestSearchRoots {
			add(filepath.Join(cwd, sub))
		}
	}
	return roots
}

// ReadTestFileAtPaths tries to read a test file from stack-trace path candidates.
func ReadTestFileAtPaths(filePath, repoPath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	candidates := []string{filePath}
	if repoPath != "" {
		candidates = append(candidates,
			filepath.Join(repoPath, filePath),
			filepath.Join(repoPath, filepath.Base(filePath)),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, filePath))
		for _, sub := range DefaultTestSearchRoots {
			candidates = append(candidates, filepath.Join(cwd, sub, filepath.Base(filePath)))
		}
		if repoPath != "" {
			candidates = append(candidates, filepath.Join(cwd, repoPath, filePath))
		}
	}
	seen := map[string]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}
	return ""
}

// FindTestSourceInRepo walks client test directories and returns file content
// whose body contains tokens from the incident test name. Works for any framework.
func FindTestSourceInRepo(repoPath, testName string) string {
	tokens := TestNameSearchTokens(testName)
	if len(tokens) == 0 {
		return ""
	}

	const maxFiles = 800
	scanned := 0
	var bestContent string
	bestScore := 0

	for _, root := range SearchRoots(repoPath) {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				if d != nil && d.IsDir() {
					base := d.Name()
					if base == "node_modules" || base == ".git" || base == "vendor" || base == "dist" || base == "build" {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if scanned >= maxFiles {
				return fs.SkipAll
			}
			if !isTestSourceFile(path) {
				return nil
			}
			scanned++
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(data)
			lower := strings.ToLower(content)
			score := 0
			for _, tok := range tokens {
				if strings.Contains(lower, strings.ToLower(tok)) {
					score += len(tok)
				}
			}
			if score > bestScore {
				bestScore = score
				bestContent = content
				slog.Info("FindTestSourceInRepo: candidate", "path", path, "score", score, "test", testName)
			}
			return nil
		})
		if bestScore > 0 {
			break
		}
	}
	return bestContent
}

func isTestSourceFile(path string) bool {
	lower := strings.ToLower(path)
	for ext := range testSourceExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
