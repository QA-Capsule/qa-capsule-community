package integrations

import (
	"context"
	"time"
)

const DefaultHTTPTimeout = 30 * time.Second

// IncidentContext is passed to every integration (auto-trigger or manual test).
type IncidentContext struct {
	Name        string
	Error       string
	ConsoleLogs string
	Status      string
	Action      string // MANUAL or AUTO_EVENT:...
}

// ProjectRouting is injected from CI/CD gateway (per pipeline).
type ProjectRouting struct {
	SlackChannel    string
	JiraProjectKey  string
	TeamsWebhookURL string
	Values          map[string]string // all routing keys from gateway entries + legacy columns
}

// Result is human-readable output for the Plugin Engine UI.
type Result struct {
	Success bool
	Logs    string
}

func (r Result) String() string {
	status := "[EXIT STATUS] SUCCESS"
	if !r.Success {
		status = "[EXIT STATUS] ERROR"
	}
	return r.Logs + "\n" + status
}

func HTTPContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, DefaultHTTPTimeout)
}

func incidentSummary(inc IncidentContext) string {
	if inc.Name != "" {
		return inc.Name
	}
	return "QA Capsule incident"
}
