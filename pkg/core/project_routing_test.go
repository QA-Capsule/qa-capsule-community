package core

import "testing"

func TestParseSRERoutingJSON_legacyLinear(t *testing.T) {
	raw := `[{"integration":"slack","file_path":"slack/slack-notifier.json","values":{"SLACK_CHANNEL":"#alerts"}}]`
	entries := ParseSRERoutingJSON(raw)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].FilePath != "slack/slack-notifier.json" {
		t.Fatalf("unexpected file_path: %s", entries[0].FilePath)
	}
	if entries[0].Values["SLACK_CHANNEL"] != "#alerts" {
		t.Fatalf("unexpected channel: %s", entries[0].Values["SLACK_CHANNEL"])
	}
}

func TestParseSRERoutingJSON_emptyAndInvalid(t *testing.T) {
	if ParseSRERoutingJSON("") != nil {
		t.Fatal("empty should be nil")
	}
	if ParseSRERoutingJSON("not-json") != nil {
		t.Fatal("invalid should be nil")
	}
}
