package quarantine

import (
	"context"
	"database/sql"
	"time"
)

type Repository interface {
	UpsertPipelineRun(ctx context.Context, projectName, runID, commitSHA, branch string) error
	GetStats(ctx context.Context, projectName, identityFP string) (*StabilityStats, error)
	UpsertStats(ctx context.Context, s StabilityStats) error
	InsertTransition(ctx context.Context, ev TransitionEvent) error
	ActiveEntry(ctx context.Context, projectName, identityFP string) (*Entry, error)
	CreateEntry(ctx context.Context, e Entry) error
	LiftEntry(ctx context.Context, projectName, identityFP, liftedBy string) error
	ListActive(ctx context.Context, projectName string) ([]Entry, error)
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) UpsertPipelineRun(ctx context.Context, projectName, runID, commitSHA, branch string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pipeline_runs (project_name, pipeline_run_id, commit_sha, branch)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(project_name, pipeline_run_id) DO UPDATE SET
			commit_sha = excluded.commit_sha,
			branch = excluded.branch`,
		projectName, runID, commitSHA, branch)
	return err
}

func (r *SQLiteRepository) GetStats(ctx context.Context, projectName, identityFP string) (*StabilityStats, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_name, test_identity_fingerprint, test_name, total_runs, fail_count, pass_count,
			flaky_count, last_status, last_commit_sha, last_pipeline_run_id, consecutive_failures, last_seen_at
		FROM test_stability_stats WHERE project_name = ? AND test_identity_fingerprint = ?`,
		projectName, identityFP)
	var s StabilityStats
	var lastSeen string
	err := row.Scan(&s.ID, &s.ProjectName, &s.TestIdentityFingerprint, &s.TestName, &s.TotalRuns, &s.FailCount,
		&s.PassCount, &s.FlakyCount, &s.LastStatus, &s.LastCommitSHA, &s.LastPipelineRunID, &s.ConsecutiveFailures, &lastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.LastSeenAt, _ = time.Parse("2006-01-02 15:04:05", lastSeen)
	return &s, nil
}

func (r *SQLiteRepository) UpsertStats(ctx context.Context, s StabilityStats) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO test_stability_stats (
			project_name, test_identity_fingerprint, test_name, total_runs, fail_count, pass_count,
			flaky_count, last_status, last_commit_sha, last_pipeline_run_id, consecutive_failures, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(project_name, test_identity_fingerprint) DO UPDATE SET
			test_name = excluded.test_name,
			total_runs = excluded.total_runs,
			fail_count = excluded.fail_count,
			pass_count = excluded.pass_count,
			flaky_count = excluded.flaky_count,
			last_status = excluded.last_status,
			last_commit_sha = excluded.last_commit_sha,
			last_pipeline_run_id = excluded.last_pipeline_run_id,
			consecutive_failures = excluded.consecutive_failures,
			last_seen_at = CURRENT_TIMESTAMP`,
		s.ProjectName, s.TestIdentityFingerprint, s.TestName, s.TotalRuns, s.FailCount, s.PassCount,
		s.FlakyCount, s.LastStatus, s.LastCommitSHA, s.LastPipelineRunID, s.ConsecutiveFailures)
	return err
}

func (r *SQLiteRepository) InsertTransition(ctx context.Context, ev TransitionEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO test_state_transitions (
			project_name, test_identity_fingerprint, pipeline_run_id, commit_sha,
			from_status, to_status, incident_fingerprint)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.ProjectName, ev.TestIdentityFingerprint, ev.PipelineRunID, ev.CommitSHA,
		ev.FromStatus, ev.ToStatus, ev.IncidentFingerprint)
	return err
}

func (r *SQLiteRepository) ActiveEntry(ctx context.Context, projectName, identityFP string) (*Entry, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_name, test_identity_fingerprint, test_name, reason, source,
			incident_id, commit_sha_at_quarantine, expires_at, is_active, created_by, created_at
		FROM test_quarantine_entries
		WHERE project_name = ? AND test_identity_fingerprint = ? AND is_active = 1
		ORDER BY id DESC LIMIT 1`, projectName, identityFP)
	var e Entry
	var incID sql.NullInt64
	var expires sql.NullString
	var created string
	err := row.Scan(&e.ID, &e.ProjectName, &e.TestIdentityFingerprint, &e.TestName, &e.Reason, &e.Source,
		&incID, &e.CommitSHAAtQuarantine, &expires, &e.IsActive, &e.CreatedBy, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if incID.Valid {
		v := incID.Int64
		e.IncidentID = &v
	}
	if expires.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", expires.String)
		e.ExpiresAt = &t
	}
	e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	return &e, nil
}

func (r *SQLiteRepository) CreateEntry(ctx context.Context, e Entry) error {
	if err := r.LiftEntry(ctx, e.ProjectName, e.TestIdentityFingerprint, "system"); err != nil {
		return err
	}
	var inc interface{}
	if e.IncidentID != nil {
		inc = *e.IncidentID
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO test_quarantine_entries (
			project_name, test_identity_fingerprint, test_name, reason, source,
			incident_id, commit_sha_at_quarantine, is_active, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)`,
		e.ProjectName, e.TestIdentityFingerprint, e.TestName, string(e.Reason), string(e.Source),
		inc, e.CommitSHAAtQuarantine, e.CreatedBy)
	return err
}

func (r *SQLiteRepository) LiftEntry(ctx context.Context, projectName, identityFP, liftedBy string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE test_quarantine_entries SET is_active = 0, lifted_at = CURRENT_TIMESTAMP, lifted_by = ?
		WHERE project_name = ? AND test_identity_fingerprint = ? AND is_active = 1`,
		liftedBy, projectName, identityFP)
	return err
}

func (r *SQLiteRepository) ListActive(ctx context.Context, projectName string) ([]Entry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_name, test_identity_fingerprint, test_name, reason, source,
			incident_id, commit_sha_at_quarantine, expires_at, is_active, created_by, created_at
		FROM test_quarantine_entries
		WHERE project_name = ? AND is_active = 1 ORDER BY created_at DESC`, projectName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var incID sql.NullInt64
		var expires sql.NullString
		var created string
		if err := rows.Scan(&e.ID, &e.ProjectName, &e.TestIdentityFingerprint, &e.TestName, &e.Reason, &e.Source,
			&incID, &e.CommitSHAAtQuarantine, &expires, &e.IsActive, &e.CreatedBy, &created); err != nil {
			continue
		}
		if incID.Valid {
			v := incID.Int64
			e.IncidentID = &v
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, e)
	}
	return out, nil
}
