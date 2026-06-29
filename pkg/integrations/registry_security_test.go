package integrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_resolveManifestPath_blocksTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.json"), []byte(`{"integration":"slack","name":"ok"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reg, err := LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.UpdateConfig("../outside.json", map[string]string{"k": "v"}, false); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}
