package core

import "testing"

func TestEnsurePipelineCrashName_failedWithoutTestName(t *testing.T) {
	a := EnsurePipelineCrashName(UnifiedAlert{Status: "failed"})
	if a.Name != PipelineCrashDefaultName {
		t.Fatalf("name = %q, want %q", a.Name, PipelineCrashDefaultName)
	}
	if a.Error == "" {
		t.Fatal("expected default error message")
	}
}

func TestEnsurePipelineCrashName_preservesExistingName(t *testing.T) {
	a := EnsurePipelineCrashName(UnifiedAlert{Name: "checkout.spec", Status: "failed"})
	if a.Name != "checkout.spec" {
		t.Fatalf("name = %q", a.Name)
	}
}

func TestEnsurePipelineCrashName_passWithoutName(t *testing.T) {
	a := EnsurePipelineCrashName(UnifiedAlert{Status: "passed"})
	if a.Name != "" {
		t.Fatalf("expected empty name on pass, got %q", a.Name)
	}
}

func TestParseAlertsFromRaw_pipelineCrashWebhook(t *testing.T) {
	raw := map[string]interface{}{
		"status":  "failed",
		"message": "npm ci exited with code 1",
		"tests":   []interface{}{},
	}
	alerts := ParseAlertsFromRaw(raw)
	if len(alerts) != 1 {
		t.Fatalf("alerts len = %d", len(alerts))
	}
	if alerts[0].Name != PipelineCrashDefaultName {
		t.Fatalf("name = %q", alerts[0].Name)
	}
	if alerts[0].Error != "npm ci exited with code 1" {
		t.Fatalf("error = %q", alerts[0].Error)
	}
}
