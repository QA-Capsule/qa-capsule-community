package core

import (
	"testing"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

func TestFlakyNameVariants(t *testing.T) {
	n1, n2, n3 := flakyNameVariants("[FLAKY] checkout payment")
	if n1 != "checkout payment" || n2 != "[FLAKY] checkout payment" || n3 != "[PERF] checkout payment" {
		t.Fatalf("variants = %q %q %q", n1, n2, n3)
	}
}

func TestHasPriorFailureForTest_requiresNormalizedMatch(t *testing.T) {
	norm := quarantine.NormalizeTestName("login spec")
	n1, n2, n3 := flakyNameVariants("login spec")
	if n1 != norm || n2 != "[FLAKY] "+norm {
		t.Fatal("variant mismatch")
	}
	_ = n3
}
