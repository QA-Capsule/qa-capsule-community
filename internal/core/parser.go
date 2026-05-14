package core

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// UnifiedAlert represents the standardized format for all telemetry events
// ADDED: ErrorLogs to cleanly separate STDOUT from STDERR/Crashes
type UnifiedAlert struct {
	Name        string `json:"name"`
	Error       string `json:"error"`
	Browser     string `json:"browser,omitempty"`
	ConsoleLogs string `json:"console_logs"` // Captures standard execution logs (STDOUT)
	ErrorLogs   string `json:"error_logs"`   // Captures stacktraces and critical errors (STDERR)
	Status      string `json:"status"`
}

// JUnitTestSuites represents the root of a JUnit XML report (for multiple suites)
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a group of test cases
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test execution result
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *JUnitFailure `xml:"failure"`
	Error     *JUnitFailure `xml:"error"`      // Support for framework using <error> instead of <failure>
	SystemOut string        `xml:"system-out"` // Captures standard console outputs
	SystemErr string        `xml:"system-err"` // Captures standard console errors
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
			Name:        fmt.Sprintf("[Playwright] %s", title),
			Error:       reason,
			Browser:     project,
			ConsoleLogs: "[INFO] Playwright Worker execution details hidden. Check full logs if needed.",
			ErrorLogs:   fmt.Sprintf("[FATAL] %s", reason),
			Status:      "CRITICAL",
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
		browser, _ := raw["browser"].(string)
		status, _ := raw["status"].(string)
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

		return UnifiedAlert{
			Name:        name,
			Error:       errStr,
			Browser:     browser,
			ConsoleLogs: strings.Join(logs, "\n"),
			ErrorLogs:   errStr, // Fallback error log
			Status:      status,
		}
	}
}

// ParseJUnitXML parses raw XML bytes and extracts ONLY failed tests into UnifiedAlerts
// It now perfectly separates STDOUT and STDERR into respective fields
func ParseJUnitXML(data []byte, framework string) []UnifiedAlert {
	var suites JUnitTestSuites
	var alerts []UnifiedAlert

	if framework == "" {
		framework = "JUnit"
	}

	err := xml.Unmarshal(data, &suites)
	if err != nil || len(suites.TestSuites) == 0 {
		var single JUnitTestSuite
		xml.Unmarshal(data, &single)
		suites.TestSuites = []JUnitTestSuite{single}
	}

	for _, suite := range suites.TestSuites {
		for _, tc := range suite.TestCases {
			var failureNode *JUnitFailure
			if tc.Failure != nil {
				failureNode = tc.Failure
			} else if tc.Error != nil {
				failureNode = tc.Error
			}

			if failureNode != nil {
				var stdoutBuilder strings.Builder
				var stderrBuilder strings.Builder

				// 1. Build the Error / Crash Log (STDERR + Stacktrace)
				stderrBuilder.WriteString(fmt.Sprintf("[INFO] Source: XML JUnit Report\n[FATAL] Type: %s\n", failureNode.Type))

				if strings.TrimSpace(failureNode.Contents) != "" {
					stderrBuilder.WriteString(fmt.Sprintf("\n--- STACKTRACE ---\n%s\n", strings.TrimSpace(failureNode.Contents)))
				}
				if strings.TrimSpace(tc.SystemErr) != "" {
					stderrBuilder.WriteString(fmt.Sprintf("\n--- CONSOLE STDERR ---\n%s\n", strings.TrimSpace(tc.SystemErr)))
				}

				// 2. Build the Standard Output Log (STDOUT)
				if strings.TrimSpace(tc.SystemOut) != "" {
					stdoutBuilder.WriteString(fmt.Sprintf("--- CONSOLE STDOUT ---\n%s\n", strings.TrimSpace(tc.SystemOut)))
				} else {
					stdoutBuilder.WriteString("[INFO] No standard output (stdout) captured for this test.")
				}

				alertName := tc.Name
				if tc.ClassName != "" {
					alertName = fmt.Sprintf("%s > %s", tc.ClassName, tc.Name)
				}

				alerts = append(alerts, UnifiedAlert{
					Name:        fmt.Sprintf("[%s] %s", framework, alertName),
					Error:       failureNode.Message,
					Browser:     "CI/Artifact",
					ConsoleLogs: stdoutBuilder.String(), // Populated with only standard logs
					ErrorLogs:   stderrBuilder.String(), // Populated with crashes and trace
					Status:      "CRITICAL",
				})
			}
		}
	}
	return alerts
}
