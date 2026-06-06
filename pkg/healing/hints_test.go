package healing

import "testing"

func TestClassifyError(t *testing.T) {
	cases := []struct {
		err  string
		want string
	}{
		{"Timeout 30000ms exceeded waiting for locator", CategoryLocator},
		{"Element is stale: stale element reference", CategoryStaleElement},
		{"locator.click: Error: strict mode violation", CategoryLocator},
		{"AssertionError: expected true to be false", CategoryAssertion},
		{"connect ECONNREFUSED 127.0.0.1:8080", CategoryNetwork},
		{"Something unexpected happened", CategoryUnknown},
	}
	for _, tc := range cases {
		if got := ClassifyError(tc.err); got != tc.want {
			t.Fatalf("ClassifyError(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestSuggestedActionsNonEmpty(t *testing.T) {
	for _, cat := range []string{CategoryTimeout, CategoryLocator, CategoryAssertion, CategoryUnknown} {
		if len(SuggestedActions(cat)) == 0 {
			t.Fatalf("no actions for %s", cat)
		}
	}
}
