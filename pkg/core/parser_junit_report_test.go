package core

import "testing"

const sampleJUnit = `<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="3" failures="1" skipped="1" time="1.5">
  <testsuite name="Suite" tests="3" failures="1" skipped="1">
    <testcase classname="Auth" name="login_ok" time="0.2"/>
    <testcase classname="Auth" name="login_fail" time="0.3">
      <failure message="assertion error">stack here</failure>
    </testcase>
    <testcase classname="Auth" name="login_skip" time="0.1">
      <skipped message="wip"/>
    </testcase>
  </testsuite>
</testsuites>`

func TestParseJUnitReport_matrixAndFailures(t *testing.T) {
	res := ParseJUnitReport([]byte(sampleJUnit), "JUnit")
	if len(res.Failures) != 1 {
		t.Fatalf("failures = %d", len(res.Failures))
	}
	if len(res.Report.Tests) != 3 {
		t.Fatalf("tests = %d", len(res.Report.Tests))
	}
	if res.Report.Summary.Total != 3 {
		t.Fatalf("total = %d", res.Report.Summary.Total)
	}
	if res.Report.Summary.Failed != 1 || res.Report.Summary.Skipped != 1 || res.Report.Summary.Passed != 1 {
		t.Fatalf("summary failed=%d skipped=%d passed=%d",
			res.Report.Summary.Failed, res.Report.Summary.Skipped, res.Report.Summary.Passed)
	}
}

func TestParseJUnitXML_backwardCompatible(t *testing.T) {
	alerts := ParseJUnitXML([]byte(sampleJUnit), "JUnit")
	if len(alerts) != 1 {
		t.Fatalf("alerts = %d", len(alerts))
	}
}

func TestParseJUnitReport_deduplicatesDuplicatedTestcases(t *testing.T) {
	dup := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="2" failures="2" skipped="0">
  <testsuite name="SuiteA">
    <testcase classname="A" name="same_test" time="0.2">
      <failure message="same failure">stack 1</failure>
    </testcase>
    <testcase classname="A" name="same_test" time="0.2">
      <failure message="same failure">stack 1</failure>
    </testcase>
  </testsuite>
</testsuites>`
	res := ParseJUnitReport([]byte(dup), "RobotFramework")
	if len(res.Report.Tests) != 1 {
		t.Fatalf("expected 1 deduplicated testcase, got %d", len(res.Report.Tests))
	}
	if len(res.Failures) != 1 {
		t.Fatalf("expected 1 deduplicated failure alert, got %d", len(res.Failures))
	}
}
