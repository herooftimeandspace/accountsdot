package symphony

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	phaseReferencePattern  = regexp.MustCompile(`(?i)\b(?:pre-phase\s*0|phase\s+[0-9][0-9a-z-]*)\b`)
	issueReferencePattern  = regexp.MustCompile(`(?:^|[^\w-])#([0-9]+)(?:$|[^\w-])`)
	targetBranchPattern    = regexp.MustCompile(`(?i)\btarget branch:\s*` + "`?" + `([A-Za-z0-9._/-]+)`)
	verificationCmdPattern = regexp.MustCompile(`\b(?:npm run [A-Za-z0-9:._/-]+|go test [^\n` + "`" + `]+|make [A-Za-z0-9:_-]+|git diff --check)\b`)
	checklistPattern       = regexp.MustCompile(`-\s+\[[ xX]\]`)
)

// ScanMarkdownCorpus walks the repository for authoritative Markdown sources.
// The Symphony CLI calls it before issue planning or dispatch so checked-in
// `.agents` and `docs` guidance affects the work graph instead of living only
// in a prompt template.
func ScanMarkdownCorpus(root string) (SourceCorpus, error) {
	corpus := SourceCorpus{
		Root:        root,
		GeneratedAt: time.Now().UTC(),
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel != "." && shouldSkipMarkdownDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type().IsRegular() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") && !shouldSkipMarkdownFile(rel) {
			source, err := scanMarkdownSource(root, rel)
			if err != nil {
				return err
			}
			corpus.Sources = append(corpus.Sources, source)
			if source.Priority <= 30 {
				corpus.PriorityFiles++
			}
		}
		return nil
	})
	if err != nil {
		return SourceCorpus{}, err
	}
	sort.Slice(corpus.Sources, func(i, j int) bool {
		if corpus.Sources[i].Priority != corpus.Sources[j].Priority {
			return corpus.Sources[i].Priority < corpus.Sources[j].Priority
		}
		return corpus.Sources[i].Path < corpus.Sources[j].Path
	})
	corpus.TotalFiles = len(corpus.Sources)
	corpus.Conflicts = detectTargetBranchConflicts(corpus.Sources)
	return corpus, nil
}

func scanMarkdownSource(root, rel string) (MarkdownSource, error) {
	file, err := os.Open(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return MarkdownSource{}, err
	}
	defer file.Close()

	source := MarkdownSource{Path: rel, Priority: markdownPriority(rel)}
	seenPhase := map[string]bool{}
	seenIssue := map[int]bool{}
	seenBranch := map[string]bool{}
	seenCommand := map[string]bool{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if heading := markdownHeading(line); heading != "" {
			source.Headings = append(source.Headings, heading)
		}
		if strings.Contains(strings.ToLower(line), "acceptance criteria") || checklistPattern.MatchString(line) {
			source.HasAcceptanceCriteria = true
		}
		if strings.Contains(strings.ToLower(line), "safety") || strings.Contains(strings.ToLower(line), "secret") || strings.Contains(strings.ToLower(line), "production write") {
			source.SafetyRuleCount++
		}
		for _, match := range phaseReferencePattern.FindAllString(line, -1) {
			key := strings.ToLower(strings.Join(strings.Fields(match), " "))
			if !seenPhase[key] {
				seenPhase[key] = true
				source.PhaseReferences = append(source.PhaseReferences, key)
			}
		}
		for _, match := range issueReferencePattern.FindAllStringSubmatch(line, -1) {
			number, err := strconv.Atoi(match[1])
			if err == nil && !seenIssue[number] {
				seenIssue[number] = true
				source.IssueReferences = append(source.IssueReferences, number)
			}
		}
		for _, match := range targetBranchPattern.FindAllStringSubmatch(line, -1) {
			branch := strings.Trim(match[1], "`.,;)")
			if branch != "" && !seenBranch[branch] {
				seenBranch[branch] = true
				source.TargetBranches = append(source.TargetBranches, branch)
			}
		}
		for _, command := range verificationCmdPattern.FindAllString(line, -1) {
			command = strings.TrimSpace(command)
			if command != "" && !seenCommand[command] {
				seenCommand[command] = true
				source.VerificationCommandIDs = append(source.VerificationCommandIDs, command)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return MarkdownSource{}, err
	}
	sort.Ints(source.IssueReferences)
	sort.Strings(source.PhaseReferences)
	sort.Strings(source.TargetBranches)
	sort.Strings(source.VerificationCommandIDs)
	return source, nil
}

func markdownHeading(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return ""
	}
	return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
}

func markdownPriority(rel string) int {
	switch {
	case rel == "README.md":
		return 1
	case rel == "AGENTS.md":
		return 2
	case rel == ".agents/AGENTS.md":
		return 3
	case rel == ".agents/WORKFLOW.md":
		return 4
	case strings.HasPrefix(rel, ".agents/skills/") && strings.HasSuffix(rel, "/SKILL.md"):
		return 5
	case strings.HasPrefix(rel, "docs/planning/"):
		return 10
	case strings.HasPrefix(rel, "docs/product/"):
		return 11
	case strings.HasPrefix(rel, "docs/testing/"):
		return 12
	case strings.HasPrefix(rel, "docs/operations/"):
		return 13
	case strings.HasPrefix(rel, "docs/agent-orchestration/"):
		return 14
	case strings.HasPrefix(rel, ".agents/"):
		return 20
	case strings.HasPrefix(rel, "docs/"):
		return 30
	default:
		return 100
	}
}

func shouldSkipMarkdownDir(rel string) bool {
	skipPrefixes := []string{
		".git",
		"node_modules",
		"frontend/dist",
		"tmp",
		".vite",
		".gocache",
		".gomodcache",
		"artifacts",
		"coverage",
		"generated",
		"accountsdot-symphony",
		"accountsdot-symphony-prs",
	}
	for _, prefix := range skipPrefixes {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return false
}

func shouldSkipMarkdownFile(rel string) bool {
	return rel == "generated" || strings.HasPrefix(rel, "generated/") || strings.Contains(rel, "/generated/") || strings.Contains(rel, "/node_modules/") || strings.Contains(rel, "/frontend/dist/")
}

func detectTargetBranchConflicts(sources []MarkdownSource) []SourceConflict {
	pathsByBranch := map[string][]string{}
	for _, source := range sources {
		for _, branch := range source.TargetBranches {
			pathsByBranch[branch] = append(pathsByBranch[branch], source.Path)
		}
	}
	if len(pathsByBranch) <= 1 {
		return nil
	}
	branches := make([]string, 0, len(pathsByBranch))
	for branch := range pathsByBranch {
		branches = append(branches, branch)
	}
	sort.Strings(branches)
	conflicts := make([]SourceConflict, 0, len(branches))
	for _, branch := range branches {
		conflicts = append(conflicts, SourceConflict{
			Kind:     "target_branch",
			Value:    branch,
			Paths:    pathsByBranch[branch],
			Decision: "follow source-of-truth priority when materializing phase work",
		})
	}
	return conflicts
}
