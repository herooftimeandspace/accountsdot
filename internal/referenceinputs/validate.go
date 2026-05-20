package referenceinputs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var requiredStartupPaths = []string{
	"docs/reference-inputs/README.md",
	"docs/reference-inputs/VENDORED_INVENTORY.md",
	"docs/reference-inputs/branding/Firefly.png",
	"docs/reference-inputs/branding/google-g.png",
	"docs/reference-inputs/branding/Wordmarks/Gold W black outline.png",
}

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

// ValidateStartup proves the repo-local reference corpus needed during service
// boot is present before the web server starts. The application entrypoint uses
// this guard so dev and staging deployments fail with an actionable missing-path
// error instead of later falling back to workstation-only reference files.
func ValidateStartup() error {
	root, err := findReferenceInputRoot()
	if err != nil {
		return err
	}
	return ValidateRepository(root)
}

// ValidateRepository checks the required reference input snapshot files and the
// relative Markdown links inside docs/reference-inputs. Tests call this helper
// with temporary repository roots to prove both the healthy and missing-snapshot
// startup paths without depending on a developer workstation layout.
func ValidateRepository(root string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("reference input validation requires a repository root")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}

	var missing []string
	for _, relativePath := range requiredStartupPaths {
		if err := requireRelativePath(root, relativePath); err != nil {
			missing = append(missing, err.Error())
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("required repo-local reference input snapshots are missing: %s", strings.Join(missing, "; "))
	}

	if err := validateReferenceMarkdownLinks(root); err != nil {
		return err
	}
	return nil
}

// RequiredStartupPaths returns the repo-relative files that current startup
// treats as mandatory reference inputs. Documentation and tests use this list to
// keep the runtime guard aligned with the inventory rather than duplicating it.
func RequiredStartupPaths() []string {
	return append([]string(nil), requiredStartupPaths...)
}

func findReferenceInputRoot() (string, error) {
	if configuredRoot := strings.TrimSpace(os.Getenv("WIZARD_REFERENCE_INPUT_ROOT")); configuredRoot != "" {
		if !directoryHasReferenceInputs(configuredRoot) {
			return "", fmt.Errorf("configured WIZARD_REFERENCE_INPUT_ROOT %s does not contain docs/reference-inputs/VENDORED_INVENTORY.md", configuredRoot)
		}
		return configuredRoot, nil
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("read working directory for reference input validation: %w", err)
	}
	for directory := workingDirectory; ; directory = filepath.Dir(directory) {
		if directoryHasReferenceInputs(directory) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("locate reference input root for validation from %s; set WIZARD_REFERENCE_INPUT_ROOT when running outside a repository checkout", workingDirectory)
		}
	}
}

func directoryHasReferenceInputs(root string) bool {
	return fileExists(filepath.Join(root, "docs", "reference-inputs", "VENDORED_INVENTORY.md"))
}

func requireRelativePath(root, relativePath string) error {
	if filepath.IsAbs(relativePath) || strings.Contains(relativePath, "..") {
		return fmt.Errorf("%s is not a safe repo-relative reference input path", relativePath)
	}
	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s", relativePath)
		}
		return fmt.Errorf("%s (%v)", relativePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, expected a file", relativePath)
	}
	return nil
}

func validateReferenceMarkdownLinks(root string) error {
	markdownFiles := []string{
		"docs/reference-inputs/README.md",
		"docs/reference-inputs/VENDORED_INVENTORY.md",
	}
	var failures []string
	for _, relativePath := range markdownFiles {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s (%v)", relativePath, err))
			continue
		}
		baseDirectory := filepath.Dir(filepath.Join(root, filepath.FromSlash(relativePath)))
		for _, match := range markdownLinkPattern.FindAllStringSubmatch(string(content), -1) {
			link := strings.TrimSpace(match[1])
			target, ok := localMarkdownTarget(root, baseDirectory, link)
			if !ok {
				continue
			}
			if !isWithinRoot(root, target) {
				failures = append(failures, fmt.Sprintf("%s links outside repository: %s", relativePath, link))
				continue
			}
			if !pathExists(target) {
				failures = append(failures, fmt.Sprintf("%s has unresolved link: %s", relativePath, link))
			}
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf("repo-local reference input documentation links are invalid: %s", strings.Join(failures, "; "))
	}
	return nil
}

func localMarkdownTarget(root, baseDirectory, link string) (string, bool) {
	if link == "" || strings.HasPrefix(link, "#") {
		return "", false
	}
	if strings.Contains(link, "://") || strings.HasPrefix(link, "mailto:") {
		return "", false
	}
	pathOnly := strings.Split(link, "#")[0]
	if pathOnly == "" {
		return "", false
	}
	if strings.HasPrefix(pathOnly, "/") {
		return filepath.Clean(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(pathOnly, "/")))), true
	}
	if filepath.IsAbs(pathOnly) {
		return filepath.Clean(pathOnly), true
	}
	return filepath.Clean(filepath.Join(baseDirectory, filepath.FromSlash(pathOnly))), true
}

func isWithinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
