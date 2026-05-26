package core

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// ── Robot Framework output.xml XML structures ─────────────────────────────────

type robotOutput struct {
	XMLName xml.Name    `xml:"robot"`
	Suite   robotSuite  `xml:"suite"`
}

type robotSuite struct {
	ID     string       `xml:"id,attr"`
	Name   string       `xml:"name,attr"`
	Source string       `xml:"source,attr"`
	Suites []robotSuite `xml:"suite"`
	Tests  []robotTest  `xml:"test"`
	Status robotStatus  `xml:"status"`
}

type robotTest struct {
	ID       string       `xml:"id,attr"`
	Name     string       `xml:"name,attr"`
	Tags     []string     `xml:"tag"`
	Keywords []robotKw    `xml:"kw"`
	Status   robotStatus  `xml:"status"`
}

type robotKw struct {
	Name     string      `xml:"name,attr"`
	Library  string      `xml:"library,attr"`
	Type     string      `xml:"type,attr"`
	Args     []string    `xml:"arg"`
	Messages []robotMsg  `xml:"msg"`
	Keywords []robotKw   `xml:"kw"`
	Status   robotStatus `xml:"status"`
}

type robotMsg struct {
	Timestamp string `xml:"timestamp,attr"`
	Level     string `xml:"level,attr"`
	Text      string `xml:",chardata"`
}

// Robot status: status attr + optional failure message as char data.
type robotStatus struct {
	Status    string `xml:"status,attr"`
	StartTime string `xml:"starttime,attr"`
	EndTime   string `xml:"endtime,attr"`
	Message   string `xml:",chardata"`
}

// ── Public API ────────────────────────────────────────────────────────────────

// IsRobotOutputXML returns true when data is a Robot Framework output.xml.
func IsRobotOutputXML(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("<?xml")) {
		idx := bytes.Index(trimmed, []byte("?>"))
		if idx >= 0 {
			trimmed = bytes.TrimSpace(trimmed[idx+2:])
		}
	}
	return bytes.HasPrefix(trimmed, []byte("<robot"))
}

// ParseRobotOutputXML parses Robot Framework's native output.xml and returns
// a JUnitParseResult with full keyword-level execution details in ConsoleLogs
// and ErrorLogs — preserving the same richness as Robot's own log.html.
func ParseRobotOutputXML(data []byte, framework string) JUnitParseResult {
	if framework == "" {
		framework = "RobotFramework"
	}

	var out robotOutput
	if err := xml.Unmarshal(data, &out); err != nil {
		return JUnitParseResult{}
	}

	report := UnifiedExecutionReport{
		SchemaVersion: executionReportSchemaVersion,
		Flags:         ExecutionFlags{Env: ExecutionEnvUnknown, Type: ExecutionTypeReal},
		Framework:     framework,
		ParsedAt:      time.Now().UTC(),
	}

	var failures []UnifiedAlert
	collectRobotSuite(&out.Suite, "", &report.Tests, &failures, framework)
	report.Summary = summarizeTests(report.Tests)

	return JUnitParseResult{Failures: failures, Report: report}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func collectRobotSuite(suite *robotSuite, parentPath string, tests *[]TestCaseResult, failures *[]UnifiedAlert, framework string) {
	path := suite.Name
	if parentPath != "" {
		path = parentPath + " > " + suite.Name
	}

	for _, test := range suite.Tests {
		tc := robotTestToResult(test, suite.Name, path, framework)
		*tests = append(*tests, tc)
		if tc.Status == "fail" {
			*failures = append(*failures, UnifiedAlert{
				Name:            tc.Name,
				Error:           tc.ErrorMessage,
				Browser:         "CI/Headless",
				ConsoleLogs:     tc.ConsoleLogs,
				ErrorLogs:       tc.ErrorLogs,
				Status:          "CRITICAL",
				ExecutionTimeMs: tc.DurationMs,
			})
		}
	}

	for i := range suite.Suites {
		collectRobotSuite(&suite.Suites[i], path, tests, failures, framework)
	}
}

func robotTestToResult(test robotTest, suiteName, suitePath, framework string) TestCaseResult {
	fullName := fmt.Sprintf("[%s] %s > %s", framework, suitePath, test.Name)

	status := strings.ToLower(strings.TrimSpace(test.Status.Status))
	switch status {
	case "pass":
		status = "pass"
	case "skip":
		status = "skip"
	default:
		status = "fail"
	}

	failureMsg := strings.TrimSpace(test.Status.Message)
	duration := robotDurationMs(test.Status.StartTime, test.Status.EndTime)

	var consoleBuf, errorBuf strings.Builder

	// Header line matching Robot's "suite > test" style
	consoleBuf.WriteString(fmt.Sprintf("[SUITE] %s\n", suitePath))
	consoleBuf.WriteString(fmt.Sprintf("[TEST] %s\n", test.Name))

	for _, kw := range test.Keywords {
		renderRobotKw(&kw, 1, &consoleBuf, &errorBuf)
	}

	if status == "fail" && failureMsg != "" {
		errorBuf.WriteString(fmt.Sprintf("\n[FAIL] %s\n", failureMsg))
	}

	consoleLogs := strings.TrimRight(consoleBuf.String(), "\n")
	if status == "pass" {
		if consoleLogs != "" {
			consoleLogs += "\n"
		}
		consoleLogs += fmt.Sprintf("[INFO] Test passed (%d ms).", duration)
	}

	errorLogs := strings.TrimSpace(errorBuf.String())

	return TestCaseResult{
		Name:         fullName,
		Suite:        suiteName,
		Status:       status,
		DurationMs:   duration,
		ErrorMessage: failureMsg,
		ConsoleLogs:  consoleLogs,
		ErrorLogs:    errorLogs,
		Fingerprint:  IncidentFingerprint(fullName, failureMsg),
	}
}

// renderRobotKw writes a keyword and all its nested content recursively.
func renderRobotKw(kw *robotKw, depth int, consoleBuf, errorBuf *strings.Builder) {
	indent := strings.Repeat("  ", depth)

	kwType := strings.ToUpper(strings.TrimSpace(kw.Type))
	if kwType == "" || kwType == "KW" {
		kwType = "KW"
	}

	kwLabel := kw.Name
	if kw.Library != "" && !strings.EqualFold(kw.Library, "BuiltIn") {
		kwLabel = kw.Library + "." + kw.Name
	}

	args := ""
	if len(kw.Args) > 0 {
		args = "  " + strings.Join(kw.Args, "  ")
	}

	duration := robotDurationMs(kw.Status.StartTime, kw.Status.EndTime)
	kwStatusStr := strings.ToUpper(strings.TrimSpace(kw.Status.Status))

	line := fmt.Sprintf("%s[%s] %s%s", indent, kwType, kwLabel, args)
	if duration > 0 {
		line += fmt.Sprintf("  (%dms)", duration)
	}
	if kwStatusStr != "" && kwStatusStr != "PASS" {
		line += fmt.Sprintf("  %s", kwStatusStr)
	}
	consoleBuf.WriteString(line + "\n")

	for _, msg := range kw.Messages {
		level := strings.ToUpper(strings.TrimSpace(msg.Level))
		text := strings.TrimSpace(msg.Text)
		if text == "" {
			continue
		}
		consoleBuf.WriteString(fmt.Sprintf("%s  [%s] %s\n", indent, level, text))
		if level == "FAIL" || level == "ERROR" || level == "WARN" {
			errorBuf.WriteString(fmt.Sprintf("[%s] %s\n", level, text))
		}
	}

	for i := range kw.Keywords {
		renderRobotKw(&kw.Keywords[i], depth+1, consoleBuf, errorBuf)
	}
}

func robotDurationMs(startTime, endTime string) int64 {
	start := parseRobotTime(startTime)
	end := parseRobotTime(endTime)
	if start.IsZero() || end.IsZero() {
		return 0
	}
	if d := end.Sub(start); d > 0 {
		return d.Milliseconds()
	}
	return 0
}

func parseRobotTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return time.Time{}
	}
	// RF < 5: "20230101 12:00:00.001"
	if t, err := time.Parse("20060102 15:04:05.999", s); err == nil {
		return t
	}
	// RF 5+: "2023-01-01T12:00:00.001000"
	if t, err := time.Parse("2006-01-02T15:04:05.999999", s); err == nil {
		return t
	}
	return time.Time{}
}
