package symphony

import "strings"

// ExtractPhaseSlices returns documented phase sections that are specific enough
// to become issue materialization candidates. The first Go implementation keeps
// extraction conservative: no acceptance criteria means no automatic issue.
func ExtractPhaseSlices(corpus SourceCorpus, phaseID string) []PhaseSlice {
	if phaseID == "" {
		return nil
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(phaseID), " "))
	var slices []PhaseSlice
	for _, source := range corpus.Sources {
		if !source.HasAcceptanceCriteria || !containsString(source.PhaseReferences, normalized) {
			continue
		}
		title := phaseID
		if len(source.Headings) > 0 {
			title = source.Headings[0]
		}
		targetBranch := ""
		if len(source.TargetBranches) > 0 {
			targetBranch = source.TargetBranches[0]
		}
		slices = append(slices, PhaseSlice{
			ID:                 normalized,
			Title:              title,
			SourcePath:         source.Path,
			Headings:           source.Headings,
			AcceptanceCriteria: []string{"See checked-in acceptance criteria in " + source.Path},
			TargetBranch:       targetBranch,
			Verification:       source.VerificationCommandIDs,
		})
	}
	return slices
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
