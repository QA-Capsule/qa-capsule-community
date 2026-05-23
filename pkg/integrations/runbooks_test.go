package integrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunbookCatalog_validatesAgainstRegistry(t *testing.T) {
	pluginsDir := filepath.Join("..", "..", "plugins")
	if _, err := os.Stat(pluginsDir); err != nil {
		t.Skip("plugins directory missing")
	}
	reg, err := LoadRegistry(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range RunbookCatalog() {
		doc := CloneRunbookDocument(&tmpl)
		if err := ValidateWorkflowStructure(doc, reg); err != nil {
			t.Fatalf("template %s: %v", tmpl.ID, err)
		}
	}
}

func TestGetRunbookTemplate(t *testing.T) {
	_, ok := GetRunbookTemplate("502-restart-pod")
	if !ok {
		t.Fatal("expected 502 template")
	}
	_, ok = GetRunbookTemplate("missing")
	if ok {
		t.Fatal("expected missing")
	}
}
