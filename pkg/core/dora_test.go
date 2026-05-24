package core

import "testing"

func TestMedianFloat64(t *testing.T) {
	if got := medianFloat64([]float64{10, 20, 30}); got != 20 {
		t.Fatalf("odd median: got %v want 20", got)
	}
	if got := medianFloat64([]float64{10, 20, 30, 40}); got != 25 {
		t.Fatalf("even median: got %v want 25", got)
	}
	if got := medianFloat64(nil); got != 0 {
		t.Fatalf("empty: got %v", got)
	}
}
