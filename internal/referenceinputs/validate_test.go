package referenceinputs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRepositoryAcceptsRequiredStartupSnapshotsAndLocalLinks(t *testing.T) {
	root := buildReferenceInputFixture(t)

	if err := ValidateRepository(root); err != nil {
		t.Fatalf("ValidateRepository returned error: %v", err)
	}
}

func TestValidateRepositoryRejectsMissingStartupSnapshot(t *testing.T) {
	root := buildReferenceInputFixture(t)
	missing := filepath.Join(root, "docs", "reference-inputs", "branding", "Firefly.png")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove fixture snapshot: %v", err)
	}

	err := ValidateRepository(root)
	if err == nil {
		t.Fatal("ValidateRepository accepted a missing required startup snapshot")
	}
	if !strings.Contains(err.Error(), "required repo-local reference input snapshots are missing") {
		t.Fatalf("error %q did not describe missing required snapshots", err)
	}
	if !strings.Contains(err.Error(), "docs/reference-inputs/branding/Firefly.png") {
		t.Fatalf("error %q did not name the missing snapshot path", err)
	}
}

func TestValidateRepositoryRejectsReferenceLinksOutsideRepo(t *testing.T) {
	root := buildReferenceInputFixture(t)
	inventoryPath := filepath.Join(root, "docs", "reference-inputs", "VENDORED_INVENTORY.md")
	if err := os.WriteFile(inventoryPath, []byte("[unsafe](/Users/example/private-note.md)\n"), 0o644); err != nil {
		t.Fatalf("write fixture inventory: %v", err)
	}

	err := ValidateRepository(root)
	if err == nil {
		t.Fatal("ValidateRepository accepted an inventory link outside the repository")
	}
	if !strings.Contains(err.Error(), "links outside repository") {
		t.Fatalf("error %q did not describe the unsafe reference link", err)
	}
}

func buildReferenceInputFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"docs/reference-inputs/README.md":                                   "# Reference Inputs\n\nUse [VENDORED_INVENTORY.md](VENDORED_INVENTORY.md).\n",
		"docs/reference-inputs/VENDORED_INVENTORY.md":                       "# Vendored Reference Input Inventory\n\nSee [README](README.md).\n",
		"docs/reference-inputs/branding/Firefly.png":                        "fixture",
		"docs/reference-inputs/branding/google-g.png":                       "fixture",
		"docs/reference-inputs/branding/Wordmarks/Gold W black outline.png": "fixture",
	}
	for relativePath, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create fixture directory for %s: %v", relativePath, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file %s: %v", relativePath, err)
		}
	}
	return root
}
