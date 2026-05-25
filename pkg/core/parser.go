package core

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// UnifiedAlert represents the standardized format for all telemetry events
// ADDED: ErrorLogs to cleanly separate STDOUT from STDERR/Crashes
type UnifiedAlert struct {
	Name            string `json:"name"`
	Error           string `json:"error"`
	Browser         string `json:"browser,omitempty"`
	OS              string `json:"os,omitempty"`
	Viewport        string `json:"viewport,omitempty"`
	ConsoleLogs     string `json:"console_logs"`
	ErrorLogs       string `json:"error_logs"`
	Status          string `json:"status"`
	ExecutionTimeMs int64  `json:"execution_time_ms,omitempty"`
	JiraIssueKey    string `json:"jira_issue_key,omitempty"`
	CommitSHA       string `json:"commit_sha,omitempty"`
	Branch          string `json:"branch,omitempty"`
}

// JUnitTestSuites represents the root of a JUnit XML report (for multiple suites)
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Skipped    int              `xml:"skipped,attr"`
	Time       float64          `xml:"time,attr"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a group of test cases (Robot/rebot may nest suites).
type JUnitTestSuite struct {
	Name       string           `xml:"name,attr"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Skipped    int              `xml:"skipped,attr"`
	Time       float64          `xml:"time,attr"`
	TestCases  []JUnitTestCase  `xml:"testcase"`
	Nested     []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestCase represents a single test execution result
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure"`
	Error     *JUnitFailure `xml:"error"`
	Skipped   *JUnitSkipped `xml:"skipped"`
	SystemOut string        `xml:"system-out"`
	SystemErr string        `xml:"system-err"`
}

// JUnitSkipped marks a skipped testcase.
type JUnitSkipped struct {
	Message string `xml:"message,attr"`
}

// JUnitFailure holds the crash details, type, and stacktrace
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"` // Captures the full stacktrace inside the tag
}

// NormalizePayload converts dynamic JSON payloads into the standard UnifiedAlert format
func NormalizePayload(raw map[string]interface{}) UnifiedAlert {
	framework, _ := raw["framework"].(string)

	switch strings.ToLower(framework) {
	case "playwright":
		title, _ := raw["title"].(string)
		reason, _ := raw["failure_reason"].(string)
		project, _ := raw["project"].(string)
		return UnifiedAlert{
			Name:            fmt.Sprintf("[Playwright] %s", title),
			Error:           reason,
			Browser:         stringField(raw, "browser", project),
			OS:              stringField(raw, "os", ""),
			Viewport:        stringField(raw, "viewport", ""),
			ConsoleLogs:     "[INFO] Playwright Worker execution details hidden. Check full logs if needed.",
			ErrorLogs:       fmt.Sprintf("[FATAL] %s", reason),
			Status:          stringField(raw, "status", "CRITICAL"),
			ExecutionTimeMs: int64Field(raw, "execution_time_ms"),
		}

	case "robotframework":
		suite, _ := raw["suite"].(string)
		test, _ := raw["test"].(string)
		msg, _ := raw["message"].(string)
		return UnifiedAlert{
			Name:        fmt.Sprintf("[RobotFW] %s - %s", suite, test),
			Error:       msg,
			Browser:     "System/Headless",
			ConsoleLogs: "[INFO] Robot Framework execution sequence finished.",
			ErrorLogs:   fmt.Sprintf("[ERROR] %s", msg),
			Status:      "CRITICAL",
		}

	default:
		name, _ := raw["name"].(string)
		errStr, _ := raw["error"].(string)
		if errStr == "" {
			if msg, ok := raw["message"].(string); ok {
				errStr = msg
			}
		}
		if errStr == "" {
			if reason, ok := raw["failure_reason"].(string); ok {
				errStr = reason
			}
		}
		browser, _ := raw["browser"].(string)
		osName, _ := raw["os"].(string)
		viewport, _ := raw["viewport"].(string)
		status, _ := raw["status"].(string)
		if status == "" {
			if st, ok := raw["state"].(string); ok && strings.TrimSpace(st) != "" {
				status = st
			}
		}
		if status == "" {
			status = "FAILED"
		}

		var logs []string
		if rawLogs, ok := raw["console_logs"].([]interface{}); ok {
			for _, l := range rawLogs {
				logs = append(logs, fmt.Sprintf("%v", l))
			}
		} else if logStr, ok := raw["console_logs"].(string); ok {
			logs = append(logs, logStr)
		}

		stack := stringField(raw, "stack", "")
		if stack == "" {
			stack = stringField(raw, "stacktrace", "")
		}
		if stack == "" {
			stack = stringField(raw, "stack_trace", "")
		}
		if stack == "" {
			stack = stringField(raw, "trace", "")
		}
		errorLogs := stack
		if errorLogs == "" {
			errorLogs = errStr
		}
		return UnifiedAlert{
			Name:            name,
			Error:           errStr,
			Browser:         browser,
			OS:              osName,
			Viewport:        viewport,
			ConsoleLogs:     strings.Join(logs, "\n"),
			ErrorLogs:       errorLogs,
			Status:          status,
			ExecutionTimeMs: int64Field(raw, "execution_time_ms"),
			JiraIssueKey:    stringField(raw, "jira_issue_key", ""),
			CommitSHA:       stringField(raw, "commit_sha", ""),
			Branch:          stringField(raw, "branch", ""),
		}
	}
}

func stringField(raw map[string]interface{}, key, def string) string {
	if v, ok := raw[key].(string); ok && v != "" {
		return v
	}
	return def
}

func int64Field(raw map[string]interface{}, key string) int64 {
	switch v := raw[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func floatField(raw map[string]interface{}, key string) float64 {
	switch v := raw[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// JUnitParseResult contains failure alerts and the full execution report.
type JUnitParseResult struct {
	Failures []UnifiedAlert
	Report   UnifiedExecutionReport
}

// ParseJUnitXML parses JUnit XML and returns failure alerts (backward compatible).
func ParseJUnitXML(data []byte, framework string) []UnifiedAlert {
	return ParseJUnitReport(data, framework).Failures
}

// flattenJUnitSuites expands nested Robot/rebot xUnit suites into leaf suites with testcases.
func flattenJUnitSuites(suites []JUnitTestSuite) []JUnitTestSuite {
	var out []JUnitTestSuite
	for _, suite := range suites {
		out = append(out, collectJUnitSuites(suite)...)
	}
	return out
}

func collectJUnitSuites(suite JUnitTestSuite) []JUnitTestSuite {
	var out []JUnitTestSuite
	if len(suite.TestCases) > 0 {
		out = append(out, suite)
	}
	for _, nested := range suite.Nested {
		out = append(out, collectJUnitSuites(nested)...)
	}
	return out
}

// ParseJUnitReport parses all testcases and builds a unified execution report.
func ParseJUnitReport(data []byte, framework string) JUnitParseResult {
	if framework == "" {
		framework = "JUnit"
	}
	var suites JUnitTestSuites
	_ = xml.Unmarshal(data, &suites)
	if len(suites.TestSuites) == 0 {
		var single JUnitTestSuite
		if xml.Unmarshal(data, &single) == nil {
			suites.TestSuites = []JUnitTestSuite{single}
		}
	}
	flatSuites := flattenJUnitSuites(suites.TestSuites)

	report := UnifiedExecutionReport{
		SchemaVersion: executionReportSchemaVersion,
		Flags:         ExecutionFlags{Env: ExecutionEnvUnknown, Type: ExecutionTypeReal},
		Framework:     framework,
		ParsedAt:      time.Now().UTC(),
	}
	var failures []UnifiedAlert

	for _, suite := range flatSuites {
		suiteName := strings.TrimSpace(suite.Name)
		for _, tc := range suite.TestCases {
			className := strings.TrimSpace(tc.ClassName)
			alertName := tc.Name
			if className != "" {
				alertName = fmt.Sprintf("%s > %s", className, tc.Name)
			} else if suiteName != "" {
				alertName = fmt.Sprintf("%s > %s", suiteName, tc.Name)
			}
			fullName := fmt.Sprintf("[%s] %s", framework, alertName)
			durationMs := int64(tc.Time * 1000)

			var failureNode *JUnitFailure
			if tc.Failure != nil {
				failureNode = tc.Failure
			} else if tc.Error != nil {
				failureNode = tc.Error
			}

			status := "passed"
			if tc.Skipped != nil {
				status = "skip"
			} else if failureNode != nil {
				status = "CRITICAL"
			}

			tcResult := TestCaseResult{
				Name:       fullName,
				Suite:      suiteName,
				ClassName:  className,
				Status:     normalizeTestMatrixStatus(status, fullName),
				DurationMs: durationMs,
			}

			if failureNode != nil {
				var stdoutBuilder strings.Builder
				var stderrBuilder strings.Builder
				stderrBuilder.WriteString(fmt.Sprintf("[INFO] Source: XML JUnit Report\n[FATAL] Type: %s\n", failureNode.Type))
				if strings.TrimSpace(failureNode.Contents) != "" {
					stderrBuilder.WriteString(fmt.Sprintf("\n--- STACKTRACE ---\n%s\n", strings.TrimSpace(failureNode.Contents)))
				}
				if strings.TrimSpace(tc.SystemErr) != "" {
					stderrBuilder.WriteString(fmt.Sprintf("\n--- CONSOLE STDERR ---\n%s\n", strings.TrimSpace(tc.SystemErr)))
				}
				if strings.TrimSpace(tc.SystemOut) != "" {
					stdoutBuilder.WriteString(fmt.Sprintf("--- CONSOLE STDOUT ---\n%s\n", strings.TrimSpace(tc.SystemOut)))
				} else {
					stdoutBuilder.WriteString("[INFO] No standard output (stdout) captured for this test.")
				}
				tcResult.ErrorMessage = failureNode.Message
				tcResult.ConsoleLogs = stdoutBuilder.String()
				tcResult.ErrorLogs = stderrBuilder.String()
				tcResult.Fingerprint = IncidentFingerprint(fullName, failureNode.Message)

				failures = append(failures, UnifiedAlert{
					Name:            fullName,
					Error:           failureNode.Message,
					Browser:         "CI/Artifact",
					ConsoleLogs:     tcResult.ConsoleLogs,
					ErrorLogs:       tcResult.ErrorLogs,
					Status:          "CRITICAL",
					ExecutionTimeMs: durationMs,
				})
			} else if tcResult.Status == "pass" || tcResult.Status == "skip" {
				tcResult.Fingerprint = IncidentFingerprint(fullName, "")
				var stdoutBuilder strings.Builder
				var stderrBuilder strings.Builder
				if strings.TrimSpace(tc.SystemOut) != "" {
					stdoutBuilder.WriteString(fmt.Sprintf("--- CONSOLE STDOUT ---\n%s\n", strings.TrimSpace(tc.SystemOut)))
				}
				if strings.TrimSpace(tc.SystemErr) != "" {
					stderrBuilder.WriteString(fmt.Sprintf("--- CONSOLE STDERR ---\n%s\n", strings.TrimSpace(tc.SystemErr)))
				}
				if tcResult.Status == "pass" {
					stdoutBuilder.WriteString(fmt.Sprintf("[INFO] Test passed (%d ms).\n", durationMs))
					if strings.TrimSpace(tc.SystemOut) == "" && strings.TrimSpace(tc.SystemErr) == "" {
						stdoutBuilder.WriteString("[INFO] JUnit has no <system-out>/<system-err> for this case. Enable Robot listener/log output or re-export output.xml with stdout enabled.\n")
					}
				} else {
					stdoutBuilder.WriteString("[INFO] Test skipped.\n")
				}
				tcResult.ConsoleLogs = stdoutBuilder.String()
				tcResult.ErrorLogs = stderrBuilder.String()
			}
			report.Tests = append(report.Tests, tcResult)
		}
	}

	report.Summary = summarizeTests(report.Tests)
	if suites.Tests > 0 && report.Summary.Total == 0 {
		report.Summary.Total = suites.Tests
	}
	if suites.Failures > 0 && report.Summary.Failed == 0 {
		report.Summary.Failed = suites.Failures
	}
	if suites.Skipped > 0 && report.Summary.Skipped == 0 {
		report.Summary.Skipped = suites.Skipped
	}
	if suites.Time > 0 && report.Summary.DurationMs == 0 {
		report.Summary.DurationMs = int64(suites.Time * 1000)
	}
	if report.Summary.Passed == 0 && report.Summary.Total > 0 {
		report.Summary.Passed = report.Summary.Total - report.Summary.Failed - report.Summary.Skipped
		if report.Summary.Passed < 0 {
			report.Summary.Passed = 0
		}
	}

	return JUnitParseResult{Failures: failures, Report: report}
}
