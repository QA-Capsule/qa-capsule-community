package core

import (
	"database/sql"
	"log/slog"
)

const perfDegradationRatio = 1.5 // 50% above historical average

// RecordExecutionMetric stores a timing sample for regression analysis.
func RecordExecutionMetric(projectName, testName, fingerprint string, executionTimeMs int64, status string) {
	if executionTimeMs <= 0 {
		return
	}
	_, err := DB.Exec(`INSERT INTO test_execution_metrics (project_name, test_name, fingerprint, execution_time_ms, status, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		projectName, testName, fingerprint, executionTimeMs, status)
	if err != nil {
		slog.Warn("failed to record execution metric", "error", err)
	}
}

// IsPerformanceDegraded returns true when duration exceeds 150% of the historical mean.
func IsPerformanceDegraded(projectName, fingerprint string, executionTimeMs int64) bool {
	if executionTimeMs <= 0 || fingerprint == "" {
		return false
	}
	var avgMs sql.NullFloat64
	err := DB.QueryRow(`
		SELECT AVG(execution_time_ms) FROM test_execution_metrics
		WHERE project_name = ? AND fingerprint = ? AND execution_time_ms > 0
		AND created_at > datetime('now', '-30 days')`,
		projectName, fingerprint).Scan(&avgMs)
	if err != nil || !avgMs.Valid || avgMs.Float64 <= 0 {
		return false
	}
	threshold := avgMs.Float64 * perfDegradationRatio
	return float64(executionTimeMs) > threshold
}
