package server

import "testing"

func TestExtractHealedLocatorFromExplanation(t *testing.T) {
	explanation := "The original test code used an incorrect locator for the submit button, which was $IBROKEN_SUBMIT. This was changed to `#submit`, the correct locator."
	got := extractHealedLocatorFromExplanation(explanation)
	if got != "#submit" {
		t.Fatalf("got %q want #submit", got)
	}
}

func TestExtractHealedLocatorFromExplanationPlainChangedTo(t *testing.T) {
	got := extractHealedLocatorFromExplanation("This was changed to #submit, the correct locator.")
	if got != "#submit" {
		t.Fatalf("got %q want #submit", got)
	}
}

func TestExtractOriginalLocatorFromExplanation(t *testing.T) {
	explanation := "The original test code used an incorrect locator for the submit button, which was $IBROKEN_SUBMIT. This was changed to `#submit`, the correct locator."
	got := extractOriginalLocatorFromExplanation(explanation)
	if got != "$IBROKEN_SUBMIT" {
		t.Fatalf("got %q want $IBROKEN_SUBMIT", got)
	}
}

func TestInferHealedLocatorFromCode(t *testing.T) {
	code := `*** Test Cases ***
TC-01 Login
    Fill Text    id=username    user
    Click    #submit
    Get Element    #submit
`
	got := inferHealedLocatorFromCode(code, "${BROKEN_SUBMIT}")
	if got != "#submit" {
		t.Fatalf("got %q want #submit", got)
	}
}

func TestInferHealedLocatorFromCodeSkipsFormFields(t *testing.T) {
	code := `    Fill Text    id=username    user
    Click    ${BROKEN_SUBMIT}
`
	got := inferHealedLocatorFromCode(code, "${BROKEN_SUBMIT}")
	if got != "" {
		t.Fatalf("got %q want empty while original still in code", got)
	}
}
