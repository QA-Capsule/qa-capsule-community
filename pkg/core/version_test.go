package core

import "testing"

func TestVersion_isSet(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	if Version != "1.0.17-beta" {
		t.Fatalf("unexpected Version: %q", Version)
	}
}
