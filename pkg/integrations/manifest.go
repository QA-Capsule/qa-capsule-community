package integrations

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Manifest describes a registered remediation integration (no shell commands).
type Manifest struct {
	ID          string            `json:"id"`
	FilePath    string            `json:"file_path"`
	Integration string            `json:"integration"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Status          string            `json:"status"`
	AutoRun         bool              `json:"auto_run"`
	RoutingEnabled  bool              `json:"routing_enabled"`
	TriggerOn       []string          `json:"trigger_on"`
	Config      map[string]string `json:"config"`
}

// inferIntegration maps legacy plugin folders / fields to a typed integration.
func inferIntegration(relPath string, raw map[string]interface{}) (string, error) {
	if t, ok := raw["integration"].(string); ok && t != "" {
		return strings.ToLower(strings.TrimSpace(t)), nil
	}
	if cmd, ok := raw["command"].(string); ok && cmd != "" {
		// Legacy manifests: never execute; only infer type from path or command name.
		lower := strings.ToLower(cmd)
		switch {
		case strings.Contains(lower, "slack"):
			return "slack", nil
		case strings.Contains(lower, "jira"):
			return "jira", nil
		case strings.Contains(lower, "teams"):
			return "teams", nil
		}
	}
	dir := strings.ToLower(strings.Trim(filepath.Dir(relPath), `/\`))
	switch dir {
	case "slack", "jira", "teams", "pagerduty", "victorops", "datadog", "webhook",
		"github", "email", "k8s", "testrail", "zephyr", "xray", "qa", "opsgenie":
		if dir == "email" {
			if strings.Contains(strings.ToLower(filepath.Base(relPath)), "smtp") {
				return "smtp", nil
			}
			return "sendgrid", nil
		}
		if dir == "qa" {
			return "webhook", nil
		}
		return dir, nil
	}
	return "", fmt.Errorf("unknown integration for %s (set \"integration\" in manifest)", relPath)
}

func mergeConfig(raw map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for _, key := range []string{"env", "config"} {
		block, ok := raw[key].(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range block {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}
