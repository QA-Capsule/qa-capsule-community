package quarantine

import (
	"context"
	"testing"
)

type stubRepo struct {
	active *Entry
}

func (s *stubRepo) UpsertPipelineRun(context.Context, string, string, string, string) error {
	return nil
}
func (s *stubRepo) GetStats(context.Context, string, string) (*StabilityStats, error) {
	return nil, nil
}
func (s *stubRepo) UpsertStats(context.Context, StabilityStats) error { return nil }
func (s *stubRepo) InsertTransition(context.Context, TransitionEvent) error {
	return nil
}
func (s *stubRepo) ActiveEntry(context.Context, string, string) (*Entry, error) {
	return s.active, nil
}
func (s *stubRepo) CreateEntry(context.Context, Entry) error { return nil }
func (s *stubRepo) LiftEntry(context.Context, string, string, string) error {
	return nil
}
func (s *stubRepo) ListActive(context.Context, string) ([]Entry, error) { return nil, nil }

func TestIsQuarantined(t *testing.T) {
	e := NewEngine(&stubRepo{active: &Entry{TestName: "checkout"}}, DefaultPolicy())
	if !e.IsQuarantined(context.Background(), "proj", "checkout") {
		t.Fatal("expected quarantined")
	}
	e2 := NewEngine(&stubRepo{active: nil}, DefaultPolicy())
	if e2.IsQuarantined(context.Background(), "proj", "checkout") {
		t.Fatal("expected not quarantined")
	}
}
