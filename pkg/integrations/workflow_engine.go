package integrations

import (
	"context"
	"log/slog"
)

// WorkflowEngine walks a DAG and runs native integration actions.
type WorkflowEngine struct {
	engine *Engine
}

<<<<<<< HEAD
// WorkflowExecutionPlan describes nodes visited during a dry-run.
type WorkflowExecutionPlan struct {
	Visited []string `json:"visited"`
	Actions []string `json:"actions"`
	Skipped []string `json:"skipped"`
}

=======
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
func NewWorkflowEngine(engine *Engine) *WorkflowEngine {
	return &WorkflowEngine{engine: engine}
}

// Execute traverses the workflow from entry and runs action nodes sequentially per branch.
func (we *WorkflowEngine) Execute(ctx context.Context, doc *WorkflowDocument, wctx WorkflowContext, routing ProjectRouting) {
	if we == nil || we.engine == nil || doc == nil || !IsWorkflowActive(doc) {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
<<<<<<< HEAD
	visited := make(map[string]bool)
	we.walk(ctx, doc, doc.Entry, wctx, routing, visited)
}

// PlanExecution dry-runs the DAG and returns which action file paths would run.
func (we *WorkflowEngine) PlanExecution(doc *WorkflowDocument, wctx WorkflowContext) WorkflowExecutionPlan {
	plan := WorkflowExecutionPlan{}
	if we == nil || doc == nil || doc.Entry == "" {
		return plan
	}
	visited := make(map[string]bool)
	we.walkPlan(doc, doc.Entry, wctx, &plan, visited)
	return plan
}

func (we *WorkflowEngine) walk(ctx context.Context, doc *WorkflowDocument, nodeID string, wctx WorkflowContext, routing ProjectRouting, visited map[string]bool) {
	if ctx.Err() != nil {
		return
	}
	if visited[nodeID] {
		slog.Warn("workflow cycle detected at runtime", "node", nodeID)
		return
	}
	visited[nodeID] = true

=======
	we.walk(ctx, doc, doc.Entry, wctx, routing)
}

func (we *WorkflowEngine) walk(ctx context.Context, doc *WorkflowDocument, nodeID string, wctx WorkflowContext, routing ProjectRouting) {
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
	node, ok := doc.Nodes[nodeID]
	if !ok {
		return
	}
	switch node.Type {
	case "trigger":
		for _, next := range outgoingEdges(doc.Edges, nodeID, "") {
<<<<<<< HEAD
			we.walk(ctx, doc, next, wctx, routing, visited)
		}
	case "condition":
		result := EvaluateCondition(node.When, wctx)
		branch := "false"
		if result {
			branch = "true"
		}
		slog.Debug("workflow condition", "node", nodeID, "result", result, "branch", branch)
		for _, next := range outgoingEdges(doc.Edges, nodeID, branch) {
			we.walk(ctx, doc, next, wctx, routing, visited)
		}
	case "action":
		skipReason := actionSkipReason(wctx, node.FilePath)
		if skipReason == "" {
			m, ok := we.engine.registry.GetByPath(node.FilePath)
			if !ok {
				slog.Warn("workflow action skipped: unknown manifest", "path", node.FilePath)
			} else {
				inc := wctx.Incident
				if inc.Action == "" {
					inc.Action = "AUTO_EVENT:" + inc.Name
				}
				slog.Info("workflow action", "path", node.FilePath, "label", node.Label)
				res := we.engine.runManifest(ctx, m, inc, routing)
				if !res.Success {
					slog.Warn("workflow action failed", "path", node.FilePath, "logs", res.Logs)
				}
			}
		} else {
			slog.Warn("workflow action skipped", "path", node.FilePath, "reason", skipReason)
		}
		for _, next := range outgoingEdges(doc.Edges, nodeID, "") {
			we.walk(ctx, doc, next, wctx, routing, visited)
		}
	}
}

func (we *WorkflowEngine) walkPlan(doc *WorkflowDocument, nodeID string, wctx WorkflowContext, plan *WorkflowExecutionPlan, visited map[string]bool) {
	if visited[nodeID] {
		return
	}
	visited[nodeID] = true
	node, ok := doc.Nodes[nodeID]
	if !ok {
		return
	}
	plan.Visited = append(plan.Visited, nodeID+":"+node.Type)
	switch node.Type {
	case "trigger":
		for _, next := range outgoingEdges(doc.Edges, nodeID, "") {
			we.walkPlan(doc, next, wctx, plan, visited)
=======
			we.walk(ctx, doc, next, wctx, routing)
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
		}
	case "condition":
		result := EvaluateCondition(node.When, wctx)
		branch := "false"
		if result {
			branch = "true"
		}
		for _, next := range outgoingEdges(doc.Edges, nodeID, branch) {
<<<<<<< HEAD
			we.walkPlan(doc, next, wctx, plan, visited)
		}
	case "action":
		if reason := actionSkipReason(wctx, node.FilePath); reason != "" {
			plan.Skipped = append(plan.Skipped, node.FilePath+": "+reason)
		} else if node.FilePath == "" {
			plan.Skipped = append(plan.Skipped, "action: missing file_path")
		} else {
			plan.Actions = append(plan.Actions, node.FilePath)
		}
		for _, next := range outgoingEdges(doc.Edges, nodeID, "") {
			we.walkPlan(doc, next, wctx, plan, visited)
		}
	}
}

func actionSkipReason(wctx WorkflowContext, filePath string) string {
	if wctx.Allowed == nil {
		return ""
	}
	if len(wctx.Allowed) == 0 {
		return "gateway has no allowed plugins"
	}
	if filePath == "" {
		return "missing file_path"
	}
	if !wctx.Allowed[filePath] {
		return "not configured on gateway"
	}
	return ""
}
=======
			we.walk(ctx, doc, next, wctx, routing)
		}
	case "action":
		if len(wctx.Allowed) > 0 && !wctx.Allowed[node.FilePath] {
			slog.Warn("workflow action skipped: not allowed on gateway", "path", node.FilePath)
			return
		}
		m, ok := we.engine.registry.GetByPath(node.FilePath)
		if !ok {
			slog.Warn("workflow action skipped: unknown manifest", "path", node.FilePath)
			return
		}
		inc := wctx.Incident
		if inc.Action == "" {
			inc.Action = "AUTO_EVENT:" + inc.Name
		}
		slog.Info("workflow action", "path", node.FilePath, "label", node.Label)
		res := we.engine.runManifest(ctx, m, inc, routing)
		if !res.Success {
			slog.Warn("workflow action failed", "path", node.FilePath, "logs", res.Logs)
		}
		for _, next := range outgoingEdges(doc.Edges, nodeID, "") {
			we.walk(ctx, doc, next, wctx, routing)
		}
	}
}
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
