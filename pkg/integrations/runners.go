package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

func mergedConfig(m *Manifest, routing ProjectRouting) map[string]string {
	out := make(map[string]string, len(m.Config)+8)
	for k, v := range m.Config {
		out[k] = v
	}
	for k, v := range routing.Values {
		if strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}
	if routing.SlackChannel != "" {
		out["SLACK_CHANNEL"] = routing.SlackChannel
	}
	if routing.JiraProjectKey != "" {
		out["JIRA_PROJECT_KEY"] = routing.JiraProjectKey
	}
	if routing.TeamsWebhookURL != "" {
		out["TEAMS_WEBHOOK_URL"] = routing.TeamsWebhookURL
	}
	return out
}

func configVal(key string, c map[string]string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return strings.TrimSpace(c[key])
}

func httpPostJSON(ctx context.Context, url string, body any, headers map[string]string) (int, string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(raw), nil
}

func runSlack(ctx context.Context, m *Manifest, inc IncidentContext, routing ProjectRouting) Result {
	c := mergedConfig(m, routing)
	webhook := configVal("SLACK_WEBHOOK_URL", c)
	if webhook == "" {
		return Result{Success: false, Logs: "[ERROR] SLACK_WEBHOOK_URL not configured (env or Plugin config)."}
	}
	channel := configVal("SLACK_CHANNEL", c)
	if channel == "" {
		channel = "#general"
	}
	payload := map[string]any{
		"channel": channel,
		"attachments": []map[string]any{{
			"color":  "#ff4444",
			"title":  fmt.Sprintf("SRE Alert: %s", incidentSummary(inc)),
			"text":   fmt.Sprintf("Error detected by QA Capsule.\n\n%s", inc.Error),
			"footer": "QA Capsule Remediation Engine",
		}},
	}
	code, body, err := httpPostJSON(ctx, webhook, payload, nil)
	if err != nil {
		return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Slack request failed: %v", err)}
	}
	if code >= 200 && code < 300 {
		return Result{Success: true, Logs: fmt.Sprintf("[SLACK] Delivered to %s (HTTP %d)\n%s", channel, code, body)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Slack HTTP %d\n%s", code, body)}
}

func runJira(ctx context.Context, m *Manifest, inc IncidentContext, routing ProjectRouting) Result {
	c := mergedConfig(m, routing)
	base := strings.TrimSuffix(configVal("JIRA_URL", c), "/")
	email := configVal("JIRA_EMAIL", c)
	token := configVal("JIRA_API_TOKEN", c)
	projectKey := configVal("JIRA_PROJECT_KEY", c)
	issueType := configVal("JIRA_ISSUE_TYPE", c)
	if issueType == "" {
		issueType = "Bug"
	}
	if base == "" || email == "" || token == "" {
		return Result{Success: false, Logs: "[ERROR] JIRA_URL, JIRA_EMAIL, JIRA_API_TOKEN required."}
	}
	if projectKey == "" {
		return Result{Success: false, Logs: "[ERROR] JIRA_PROJECT_KEY required (Plugin config or CI/CD Gateway)."}
	}
	payload := map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": projectKey},
			"summary":   fmt.Sprintf("[QA Capsule] %s", incidentSummary(inc)),
			"description": fmt.Sprintf("Incident from QA Capsule.\n\n%s", inc.Error),
			"issuetype": map[string]string{"name": issueType},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/rest/api/2/issue", bytes.NewReader(body))
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Jira request failed: %v", err)}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == 201 {
		return Result{Success: true, Logs: fmt.Sprintf("[JIRA] Ticket created (HTTP 201)\n%s\nBrowse: %s/browse/...", base, base)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Jira HTTP %d\n%s", resp.StatusCode, raw)}
}

func runTeams(ctx context.Context, m *Manifest, inc IncidentContext, routing ProjectRouting) Result {
	c := mergedConfig(m, routing)
	webhook := configVal("TEAMS_WEBHOOK_URL", c)
	if webhook == "" {
		return Result{Success: false, Logs: "[ERROR] TEAMS_WEBHOOK_URL not configured."}
	}
	payload := map[string]any{
		"@type":    "MessageCard",
		"@context": "http://schema.org/extensions",
		"themeColor": "E81123",
		"summary":  "QA Capsule alert",
		"sections": []map[string]any{{
			"activityTitle":    fmt.Sprintf("SRE incident: %s", incidentSummary(inc)),
			"activitySubtitle": fmt.Sprintf("Status: %s", inc.Status),
			"text":             inc.Error,
			"markdown":         true,
		}},
	}
	code, body, err := httpPostJSON(ctx, webhook, payload, nil)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	if code == 200 || code == 202 {
		return Result{Success: true, Logs: fmt.Sprintf("[TEAMS] Success HTTP %d\n%s", code, body)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Teams HTTP %d\n%s", code, body)}
}

func runPagerDuty(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	key := configVal("PAGERDUTY_ROUTING_KEY", c)
	if key == "" {
		return Result{Success: false, Logs: "[ERROR] PAGERDUTY_ROUTING_KEY not configured."}
	}
	apiURL := configVal("PAGERDUTY_API_URL", c)
	if apiURL == "" {
		apiURL = "https://events.pagerduty.com/v2/enqueue"
	}
	payload := map[string]any{
		"routing_key": key,
		"event_action": "trigger",
		"payload": map[string]any{
			"summary":  fmt.Sprintf("[QA Capsule] %s", incidentSummary(inc)),
			"source":   "qa-capsule",
			"severity": "critical",
		},
	}
	code, body, err := httpPostJSON(ctx, apiURL, payload, nil)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	if code == 200 || code == 202 {
		return Result{Success: true, Logs: fmt.Sprintf("[PAGERDUTY] Queued (HTTP %d)\n%s", code, body)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] PagerDuty HTTP %d\n%s", code, body)}
}

func runOpsgenie(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	key := configVal("OPSGENIE_API_KEY", c)
	if key == "" {
		return Result{Success: false, Logs: "[ERROR] OPSGENIE_API_KEY not configured (env or Plugin config)."}
	}
	apiURL := configVal("OPSGENIE_API_URL", c)
	if apiURL == "" {
		apiURL = "https://api.opsgenie.com"
	}
	u := strings.TrimSuffix(apiURL, "/") + "/v2/alerts"
	payload := map[string]any{
		"message":     fmt.Sprintf("[QA Capsule] %s", incidentSummary(inc)),
		"description": inc.Error,
		"priority":    "P1",
		"source":      "qa-capsule",
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, mustJSON(payload))
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	req.Header.Set("Authorization", "GenieKey "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return Result{Success: true, Logs: fmt.Sprintf("[OPSGENIE] Alert created (HTTP %d)\n%s", resp.StatusCode, raw)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Opsgenie HTTP %d\n%s", resp.StatusCode, raw)}
}

func runVictorOps(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	u := configVal("VICTOROPS_ROUTING_URL", c)
	if u == "" {
		return Result{Success: false, Logs: "[ERROR] VICTOROPS_ROUTING_URL not configured."}
	}
	payload := map[string]any{
		"message_type":        "CRITICAL",
		"entity_display_name": "QA Capsule",
		"state_message":       fmt.Sprintf("[QA Capsule] %s", incidentSummary(inc)),
	}
	code, body, err := httpPostJSON(ctx, u, payload, nil)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	if code >= 200 && code < 300 {
		return Result{Success: true, Logs: fmt.Sprintf("[VICTOROPS] Success HTTP %d\n%s", code, body)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] VictorOps HTTP %d\n%s", code, body)}
}

func runWebhook(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	u := configVal("WEBHOOK_URL", c)
	if u == "" {
		u = configVal("QA_WEBHOOK_URL", c)
	}
	if u == "" {
		return Result{Success: false, Logs: "[ERROR] WEBHOOK_URL not configured."}
	}
	payload := map[string]any{
		"source":   "qa-capsule",
		"event":    "incident.detected",
		"incident": incidentSummary(inc),
		"error":    inc.Error,
		"status":   inc.Status,
		"action":   inc.Action,
	}
	headers := map[string]string{}
	if auth := configVal("WEBHOOK_AUTH_HEADER", c); auth != "" {
		headers["Authorization"] = auth
	}
	code, body, err := httpPostJSON(ctx, u, payload, headers)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	if code >= 200 && code < 300 {
		return Result{Success: true, Logs: fmt.Sprintf("[WEBHOOK] Delivered HTTP %d\n%s", code, body)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Webhook HTTP %d\n%s", code, body)}
}

func runDatadog(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	apiKey := configVal("DD_API_KEY", c)
	if apiKey == "" {
		return Result{Success: false, Logs: "[ERROR] DD_API_KEY not configured."}
	}
	site := configVal("DD_SITE", c)
	if site == "" {
		site = "datadoghq.com"
	}
	payload := map[string]any{
		"title": fmt.Sprintf("QA Capsule: %s", incidentSummary(inc)),
		"text":  inc.Error,
		"alert_type": "error",
		"source_type_name": "qa-capsule",
	}
	url := fmt.Sprintf("https://api.%s/api/v1/events", site)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, mustJSON(payload))
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	req.Header.Set("DD-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 202 {
		return Result{Success: true, Logs: fmt.Sprintf("[DATADOG] Event created\n%s", raw)}
	}
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] Datadog HTTP %d\n%s", resp.StatusCode, raw)}
}

func mustJSON(v any) io.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func runSendGrid(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	apiKey := configVal("SENDGRID_API_KEY", c)
	from := configVal("SENDGRID_FROM", c)
	to := configVal("SENDGRID_TO", c)
	if apiKey == "" || from == "" || to == "" {
		return Result{Success: false, Logs: "[ERROR] SENDGRID_API_KEY, SENDGRID_FROM, SENDGRID_TO required."}
	}
	payload := map[string]any{
		"personalizations": []map[string]any{{"to": []map[string]string{{"email": to}}}},
		"from":             map[string]string{"email": from},
		"subject":          fmt.Sprintf("[QA Capsule] %s", incidentSummary(inc)),
		"content":          []map[string]string{{"type": "text/plain", "value": inc.Error}},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", mustJSON(payload))
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 202 || resp.StatusCode == 200 {
		return Result{Success: true, Logs: "[SENDGRID] Email accepted."}
	}
	raw, _ := io.ReadAll(resp.Body)
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] SendGrid HTTP %d\n%s", resp.StatusCode, raw)}
}

func runGitHub(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	token := configVal("GITHUB_TOKEN", c)
	owner := configVal("GITHUB_OWNER", c)
	repo := configVal("GITHUB_REPO", c)
	workflow := configVal("GITHUB_WORKFLOW_ID", c)
	ref := configVal("GITHUB_REF", c)
	if ref == "" {
		ref = "main"
	}
	if token == "" || owner == "" || repo == "" || workflow == "" {
		return Result{Success: false, Logs: "[ERROR] GITHUB_TOKEN, GITHUB_OWNER, GITHUB_REPO, GITHUB_WORKFLOW_ID required."}
	}
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflow)
	payload := map[string]any{
		"ref": ref,
		"inputs": map[string]string{
			"triggered_by": "qa-capsule",
			"incident":     incidentSummary(inc),
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, mustJSON(payload))
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Success: false, Logs: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		return Result{Success: true, Logs: "[GITHUB] Workflow dispatch accepted."}
	}
	raw, _ := io.ReadAll(resp.Body)
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] GitHub HTTP %d\n%s", resp.StatusCode, raw)}
}

func runSMTP(ctx context.Context, m *Manifest, inc IncidentContext, _ ProjectRouting) Result {
	c := mergedConfig(m, ProjectRouting{})
	host := configVal("SMTP_HOST", c)
	port := configVal("SMTP_PORT", c)
	if port == "" {
		port = "587"
	}
	user := configVal("SMTP_USER", c)
	pass := configVal("SMTP_PASS", c)
	from := configVal("SMTP_FROM", c)
	to := configVal("SMTP_TO", c)
	if host == "" || from == "" || to == "" {
		return Result{Success: false, Logs: "[ERROR] SMTP_HOST, SMTP_FROM, SMTP_TO required."}
	}
	subject := fmt.Sprintf("[QA Capsule] %s", incidentSummary(inc))
	body := fmt.Sprintf("Incident: %s\n\n%s", incidentSummary(inc), inc.Error)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body))
	addr := host + ":" + port
	auth := smtp.PlainAuth("", user, pass, host)
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, from, []string{to}, msg)
	}()
	select {
	case <-ctx.Done():
		return Result{Success: false, Logs: "[ERROR] SMTP timeout."}
	case err := <-done:
		if err != nil {
			return Result{Success: false, Logs: fmt.Sprintf("[ERROR] SMTP: %v", err)}
		}
		return Result{Success: true, Logs: "[SMTP] Message sent."}
	}
}

func runK8sStub(_ context.Context, _ *Manifest, _ IncidentContext, _ ProjectRouting) Result {
	return Result{
		Success: false,
		Logs: `[ERROR] Kubernetes rollout is not available via shell anymore.
[HINT] Use a webhook integration pointing to your GitOps/operator API, or add in-cluster client-go in a future release.`,
	}
}

func runUnsupported(name string) Result {
	return Result{Success: false, Logs: fmt.Sprintf("[ERROR] integration %q not implemented in Go engine yet.", name)}
}
