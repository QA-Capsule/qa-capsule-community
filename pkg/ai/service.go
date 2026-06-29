package ai

import (
	"context"
	"fmt"
)

// Service wraps the AI repository and analyzer to expose high-level operations
// for config management and on-demand fix proposal generation.
type Service struct {
	repo     Repository
	analyzer Analyzer
}

// NewService constructs a Service. If analyzer is nil, HTTPAnalyzer is used as default.
func NewService(repo Repository, analyzer Analyzer) *Service {
	if analyzer == nil {
		analyzer = HTTPAnalyzer{}
	}
	return &Service{repo: repo, analyzer: analyzer}
}

// defaultModel returns the recommended default model identifier for a given provider.
func defaultModel(p ProviderKind) string {
	switch p {
	case ProviderOllama:
		return "llama3.2"
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderAnthropic:
		return "claude-3-5-haiku-20241022"
	case ProviderGemini:
		return "gemini-1.5-flash"
	case ProviderMistral:
		return "mistral-small-latest"
	case ProviderGroq:
		return "llama-3.1-8b-instant"
	case ProviderOpenRouter:
		return "openai/gpt-4o-mini"
	case ProviderAzure:
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

// GenerateFixProposal loads AI config and asks the LLM for a framework-agnostic patch.
func (s *Service) GenerateFixProposal(ctx context.Context, incident Incident, fileContent string) (code string, explanation string, err error) {
	if s == nil {
		return "", "", fmt.Errorf("AI service not initialized")
	}
	cfg, err := s.repo.LoadConfig(ctx)
	if err != nil {
		return "", "", err
	}
	prop, err := s.analyzer.GenerateFixProposal(ctx, cfg, incident, fileContent)
	if err != nil {
		return "", "", err
	}
	return prop.Code, prop.Explanation, nil
}

// ProposeFixFromIncidentID loads incident telemetry then generates a fix proposal.
func (s *Service) ProposeFixFromIncidentID(ctx context.Context, incidentID int64, fileContent string) (code string, explanation string, err error) {
	inc, err := s.repo.LoadIncidentRecord(ctx, incidentID)
	if err != nil {
		return "", "", err
	}
	return s.GenerateFixProposal(ctx, inc, fileContent)
}
