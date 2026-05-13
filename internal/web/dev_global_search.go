package web

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"
)

type devGlobalSearchPayload struct {
	PageID      string                     `json:"page_id"`
	Persona     devPersona                 `json:"persona"`
	Shell       devShellPayload            `json:"shell"`
	GeneratedAt string                     `json:"generated_at"`
	Page        devGlobalSearchPagePayload `json:"page"`
}

type devGlobalSearchPagePayload struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	LastRefreshed string                 `json:"last_refreshed"`
	Query         string                 `json:"query"`
	Groups        []devGlobalSearchGroup `json:"groups"`
}

type devGlobalSearchGroup struct {
	ID      string                  `json:"id"`
	Title   string                  `json:"title"`
	Results []devGlobalSearchResult `json:"results"`
}

type devGlobalSearchResult struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Context     string `json:"context"`
	Destination string `json:"destination"`
	Source      string `json:"source"`
}

type rankedGlobalSearchResult struct {
	GroupID       string
	GroupTitle    string
	Result        devGlobalSearchResult
	MatchRank     int
	NormalizedKey string
}

// handleDevGlobalSearch handles the request path for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleDevGlobalSearch(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can search.",
		})
		return
	}
	if !routeAllowed(config, devGlobalSearchRoute) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Global search is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, devGlobalSearchPayload{
		PageID:      "global-search",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: devGlobalSearchPagePayload{
			Title:         "Search",
			Description:   "Global search results across the DEV projections this persona can access.",
			LastRefreshed: "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			Query:         query,
			Groups:        buildDevGlobalSearchGroups(config, query, now),
		},
	})
}

// buildDevGlobalSearchGroups builds the value used by internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func buildDevGlobalSearchGroups(config devPersonaConfig, query string, now time.Time) []devGlobalSearchGroup {
	normalizedQuery := normalizeSearchValue(query)
	if normalizedQuery == "" {
		return []devGlobalSearchGroup{}
	}

	ranked := []rankedGlobalSearchResult{}
	if allowedRoutesContainPrefix(config, "/phone-directory/") {
		ranked = append(ranked, globalSearchPhoneDirectoryResults(config, normalizedQuery)...)
	}
	if routeAllowed(config, "/onboarding") {
		ranked = append(ranked, globalSearchOnboardingResults(normalizedQuery, now)...)
	}
	if routeAllowed(config, "/offboarding") {
		ranked = append(ranked, globalSearchOffboardingResults(config, normalizedQuery)...)
	}
	ranked = append(ranked, globalSearchWorkflowActionResults(config, normalizedQuery, now)...)
	if routeAllowed(config, devDepartingSeniorsRoute) {
		ranked = append(ranked, globalSearchDepartingSeniorResults(normalizedQuery, now)...)
		ranked = append(ranked, globalSearchDeviceResults(normalizedQuery, now)...)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if left.GroupID != right.GroupID {
			return globalSearchGroupRank(left.GroupID) < globalSearchGroupRank(right.GroupID)
		}
		if left.MatchRank != right.MatchRank {
			return left.MatchRank < right.MatchRank
		}
		if left.NormalizedKey != right.NormalizedKey {
			return left.NormalizedKey < right.NormalizedKey
		}
		return left.Result.ID < right.Result.ID
	})

	groupByID := map[string]*devGlobalSearchGroup{}
	groups := []devGlobalSearchGroup{}
	for _, item := range ranked {
		group := groupByID[item.GroupID]
		if group == nil {
			groups = append(groups, devGlobalSearchGroup{
				ID:      item.GroupID,
				Title:   item.GroupTitle,
				Results: []devGlobalSearchResult{},
			})
			group = &groups[len(groups)-1]
			groupByID[item.GroupID] = group
		}
		group.Results = append(group.Results, item.Result)
	}

	return groups
}

// globalSearchPhoneDirectoryResults documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchPhoneDirectoryResults(config devPersonaConfig, normalizedQuery string) []rankedGlobalSearchResult {
	canSeeEmployeeID := config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
	seen := map[string]bool{}
	results := []rankedGlobalSearchResult{}
	for _, entry := range devPhoneDirectoryEntries {
		searchValues := []string{entry.Title, entry.TypeLabel, entry.SiteName, entry.Role, entry.Department, entry.Location, entry.Email, entry.Phone, entry.Extension}
		if canSeeEmployeeID {
			searchValues = append(searchValues, entry.Identifier)
		}
		match := bestGlobalSearchMatch(searchValues, normalizedQuery)
		if match == nil {
			continue
		}

		groupID, groupTitle, destination := globalSearchDirectoryTarget(entry)
		key := groupID + ":" + entry.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		contextParts := []string{entry.SiteName}
		if entry.Extension != "" {
			contextParts = append(contextParts, "Extension "+entry.Extension)
		}
		if entry.Phone != "" {
			contextParts = append(contextParts, entry.Phone)
		}
		if canSeeEmployeeID && entry.Identifier != "" {
			contextParts = append(contextParts, entry.Identifier)
		}
		results = append(results, rankedGlobalSearchResult{
			GroupID:    groupID,
			GroupTitle: groupTitle,
			Result: devGlobalSearchResult{
				ID:          entry.ID,
				Type:        entry.TypeLabel,
				Title:       entry.Title,
				Subtitle:    strings.TrimSpace(firstNonEmpty(entry.Role, entry.Department, entry.Location, entry.TypeLabel)),
				Context:     strings.Join(contextParts, " · "),
				Destination: destination,
				Source:      "Phone Directory",
			},
			MatchRank:     match.Rank,
			NormalizedKey: normalizeSearchValue(entry.Title),
		})
	}
	return results
}

// globalSearchDirectoryTarget documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchDirectoryTarget(entry devPhoneDirectoryEntry) (string, string, string) {
	query := "?q=" + urlQueryEscape(firstNonEmpty(entry.Extension, entry.Title))
	switch entry.Type {
	case phoneDirectoryTypePerson:
		return "people", "People", "/phone-directory/by-person" + query
	case phoneDirectoryTypeCommonArea, phoneDirectoryTypeClassroomSLG:
		return "rooms", "Rooms and Extensions", "/phone-directory/by-room" + query
	case phoneDirectoryTypeDepartmentSLG, phoneDirectoryTypeCallQueue, phoneDirectoryTypeAutoAttendant:
		return "departments", "Departments and Lines", "/phone-directory/by-department" + query
	default:
		return "directory", "Phone Directory", "/phone-directory/by-person" + query
	}
}

// globalSearchOnboardingResults documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchOnboardingResults(normalizedQuery string, now time.Time) []rankedGlobalSearchResult {
	rows := devOnboardingStore.rows(now)
	results := make([]rankedGlobalSearchResult, 0, len(rows))
	for _, row := range rows {
		values := []string{row.Person, row.Site, row.AssignedEmail, row.EmployeeNumber, row.WorkflowStatus, row.CurrentStep, row.IssueAction, row.AeriesTicket, row.VerkadaTicket}
		match := bestGlobalSearchMatch(values, normalizedQuery)
		if match == nil {
			continue
		}
		results = append(results, rankedGlobalSearchResult{
			GroupID:    "onboarding",
			GroupTitle: "Onboarding",
			Result: devGlobalSearchResult{
				ID:          row.ID,
				Type:        row.WorkflowStatus,
				Title:       row.Person,
				Subtitle:    row.CurrentStep,
				Context:     strings.Join(nonEmptyStrings(row.Site, row.IssueAction, row.AssignedEmail), " · "),
				Destination: "/onboarding",
				Source:      "Onboarding",
			},
			MatchRank:     match.Rank,
			NormalizedKey: normalizeSearchValue(row.Person),
		})
	}
	return results
}

// globalSearchOffboardingResults documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchOffboardingResults(config devPersonaConfig, normalizedQuery string) []rankedGlobalSearchResult {
	rows := devOffboardingStore.rows(config)
	results := make([]rankedGlobalSearchResult, 0, len(rows))
	for _, row := range rows {
		values := []string{row.Person, row.Email, row.Site, row.Status, row.NextAction, row.AssetWork, row.ExternalReference}
		if canSeeOffboardingEmployeeIDs(config) {
			values = append(values, row.EmployeeID)
		}
		for _, action := range row.Actions {
			values = append(values, action.Name, action.Owner, action.Status, action.Detail, action.Resolution)
		}
		match := bestGlobalSearchMatch(values, normalizedQuery)
		if match == nil {
			continue
		}
		results = append(results, rankedGlobalSearchResult{
			GroupID:    "offboarding",
			GroupTitle: "Offboarding",
			Result: devGlobalSearchResult{
				ID:          row.ID,
				Type:        row.Status,
				Title:       row.Person,
				Subtitle:    row.NextAction,
				Context:     strings.Join(nonEmptyStrings(row.Site, row.Email, row.AssetWork), " · "),
				Destination: "/offboarding",
				Source:      "Offboarding",
			},
			MatchRank:     match.Rank,
			NormalizedKey: normalizeSearchValue(row.Person),
		})
	}
	return results
}

// globalSearchWorkflowActionResults documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchWorkflowActionResults(config devPersonaConfig, normalizedQuery string, now time.Time) []rankedGlobalSearchResult {
	results := []rankedGlobalSearchResult{}
	if routeAllowed(config, "/onboarding") {
		for _, row := range devOnboardingStore.rows(now) {
			for _, step := range row.WorkflowSteps {
				for _, action := range step.Actions {
					values := []string{row.Person, row.Site, row.AssignedEmail, step.Name, step.Status, step.Detail, action.Label, action.Resolution, action.System, action.Href}
					match := bestGlobalSearchMatch(values, normalizedQuery)
					if match == nil {
						continue
					}
					results = append(results, rankedGlobalSearchResult{
						GroupID:    "workflow-actions",
						GroupTitle: "Workflow Actions",
						Result: devGlobalSearchResult{
							ID:          row.ID + ":" + step.Name + ":" + action.Label,
							Type:        step.Status,
							Title:       action.Label,
							Subtitle:    row.Person,
							Context:     strings.Join(nonEmptyStrings(row.Site, action.System, action.Resolution), " · "),
							Destination: "/onboarding",
							Source:      "Onboarding",
						},
						MatchRank:     match.Rank,
						NormalizedKey: normalizeSearchValue(row.Person + " " + action.Label),
					})
				}
			}
		}
	}
	if routeAllowed(config, "/offboarding") {
		for _, row := range devOffboardingStore.rows(config) {
			for _, action := range row.Actions {
				values := []string{row.Person, row.Email, row.Site, action.Name, action.Owner, action.Status, action.Detail, action.Resolution}
				match := bestGlobalSearchMatch(values, normalizedQuery)
				if match == nil {
					continue
				}
				results = append(results, rankedGlobalSearchResult{
					GroupID:    "workflow-actions",
					GroupTitle: "Workflow Actions",
					Result: devGlobalSearchResult{
						ID:          row.ID + ":" + action.Name,
						Type:        action.Status,
						Title:       action.Name,
						Subtitle:    row.Person,
						Context:     strings.Join(nonEmptyStrings(row.Site, action.Owner, action.Resolution), " · "),
						Destination: "/offboarding",
						Source:      "Offboarding",
					},
					MatchRank:     match.Rank,
					NormalizedKey: normalizeSearchValue(row.Person + " " + action.Name),
				})
			}
		}
	}
	return results
}

// globalSearchDepartingSeniorResults documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchDepartingSeniorResults(normalizedQuery string, now time.Time) []rankedGlobalSearchResult {
	rows := devDepartingSeniorsStore.rows(currentSeniorGraduationYear(now))
	results := make([]rankedGlobalSearchResult, 0, len(rows))
	for _, row := range rows {
		values := []string{row.DisplayName, row.FirstName, row.LastName, row.Email, row.StudentID, row.Site, row.GraduationYear, row.Status}
		for _, device := range row.OutstandingDevices {
			values = append(values, device.AssetID, device.Serial, device.Type)
		}
		match := bestGlobalSearchMatch(values, normalizedQuery)
		if match == nil {
			continue
		}
		deviceSummary := "No outstanding devices"
		if count := len(row.OutstandingDevices); count == 1 {
			deviceSummary = "1 outstanding device"
		} else if count > 1 {
			deviceSummary = fmt.Sprintf("%d outstanding devices", count)
		}
		results = append(results, rankedGlobalSearchResult{
			GroupID:    "departing-seniors",
			GroupTitle: "Departing Seniors",
			Result: devGlobalSearchResult{
				ID:          row.ID,
				Type:        row.Status,
				Title:       row.DisplayName,
				Subtitle:    "Graduation year " + row.GraduationYear,
				Context:     strings.Join(nonEmptyStrings(row.Site, row.Email, row.StudentID, deviceSummary), " · "),
				Destination: "/departing-seniors",
				Source:      "Departing Seniors",
			},
			MatchRank:     match.Rank,
			NormalizedKey: normalizeSearchValue(row.DisplayName),
		})
	}
	return results
}

// globalSearchDeviceResults documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchDeviceResults(normalizedQuery string, now time.Time) []rankedGlobalSearchResult {
	rows := devDepartingSeniorsStore.rows(currentSeniorGraduationYear(now))
	results := []rankedGlobalSearchResult{}
	for _, row := range rows {
		for _, device := range row.OutstandingDevices {
			values := []string{device.AssetID, device.Serial, device.Type, row.DisplayName, row.Email, row.StudentID, row.Site, row.GraduationYear}
			match := bestGlobalSearchMatch(values, normalizedQuery)
			if match == nil {
				continue
			}
			results = append(results, rankedGlobalSearchResult{
				GroupID:    "devices-assets",
				GroupTitle: "Devices and Assets",
				Result: devGlobalSearchResult{
					ID:          row.ID + ":" + device.AssetID,
					Type:        device.Type,
					Title:       device.AssetID,
					Subtitle:    "Assigned to " + row.DisplayName,
					Context:     strings.Join(nonEmptyStrings(row.Site, "Serial "+device.Serial, row.StudentID), " · "),
					Destination: "/departing-seniors",
					Source:      "IncidentIQ Devices",
				},
				MatchRank:     match.Rank,
				NormalizedKey: normalizeSearchValue(row.DisplayName + " " + device.AssetID),
			})
		}
	}
	return results
}

// bestGlobalSearchMatch documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func bestGlobalSearchMatch(values []string, normalizedQuery string) *phoneDirectorySearchMatch {
	if normalizedQuery == "" {
		return nil
	}
	entry := devPhoneDirectoryEntry{Searchable: values}
	return bestPhoneDirectoryMatch(entry, normalizedQuery)
}

// globalSearchGroupRank documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func globalSearchGroupRank(groupID string) int {
	order := []string{"people", "rooms", "departments", "onboarding", "offboarding", "workflow-actions", "departing-seniors", "devices-assets"}
	index := slices.Index(order, groupID)
	if index < 0 {
		return len(order)
	}
	return index
}

// allowedRoutesContainPrefix resolves decision data for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func allowedRoutesContainPrefix(config devPersonaConfig, prefix string) bool {
	return slices.ContainsFunc(config.Allowed, func(route string) bool {
		return strings.HasPrefix(route, prefix)
	})
}

// firstNonEmpty documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// nonEmptyStrings documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func nonEmptyStrings(values ...string) []string {
	result := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// urlQueryEscape documents the data flow for internal/web/dev_global_search.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func urlQueryEscape(value string) string {
	return url.QueryEscape(strings.TrimSpace(value))
}
