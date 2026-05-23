package core

import (
	"database/sql"
	"fmt"
	"math"
	"time"
)

// DORAMetrics aggregates the four DORA indicators for a time window.
type DORAMetrics struct {
	WindowDays                int           `json:"window_days"`
	Project                   string        `json:"project,omitempty"`
	Deployments               int           `json:"deployments"`
	FailedDeployments         int           `json:"failed_deployments"`
	DeploymentFrequencyPerDay float64       `json:"deployment_frequency_per_day"`
	LeadTimeMinutesMedian     float64       `json:"lead_time_minutes_median"`
	ChangeFailureRate         float64       `json:"change_failure_rate"`
	MTTRMinutes               float64       `json:"mttr_minutes"`
	Series                    []DORABucket  `json:"series"`
	ExternalSignals           int           `json:"external_signals"`
	CorrelatedIncidents       int           `json:"correlated_incidents"`
}

// DORABucket is one chart bucket (week by default).
type DORABucket struct {
	PeriodStart       string  `json:"period_start"`
	Deployments       int     `json:"deployments"`
	FailedDeployments int     `json:"failed_deployments"`
	ChangeFailureRate float64 `json:"change_failure_rate"`
	MTTRMinutes       float64 `json:"mttr_minutes"`
}

// ComputeDORAMetrics calculates DORA KPIs from pipeline_runs and incidents.
func ComputeDORAMetrics(project string, from, to time.Time) DORAMetrics {
	days := to.Sub(from).Hours() / 24
	if days < 1 {
		days = 1
	}
	out := DORAMetrics{
		WindowDays: int(math.Ceil(days)),
		Project:    project,
	}
	if DB == nil {
		return out
	}
	fromS := from.UTC().Format("2006-01-02 15:04:05")
	toS := to.UTC().Format("2006-01-02 15:04:05")

	projFilter := ""
	args := []interface{}{fromS, toS}
	if project != "" {
		projFilter = " AND project_name = ? "
		args = append(args, project)
	}

	var deploys, failed int
	qDeploy := fmt.Sprintf(`SELECT COUNT(*) FROM pipeline_runs WHERE started_at >= ? AND started_at <= ? %s`, projFilter)
	_ = DB.QueryRow(qDeploy, args...).Scan(&deploys)

	qFailed := fmt.Sprintf(`SELECT COUNT(*) FROM pipeline_runs WHERE started_at >= ? AND started_at <= ? AND outcome = 'failure' %s`, projFilter)
	_ = DB.QueryRow(qFailed, args...).Scan(&failed)

	out.Deployments = deploys
	out.FailedDeployments = failed
	out.DeploymentFrequencyPerDay = float64(deploys) / days
	if deploys > 0 {
		out.ChangeFailureRate = float64(failed) / float64(deploys)
	}

	leadArgs := []interface{}{fromS, toS}
	leadFilter := ""
	if project != "" {
		leadFilter = " AND i.project_name = ? "
		leadArgs = append(leadArgs, project)
	}
	var leadMedian sql.NullFloat64
	_ = DB.QueryRow(fmt.Sprintf(`
		SELECT AVG((julianday(i.created_at) - julianday(p.started_at)) * 24 * 60)
		FROM incidents i
		INNER JOIN pipeline_runs p ON p.project_name = i.project_name AND p.pipeline_run_id = i.pipeline_run_id
		WHERE i.created_at >= ? AND i.created_at <= ? %s`, leadFilter), leadArgs...).Scan(&leadMedian)
	if leadMedian.Valid {
		out.LeadTimeMinutesMedian = leadMedian.Float64
	}

	mttrArgs := []interface{}{fromS, toS}
	mttrFilter := " AND created_at >= ? AND created_at <= ? "
	if project != "" {
		mttrFilter += " AND project_name = ? "
		mttrArgs = append(mttrArgs, project)
	}
	var mttr sql.NullFloat64
	_ = DB.QueryRow(`
		SELECT AVG((julianday(resolved_at) - julianday(created_at)) * 24 * 60)
		FROM incidents
		WHERE is_resolved = 1 AND resolved_at IS NOT NULL`+mttrFilter, mttrArgs...).Scan(&mttr)
	if mttr.Valid {
		out.MTTRMinutes = mttr.Float64
	}

	out.Series = loadDORASeries(project, from, to)

	sigArgs := []interface{}{fromS, toS}
	sigFilter := ""
	if project != "" {
		sigFilter = " AND project_name = ? "
		sigArgs = append(sigArgs, project)
	}
	_ = DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM external_signals WHERE fired_at >= ? AND fired_at <= ? %s`, sigFilter), sigArgs...).Scan(&out.ExternalSignals)
	out.CorrelatedIncidents = countCorrelatedIncidents(project, from, to)

	return out
}

func loadDORASeries(project string, from, to time.Time) []DORABucket {
	bucketExpr := `(strftime('%Y-%W', started_at))`
	fromS := from.UTC().Format("2006-01-02 15:04:05")
	toS := to.UTC().Format("2006-01-02 15:04:05")
	args := []interface{}{fromS, toS}
	filter := ""
	if project != "" {
		filter = " AND project_name = ? "
		args = append(args, project)
	}
	rows, err := DB.Query(fmt.Sprintf(`
		SELECT %s AS period,
			COUNT(*) AS deployments,
			SUM(CASE WHEN outcome = 'failure' THEN 1 ELSE 0 END) AS failed
		FROM pipeline_runs
		WHERE started_at >= ? AND started_at <= ? %s
		GROUP BY period ORDER BY period`, bucketExpr, filter), args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var series []DORABucket
	for rows.Next() {
		var b DORABucket
		rows.Scan(&b.PeriodStart, &b.Deployments, &b.FailedDeployments)
		if b.Deployments > 0 {
			b.ChangeFailureRate = float64(b.FailedDeployments) / float64(b.Deployments)
		}
		series = append(series, b)
	}
	return series
}

func countCorrelatedIncidents(project string, from, to time.Time) int {
	fromS := from.UTC().Format("2006-01-02 15:04:05")
	toS := to.UTC().Format("2006-01-02 15:04:05")
	args := []interface{}{fromS, toS}
	filter := ""
	if project != "" {
		filter = " AND e.project_name = ? "
		args = append(args, project)
	}
	var n int
	_ = DB.QueryRow(fmt.Sprintf(`
		SELECT COUNT(DISTINCT c.incident_id)
		FROM external_signal_correlations c
		INNER JOIN external_signals e ON e.id = c.signal_id
		WHERE e.fired_at >= ? AND e.fired_at <= ? %s`, filter), args...).Scan(&n)
	return n
}
