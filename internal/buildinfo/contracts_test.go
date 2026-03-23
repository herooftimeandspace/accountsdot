package buildinfo_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadmeContainsLLMDisclaimer(t *testing.T) {
	root := projectRoot(t)
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("failed reading README.md: %v", err)
	}
	text := string(readme)
	if !strings.Contains(text, "LLM Usage Disclaimer") || !strings.Contains(text, "LLM-driven project") {
		t.Fatal("README.md must contain the required LLM usage disclaimer")
	}
}

func TestAllowedModules(t *testing.T) {
	root := projectRoot(t)
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("failed reading go.mod: %v", err)
	}
	allowed := []string{
		"github.com/google/uuid",
		"github.com/jackc/pgx/v5",
	}
	for _, line := range strings.Split(string(goMod), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "github.com/") && !strings.HasPrefix(line, "golang.org/") {
			continue
		}
		if strings.Contains(line, "// indirect") {
			continue
		}
		ok := false
		for _, module := range allowed {
			if strings.HasPrefix(line, module+" ") || line == module {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("go.mod contains unauthorized direct dependency line: %q", line)
		}
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
