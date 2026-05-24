package core

import "testing"

func TestGlobalReportStatus(t *testing.T) {
	if globalReportStatus(ExecutionSummary{Total: 10, Passed: 10}, "success") != "passed" {
		t.Fatal("expected passed")
	}
	if globalReportStatus(ExecutionSummary{Total: 10, Passed: 8, Failed: 2}, "success") != "failed" {
		t.Fatal("expected failed from failed count")
	}
	if globalReportStatus(ExecutionSummary{}, "failure") != "failed" {
		t.Fatal("expected failed from outcome")
	}
}

func TestMergeReportTests_enrichesIncidentID(t *testing.T) {
	primary := []TestCaseResult{
		{Name: "login", Status: "fail", Fingerprint: "fp1", ErrorMessage: "timeout"},
	}
	incidents := []TestCaseResult{
		{Name: "[FLAKY] login", Status: "flaky", Fingerprint: "fp1", IncidentID: 42, ErrorLogs: "stack"},
	}
	merged := mergeReportTests(primary, incidents)
	if len(merged) != 1 {
		t.Fatalf("len = %d", len(merged))
	}
	if merged[0].IncidentID != 42 {
		t.Fatalf("incident_id = %d", merged[0].IncidentID)
	}
	if merged[0].ErrorLogs != "stack" {
		t.Fatal("expected error logs from incident")
	}
}
