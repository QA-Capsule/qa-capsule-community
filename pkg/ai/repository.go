package ai

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Repository interface {
	LoadConfig(ctx context.Context) (ProviderConfig, error)
	SaveConfig(ctx context.Context, cfg ProviderConfig) error
	CreateJob(ctx context.Context, incidentID int64, provider ProviderKind, model string) error
	UpdateJobStatus(ctx context.Context, incidentID int64, status JobStatus, errMsg string) error
	SaveReport(ctx context.Context, incidentID int64, res AnalysisResult) error
	GetReport(ctx context.Context, incidentID int64) (*RCAReport, error)
	ListInsights(ctx context.Context, projectName string, limit int) ([]InsightRow, error)
	LoadIncident(ctx context.Context, incidentID int64) (AnalysisInput, error)
	LoadIncidentRecord(ctx context.Context, incidentID int64) (Incident, error)
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) migrateAPIKeyColumn(ctx context.Context) {
	// Idempotent — adds the api_key column only if it does not already exist.
	_, _ = r.db.ExecContext(ctx, `ALTER TABLE ai_provider_config ADD COLUMN api_key TEXT NOT NULL DEFAULT ''`)
}

func (r *SQLiteRepository) LoadConfig(ctx context.Context) (ProviderConfig, error) {
	r.migrateAPIKeyColumn(ctx)
	row := r.db.QueryRowContext(ctx, `
		SELECT provider, model, base_url, api_key_env, COALESCE(api_key,''), max_tokens, timeout_seconds, enabled
		FROM ai_provider_config WHERE id = 1`)
	var cfg ProviderConfig
	var enabled int
	err := row.Scan(&cfg.Provider, &cfg.Model, &cfg.BaseURL, &cfg.APIKeyEnv, &cfg.APIKey, &cfg.MaxTokens, &cfg.TimeoutSeconds, &enabled)
	if err == sql.ErrNoRows {
		return ProviderConfig{Provider: ProviderDisabled, APIKeyEnv: "OPENAI_API_KEY", MaxTokens: 1024, TimeoutSeconds: 45}, nil
	}
	cfg.Enabled = enabled == 1
	return cfg, err
}

func (r *SQLiteRepository) SaveConfig(ctx context.Context, cfg ProviderConfig) error {
	r.migrateAPIKeyColumn(ctx)
	en := 0
	if cfg.Enabled {
		en = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ai_provider_config (id, provider, model, base_url, api_key_env, api_key, max_tokens, timeout_seconds, enabled, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			provider        = excluded.provider,
			model           = excluded.model,
			base_url        = excluded.base_url,
			api_key_env     = excluded.api_key_env,
			api_key         = CASE WHEN excluded.api_key != '' THEN excluded.api_key ELSE api_key END,
			max_tokens      = excluded.max_tokens,
			timeout_seconds = excluded.timeout_seconds,
			enabled         = excluded.enabled,
			updated_at      = CURRENT_TIMESTAMP`,
		string(cfg.Provider), cfg.Model, cfg.BaseURL, cfg.APIKeyEnv, cfg.APIKey,
		cfg.MaxTokens, cfg.TimeoutSeconds, en)
	return err
}

func (r *SQLiteRepository) CreateJob(ctx context.Context, incidentID int64, provider ProviderKind, model string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ai_analysis_jobs (incident_id, status, provider, model, prompt_version)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(incident_id) DO UPDATE SET status = 'pending', provider = excluded.provider, model = excluded.model`,
		incidentID, JobPending, string(provider), model, PromptVersion)
	if err != nil {
		return err
	}
	_, _ = r.db.ExecContext(ctx, `UPDATE incidents SET rca_status = 'pending' WHERE id = ?`, incidentID)
	return nil
}

func (r *SQLiteRepository) UpdateJobStatus(ctx context.Context, incidentID int64, status JobStatus, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ai_analysis_jobs SET status = ?, error_message = ?,
			started_at = CASE WHEN ? = 'running' THEN CURRENT_TIMESTAMP ELSE started_at END,
			completed_at = CASE WHEN ? IN ('completed','failed','skipped') THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE incident_id = ?`,
		string(status), errMsg, string(status), string(status), incidentID)
	return err
}

func (r *SQLiteRepository) SaveReport(ctx context.Context, incidentID int64, res AnalysisResult) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO incident_rca_reports (incident_id, summary, root_cause, suggested_fix, selector_hint, confidence, raw_response_json, tokens_input, tokens_output, latency_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(incident_id) DO UPDATE SET
			summary = excluded.summary, root_cause = excluded.root_cause,
			suggested_fix = excluded.suggested_fix, selector_hint = excluded.selector_hint,
			confidence = excluded.confidence, raw_response_json = excluded.raw_response_json,
			tokens_input = excluded.tokens_input, tokens_output = excluded.tokens_output,
			latency_ms = excluded.latency_ms`,
		incidentID, res.Summary, res.RootCause, res.SuggestedFix, res.SelectorHint, res.Confidence,
		res.RawJSON, res.TokensIn, res.TokensOut, res.LatencyMs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE incidents SET rca_status = 'ready', has_rca = 1 WHERE id = ?`, incidentID)
	return err
}

func (r *SQLiteRepository) GetReport(ctx context.Context, incidentID int64) (*RCAReport, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.incident_id, i.project_name, i.name, r.summary, r.root_cause, r.suggested_fix, r.selector_hint, r.confidence, r.created_at
		FROM incident_rca_reports r
		JOIN incidents i ON i.id = r.incident_id
		WHERE r.incident_id = ?`, incidentID)
	var rep RCAReport
	var created string
	err := row.Scan(&rep.ID, &rep.IncidentID, &rep.ProjectName, &rep.TestName, &rep.Summary, &rep.RootCause,
		&rep.SuggestedFix, &rep.SelectorHint, &rep.Confidence, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	rep.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	return &rep, err
}

func (r *SQLiteRepository) ListInsights(ctx context.Context, projectName string, limit int) ([]InsightRow, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `
		SELECT i.id, i.project_name, i.name, i.status, COALESCE(i.rca_status,''), COALESCE(r.summary,''), i.created_at
		FROM incidents i
		LEFT JOIN incident_rca_reports r ON r.incident_id = i.id
		WHERE i.has_rca = 1 OR i.rca_status != ''`
	args := []interface{}{}
	if projectName != "" {
		q += ` AND i.project_name = ?`
		args = append(args, projectName)
	}
	q += ` ORDER BY i.created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InsightRow
	for rows.Next() {
		var row InsightRow
		var created string
		if err := rows.Scan(&row.IncidentID, &row.ProjectName, &row.TestName, &row.Status, &row.RCAStatus, &row.Summary, &created); err != nil {
			continue
		}
		row.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, row)
	}
	return out, nil
}

func (r *SQLiteRepository) LoadIncident(ctx context.Context, incidentID int64) (AnalysisInput, error) {
	rec, err := r.LoadIncidentRecord(ctx, incidentID)
	if err != nil {
		return AnalysisInput{}, err
	}
	return AnalysisInput{
		IncidentID: rec.ID, ProjectName: rec.ProjectName, TestName: rec.TestName,
		Status: rec.Status, ErrorMessage: rec.ErrorMessage, ConsoleLogs: rec.ConsoleLogs,
		Browser: rec.Browser, OS: rec.OS, Viewport: rec.Viewport,
	}, nil
}

func (r *SQLiteRepository) LoadIncidentRecord(ctx context.Context, incidentID int64) (Incident, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_name, name, status, error_message, console_logs, COALESCE(error_logs,''),
			COALESCE(fingerprint,''), COALESCE(pipeline_run_id,''), COALESCE(execution_time_ms,0),
			COALESCE(browser,''), COALESCE(os,''), COALESCE(viewport,'')
		FROM incidents WHERE id = ?`, incidentID)
	var inc Incident
	err := row.Scan(&inc.ID, &inc.ProjectName, &inc.TestName, &inc.Status, &inc.ErrorMessage, &inc.ConsoleLogs,
		&inc.StackTrace, &inc.Fingerprint, &inc.PipelineRunID, &inc.ExecutionTimeMs, &inc.Browser, &inc.OS, &inc.Viewport)
	if err == sql.ErrNoRows {
		return Incident{}, fmt.Errorf("incident not found")
	}
	if strings.TrimSpace(inc.StackTrace) == "" {
		inc.StackTrace = strings.TrimSpace(inc.ErrorMessage)
	}
	return inc, err
}
