package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// ==========================================
// JUNIT XML DATA STRUCTURES
// ==========================================

// JUnitTestSuites represents the root element (common in multi-suite reports)
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single suite of tests
type JUnitTestSuite struct {
	XMLName   xml.Name         `xml:"testsuite"`
	Name      string           `xml:"name,attr"`
	TestCases []JUnitTestCase  `xml:"testcase"`
	Suites    []JUnitTestSuite `xml:"testsuite"` // Supports nested suites
}

// JUnitTestCase represents an individual test execution
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure"`
	Error     *JUnitFailure `xml:"error"`
	SystemOut string        `xml:"system-out"`
	SystemErr string        `xml:"system-err"`
}

// JUnitFailure holds the crash details, stacktrace, and error message
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"` // Extracts the inner text (usually the stacktrace)
}

// ==========================================
// BACKEND API STRUCTURE
// ==========================================

// UnifiedAlert is the payload expected by the QA Capsule Webhook
type UnifiedAlert struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	ConsoleLogs string `json:"console_logs"`
}

// ==========================================
// CLI AGENT LOGIC
// ==========================================

func main() {
	// 1. Define command-line arguments for CI integration
	reportPath := flag.String("file", "", "Path to the JUnit XML report file")
	apiKey := flag.String("token", "", "Project API Key from QA Capsule")
	serverURL := flag.String("url", "http://localhost:8080/api/webhooks/", "QA Capsule Webhook URL")
	framework := flag.String("framework", "CI/CD", "Name of the testing framework (e.g., Cypress, Playwright)")

	flag.Parse()

	// 2. Validate inputs
	if *reportPath == "" || *apiKey == "" {
		log.Fatal("[AGENT ERROR] Missing required arguments: --file and --token are mandatory.")
	}

	// 3. Read the JUnit XML file
	xmlFile, err := os.Open(*reportPath)
	if err != nil {
		log.Fatalf("[AGENT ERROR] Cannot open report file: %v", err)
	}
	defer xmlFile.Close()

	byteValue, err := io.ReadAll(xmlFile)
	if err != nil {
		log.Fatalf("[AGENT ERROR] Cannot read report file: %v", err)
	}

	// 4. Parse the XML
	// Note: JUnit XML can have <testsuites> or <testsuite> as the root element.
	// We handle both possibilities to ensure universal compatibility.
	var rootSuites JUnitTestSuites
	var singleSuite JUnitTestSuite

	var suitesToProcess []JUnitTestSuite

	if err := xml.Unmarshal(byteValue, &rootSuites); err == nil && len(rootSuites.Suites) > 0 {
		suitesToProcess = rootSuites.Suites
	} else if err := xml.Unmarshal(byteValue, &singleSuite); err == nil && singleSuite.Name != "" {
		suitesToProcess = append(suitesToProcess, singleSuite)
	} else {
		log.Fatalf("[AGENT ERROR] Failed to parse XML as valid JUnit format.")
	}

	// 5. Analyze tests and extract ONLY failures
	failedTestsCount := 0
	log.Printf("[AGENT] Parsing report for %s...", *framework)

	for _, suite := range suitesToProcess {
		processSuite(suite, *framework, *serverURL, *apiKey, &failedTestsCount)
	}

	log.Printf("[AGENT SUCCESS] Analysis complete. Sent %d failed tests to QA Capsule Dashboard.", failedTestsCount)
}

// processSuite recursively searches for failed test cases within a suite
func processSuite(suite JUnitTestSuite, framework string, url string, apiKey string, failedCount *int) {
	// Process tests in the current suite
	for _, test := range suite.TestCases {
		// We only care about tests that actually failed (presence of <failure> or <error> node)
		if test.Failure != nil || test.Error != nil {
			*failedCount++
			sendIncident(suite.Name, test, framework, url, apiKey)
		}
	}

	// Recursively process nested suites (common in Playwright and RobotFramework)
	for _, nestedSuite := range suite.Suites {
		processSuite(nestedSuite, framework, url, apiKey, failedCount)
	}
}

// sendIncident constructs the payload and dispatches it to the control plane
func sendIncident(suiteName string, test JUnitTestCase, framework string, url string, apiKey string) {
	var failureNode *JUnitFailure
	if test.Failure != nil {
		failureNode = test.Failure
	} else {
		failureNode = test.Error
	}

	// Clean up the error body (remove leading/trailing whitespaces)
	stackTrace := strings.TrimSpace(failureNode.Body)
	if stackTrace == "" {
		stackTrace = "No stacktrace provided by the framework."
	}

	// Compile specific logs for this failing test
	var consoleLogs strings.Builder
	if test.SystemOut != "" {
		consoleLogs.WriteString("--- STDOUT ---\n" + strings.TrimSpace(test.SystemOut) + "\n")
	}
	if test.SystemErr != "" {
		consoleLogs.WriteString("--- STDERR ---\n" + strings.TrimSpace(test.SystemErr) + "\n")
	}

	// Create a clear, normalized name for the dashboard
	// Format: [Framework] SuiteName > TestName
	normalizedName := fmt.Sprintf("[%s] %s > %s", framework, suiteName, test.Name)

	// Create the payload
	alert := UnifiedAlert{
		Name:        normalizedName,
		Status:      "FAILED",
		Error:       fmt.Sprintf("%s\n\nStackTrace:\n%s", failureNode.Message, stackTrace),
		ConsoleLogs: consoleLogs.String(),
	}

	payloadBytes, err := json.Marshal(alert)
	if err != nil {
		log.Printf("[AGENT WARNING] Failed to serialize alert for test %s: %v", test.Name, err)
		return
	}

	// Execute HTTP POST Request to QA Capsule
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("[AGENT WARNING] Failed to create HTTP request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[AGENT WARNING] Network error while sending alert to QA Capsule: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
		log.Printf("  -> [SENT] %s", normalizedName)
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("  -> [FAILED TO SEND] %s - HTTP %d: %s", normalizedName, resp.StatusCode, string(body))
	}
}