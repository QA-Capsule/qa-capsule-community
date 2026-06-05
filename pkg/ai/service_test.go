package ai

import (
	"context"
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// Fake implementations for testing without a real database or network
// ---------------------------------------------------------------------------

// fakeRepo is an in-memory Repository stub used only in tests.
type fakeRepo struct {
	cfg       ProviderConfig
	saveErr   error
	loadErr   error
	incident  Incident
	incErr    error
}

func (r *fakeRepo) LoadConfig(_ context.Context) (ProviderConfig, error) {
	return r.cfg, r.loadErr
}
func (r *fakeRepo) SaveConfig(_ context.Context, cfg ProviderConfig) error {
	r.cfg = cfg
	return r.saveErr
}
func (r *fakeRepo) CreateJob(_ context.Context, _ int64, _ ProviderKind, _ string) error {
	return nil
}
func (r *fakeRepo) UpdateJobStatus(_ context.Context, _ int64, _ JobStatus, _ string) error {
	return nil
}
func (r *fakeRepo) SaveReport(_ context.Context, _ int64, _ AnalysisResult) error {
	return nil
}
func (r *fakeRepo) GetReport(_ context.Context, _ int64) (*RCAReport, error) {
	return nil, nil
}
func (r *fakeRepo) ListInsights(_ context.Context, _ string, _ int) ([]InsightRow, error) {
	return nil, nil
}
func (r *fakeRepo) LoadIncident(_ context.Context, _ int64) (AnalysisInput, error) {
	return AnalysisInput{}, nil
}
func (r *fakeRepo) LoadIncidentRecord(_ context.Context, _ int64) (Incident, error) {
	return r.incident, r.incErr
}

// fixedAnalyzer returns a predetermined FixProposal regardless of input.
type fixedAnalyzer struct {
	proposal FixProposal
	err      error
}

func (a fixedAnalyzer) Analyze(_ context.Context, _ ProviderConfig, _ AnalysisInput) (AnalysisResult, error) {
	return AnalysisResult{}, nil
}
func (a fixedAnalyzer) GenerateFixProposal(_ context.Context, _ ProviderConfig, _ Incident, _ string) (FixProposal, error) {
	return a.proposal, a.err
}

// ---------------------------------------------------------------------------
// NewService tests
// ---------------------------------------------------------------------------

func TestNewService_NilAnalyzer_UsesHTTPAnalyzer(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if _, ok := svc.analyzer.(HTTPAnalyzer); !ok {
		t.Errorf("expected HTTPAnalyzer when nil passed, got %T", svc.analyzer)
	}
}

func TestNewService_CustomAnalyzer_IsPreserved(t *testing.T) {
	custom := fixedAnalyzer{}
	svc := NewService(&fakeRepo{}, custom)
	if _, ok := svc.analyzer.(fixedAnalyzer); !ok {
		t.Errorf("expected fixedAnalyzer to be preserved, got %T", svc.analyzer)
	}
}

// ---------------------------------------------------------------------------
// GetConfig / SaveConfig tests
// ---------------------------------------------------------------------------

func TestService_GetConfig_ReturnsRepoValue(t *testing.T) {
	want := ProviderConfig{Provider: ProviderGroq, Model: "llama-3.1-8b-instant", Enabled: true}
	repo := &fakeRepo{cfg: want}
	svc := NewService(repo, nil)

	got, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Provider != want.Provider {
		t.Errorf("provider mismatch: want %q, got %q", want.Provider, got.Provider)
	}
	if got.Model != want.Model {
		t.Errorf("model mismatch: want %q, got %q", want.Model, got.Model)
	}
}

func TestService_GetConfig_PropagatesRepoError(t *testing.T) {
	repo := &fakeRepo{loadErr: errors.New("db locked")}
	svc := NewService(repo, nil)
	_, err := svc.GetConfig(context.Background())
	if err == nil {
		t.Fatal("expected error to be propagated")
	}
}

func TestService_SaveConfig_PersistsToRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	cfg := ProviderConfig{Provider: ProviderGemini, Model: "gemini-1.5-flash", Enabled: true}
	if err := svc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.cfg.Provider != ProviderGemini {
		t.Errorf("expected config to be persisted, got: %+v", repo.cfg)
	}
}

func TestService_SaveConfig_PropagatesRepoError(t *testing.T) {
	repo := &fakeRepo{saveErr: errors.New("disk full")}
	svc := NewService(repo, nil)
	err := svc.SaveConfig(context.Background(), ProviderConfig{})
	if err == nil {
		t.Fatal("expected save error to be propagated")
	}
}

// ---------------------------------------------------------------------------
// GenerateFixProposal tests
// ---------------------------------------------------------------------------

func TestService_GenerateFixProposal_ReturnsAnalyzerOutput(t *testing.T) {
	expected := FixProposal{Code: "cy.get('[data-testid=btn]').click()", Explanation: "use stable selector", Confidence: 0.9}
	repo := &fakeRepo{cfg: ProviderConfig{Provider: ProviderGroq, Enabled: true, APIKey: "k"}}
	svc := NewService(repo, fixedAnalyzer{proposal: expected})

	code, explanation, err := svc.GenerateFixProposal(context.Background(), Incident{TestName: "T"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != expected.Code {
		t.Errorf("code mismatch: want %q, got %q", expected.Code, code)
	}
	if explanation != expected.Explanation {
		t.Errorf("explanation mismatch: want %q, got %q", expected.Explanation, explanation)
	}
}

func TestService_GenerateFixProposal_NilReceiver_ReturnsError(t *testing.T) {
	var svc *Service
	_, _, err := svc.GenerateFixProposal(context.Background(), Incident{}, "")
	if err == nil {
		t.Fatal("expected error for nil receiver")
	}
}

func TestService_GenerateFixProposal_PropagatesAnalyzerError(t *testing.T) {
	repo := &fakeRepo{cfg: ProviderConfig{Provider: ProviderGroq, Enabled: true, APIKey: "k"}}
	svc := NewService(repo, fixedAnalyzer{err: errors.New("quota exceeded")})
	_, _, err := svc.GenerateFixProposal(context.Background(), Incident{TestName: "T"}, "")
	if err == nil {
		t.Fatal("expected analyzer error to propagate")
	}
}

// ---------------------------------------------------------------------------
// ProposeFixFromIncidentID tests
// ---------------------------------------------------------------------------

func TestService_ProposeFixFromIncidentID_LoadsIncidentAndCallsAnalyzer(t *testing.T) {
	inc := Incident{
		ID:           42,
		TestName:     "Payment_Test",
		ErrorMessage: "Element not found",
		Status:       "FAILED",
	}
	expected := FixProposal{Code: "fixed code", Explanation: "all good", Confidence: 0.85}
	repo := &fakeRepo{
		cfg:      ProviderConfig{Provider: ProviderGroq, Enabled: true, APIKey: "k"},
		incident: inc,
	}
	svc := NewService(repo, fixedAnalyzer{proposal: expected})

	code, explanation, err := svc.ProposeFixFromIncidentID(context.Background(), 42, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != expected.Code {
		t.Errorf("code mismatch: want %q, got %q", expected.Code, code)
	}
	if explanation != expected.Explanation {
		t.Errorf("explanation mismatch: want %q, got %q", expected.Explanation, explanation)
	}
}

func TestService_ProposeFixFromIncidentID_IncidentLoadError(t *testing.T) {
	repo := &fakeRepo{
		cfg:    ProviderConfig{Provider: ProviderGroq, Enabled: true, APIKey: "k"},
		incErr: errors.New("incident not found"),
	}
	svc := NewService(repo, fixedAnalyzer{})
	_, _, err := svc.ProposeFixFromIncidentID(context.Background(), 999, "")
	if err == nil {
		t.Fatal("expected error when incident cannot be loaded")
	}
}

// ---------------------------------------------------------------------------
// defaultModel tests
// ---------------------------------------------------------------------------

func TestDefaultModel_KnownProviders(t *testing.T) {
	cases := []struct {
		provider ProviderKind
		wantNE   string // must not be empty
	}{
		{ProviderGroq, ""},
		{ProviderOpenAI, ""},
		{ProviderGemini, ""},
		{ProviderAnthropic, ""},
		{ProviderMistral, ""},
		{ProviderOllama, ""},
		{ProviderOpenRouter, ""},
		{ProviderAzure, ""},
	}
	for _, c := range cases {
		got := defaultModel(c.provider)
		if got == "" {
			t.Errorf("defaultModel(%q) returned empty string", c.provider)
		}
	}
}

func TestDefaultModel_DisabledProvider_ReturnsEmpty(t *testing.T) {
	if got := defaultModel(ProviderDisabled); got != "" {
		t.Errorf("expected empty for ProviderDisabled, got: %q", got)
	}
}
