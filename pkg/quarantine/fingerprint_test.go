package quarantine

import "testing"

func TestTestIdentityFingerprint_stable(t *testing.T) {
	a := TestIdentityFingerprint("api-tests", "[FLAKY] checkout payment")
	b := TestIdentityFingerprint("api-tests", "checkout payment")
	if a != b {
		t.Fatalf("expected same identity fingerprint, got %s vs %s", a, b)
	}
}

func TestNormalizeTestName(t *testing.T) {
	if NormalizeTestName("[FLAKY] foo") != "foo" {
		t.Fatal("prefix not stripped")
	}
}
