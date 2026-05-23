package core

import (
	"regexp"
	"strings"
)

var (
	jiraTagPattern = regexp.MustCompile(`(?i)@jira[-_]?([A-Za-z0-9]+-\d+)`)
	jiraKeyPattern = regexp.MustCompile(`@([A-Z][A-Z0-9]+-\d+)`)
)

// ExtractJiraIssueKey pulls @jira-PROJ-123 or @PROJ-123 from a test name.
func ExtractJiraIssueKey(testName string) string {
	if m := jiraTagPattern.FindStringSubmatch(testName); len(m) > 1 {
		return strings.ToUpper(m[1])
	}
	if m := jiraKeyPattern.FindStringSubmatch(testName); len(m) > 1 {
		return m[1]
	}
	return ""
}
