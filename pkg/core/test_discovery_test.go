package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestNameSearchTokens_RobotNestedSuite(t *testing.T) {
	tokens := TestNameSearchTokens("[RobotFramework] Practice Login > TC-04 Submit Button With Broken Locator")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}
	found := false
	for _, tok := range tokens {
		if strings.Contains(tok, "TC-04") || strings.Contains(tok, "Submit Button") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TC-04 or Submit Button in tokens: %v", tokens)
	}
}

func TestTestNameSearchTokens_Playwright(t *testing.T) {
	tokens := TestNameSearchTokens("[Playwright] checkout.spec.ts > should complete payment")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}
}

func TestFindTestSourceInRepo_WalksTestDirs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "e2e", "checkout")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `test('should pay with broken selector', async ({ page }) => {
  await page.locator('button[data-qa="pay-v1"]').click();
});`
	path := filepath.Join(dir, "checkout.spec.ts")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := FindTestSourceInRepo(root, "[Playwright] checkout.spec.ts > should pay with broken selector")
	if got == "" {
		t.Fatal("expected to find test source file")
	}
	if !strings.Contains(got, `button[data-qa="pay-v1"]`) {
		t.Errorf("expected broken selector in content, got: %q", got)
	}
}

func TestReadTestFileAtPaths_RelativePath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "tests", "login")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	want := "Click    #submit"
	path := filepath.Join(sub, "login.robot")
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ReadTestFileAtPaths("tests/login/login.robot", root)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
