package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJUnitReport_RobotNestedSuites(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "results-local", "robot-junit.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample robot-junit.xml not present:", err)
	}

	got := ParseJUnitReport(data, "RobotFramework")
	if len(got.Report.Tests) == 0 {
		t.Fatal("expected testcases from nested Robot xUnit suites")
	}
	if got.Report.Summary.Failed < 1 {
		t.Fatalf("expected at least 1 failed test, got summary=%+v", got.Report.Summary)
	}
	if len(got.Failures) < 1 {
		t.Fatal("expected at least one failure alert")
	}
	foundDemo := false
	for _, f := range got.Failures {
		if strings.Contains(f.Name, "Payment") || strings.Contains(f.Error, "400") {
			foundDemo = true
			break
		}
	}
	if !foundDemo {
		t.Fatalf("expected demo failure alert, got failures=%+v", got.Failures)
	}
}
