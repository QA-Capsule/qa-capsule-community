package integrations

// RoutingField describes one project-level routing input for a CI/CD gateway.
type RoutingField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	InputType   string `json:"input_type"` // text, url, password
}

// RoutingFieldsForIntegration returns per-project routing inputs shown in the gateway UI.
func RoutingFieldsForIntegration(integration string) []RoutingField {
	switch integration {
	case "slack":
		return []RoutingField{{Key: "SLACK_CHANNEL", Label: "Slack Channel", Placeholder: "#alerts-backend"}}
	case "jira":
		return []RoutingField{{Key: "JIRA_PROJECT_KEY", Label: "Jira Project Key", Placeholder: "e.g. PAY or SCRUM"}}
	case "teams":
		return []RoutingField{{Key: "TEAMS_WEBHOOK_URL", Label: "MS Teams Webhook URL", Placeholder: "https://...", InputType: "url"}}
	case "pagerduty":
		return []RoutingField{{Key: "PAGERDUTY_ROUTING_KEY", Label: "PagerDuty Routing Key", Placeholder: "Integration key"}}
	case "opsgenie":
		return []RoutingField{{Key: "OPSGENIE_TEAM", Label: "Opsgenie Team (optional)", Placeholder: "team-name"}}
	case "victorops":
		return []RoutingField{{Key: "VICTOROPS_ROUTING_URL", Label: "VictorOps Routing URL", Placeholder: "https://...", InputType: "url"}}
	case "datadog":
		return []RoutingField{{Key: "DATADOG_TAGS", Label: "Datadog Tags (optional)", Placeholder: "env:ci,service:api"}}
	case "webhook":
		return []RoutingField{{Key: "WEBHOOK_URL", Label: "Custom Webhook URL", Placeholder: "https://...", InputType: "url"}}
	case "sendgrid":
		return []RoutingField{{Key: "SENDGRID_TO", Label: "Alert Email To", Placeholder: "oncall@company.com"}}
	case "smtp":
		return []RoutingField{{Key: "SMTP_TO", Label: "SMTP Alert To", Placeholder: "oncall@company.com"}}
	case "github":
		return []RoutingField{
			{Key: "GITHUB_OWNER", Label: "GitHub Owner", Placeholder: "org"},
			{Key: "GITHUB_REPO", Label: "GitHub Repository", Placeholder: "repo"},
			{Key: "GITHUB_WORKFLOW_ID", Label: "Workflow ID or file name", Placeholder: "ci.yml"},
		}
	case "testrail", "zephyr", "xray", "k8s":
		return []RoutingField{{Key: "WEBHOOK_URL", Label: "Webhook URL", Placeholder: "https://...", InputType: "url"}}
	default:
		return nil
	}
}
