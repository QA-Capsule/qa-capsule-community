package ai

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Service struct {
	repo     Repository
	analyzer Analyzer
}

func NewService(repo Repository, analyzer Analyzer) *Service {
	if analyzer == nil {
		analyzer = HTTPAnalyzer{}
	}
	return &Service{repo: repo, analyzer: analyzer}
}

func (s *Service) EnqueueForIncident(incidentID int64) {
	if s == nil || incidentID <= 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.RunAnalysis(ctx, incidentID); err != nil {
			slog.Warn("RCA analysis failed", "incident_id", incidentID, "error", err)
		}
	}()
}

func (s *Service) RunAnalysis(ctx context.Context, incidentID int64) error {
	cfg, err := s.repo.LoadConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled || cfg.Provider == ProviderDisabled {
		_ = s.repo.CreateJob(ctx, incidentID, ProviderDisabled, "")
		_ = s.repo.UpdateJobStatus(ctx, incidentID, JobSkipped, "AI disabled")
		return nil
	}

	in, err := s.repo.LoadIncident(ctx, incidentID)
	if err != nil {
		return err
	}
	if !shouldAnalyze(in.Status) {
		_ = s.repo.UpdateJobStatus(ctx, incidentID, JobSkipped, "non-failure status")
		return nil
	}

	model := cfg.Model
	if model == "" {
		model = defaultModel(cfg.Provider)
	}
	_ = s.repo.CreateJob(ctx, incidentID, cfg.Provider, model)
	_ = s.repo.UpdateJobStatus(ctx, incidentID, JobRunning, "")

	start := time.Now()
	var analyzer Analyzer = s.analyzer
	if cfg.Provider == ProviderDisabled {
		analyzer = StubAnalyzer{}
	}

	res, err := analyzer.Analyze(ctx, cfg, in)
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		_ = s.repo.UpdateJobStatus(ctx, incidentID, JobFailed, err.Error())
		stub, _ := StubAnalyzer{}.Analyze(ctx, cfg, in)
		stub.Summary = "AI provider error: " + err.Error() + ". " + stub.Summary
		_ = s.repo.SaveReport(ctx, incidentID, stub)
		return err
	}
	if err := s.repo.SaveReport(ctx, incidentID, res); err != nil {
		return err
	}
	return s.repo.UpdateJobStatus(ctx, incidentID, JobCompleted, "")
}

func shouldAnalyze(status string) bool {
	u := strings.ToUpper(strings.TrimSpace(status))
	return u != "PASSED" && u != "PASS" && u != ""
}

func defaultModel(p ProviderKind) string {
	switch p {
	case ProviderOllama:
		return "llama3.2"
	case ProviderOpenAI:
		return "gpt-4o-mini"
	default:
		return ""
	}
}

func (s *Service) GetConfig(ctx context.Context) (ProviderConfig, error) {
	return s.repo.LoadConfig(ctx)
}

func (s *Service) SaveConfig(ctx context.Context, cfg ProviderConfig) error {
	return s.repo.SaveConfig(ctx, cfg)
}

func (s *Service) GetReport(ctx context.Context, incidentID int64) (*RCAReport, error) {
	return s.repo.GetReport(ctx, incidentID)
}

func (s *Service) ListInsights(ctx context.Context, projectName string, limit int) ([]InsightRow, error) {
	return s.repo.ListInsights(ctx, projectName, limit)
}
