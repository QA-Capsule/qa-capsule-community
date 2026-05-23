package core

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// ExternalSignal is a normalized observability alert (e.g. Prometheus).
type ExternalSignal struct {
	ID          int64             `json:"id"`
	ProjectName string            `json:"project_name"`
	Source      string            `json:"source"`
	SignalName  string            `json:"signal_name"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	FiredAt     time.Time         `json:"fired_at"`
}

// IngestPrometheusWebhook stores Alertmanager-style payloads and correlates to incidents.
func IngestPrometheusWebhook(projectName string, raw []byte) (signalID int64, correlated int, err error) {
	var payload struct {
		Status string `json:"status"`
		Alerts []struct {
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			StartsAt    string            `json:"startsAt"`
		} `json:"alerts"`
	}
	if err = json.Unmarshal(raw, &payload); err != nil {
		return 0, 0, err
	}
	if len(payload.Alerts) == 0 {
		return 0, 0, nil
	}
	a := payload.Alerts[0]
	if projectName == "" {
		projectName = a.Labels["project"]
		if projectName == "" {
			projectName = a.Labels["project_name"]
		}
	}
	name := a.Labels["alertname"]
	if name == "" {
		name = "prometheus_alert"
	}
	sev := a.Labels["severity"]
	if sev == "" {
		sev = "warning"
	}
	summary := a.Annotations["summary"]
	if summary == "" {
		summary = a.Annotations["description"]
	}
	firedAt := time.Now().UTC()
	if a.StartsAt != "" {
		if t, e := time.Parse(time.RFC3339, a.StartsAt); e == nil {
			firedAt = t.UTC()
		}
	}
	labelsJSON, _ := json.Marshal(a.Labels)
	res, err := DB.Exec(`
		INSERT INTO external_signals (project_name, source, signal_name, severity, labels_json, summary, fired_at)
		VALUES (?, 'prometheus', ?, ?, ?, ?, ?)`,
		projectName, name, sev, string(labelsJSON), summary, firedAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, 0, err
	}
	signalID, _ = res.LastInsertId()
	if projectName != "" {
		correlated = correlateSignalToIncidents(signalID, projectName, firedAt)
	}
	return signalID, correlated, nil
}

func correlateSignalToIncidents(signalID int64, projectName string, firedAt time.Time) int {
	windowStart := firedAt.Add(-15 * time.Minute).Format("2006-01-02 15:04:05")
	windowEnd := firedAt.Add(15 * time.Minute).Format("2006-01-02 15:04:05")
	rows, err := DB.Query(`
		SELECT id FROM incidents
		WHERE project_name = ? AND created_at >= ? AND created_at <= ?`,
		projectName, windowStart, windowEnd)
	if err != nil {
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var incID int64
		if rows.Scan(&incID) != nil {
			continue
		}
		_, _ = DB.Exec(`INSERT OR IGNORE INTO external_signal_correlations (signal_id, incident_id) VALUES (?, ?)`, signalID, incID)
		n++
	}
	return n
}

// ListExternalSignals returns recent signals for API/UI.
func ListExternalSignals(ctx context.Context, project string, limit int) ([]ExternalSignal, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, project_name, source, signal_name, severity, labels_json, summary, fired_at
		FROM external_signals`
	args := []interface{}{}
	if project != "" {
		q += ` WHERE project_name = ?`
		args = append(args, project)
	}
	q += ` ORDER BY fired_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExternalSignal
	for rows.Next() {
		var s ExternalSignal
		var labelsJSON, summary string
		var fired string
		if err := rows.Scan(&s.ID, &s.ProjectName, &s.Source, &s.SignalName, &s.Severity, &labelsJSON, &summary, &fired); err != nil {
			continue
		}
		s.Summary = summary
		_ = json.Unmarshal([]byte(labelsJSON), &s.Labels)
		if t, e := time.Parse("2006-01-02 15:04:05", fired); e == nil {
			s.FiredAt = t
		}
		out = append(out, s)
	}
	return out, nil
}

// SignalCorrelationRow links a signal to an incident for the DORA UI.
type SignalCorrelationRow struct {
	SignalID    int64  `json:"signal_id"`
	IncidentID  int64  `json:"incident_id"`
	SignalName  string `json:"signal_name"`
	IncidentName string `json:"incident_name"`
	FiredAt     string `json:"fired_at"`
}

// ListSignalCorrelations returns recent correlations.
func ListSignalCorrelations(project string, limit int) []SignalCorrelationRow {
	if limit <= 0 {
		limit = 30
	}
	filter := ""
	args := []interface{}{}
	if project != "" {
		filter = " WHERE e.project_name = ? "
		args = append(args, project)
	}
	args = append(args, limit)
	rows, err := DB.Query(`
		SELECT c.signal_id, c.incident_id, e.signal_name, i.name, e.fired_at
		FROM external_signal_correlations c
		INNER JOIN external_signals e ON e.id = c.signal_id
		INNER JOIN incidents i ON i.id = c.incident_id`+filter+`
		ORDER BY e.fired_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []SignalCorrelationRow
	for rows.Next() {
		var r SignalCorrelationRow
		var fired string
		rows.Scan(&r.SignalID, &r.IncidentID, &r.SignalName, &r.IncidentName, &fired)
		r.FiredAt = fired
		out = append(out, r)
	}
	return out
}

// ResolvePrometheusProject extracts project from labels or query.
func ResolvePrometheusProject(labels map[string]string, queryProject string) string {
	if p := strings.TrimSpace(queryProject); p != "" {
		return p
	}
	if labels == nil {
		return ""
	}
	if p := labels["project"]; p != "" {
		return p
	}
	return labels["project_name"]
}
