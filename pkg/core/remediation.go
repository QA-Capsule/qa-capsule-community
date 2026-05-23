package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/integrations"
)

var remediationEngine *integrations.Engine

// InitRemediationEngine loads integrations once at startup.
func InitRemediationEngine(pluginsDir string) error {
	return ReloadRemediationRegistry(pluginsDir)
}

// ReloadRemediationRegistry re-reads plugin manifests from disk (picks up routing_enabled / config changes).
func ReloadRemediationRegistry(pluginsDir string) error {
	reg, err := integrations.LoadRegistry(pluginsDir)
	if err != nil {
		return err
	}
	remediationEngine = integrations.NewEngine(reg)
	return nil
}

func requireEngine() (*integrations.Engine, error) {
	if remediationEngine == nil {
		return nil, fmt.Errorf("remediation engine not initialized")
	}
	return remediationEngine, nil
}

// EvaluateAlertRules runs native Go integrations when triggers match (no shell).
// allowedPluginPaths limits auto-run to plugins configured on the project gateway (nil = all auto_run plugins).
func EvaluateAlertRules(config Config, alert UnifiedAlert, projectContext map[string]string, allowedPluginPaths map[string]bool) {
	engine, err := requireEngine()
	if err != nil {
		return
	}
	routing := mapToRouting(projectContext)
	engine.EvaluateAlertRules(alert.Name, alert.Error, alert.ConsoleLogs, alert.Status, routing, allowedPluginPaths)
}

// RunSinglePlugin executes an integration by manifest path (API name kept for compatibility).
func RunSinglePlugin(_ Config, pluginRelPath string, action string, projectContext map[string]string) (string, error) {
	engine, err := requireEngine()
	if err != nil {
		return "", err
	}
	inc := integrations.IncidentContext{
		Name:   "Manual test (QA Capsule Plugin Engine)",
		Status: "CRITICAL",
		Action: action,
	}
	if strings.HasPrefix(action, "AUTO_EVENT:") {
		inc.Name = strings.TrimSpace(strings.TrimPrefix(action, "AUTO_EVENT:"))
	}
	routing := mapToRouting(projectContext)
	res, err := engine.Run(context.Background(), pluginRelPath, inc, routing)
	if err != nil {
		return res.Logs, err
	}
	if !res.Success {
		return res.String(), fmt.Errorf("integration execution failed")
	}
	return res.String(), nil
}

// RemediationRegistry exposes the loaded registry for HTTP handlers.
func RemediationRegistry() *integrations.Registry {
	if remediationEngine == nil {
		return nil
	}
	return remediationEngine.Registry()
}

func mapToRouting(ctx map[string]string) integrations.ProjectRouting {
	if ctx == nil {
		return integrations.ProjectRouting{}
	}
	values := make(map[string]string, len(ctx))
	for k, v := range ctx {
		if strings.HasPrefix(k, "__") {
			continue
		}
		if strings.TrimSpace(v) != "" {
			values[k] = v
		}
	}
	return integrations.ProjectRouting{
		SlackChannel:    ctx["SLACK_CHANNEL"],
		JiraProjectKey:  ctx["JIRA_PROJECT_KEY"],
		TeamsWebhookURL: firstNonEmpty(ctx["TEAMS_WEBHOOK_URL"], ctx["TEAMS_WEBHOOK"]),
		Values:          values,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
