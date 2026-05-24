package core

import "testing"

func TestUnifiedReporter_JSON_pytestStyle(t *testing.T) {
	raw := map[string]interface{}{
		"execution_env":  "prod",
		"execution_type": "nightly",
		"tests": []interface{}{
			map[string]interface{}{
				"nodeid":   "tests/test_auth.py::test_login",
				"outcome":  "passed",
				"duration": 0.42,
			},
			map[string]interface{}{
				"nodeid":   "tests/test_auth.py::test_logout",
				"outcome":  "failed",
				"longrepr": "AssertionError",
			},
		},
	}
	res := DefaultUnifiedReporter.Normalize(IngestPayload{
		Format: ReportFormatJSON,
		JSON:   raw,
	})
	if len(res.Report.Tests) != 2 {
		t.Fatalf("tests = %d", len(res.Report.Tests))
	}
	if res.Report.Flags.Env != ExecutionEnvProd || res.Report.Flags.Type != ExecutionTypeNightly {
		t.Fatalf("flags env=%s type=%s", res.Report.Flags.Env, res.Report.Flags.Type)
	}
	if len(res.Failures) < 1 {
		t.Fatal("expected at least one failure alert")
	}
}

func TestUnifiedReporter_JUnit(t *testing.T) {
	res := DefaultUnifiedReporter.Normalize(IngestPayload{
		Format:    ReportFormatJUnitXML,
		Framework: "pytest",
		XML:       []byte(sampleJUnit),
	})
	if len(res.Report.Tests) != 3 {
		t.Fatalf("tests = %d", len(res.Report.Tests))
	}
	if len(res.Failures) != 1 {
		t.Fatalf("failures = %d", len(res.Failures))
	}
}

func TestDetectReportFormat(t *testing.T) {
	if DetectReportFormat("/api/webhooks/foo/upload", "") != ReportFormatJUnitXML {
		t.Fatal("upload path should be junit")
	}
	if DetectReportFormat("/api/webhooks/foo", "application/json") != ReportFormatJSON {
		t.Fatal("json path should be json")
	}
}
