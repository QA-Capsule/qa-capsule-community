package quarantine

import (
	"context"
	"testing"
	"time"
)

type ciStubRepo struct {
	active map[string]*Entry
}

func (s *ciStubRepo) key(project, fp string) string { return project + "|" + fp }

func (s *ciStubRepo) UpsertPipelineRun(context.Context, string, string, string, string) error {
	return nil
}
func (s *ciStubRepo) GetStats(context.Context, string, string) (*StabilityStats, error) {
	return nil, nil
}
func (s *ciStubRepo) UpsertStats(context.Context, StabilityStats) error { return nil }
func (s *ciStubRepo) InsertTransition(context.Context, TransitionEvent) error {
	return nil
}
func (s *ciStubRepo) ActiveEntry(_ context.Context, project, fp string) (*Entry, error) {
	if s.active == nil {
		return nil, nil
	}
	return s.active[s.key(project, fp)], nil
}
func (s *ciStubRepo) CreateEntry(context.Context, Entry) error { return nil }
func (s *ciStubRepo) LiftEntry(context.Context, string, string, string) error { return nil }
func (s *ciStubRepo) ListActive(context.Context, string) ([]Entry, error) { return nil, nil }

func TestResolveCIIdentity(t *testing.T) {
	fp := TestIdentityFingerprint("proj", "Checkout.Payment")
	validHash := fp

	t.Run("by test name", func(t *testing.T) {
		gotFP, name, err := ResolveCIIdentity("proj", "", "Checkout.Payment")
		if err != nil {
			t.Fatal(err)
		}
		if gotFP != validHash {
			t.Fatalf("fingerprint mismatch: %s", gotFP)
		}
		if name != "Checkout.Payment" {
			t.Fatalf("name: %q", name)
		}
	})

	t.Run("by hash", func(t *testing.T) {
		gotFP, _, err := ResolveCIIdentity("proj", validHash, "")
		if err != nil {
			t.Fatal(err)
		}
		if gotFP != validHash {
			t.Fatalf("expected hash passthrough")
		}
	})

	t.Run("missing", func(t *testing.T) {
		_, _, err := ResolveCIIdentity("proj", "", "")
		if err != ErrMissingCIIdentifier {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("bad hash", func(t *testing.T) {
		_, _, err := ResolveCIIdentity("proj", "not-a-fingerprint", "")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCheckCIStatus(t *testing.T) {
	project := "my-suite"
	testName := "login flaky"
	fp := TestIdentityFingerprint(project, testName)

	repo := &ciStubRepo{active: map[string]*Entry{}}
	e := NewEngine(repo, DefaultPolicy())

	resp, err := e.CheckCIStatus(context.Background(), project, "", testName)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Quarantined || resp.Skip {
		t.Fatal("expected not quarantined")
	}

	repo.active[repo.key(project, fp)] = &Entry{
		ProjectName:             project,
		TestIdentityFingerprint: fp,
		TestName:                testName,
		Reason:                  ReasonFlaky,
		Source:                  SourceAuto,
		CreatedAt:               time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC),
	}

	resp, err = e.CheckCIStatus(context.Background(), project, fp, "")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Quarantined || !resp.Skip {
		t.Fatal("expected skip")
	}
	if resp.Reason != string(ReasonFlaky) {
		t.Fatalf("reason: %q", resp.Reason)
	}
}
