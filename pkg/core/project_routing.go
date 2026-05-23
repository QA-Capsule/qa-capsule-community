package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/integrations"
)

// SRERoutingEntry is one plugin routing block on a CI/CD gateway project.
type SRERoutingEntry struct {
	Integration string            `json:"integration"`
	FilePath    string            `json:"file_path"`
	Name        string            `json:"name,omitempty"`
	Values      map[string]string `json:"values"`
}

// ProjectAlertContext loads routing key/value pairs and optional allowed plugin paths for auto-trigger.
func ProjectAlertContext(projectName string) (map[string]string, map[string]bool) {
	ctx := make(map[string]string)
	var slack, jira, teams, routingJSON sql.NullString
	err := DB.QueryRow(`SELECT slack_channel, jira_project_key, teams_webhook, sre_routing_json FROM projects WHERE name = ?`,
		projectName).Scan(&slack, &jira, &teams, &routingJSON)
	if err != nil {
		return ctx, nil
	}
	if slack.Valid && slack.String != "" {
		ctx["SLACK_CHANNEL"] = slack.String
	}
	if jira.Valid && jira.String != "" {
		ctx["JIRA_PROJECT_KEY"] = jira.String
	}
	if teams.Valid && teams.String != "" {
		ctx["TEAMS_WEBHOOK_URL"] = teams.String
	}

	var allowed map[string]bool
	if routingJSON.Valid {
		entries := ParseSRERoutingJSON(routingJSON.String)
		allowed = make(map[string]bool)
		for _, e := range entries {
			if e.FilePath != "" {
				allowed[e.FilePath] = true
			}
			for k, v := range e.Values {
				if strings.TrimSpace(v) != "" {
					ctx[k] = v
				}
			}
		}
		if len(allowed) == 0 {
			if slack.Valid && slack.String != "" {
				allowed["slack/slack-notifier.json"] = true
			}
			if jira.Valid && jira.String != "" {
				allowed["jira/jira-ticket.json"] = true
			}
			if teams.Valid && teams.String != "" {
				allowed["teams/teams.json"] = true
			}
		}
	}
	return ctx, allowed
}

// ParseSRERoutingJSON decodes stored routing or returns empty slice.
func ParseSRERoutingJSON(raw string) []SRERoutingEntry {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var entries []SRERoutingEntry
	if json.Unmarshal([]byte(raw), &entries) != nil {
		return nil
	}
	return entries
}

// ValidateSRERoutingEntries rejects routing blocks for integrations not activated by a Manager (Active + auto_run).
func ValidateSRERoutingEntries(entries []SRERoutingEntry) error {
	if len(entries) == 0 {
		return nil
	}
	reg := RemediationRegistry()
	if reg == nil {
		return fmt.Errorf("remediation engine not initialized")
	}
	for _, e := range entries {
		if e.FilePath == "" {
			continue
		}
		m, ok := reg.GetByPath(e.FilePath)
		if !ok {
			return fmt.Errorf("unknown integration: %s", e.FilePath)
		}
		if !integrations.IsActiveForRouting(m.Status, m.RoutingEnabled, m.AutoRun) {
			return fmt.Errorf("integration %q is not activated by a Manager in Plugin Engine", m.Name)
		}
	}
	return nil
}

// MarshalSRERoutingJSON encodes routing entries for persistence.
func MarshalSRERoutingJSON(entries []SRERoutingEntry) string {
	if len(entries) == 0 {
		return "[]"
	}
	b, err := json.Marshal(entries)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// SyncLegacyRoutingColumns copies first matching entry values into projects columns for backward compatibility.
func SyncLegacyRoutingColumns(entries []SRERoutingEntry) (slack, jira, teams string) {
	for _, e := range entries {
		if v := e.Values["SLACK_CHANNEL"]; slack == "" && v != "" {
			slack = v
		}
		if v := e.Values["JIRA_PROJECT_KEY"]; jira == "" && v != "" {
			jira = v
		}
		if v := e.Values["TEAMS_WEBHOOK_URL"]; teams == "" && v != "" {
			teams = v
		}
		if v := e.Values["TEAMS_WEBHOOK"]; teams == "" && v != "" {
			teams = v
		}
	}
	return slack, jira, teams
}
