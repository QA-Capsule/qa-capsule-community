package integrations

import (
	"context"
	"log/slog"
)

// WorkflowEngine walks a DAG and runs native integration actions.
type WorkflowEngine struct {
	engine *Engine
}

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
	we.walk(ctx, doc, doc.Entry, wctx, routing)
}

func (we *WorkflowEngine) walk(ctx context.Context, doc *WorkflowDocument, nodeID string, wctx WorkflowContext, routing ProjectRouting) {
	node, ok := doc.Nodes[nodeID]
	if !ok {
		return
	}
	switch node.Type {
	case "trigger":
		for _, next := range outgoingEdges(doc.Edges, nodeID, "") {
			we.walk(ctx, doc, next, wctx, routing)
		}
	case "condition":
		result := EvaluateCondition(node.When, wctx)
		branch := "false"
		if result {
			branch = "true"
		}
		for _, next := range outgoingEdges(doc.Edges, nodeID, branch) {
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
