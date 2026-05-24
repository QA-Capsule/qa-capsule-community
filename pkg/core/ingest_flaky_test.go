package core

import (
	"testing"
)

func TestHasPassFailPassSequence(t *testing.T) {
	if !hasPassFailPassSequence([]string{"pass", "fail", "pass"}) {
		t.Fatal("expected pass-fail-pass")
	}
	if !hasPassFailPassSequence([]string{"fail", "pass", "fail"}) {
		t.Fatal("expected fail-pass-fail")
	}
	if hasPassFailPassSequence([]string{"pass", "fail"}) {
		t.Fatal("expected false for short sequence")
	}
	if hasPassFailPassSequence([]string{"pass", "pass", "fail"}) {
		t.Fatal("expected false without oscillation")
	}
}

func TestStatusBucket(t *testing.T) {
	if statusBucket("PASSED") != "pass" {
		t.Fatal("PASSED should be pass bucket")
	}
	if statusBucket("CRITICAL") != "fail" {
		t.Fatal("CRITICAL should be fail bucket")
	}
	if statusBucket("PERF_DEGRADATION") != "" {
		t.Fatal("perf should be ignored in oscillation")
	}
}

func TestIsPassStatus(t *testing.T) {
	if !isPassStatus("passed") || !isPassStatus("PASS") {
		t.Fatal("pass variants")
	}
	if isPassStatus("CRITICAL") {
		t.Fatal("failure is not pass")
	}
}
