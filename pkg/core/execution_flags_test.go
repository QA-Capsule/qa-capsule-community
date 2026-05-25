package core

import "testing"

func TestNormalizeExecutionEnv(t *testing.T) {
	cases := []struct{ in string; want ExecutionEnv }{
		{"prod", ExecutionEnvProd},
		{"PRODUCTION", ExecutionEnvProd},
		{"staging", ExecutionEnvStaging},
		{"integration", ExecutionEnvIntegration},
		{"CANARY", ExecutionEnvIntegration},
		{"", ExecutionEnvUnknown},
		{"invalid", ExecutionEnvUnknown},
	}
	for _, tc := range cases {
		if got := NormalizeExecutionEnv(tc.in); got != tc.want {
			t.Fatalf("NormalizeExecutionEnv(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeExecutionType(t *testing.T) {
	cases := []struct{ in string; want ExecutionType }{
		{"test-run", ExecutionTypeTestRun},
		{"TEST_RUN", ExecutionTypeTestRun},
		{"nightly", ExecutionTypeNightly},
		{"", ExecutionTypeUnknown},
	}
	for _, tc := range cases {
		if got := NormalizeExecutionType(tc.in); got != tc.want {
			t.Fatalf("NormalizeExecutionType(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestExecutionFlagsFromPayload_defaults(t *testing.T) {
	flags := ExecutionFlagsFromPayload(map[string]interface{}{})
	if flags.Env != ExecutionEnvUnknown {
		t.Fatalf("env = %q", flags.Env)
	}
	if flags.Type != ExecutionTypeReal {
		t.Fatalf("type = %q", flags.Type)
	}
}

func TestBuildReportFromPayload_testsArray(t *testing.T) {
	raw := map[string]interface{}{
		"execution_env":  "STAGING",
		"execution_type": "SMOKE",
		"tests": []interface{}{
			map[string]interface{}{"name": "login", "status": "passed"},
			map[string]interface{}{"name": "checkout", "status": "failed", "error": "timeout"},
		},
	}
	report := BuildReportFromPayload(raw, ExecutionFlagsFromPayload(raw), "Playwright")
	if report.Summary.Total != 2 {
		t.Fatalf("total = %d", report.Summary.Total)
	}
	if report.Summary.Failed != 1 || report.Summary.Passed != 1 {
		t.Fatalf("summary failed=%d passed=%d", report.Summary.Failed, report.Summary.Passed)
	}
	if report.Flags.Env != ExecutionEnvStaging || report.Flags.Type != ExecutionTypeSmoke {
		t.Fatalf("flags env=%s type=%s", report.Flags.Env, report.Flags.Type)
	}
}
