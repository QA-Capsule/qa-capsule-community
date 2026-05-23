package integrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanExecution_flakyTrueBranch(t *testing.T) {
	pluginsDir := filepath.Join("..", "..", "plugins")
	if _, err := os.Stat(pluginsDir); err != nil {
		t.Skip("plugins directory missing")
	}
	reg, err := LoadRegistry(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(reg)
	doc := &WorkflowDocument{
		Version: 1,
		Enabled: true,
		Entry:   "start",
		Nodes: map[string]WorkflowNode{
			"start": {Type: "trigger"},
			"cond": {
				Type: "condition",
				When: &ConditionExpr{Op: "tag", Match: "prefix", Value: "[FLAKY]", Field: "incident.name"},
			},
			"slack": {Type: "action", FilePath: "slack/slack-notifier.json"},
			"jira":  {Type: "action", FilePath: "jira/jira-ticket.json"},
		},
		Edges: []WorkflowEdge{
			{From: "start", To: "cond"},
			{From: "cond", To: "slack", When: "true"},
			{From: "cond", To: "jira", When: "false"},
		},
	}
	wctx := WorkflowContext{
		Incident: IncidentContext{Name: "[FLAKY] pay", Status: "CRITICAL"},
		Tags:     []string{"FLAKY"},
		Allowed:  map[string]bool{"slack/slack-notifier.json": true, "jira/jira-ticket.json": true},
	}
	plan := NewWorkflowEngine(engine).PlanExecution(doc, wctx)
	if len(plan.Actions) != 1 || plan.Actions[0] != "slack/slack-notifier.json" {
		t.Fatalf("expected slack on true branch, got %#v skipped=%#v", plan.Actions, plan.Skipped)
	}
}

func TestPlanExecution_continuesAfterSkippedAction(t *testing.T) {
	pluginsDir := filepath.Join("..", "..", "plugins")
	if _, err := os.Stat(pluginsDir); err != nil {
		t.Skip("plugins directory missing")
	}
	reg, err := LoadRegistry(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(reg)
	doc := &WorkflowDocument{
		Version: 1,
		Enabled: true,
		Entry:   "start",
		Nodes: map[string]WorkflowNode{
			"start": {Type: "trigger"},
			"slack": {Type: "action", FilePath: "slack/slack-notifier.json"},
			"jira":  {Type: "action", FilePath: "jira/jira-ticket.json"},
		},
		Edges: []WorkflowEdge{
			{From: "start", To: "slack"},
			{From: "slack", To: "jira"},
		},
	}
	wctx := WorkflowContext{
		Incident: IncidentContext{Name: "test", Status: "CRITICAL"},
		Allowed:  map[string]bool{"jira/jira-ticket.json": true},
	}
	plan := NewWorkflowEngine(engine).PlanExecution(doc, wctx)
	if len(plan.Actions) != 1 || plan.Actions[0] != "jira/jira-ticket.json" {
		t.Fatalf("expected jira after skipped slack, got %#v skipped=%#v", plan.Actions, plan.Skipped)
	}
}
