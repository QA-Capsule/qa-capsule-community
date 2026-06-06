package server

import (
	"strings"
	"testing"
)

func TestApplyLocatorFixToSource_RobotBrokenVariable(t *testing.T) {
	source := `*** Variables ***
${BROKEN_SUBMIT}    button[data-qa="submit-v1"]

*** Test Cases ***
TC-04 Broken
    Click        ${BROKEN_SUBMIT}
`
	got := applyLocatorFixToSource(source, `button[data-qa="submit-v1"]`, "#submit")
	if !strings.Contains(got, "Click        #submit") {
		t.Fatalf("expected fixed click line, got:\n%s", got)
	}
}

func TestBuildFixHighlights_RobotFix(t *testing.T) {
	before := "    Click        ${BROKEN_SUBMIT}\n"
	after := "    Click        #submit\n"
	hl := buildFixHighlights(before, after, "#submit")
	if len(hl) != 1 || hl[0]["line"] != 1 || hl[0]["token"] != "#submit" {
		t.Fatalf("unexpected highlights: %#v", hl)
	}
}
