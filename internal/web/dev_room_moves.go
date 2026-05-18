package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	roomMoveTypeSingle      = "mid_year_targeted_move"
	roomMoveTypeBulkRoster  = "end_of_year_site_move"
	roomMoveTypeBuildList   = "manual_move_list"
	roomMoveDraftStatusOpen = "draft"
)

var devRoomMoveStore = newDevRoomMoveStore()

type roomMovesPagePayload struct {
	PageID      string               `json:"page_id"`
	Persona     devPersona           `json:"persona"`
	Shell       devShellPayload      `json:"shell"`
	GeneratedAt string               `json:"generated_at"`
	Page        roomMovesPageContent `json:"page"`
}

type roomMovesPageContent struct {
	Title                 string                 `json:"title"`
	Description           string                 `json:"description"`
	LastRefreshed         string                 `json:"last_refreshed"`
	CanManageDistrict     bool                   `json:"can_manage_district"`
	ScopeSite             devSiteContext         `json:"scope_site"`
	Sites                 []devSiteContext       `json:"sites"`
	Rooms                 []roomMoveRoomOption   `json:"rooms"`
	People                []roomMovePersonOption `json:"people"`
	SummaryCards          []summaryCardPayload   `json:"summary_cards"`
	Rows                  []roomMoveReviewRow    `json:"rows"`
	DefaultBulkRosterHref string                 `json:"default_bulk_roster_href"`
	DefaultBuildListHref  string                 `json:"default_build_list_href"`
}

type roomMovesBulkDraftPayload struct {
	PageID      string                    `json:"page_id"`
	Persona     devPersona                `json:"persona"`
	Shell       devShellPayload           `json:"shell"`
	GeneratedAt string                    `json:"generated_at"`
	Page        roomMovesBulkDraftContent `json:"page"`
}

type roomMovesBulkDraftContent struct {
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	LastRefreshed     string                 `json:"last_refreshed"`
	CanManageDistrict bool                   `json:"can_manage_district"`
	ScopeSite         devSiteContext         `json:"scope_site"`
	Sites             []devSiteContext       `json:"sites"`
	Rooms             []roomMoveRoomOption   `json:"rooms"`
	People            []roomMovePersonOption `json:"people"`
	Draft             roomMoveDraftPayload   `json:"draft"`
}

type roomMoveReviewRow struct {
	ID                 string   `json:"id"`
	DraftID            string   `json:"draft_id"`
	MoveType           string   `json:"move_type"`
	Person             string   `json:"person"`
	Email              string   `json:"email"`
	EmployeeID         string   `json:"employee_id"`
	CurrentSiteID      string   `json:"current_site_id"`
	CurrentSite        string   `json:"current_site"`
	CurrentRoom        string   `json:"current_room"`
	DestinationSiteID  string   `json:"destination_site_id"`
	DestinationSite    string   `json:"destination_site"`
	DestinationRoomID  string   `json:"destination_room_id"`
	DestinationRoom    string   `json:"destination_room"`
	Phone              string   `json:"phone"`
	Author             string   `json:"author"`
	AuthorID           string   `json:"author_id,omitempty"`
	State              string   `json:"state"`
	ScheduledFor       string   `json:"scheduled_for,omitempty"`
	CanEdit            bool     `json:"can_edit"`
	CanCancel          bool     `json:"can_cancel"`
	Warning            string   `json:"warning,omitempty"`
	WarningLevel       string   `json:"warning_level,omitempty"`
	AttentionReason    string   `json:"attention_reason,omitempty"`
	AutomationOutcome  string   `json:"automation_outcome,omitempty"`
	ManualActionOwner  string   `json:"manual_action_owner,omitempty"`
	ManualActionReason string   `json:"manual_action_reason,omitempty"`
	ResolutionSteps    []string `json:"resolution_steps,omitempty"`
	ExternalSystems    []string `json:"external_systems,omitempty"`
	FallbackTicket     string   `json:"fallback_ticket,omitempty"`
	FallbackTicketHref string   `json:"fallback_ticket_href,omitempty"`
	FallbackStatus     string   `json:"fallback_status,omitempty"`
	TechnicalOutcome   string   `json:"technical_outcome,omitempty"`
}

type roomMoveDraftPayload struct {
	ID                string             `json:"id"`
	Mode              string             `json:"mode"`
	Status            string             `json:"status"`
	ScopeSiteID       string             `json:"scope_site_id"`
	ScopeSite         string             `json:"scope_site"`
	EffectiveDate     string             `json:"effective_date"`
	ScheduledFor      string             `json:"scheduled_for,omitempty"`
	Author            string             `json:"author"`
	AuthorID          string             `json:"author_id,omitempty"`
	Warnings          []string           `json:"warnings"`
	Rows              []roomMoveDraftRow `json:"rows"`
	CanEdit           bool               `json:"can_edit"`
	CanDelete         bool               `json:"can_delete"`
	CanManageDistrict bool               `json:"can_manage_district"`
}

type roomMoveCompletedJobPayload struct {
	ID            string             `json:"id"`
	SourceDraftID string             `json:"source_draft_id"`
	Mode          string             `json:"mode"`
	ScopeSiteID   string             `json:"scope_site_id"`
	ScopeSite     string             `json:"scope_site"`
	CompletedAt   string             `json:"completed_at"`
	CompletedBy   string             `json:"completed_by"`
	RowCount      int                `json:"row_count"`
	Rows          []roomMoveDraftRow `json:"rows"`
	CanRevert     bool               `json:"can_revert"`
	RevertDraftID string             `json:"revert_draft_id,omitempty"`
	RevertStatus  string             `json:"revert_status,omitempty"`
}

type roomMoveCompletedJobsResponse struct {
	Jobs []roomMoveCompletedJobPayload `json:"jobs"`
}

type roomMoveDraftRow struct {
	ID                 string   `json:"id"`
	PersonID           string   `json:"person_id"`
	Person             string   `json:"person"`
	Email              string   `json:"email"`
	EmployeeID         string   `json:"employee_id"`
	CurrentSiteID      string   `json:"current_site_id"`
	CurrentSite        string   `json:"current_site"`
	CurrentRoomID      string   `json:"current_room_id"`
	CurrentRoom        string   `json:"current_room"`
	SourceRole         string   `json:"-"`
	DestinationSiteID  string   `json:"destination_site_id"`
	DestinationSite    string   `json:"destination_site"`
	DestinationRoomID  string   `json:"destination_room_id"`
	DestinationRoom    string   `json:"destination_room"`
	DestinationRole    string   `json:"destination_role,omitempty"`
	Phone              string   `json:"phone"`
	Action             string   `json:"action"`
	Warning            string   `json:"warning,omitempty"`
	AttentionReason    string   `json:"attention_reason,omitempty"`
	AutomationOutcome  string   `json:"automation_outcome,omitempty"`
	ManualActionOwner  string   `json:"manual_action_owner,omitempty"`
	ManualActionReason string   `json:"manual_action_reason,omitempty"`
	ResolutionSteps    []string `json:"resolution_steps,omitempty"`
	ExternalSystems    []string `json:"external_systems,omitempty"`
	FallbackTicket     string   `json:"fallback_ticket,omitempty"`
	FallbackTicketHref string   `json:"fallback_ticket_href,omitempty"`
	FallbackStatus     string   `json:"fallback_status,omitempty"`
	TechnicalOutcome   string   `json:"technical_outcome,omitempty"`
}

type roomMovePersonOption struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmployeeID    string `json:"employee_id"`
	Role          string `json:"role"`
	SiteID        string `json:"site_id"`
	Site          string `json:"site"`
	CurrentRoomID string `json:"current_room_id"`
	CurrentRoom   string `json:"current_room"`
	SourceRole    string `json:"source_role,omitempty"`
	Phone         string `json:"phone"`
}

type roomMoveRoomOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	SiteID string `json:"site_id"`
	Site   string `json:"site"`
}

type roomMoveDraftRequest struct {
	Mode          string             `json:"mode"`
	PersonID      string             `json:"person_id"`
	ScopeSiteID   string             `json:"scope_site_id"`
	EffectiveDate string             `json:"effective_date"`
	ScheduledFor  string             `json:"scheduled_for"`
	Rows          []roomMoveDraftRow `json:"rows"`
}

type roomMoveDraftResponse struct {
	Draft roomMoveDraftPayload `json:"draft"`
}

type devRoomMoveStoreState struct {
	mu        sync.Mutex
	nextID    int
	drafts    map[string]roomMoveDraftPayload
	completed map[string]bool
	canceled  map[string]bool
	jobs      map[string]roomMoveCompletedJobPayload
}

func newDevRoomMoveStore() *devRoomMoveStoreState {
	store := &devRoomMoveStoreState{
		nextID:    1000,
		drafts:    map[string]roomMoveDraftPayload{},
		completed: map[string]bool{},
		canceled:  map[string]bool{},
		jobs:      map[string]roomMoveCompletedJobPayload{},
	}
	for _, job := range seedCompletedRoomMoveJobs() {
		store.jobs[job.ID] = job
	}
	return store
}

func handleDevRoomMovesPage(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMovesPersona(w, r)
	if !ok {
		return
	}
	page := devRoomMoveStore.page(config)
	writeJSON(w, http.StatusOK, roomMovesPagePayload{
		PageID:      "room-moves",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Page:        page,
	})
}

func handleDevRoomMovesBulkDraftPage(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMovesPersona(w, r)
	if !ok {
		return
	}
	draftID := strings.TrimSpace(r.URL.Query().Get("draft_id"))
	draft := devRoomMoveStore.ensureBulkDraft(config, draftID, roomMoveTypeBulkRoster)
	title := roomMoveBulkDraftTitle(draft.Mode)
	writeJSON(w, http.StatusOK, roomMovesBulkDraftPayload{
		PageID:      "room-moves-bulk-draft",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Page: roomMovesBulkDraftContent{
			Title:             title,
			Description:       "Draft, review, schedule, and commit room changes for employees and contractors.",
			LastRefreshed:     "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			CanManageDistrict: canManageDistrictRoomMoves(config),
			ScopeSite:         roomMovesScopeSite(config),
			Sites:             roomMoveVisibleSites(config),
			Rooms:             roomMoveRoomsForConfig(config),
			People:            roomMovePeopleForConfig(config),
			Draft:             draft,
		},
	})
}

func roomMoveBulkDraftTitle(mode string) string {
	if mode == roomMoveTypeBuildList {
		return "Batch Move"
	}
	return "Site Rollover"
}

// handleDevRoomMoveDrafts creates a draft in the in-memory DEV room-move store.
// The frontend single-move drawer posts one JSON draft request here; validation
// errors return field messages, while success returns the stored draft payload.
func handleDevRoomMoveDrafts(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMovesPersona(w, r)
	if !ok {
		return
	}
	var request roomMoveDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_json", "message": "Request body must be valid JSON."})
		return
	}
	draft, status, errors := devRoomMoveStore.createDraft(config, request)
	if status != http.StatusOK {
		writeJSON(w, status, map[string]any{"code": "room_move_draft_rejected", "message": "The room move draft could not be created.", "errors": errors})
		return
	}
	writeJSON(w, http.StatusOK, roomMoveDraftResponse{Draft: draft})
}

// handleDevRoomMoveDraft routes PUT, transition, cancel, and delete actions for
// an existing DEV room-move draft. Each branch checks the authenticated persona
// before mutating the in-memory store so site-scoped users cannot alter another
// site's draft or a visible draft authored by IT or another operator.
func handleDevRoomMoveDraft(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMovesPersona(w, r)
	if !ok {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/dev/room-moves/drafts/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	draftID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodPut && action == "":
		var request roomMoveDraftRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_json", "message": "Request body must be valid JSON."})
			return
		}
		draft, status, errors := devRoomMoveStore.updateDraft(config, draftID, request)
		if status != http.StatusOK {
			writeJSON(w, status, map[string]any{"code": "room_move_draft_rejected", "message": "The room move draft could not be updated.", "errors": errors})
			return
		}
		writeJSON(w, http.StatusOK, roomMoveDraftResponse{Draft: draft})
	case r.Method == http.MethodPost && action == "cancel":
		draft, status, errors := devRoomMoveStore.cancelDraft(config, draftID)
		if status != http.StatusOK {
			writeJSON(w, status, map[string]any{"code": "room_move_draft_rejected", "message": "The room move draft could not be canceled.", "errors": errors})
			return
		}
		writeJSON(w, http.StatusOK, roomMoveDraftResponse{Draft: draft})
	case r.Method == http.MethodPost && (action == "schedule" || action == "apply"):
		draft, status, errors := devRoomMoveStore.transitionDraft(config, draftID, action)
		if status != http.StatusOK {
			writeJSON(w, status, map[string]any{"code": "room_move_draft_rejected", "message": "The room move draft could not be changed.", "errors": errors})
			return
		}
		writeJSON(w, http.StatusOK, roomMoveDraftResponse{Draft: draft})
	case r.Method == http.MethodDelete && action == "":
		status, errors := devRoomMoveStore.deleteDraft(config, draftID)
		if status != http.StatusNoContent {
			writeJSON(w, status, map[string]any{"code": "room_move_draft_rejected", "message": "The room move draft could not be deleted.", "errors": errors})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func handleDevRoomMoveCompletedJobs(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMoveRevertPersona(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, roomMoveCompletedJobsResponse{Jobs: devRoomMoveStore.completedJobs(config)})
}

func handleDevRoomMoveCompletedJob(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMoveRevertPersona(w, r)
	if !ok {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/dev/room-moves/completed/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "revert" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	draft, status, errors := devRoomMoveStore.scheduleRevert(config, parts[0])
	if status != http.StatusOK {
		writeJSON(w, status, map[string]any{"code": "room_move_revert_rejected", "message": "The room move job could not be reverted.", "errors": errors})
		return
	}
	writeJSON(w, http.StatusOK, roomMoveDraftResponse{Draft: draft})
}

func authenticatedRoomMovesPersona(w http.ResponseWriter, r *http.Request) (devPersonaConfig, bool) {
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "not_authorized", "message": "You need to sign in before you can view this page."})
		return devPersonaConfig{}, false
	}
	if !routeAllowed(r.Context(), config, "/room-moves") {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": "forbidden", "message": "Room Moves is not available for this role.", "persona": config.Persona})
		return devPersonaConfig{}, false
	}
	return config, true
}

func authenticatedRoomMoveRevertPersona(w http.ResponseWriter, r *http.Request) (devPersonaConfig, bool) {
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "not_authorized", "message": "You need to sign in before you can view this page."})
		return devPersonaConfig{}, false
	}
	if !canManageDistrictRoomMoves(config) || !routeAllowed(r.Context(), config, "/admin") {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": "forbidden", "message": "Only IT Admin can revert completed room move jobs.", "persona": config.Persona})
		return devPersonaConfig{}, false
	}
	return config, true
}

func (s *devRoomMoveStoreState) page(config devPersonaConfig) roomMovesPageContent {
	rows := s.reviewRows(config)
	return roomMovesPageContent{
		Title:                 "Room Moves",
		Description:           "Draft, review, schedule, and commit room and phone changes with warnings before automation writes.",
		LastRefreshed:         "Last refreshed:\nMay 3, 2026 9:00 AM PT",
		CanManageDistrict:     canManageDistrictRoomMoves(config),
		ScopeSite:             roomMovesScopeSite(config),
		Sites:                 roomMoveVisibleSites(config),
		Rooms:                 roomMoveRoomsForConfig(config),
		People:                roomMovePeopleForConfig(config),
		SummaryCards:          roomMoveSummaryCards(rows),
		Rows:                  rows,
		DefaultBulkRosterHref: "/room-moves/bulk-draft?mode=bulk_site_roster",
		DefaultBuildListHref:  "/room-moves/bulk-draft?mode=build_move_list",
	}
}

func (s *devRoomMoveStoreState) reviewRows(config devPersonaConfig) []roomMoveReviewRow {
	base := []roomMoveReviewRow{
		seedRoomMoveReviewRow("single-alex-ramirez", "single-alex-ramirez", roomMoveTypeSingle, "Alex Ramirez", "alex.ramirez@wusd.org", "103118", "clover-hs", "A-104", "clover-hs", "A-108", "Move ext 51042", "Alex Ramirez", "it_admin", "Ready", ""),
		primaryConflictReviewRow("single-morgan-lee", "single-morgan-lee", "Morgan Lee", "morgan.lee@wusd.org", "103442", "clover-hs", "B-210", "clover-hs", "B-204", "Avery Shah"),
		manualFallbackReviewRow("single-taylor-chen", "single-taylor-chen", "Taylor Chen", "taylor.chen@wusd.org", "103884", "clover-hs", "C-202", "clover-hs", "C-214", "Avery Shah"),
		seedRoomMoveReviewRow("bulk-clover-summer", "rm-draft-103", roomMoveTypeBulkRoster, "Bulk Move", "", "", "clover-hs", "Multiple", "clover-hs", "Multiple", "Batch cutover", "Alex Ramirez", "it_admin", "Scheduled", "Two rows need review before scheduling").withScheduledFor("2026-07-27T20:00:00-07:00"),
		seedRoomMoveReviewRow("single-jamie-reed", "single-jamie-reed", roomMoveTypeSingle, "Jamie Reed", "jamie.reed@wusd.org", "103772", "desert-view", "C-118", "desert-view", "None", "Remove phone and SLGs; convert room to common area", "Alex Ramirez", "it_admin", "Ready", ""),
		seedRoomMoveReviewRow("single-nia-brooks", "single-nia-brooks", roomMoveTypeSingle, "Nia Brooks", "nia.brooks@wusd.org", "104012", "franklin-ms", "D-102", "franklin-ms", "D-112", "Assign line", "Avery Shah", "other_site_operator", "Ready", ""),
	}
	baseDraftIDs := map[string]bool{}
	for _, row := range base {
		baseDraftIDs[row.DraftID] = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	suppressedDraftIDs := map[string]bool{}
	for _, draft := range s.drafts {
		suppressedDraftIDs[draft.ID] = true
		if s.completed[draft.ID] || s.canceled[draft.ID] {
			continue
		}
		if !canAccessRoomMoveSite(config, draft.ScopeSiteID) {
			continue
		}
		if len(draft.Rows) == 1 && draft.Mode == roomMoveTypeSingle {
			row := draft.Rows[0]
			base = append(base, roomMoveReviewRow{
				ID:                 row.ID,
				DraftID:            draft.ID,
				MoveType:           roomMoveTypeSingle,
				Person:             row.Person,
				Email:              row.Email,
				EmployeeID:         row.EmployeeID,
				CurrentSiteID:      row.CurrentSiteID,
				CurrentSite:        row.CurrentSite,
				CurrentRoom:        row.CurrentRoom,
				DestinationSiteID:  row.DestinationSiteID,
				DestinationSite:    row.DestinationSite,
				DestinationRoomID:  row.DestinationRoomID,
				DestinationRoom:    row.DestinationRoom,
				Phone:              row.Phone,
				Author:             draft.Author,
				AuthorID:           draft.AuthorID,
				State:              draftStatusLabel(draft.Status),
				ScheduledFor:       draft.ScheduledFor,
				CanEdit:            canMutateRoomMoveDraft(config, draft),
				CanCancel:          canMutateRoomMoveDraft(config, draft),
				Warning:            row.Warning,
				WarningLevel:       warningLevel(row.Warning),
				AttentionReason:    row.AttentionReason,
				AutomationOutcome:  row.AutomationOutcome,
				ManualActionOwner:  row.ManualActionOwner,
				ManualActionReason: row.ManualActionReason,
				ResolutionSteps:    row.ResolutionSteps,
				ExternalSystems:    row.ExternalSystems,
				FallbackTicket:     row.FallbackTicket,
				FallbackTicketHref: row.FallbackTicketHref,
				FallbackStatus:     row.FallbackStatus,
				TechnicalOutcome:   row.TechnicalOutcome,
			})
			continue
		}
		state := draftStatusLabel(draft.Status)
		scheduledFor := draft.ScheduledFor
		warning := strings.Join(draft.Warnings, " ")
		phone := fmt.Sprintf("%d rows", len(draft.Rows))
		author := draft.Author
		if seed, ok := seedRoomMoveReviewRowByDraftID(draft.ID); ok && seed.ID != seed.DraftID && draft.Status == roomMoveDraftStatusOpen {
			state = seed.State
			scheduledFor = seed.ScheduledFor
			warning = seed.Warning
			phone = seed.Phone
			author = seed.Author
		}
		base = append(base, roomMoveReviewRow{
			ID:                "bulk-" + draft.ID,
			DraftID:           draft.ID,
			MoveType:          draft.Mode,
			Person:            "Bulk Move",
			Email:             "",
			EmployeeID:        "",
			CurrentSiteID:     draft.ScopeSiteID,
			CurrentSite:       draft.ScopeSite,
			CurrentRoom:       "Multiple",
			DestinationSiteID: draft.ScopeSiteID,
			DestinationSite:   draft.ScopeSite,
			DestinationRoomID: "",
			DestinationRoom:   "Multiple",
			Phone:             phone,
			Author:            author,
			AuthorID:          draft.AuthorID,
			State:             state,
			ScheduledFor:      scheduledFor,
			CanEdit:           canMutateRoomMoveDraft(config, draft),
			CanCancel:         canMutateRoomMoveDraft(config, draft),
			Warning:           warning,
			WarningLevel:      warningLevel(warning),
		})
	}

	filtered := base[:0]
	rowIndexByDraftID := map[string]int{}
	for _, row := range base {
		if baseDraftIDs[row.DraftID] && suppressedDraftIDs[row.DraftID] && isSeededRoomMoveReviewRow(row) {
			continue
		}
		if canAccessRoomMoveSite(config, row.CurrentSiteID) && !s.canceled[row.DraftID] {
			row.CanEdit = canMutateRoomMoveReviewRow(config, row)
			row.CanCancel = row.CanEdit
			if index, ok := rowIndexByDraftID[row.DraftID]; ok {
				filtered[index] = row
				continue
			}
			rowIndexByDraftID[row.DraftID] = len(filtered)
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (row roomMoveReviewRow) withScheduledFor(scheduledFor string) roomMoveReviewRow {
	row.ScheduledFor = scheduledFor
	return row
}

func seedRoomMoveReviewRow(id string, draftID string, moveType string, person string, email string, employeeID string, currentSiteID string, currentRoom string, destinationSiteID string, destinationRoom string, phone string, author string, authorID string, state string, warning string) roomMoveReviewRow {
	currentSite := siteByID(currentSiteID)
	destinationSite := siteByID(destinationSiteID)
	return roomMoveReviewRow{
		ID:                id,
		DraftID:           draftID,
		MoveType:          moveType,
		Person:            person,
		Email:             email,
		EmployeeID:        employeeID,
		CurrentSiteID:     currentSiteID,
		CurrentSite:       currentSite.Name,
		CurrentRoom:       currentRoom,
		DestinationSiteID: destinationSiteID,
		DestinationSite:   destinationSite.Name,
		DestinationRoomID: roomMoveRoomIDByLabel(destinationSiteID, destinationRoom),
		DestinationRoom:   destinationRoom,
		Phone:             phone,
		Author:            author,
		AuthorID:          authorID,
		State:             state,
		Warning:           warning,
		WarningLevel:      warningLevel(warning),
	}
}

// primaryConflictReviewRow builds the seeded Morgan Lee-style row that issue
// #54 uses as browser evidence. It returns the same review-row contract as
// normal draft rows, but pre-populates the primary-room conflict explanation so
// the page demonstrates the shared-line-group automation path before a user
// creates a new draft in the DEV mock store.
func primaryConflictReviewRow(id string, draftID string, person string, email string, employeeID string, currentSiteID string, currentRoom string, destinationSiteID string, destinationRoom string, author string) roomMoveReviewRow {
	row := seedRoomMoveReviewRow(
		id,
		draftID,
		roomMoveTypeSingle,
		person,
		email,
		employeeID,
		currentSiteID,
		currentRoom,
		destinationSiteID,
		destinationRoom,
		"Add to room shared line group; keep primary phone owner",
		author,
		"other_site_operator",
		"Ready",
		primaryConflictWarning(person, destinationRoom, "Jordan Patel"),
	)
	row.AttentionReason = primaryConflictAttentionReason(person, destinationRoom, "Jordan Patel")
	row.AutomationOutcome = primaryConflictAutomationOutcome(person, destinationRoom)
	row.ResolutionSteps = primaryConflictResolutionSteps(person, destinationRoom)
	return row
}

// manualFallbackReviewRow seeds the Room Moves owner surface with a fallback
// ticket that is already closed and technically verified. The DEV payload keeps
// the owner, reason, resolution steps, linked systems, ticket status, and
// verification result together so the React drawer demonstrates auto-resolution
// without reviving the retired standalone human-work report route.
func manualFallbackReviewRow(id string, draftID string, person string, email string, employeeID string, currentSiteID string, currentRoom string, destinationSiteID string, destinationRoom string, author string) roomMoveReviewRow {
	row := seedRoomMoveReviewRow(
		id,
		draftID,
		roomMoveTypeSingle,
		person,
		email,
		employeeID,
		currentSiteID,
		currentRoom,
		destinationSiteID,
		destinationRoom,
		"Verified by fallback ticket",
		author,
		"other_site_operator",
		"Resolved",
		"Manual fallback ticket closed after Zoom and IncidentIQ room state verification.",
	)
	row.AttentionReason = "Automation could not verify the Zoom shared line group during execution, so IT completed and verified the room move through IncidentIQ."
	row.AutomationOutcome = "Resolved automatically after the linked ticket closed and the room/phone technical outcome was verified."
	row.ManualActionOwner = "IT Service Desk"
	row.ManualActionReason = "Zoom shared line group membership verification failed during room-move cutover."
	row.ResolutionSteps = []string{
		"Confirm the destination room shared line group includes the moving user.",
		"Confirm the IncidentIQ room association reflects the destination room.",
		"Close the fallback ticket only after both technical checks pass.",
	}
	row.ExternalSystems = []string{"Zoom room shared line group", "IncidentIQ room association"}
	row.FallbackTicket = "IT-12977"
	row.FallbackTicketHref = "https://mock.wusd.local/incidentiq/tickets/IT-12977"
	row.FallbackStatus = "Closed"
	row.TechnicalOutcome = "Verified Zoom shared line group and IncidentIQ room association; row auto-resolved."
	return row
}

func seedRoomMoveReviewRowByDraftID(draftID string) (roomMoveReviewRow, bool) {
	for _, row := range []roomMoveReviewRow{
		seedRoomMoveReviewRow("single-alex-ramirez", "single-alex-ramirez", roomMoveTypeSingle, "Alex Ramirez", "alex.ramirez@wusd.org", "103118", "clover-hs", "A-104", "clover-hs", "A-108", "Move ext 51042", "Alex Ramirez", "it_admin", "Ready", ""),
		primaryConflictReviewRow("single-morgan-lee", "single-morgan-lee", "Morgan Lee", "morgan.lee@wusd.org", "103442", "clover-hs", "B-210", "clover-hs", "B-204", "Avery Shah"),
		manualFallbackReviewRow("single-taylor-chen", "single-taylor-chen", "Taylor Chen", "taylor.chen@wusd.org", "103884", "clover-hs", "C-202", "clover-hs", "C-214", "Avery Shah"),
		seedRoomMoveReviewRow("bulk-clover-summer", "rm-draft-103", roomMoveTypeBulkRoster, "Bulk Move", "", "", "clover-hs", "Multiple", "clover-hs", "Multiple", "Batch cutover", "Alex Ramirez", "it_admin", "Scheduled", "Two rows need review before scheduling").withScheduledFor("2026-07-27T20:00:00-07:00"),
		seedRoomMoveReviewRow("single-jamie-reed", "single-jamie-reed", roomMoveTypeSingle, "Jamie Reed", "jamie.reed@wusd.org", "103772", "desert-view", "C-118", "desert-view", "None", "Remove phone and SLGs; convert room to common area", "Alex Ramirez", "it_admin", "Ready", ""),
		seedRoomMoveReviewRow("single-nia-brooks", "single-nia-brooks", roomMoveTypeSingle, "Nia Brooks", "nia.brooks@wusd.org", "104012", "franklin-ms", "D-102", "franklin-ms", "D-112", "Assign line", "Avery Shah", "other_site_operator", "Ready", ""),
	} {
		if row.DraftID == draftID {
			return row, true
		}
	}
	return roomMoveReviewRow{}, false
}

func isSeededRoomMoveReviewRow(row roomMoveReviewRow) bool {
	seed, ok := seedRoomMoveReviewRowByDraftID(row.DraftID)
	return ok && row.ID == seed.ID
}

func seedCompletedRoomMoveJobs() []roomMoveCompletedJobPayload {
	alex, _ := roomMovePersonByID("alex-ramirez")
	morgan, _ := roomMovePersonByID("morgan-lee")
	rows := []roomMoveDraftRow{
		draftRowFromPerson(alex, "clover-hs", "cla-a108"),
		draftRowFromPerson(morgan, "clover-hs", "cla-b204"),
	}
	return []roomMoveCompletedJobPayload{{
		ID:            "rm-job-090",
		SourceDraftID: "rm-draft-090",
		Mode:          roomMoveTypeBulkRoster,
		ScopeSiteID:   "clover-hs",
		ScopeSite:     siteByID("clover-hs").Name,
		CompletedAt:   "May 10, 2026 8:30 PM PT",
		CompletedBy:   "IT Admin",
		RowCount:      len(rows),
		Rows:          rows,
		CanRevert:     true,
	}}
}

// createDraft validates the posted row data, allocates a deterministic DEV-only
// draft id, and stores the resulting payload in memory for later update,
// schedule, apply, cancel, or delete requests.
func (s *devRoomMoveStoreState) createDraft(config devPersonaConfig, request roomMoveDraftRequest) (roomMoveDraftPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	draftID := fmt.Sprintf("rm-draft-%03d", s.nextID)
	draft, status, errors := buildRoomMoveDraft(config, draftID, request)
	if status != http.StatusOK {
		return roomMoveDraftPayload{}, status, errors
	}
	s.drafts[draft.ID] = draft
	return draft, http.StatusOK, nil
}

// updateDraft replaces an existing DEV draft after confirming the current
// persona can access the draft's scoped site. Missing mode, scope-site, or
// effective-date fields keep their prior values so partial frontend saves do
// not reset workflow context.
func (s *devRoomMoveStoreState) updateDraft(config devPersonaConfig, draftID string, request roomMoveDraftRequest) (roomMoveDraftPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.drafts[draftID]
	if !ok {
		if seed, seedOK := seedRoomMoveReviewRowByDraftID(draftID); seedOK {
			existing = roomMoveDraftPayload{
				ID:                draftID,
				Mode:              seed.MoveType,
				Status:            roomMoveDraftStatusOpen,
				ScopeSiteID:       seed.CurrentSiteID,
				ScopeSite:         seed.CurrentSite,
				EffectiveDate:     "2026-07-27",
				Author:            seed.Author,
				AuthorID:          seed.AuthorID,
				Rows:              []roomMoveDraftRow{{PersonID: roomMovePersonIDByEmail(seed.Email), DestinationSiteID: seed.DestinationSiteID, DestinationRoomID: seed.DestinationRoomID}},
				CanEdit:           canMutateRoomMoveReviewRow(config, seed),
				CanDelete:         canMutateRoomMoveReviewRow(config, seed),
				CanManageDistrict: canManageDistrictRoomMoves(config),
			}
		} else {
			return roomMoveDraftPayload{}, http.StatusNotFound, map[string]string{"draft": "Draft not found."}
		}
	}
	if !canAccessRoomMoveSite(config, existing.ScopeSiteID) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot update another site's room move draft."}
	}
	if !canMutateRoomMoveDraft(config, existing) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"author": "This persona can view this room move but only the original author or IT Admin can update it."}
	}
	if request.ScopeSiteID == "" {
		request.ScopeSiteID = existing.ScopeSiteID
	}
	draft, status, errors := buildRoomMoveDraft(config, draftID, request)
	if status != http.StatusOK {
		return roomMoveDraftPayload{}, status, errors
	}
	if request.Mode == "" {
		draft.Mode = existing.Mode
	}
	if request.EffectiveDate == "" {
		draft.EffectiveDate = existing.EffectiveDate
	}
	draft.Author = existing.Author
	draft.AuthorID = existing.AuthorID
	draft.Status = existing.Status
	s.drafts[draft.ID] = draft
	return roomMoveDraftWithPermissions(config, draft), http.StatusOK, nil
}

// transitionDraft records the DEV-only schedule or completion state for a draft.
// Applying a draft also creates a completed-job record that the Admin revert
// overlay can read back without calling a live provider.
func (s *devRoomMoveStoreState) transitionDraft(config devPersonaConfig, draftID string, action string) (roomMoveDraftPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[draftID]
	if !ok {
		return roomMoveDraftPayload{}, http.StatusNotFound, map[string]string{"draft": "Draft not found."}
	}
	if !canAccessRoomMoveSite(config, draft.ScopeSiteID) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot update another site's room move draft."}
	}
	if !canMutateRoomMoveDraft(config, draft) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"author": "This persona can view this room move but only the original author or IT Admin can apply or schedule it."}
	}
	if s.canceled[draft.ID] {
		return roomMoveDraftPayload{}, http.StatusConflict, map[string]string{"draft": "Canceled drafts cannot be scheduled or applied."}
	}
	if status, errors := validateRoomMoveRowsForTransition(draft.Rows); status != http.StatusOK {
		return roomMoveDraftPayload{}, status, errors
	}
	if action == "schedule" {
		draft.Status = "scheduled"
		draft.ScheduledFor = "2026-07-27T20:00:00-07:00"
	} else {
		draft.Status = "complete"
		s.completed[draft.ID] = true
		jobID := "rm-job-" + strings.TrimPrefix(draft.ID, "rm-draft-")
		s.jobs[jobID] = roomMoveCompletedJobPayload{
			ID:            jobID,
			SourceDraftID: draft.ID,
			Mode:          draft.Mode,
			ScopeSiteID:   draft.ScopeSiteID,
			ScopeSite:     draft.ScopeSite,
			CompletedAt:   "May 11, 2026 9:30 AM PT",
			CompletedBy:   config.Persona.Label,
			RowCount:      len(draft.Rows),
			Rows:          draft.Rows,
			CanRevert:     true,
		}
	}
	s.drafts[draft.ID] = draft
	return roomMoveDraftWithPermissions(config, draft), http.StatusOK, nil
}

// cancelDraft marks draft-review and saved drafts as canceled in the DEV store.
// Completed drafts are rejected because production parity requires reversal
// through the completed-job revert flow instead of deleting completed history.
func (s *devRoomMoveStoreState) cancelDraft(config devPersonaConfig, draftID string) (roomMoveDraftPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.completed[draftID] {
		return roomMoveDraftPayload{}, http.StatusConflict, map[string]string{"draft": "This room move already ran and must be reverted from Admin."}
	}
	if draft, ok := s.drafts[draftID]; ok {
		if !canAccessRoomMoveSite(config, draft.ScopeSiteID) {
			return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot cancel another site's room move draft."}
		}
		if !canMutateRoomMoveDraft(config, draft) {
			return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"author": "This persona can view this room move but only the original author or IT Admin can cancel it."}
		}
		draft.Status = "canceled"
		s.drafts[draft.ID] = draft
		s.canceled[draft.ID] = true
		return roomMoveDraftWithPermissions(config, draft), http.StatusOK, nil
	}
	if seed, ok := seedRoomMoveReviewRowByDraftID(draftID); ok {
		if !canAccessRoomMoveSite(config, seed.CurrentSiteID) {
			return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot cancel another site's room move draft."}
		}
		if !canMutateRoomMoveReviewRow(config, seed) {
			return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"author": "This persona can view this room move but only the original author or IT Admin can cancel it."}
		}
		s.canceled[draftID] = true
		return roomMoveDraftPayload{
			ID:          draftID,
			Mode:        seed.MoveType,
			Status:      "canceled",
			ScopeSiteID: seed.CurrentSiteID,
			ScopeSite:   seed.CurrentSite,
			Rows: []roomMoveDraftRow{{
				ID:                 seed.ID,
				Person:             seed.Person,
				Email:              seed.Email,
				EmployeeID:         seed.EmployeeID,
				CurrentSiteID:      seed.CurrentSiteID,
				CurrentSite:        seed.CurrentSite,
				CurrentRoom:        seed.CurrentRoom,
				DestinationSiteID:  seed.DestinationSiteID,
				DestinationSite:    seed.DestinationSite,
				DestinationRoomID:  seed.DestinationRoomID,
				DestinationRoom:    seed.DestinationRoom,
				Phone:              seed.Phone,
				Warning:            seed.Warning,
				AttentionReason:    seed.AttentionReason,
				AutomationOutcome:  seed.AutomationOutcome,
				ManualActionOwner:  seed.ManualActionOwner,
				ManualActionReason: seed.ManualActionReason,
				ResolutionSteps:    seed.ResolutionSteps,
				ExternalSystems:    seed.ExternalSystems,
			}},
			Author:            seed.Author,
			AuthorID:          seed.AuthorID,
			CanEdit:           canMutateRoomMoveReviewRow(config, seed),
			CanDelete:         canMutateRoomMoveReviewRow(config, seed),
			CanManageDistrict: canManageDistrictRoomMoves(config),
		}, http.StatusOK, nil
	}
	return roomMoveDraftPayload{}, http.StatusNotFound, map[string]string{"draft": "Draft not found."}
}

// deleteDraft removes an unsent draft and its local status markers from the DEV
// store. The endpoint returns 204 on success so drawer cleanup can treat deletion
// as a silent discard rather than a new workflow state.
func (s *devRoomMoveStoreState) deleteDraft(config devPersonaConfig, draftID string) (int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[draftID]
	if !ok {
		return http.StatusNotFound, map[string]string{"draft": "Draft not found."}
	}
	if !canAccessRoomMoveSite(config, draft.ScopeSiteID) {
		return http.StatusForbidden, map[string]string{"scope": "This persona cannot delete another site's room move draft."}
	}
	if !canMutateRoomMoveDraft(config, draft) {
		return http.StatusForbidden, map[string]string{"author": "This persona can view this room move but only the original author or IT Admin can delete it."}
	}
	delete(s.drafts, draftID)
	delete(s.completed, draftID)
	delete(s.canceled, draftID)
	return http.StatusNoContent, nil
}

func (s *devRoomMoveStoreState) completedJobs(config devPersonaConfig) []roomMoveCompletedJobPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]roomMoveCompletedJobPayload, 0, len(s.jobs))
	for _, job := range s.jobs {
		if canAccessRoomMoveSite(config, job.ScopeSiteID) {
			jobs = append(jobs, job)
		}
	}
	slices.SortFunc(jobs, func(a, b roomMoveCompletedJobPayload) int {
		return strings.Compare(a.ID, b.ID)
	})
	return jobs
}

// scheduleRevert builds one scheduled DEV draft from a completed job by swapping
// every row's destination back to its previous room. Repeated calls return the
// same revert draft id when one is already attached to the job.
func (s *devRoomMoveStoreState) scheduleRevert(config devPersonaConfig, jobID string) (roomMoveDraftPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return roomMoveDraftPayload{}, http.StatusNotFound, map[string]string{"job": "Completed room move job not found."}
	}
	if !canAccessRoomMoveSite(config, job.ScopeSiteID) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot revert another site's room move job."}
	}
	if job.RevertDraftID != "" {
		if draft, ok := s.drafts[job.RevertDraftID]; ok {
			return draft, http.StatusOK, nil
		}
	}
	s.nextID++
	revertID := fmt.Sprintf("rm-draft-%03d", s.nextID)
	rows := make([]roomMoveDraftRow, 0, len(job.Rows))
	for _, row := range job.Rows {
		rows = append(rows, roomMoveDraftRow{
			ID:                "revert-" + row.ID,
			PersonID:          row.PersonID,
			DestinationSiteID: row.CurrentSiteID,
			DestinationRoomID: row.CurrentRoomID,
			Action:            "revert",
		})
	}
	draft, status, errors := buildRoomMoveDraft(config, revertID, roomMoveDraftRequest{
		Mode:          job.Mode,
		ScopeSiteID:   job.ScopeSiteID,
		EffectiveDate: "2026-05-11",
		Rows:          rows,
	})
	if status != http.StatusOK {
		return roomMoveDraftPayload{}, status, errors
	}
	draft.Status = "scheduled"
	draft.ScheduledFor = "2026-05-11T20:00:00-07:00"
	s.drafts[draft.ID] = draft
	job.RevertDraftID = draft.ID
	job.RevertStatus = draft.Status
	s.jobs[jobID] = job
	return draft, http.StatusOK, nil
}

func (s *devRoomMoveStoreState) ensureBulkDraft(config devPersonaConfig, draftID string, defaultMode string) roomMoveDraftPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	if draft, ok := s.drafts[draftID]; ok && canAccessRoomMoveSite(config, draft.ScopeSiteID) {
		return roomMoveDraftWithPermissions(config, draft)
	}
	mode := defaultMode
	if mode == "" {
		mode = roomMoveTypeBulkRoster
	}
	newID := strings.TrimSpace(draftID)
	if newID == "" {
		s.nextID++
		newID = fmt.Sprintf("rm-draft-%03d", s.nextID)
	}
	scopeSiteID := roomMovesScopeSite(config).ID
	if newID == "rm-draft-103" {
		scopeSiteID = "clover-hs"
	}
	request := roomMoveDraftRequest{Mode: mode, ScopeSiteID: scopeSiteID}
	if newID == "rm-draft-103" {
		request.Rows = seededBulkDraftFeedbackRows()
	}
	draft, _, _ := buildRoomMoveDraft(config, newID, request)
	if seed, ok := seedRoomMoveReviewRowByDraftID(newID); ok {
		draft.Author = seed.Author
		draft.AuthorID = seed.AuthorID
	}
	s.drafts[draft.ID] = draft
	return roomMoveDraftWithPermissions(config, draft)
}

func buildRoomMoveDraft(config devPersonaConfig, draftID string, request roomMoveDraftRequest) (roomMoveDraftPayload, int, map[string]string) {
	mode := strings.TrimSpace(request.Mode)
	if mode == "" {
		mode = roomMoveTypeSingle
	}
	if mode != roomMoveTypeSingle && mode != roomMoveTypeBulkRoster && mode != roomMoveTypeBuildList {
		return roomMoveDraftPayload{}, http.StatusBadRequest, map[string]string{"mode": "Unsupported room move draft mode."}
	}
	scopeSite := resolveRoomMoveScopeSite(config, request.ScopeSiteID)
	if !canAccessRoomMoveSite(config, scopeSite.ID) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope_site_id": "This persona cannot create a draft for that site."}
	}
	effectiveDate := request.EffectiveDate
	if effectiveDate == "" {
		effectiveDate = "2026-07-27"
	}
	rows := request.Rows
	if mode == roomMoveTypeSingle && len(rows) == 0 && request.PersonID != "" {
		person, ok := roomMovePersonByID(request.PersonID)
		if !ok {
			return roomMoveDraftPayload{}, http.StatusBadRequest, map[string]string{"person_id": "Unknown person."}
		}
		rows = []roomMoveDraftRow{draftRowFromPerson(person, person.SiteID, person.CurrentRoomID)}
	}
	if mode == roomMoveTypeBulkRoster && len(rows) == 0 {
		for _, person := range roomMovePeopleForSite(scopeSite.ID) {
			rows = append(rows, draftRowFromPerson(person, person.SiteID, "none"))
		}
	}
	if mode == roomMoveTypeBuildList && rows == nil {
		rows = []roomMoveDraftRow{}
	}
	normalizedRows, warnings, status, errors := normalizeRoomMoveRows(config, scopeSite, rows)
	if status != http.StatusOK {
		return roomMoveDraftPayload{}, status, errors
	}
	return roomMoveDraftPayload{
		ID:                draftID,
		Mode:              mode,
		Status:            roomMoveDraftStatusOpen,
		ScopeSiteID:       scopeSite.ID,
		ScopeSite:         scopeSite.Name,
		EffectiveDate:     effectiveDate,
		Author:            config.Persona.DisplayName,
		AuthorID:          config.Persona.ID,
		Warnings:          warnings,
		Rows:              normalizedRows,
		CanEdit:           true,
		CanDelete:         true,
		CanManageDistrict: canManageDistrictRoomMoves(config),
	}, http.StatusOK, nil
}

// normalizeRoomMoveRows converts client-supplied draft rows into the canonical DEV mock payload saved by createDraft
// and updateDraft. The action-specific branches are user-visible: add rows clear prior room state, and removal rows
// always save destination room None so reloads continue to represent removal from room phones, SLGs, and queues. The
// repeated-user pass preserves every same-person row and models the Phase 3 planner contract for one primary desk-phone
// owner plus secondary, tertiary, or later-order shared-line-group memberships.
func normalizeRoomMoveRows(config devPersonaConfig, scopeSite devSiteContext, rows []roomMoveDraftRow) ([]roomMoveDraftRow, []string, int, map[string]string) {
	normalized := make([]roomMoveDraftRow, 0, len(rows))
	warnings := []string{}
	for index, row := range rows {
		person, ok := roomMovePersonByID(row.PersonID)
		if !ok {
			return nil, nil, http.StatusBadRequest, map[string]string{fmt.Sprintf("rows.%d.person_id", index): "Unknown person."}
		}
		if !canAccessRoomMoveSite(config, person.SiteID) {
			return nil, nil, http.StatusForbidden, map[string]string{fmt.Sprintf("rows.%d.person_id", index): "This persona cannot move that person."}
		}
		destinationSiteID := row.DestinationSiteID
		if destinationSiteID == "" {
			destinationSiteID = person.SiteID
		}
		if !canManageDistrictRoomMoves(config) {
			destinationSiteID = scopeSite.ID
		}
		destinationSite := siteByID(destinationSiteID)
		action := row.Action
		if action == "" {
			action = "change"
		}
		destinationRoomID := row.DestinationRoomID
		if action == "removal" {
			destinationRoomID = "none"
		} else if destinationSiteID != person.SiteID {
			destinationRoomID = "none"
		} else if destinationRoomID == "" {
			destinationRoomID = person.CurrentRoomID
		}
		destinationRole := normalizeRoomMoveDestinationRole(row.DestinationRole)
		if action == "removal" {
			destinationRole = "removal"
		}
		if roomMoveIsSameStableRoom(person, destinationSiteID, destinationRoomID, action) {
			return nil, nil, http.StatusBadRequest, map[string]string{fmt.Sprintf("rows.%d.destination_room_id", index): fmt.Sprintf("%s is already in %s. Choose a different destination room.", person.Name, person.CurrentRoom)}
		}
		room := roomMoveRoomByID(destinationRoomID, destinationSiteID)
		warning := row.Warning
		attentionReason := row.AttentionReason
		automationOutcome := row.AutomationOutcome
		manualActionOwner := row.ManualActionOwner
		manualActionReason := row.ManualActionReason
		resolutionSteps := row.ResolutionSteps
		externalSystems := row.ExternalSystems
		fallbackTicket := row.FallbackTicket
		fallbackTicketHref := sanitizeRoomMoveFallbackTicketHref(row.FallbackTicketHref)
		fallbackStatus := row.FallbackStatus
		technicalOutcome := row.TechnicalOutcome
		phoneOutcome := person.Phone
		if destinationRoomID == "none" && person.CurrentRoomID != "none" && !roomMoveIsNeutralRolloverPlaceholder(person, destinationSiteID, destinationRoomID, action) {
			warning = fmt.Sprintf("Destination room for %s is None; phone and room assignments will be removed.", person.Name)
			warnings = appendUniqueString(warnings, warning)
			phoneOutcome = "Remove phone and SLGs; convert room to common area"
		}
		if destinationSiteID != person.SiteID {
			warning = "Inter-site move: destination room is set to none until the destination site confirms the room."
			warnings = appendUniqueString(warnings, warning)
		}
		if primaryOwner, ok := roomMoveActivePrimaryOwner(destinationRoomID, person.ID); ok {
			warning = primaryConflictWarning(person.Name, room.Label, primaryOwner)
			attentionReason = primaryConflictAttentionReason(person.Name, room.Label, primaryOwner)
			automationOutcome = primaryConflictAutomationOutcome(person.Name, room.Label)
			resolutionSteps = primaryConflictResolutionSteps(person.Name, room.Label)
			externalSystems = nil
			phoneOutcome = "Add to room shared line group; keep primary phone owner"
			manualActionOwner = ""
			manualActionReason = ""
			warnings = appendUniqueString(warnings, warning)
		}
		currentRoomID := person.CurrentRoomID
		currentRoom := person.CurrentRoom
		if action == "add" {
			currentRoomID = "none"
			currentRoom = ""
		}
		normalized = append(normalized, roomMoveDraftRow{
			ID:                 firstNonEmpty(row.ID, fmt.Sprintf("row-%02d-%s", index+1, person.ID)),
			PersonID:           person.ID,
			Person:             person.Name,
			Email:              person.Email,
			EmployeeID:         person.EmployeeID,
			CurrentSiteID:      person.SiteID,
			CurrentSite:        person.Site,
			CurrentRoomID:      currentRoomID,
			CurrentRoom:        currentRoom,
			SourceRole:         person.SourceRole,
			DestinationSiteID:  destinationSiteID,
			DestinationSite:    destinationSite.Name,
			DestinationRoomID:  destinationRoomID,
			DestinationRoom:    room.Label,
			DestinationRole:    destinationRole,
			Phone:              phoneOutcome,
			Action:             action,
			Warning:            warning,
			AttentionReason:    attentionReason,
			AutomationOutcome:  automationOutcome,
			ManualActionOwner:  manualActionOwner,
			ManualActionReason: manualActionReason,
			ResolutionSteps:    resolutionSteps,
			ExternalSystems:    externalSystems,
			FallbackTicket:     fallbackTicket,
			FallbackTicketHref: fallbackTicketHref,
			FallbackStatus:     fallbackStatus,
			TechnicalOutcome:   technicalOutcome,
		})
	}
	normalized, repeatedWarnings := applyRepeatedUserRoomMovePlanning(normalized)
	warnings = roomMoveWarningsFromRows(normalized)
	for _, warning := range repeatedWarnings {
		warnings = appendUniqueString(warnings, warning)
	}
	return normalized, warnings, http.StatusOK, nil
}

func normalizeRoomMoveDestinationRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "primary":
		return "primary"
	case "secondary":
		return "secondary"
	case "tertiary":
		return "tertiary"
	case "slg_only", "slg-only", "member":
		return "member"
	default:
		return ""
	}
}

func sanitizeRoomMoveFallbackTicketHref(href string) string {
	parsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch parsed.Scheme {
	case "http", "https":
		return parsed.String()
	default:
		return ""
	}
}

// roomMoveWarningsFromRows rebuilds draft-level warning copy after planner passes finish mutating rows. The DEV draft
// create/update handlers use the returned list for drawer and summary gating, so it must mirror the warnings still
// attached to saved rows instead of retaining conflicts that a later planner pass downgraded to shared-line-group work.
func roomMoveWarningsFromRows(rows []roomMoveDraftRow) []string {
	warnings := []string{}
	for _, row := range rows {
		warnings = appendUniqueString(warnings, row.Warning)
	}
	return warnings
}

// applyRepeatedUserRoomMovePlanning is the DEV mock stand-in for the future live Room Moves planner. It handles
// same-person bulk rows as one planning group so rows are not dropped and secondary destinations never steal the desk
// phone from the one resolved primary room. When no row is explicitly primary, it first uses source-role and current-room
// data to infer the retained source room as primary when that choice is unique. Ambiguous repeated-primary input is
// returned as actionable review output instead of silently selecting one room.
func applyRepeatedUserRoomMovePlanning(rows []roomMoveDraftRow) ([]roomMoveDraftRow, []string) {
	indexesByPerson := map[string][]int{}
	for index, row := range rows {
		indexesByPerson[row.PersonID] = append(indexesByPerson[row.PersonID], index)
	}
	warnings := []string{}
	for _, indexes := range indexesByPerson {
		if len(indexes) < 2 {
			continue
		}
		activeIndexes := []int{}
		primaryIndexes := []int{}
		for _, index := range indexes {
			row := rows[index]
			if row.Action == "removal" || row.DestinationRoomID == "none" {
				continue
			}
			activeIndexes = append(activeIndexes, index)
			if row.DestinationRole == "primary" {
				primaryIndexes = append(primaryIndexes, index)
			}
		}
		if len(activeIndexes) < 2 {
			continue
		}
		personName := rows[indexes[0]].Person
		if len(primaryIndexes) == 0 {
			if inferredPrimaryIndex, ok := inferRepeatedUserPrimaryIndex(rows, activeIndexes); ok {
				rows[inferredPrimaryIndex].DestinationRole = "primary"
				primaryIndexes = append(primaryIndexes, inferredPrimaryIndex)
			}
		}
		switch {
		case len(primaryIndexes) > 1:
			roomLabels := repeatedRoomLabels(rows, primaryIndexes)
			warning := fmt.Sprintf("Ambiguous repeated-user primary room: %s is marked primary for %s; choose one primary before execution.", personName, strings.Join(roomLabels, ", "))
			for _, index := range activeIndexes {
				rows[index].Warning = warning
				rows[index].AttentionReason = fmt.Sprintf("%s appears in multiple primary destination rows, so automation cannot safely decide which room owns the desk phone.", personName)
				rows[index].AutomationOutcome = "Hold primary phone assignment until one primary room is selected; preserve shared-line-group membership planning for the remaining destination rooms."
				rows[index].Phone = "Review required before primary phone assignment"
				rows[index].ResolutionSteps = []string{
					"Select exactly one destination row as primary.",
					"Keep additional destination rows as secondary, tertiary, or later-order shared-line-group memberships.",
					"Re-run the Room Moves review before scheduling or applying the batch.",
				}
				rows[index].ExternalSystems = []string{"Zoom primary phone assignment", "Zoom room shared line group", "IncidentIQ room association"}
			}
			warnings = appendUniqueString(warnings, warning)
		case len(primaryIndexes) == 1:
			primaryIndex := primaryIndexes[0]
			memberOrdinal := 0
			for _, index := range activeIndexes {
				if index == primaryIndex {
					rows[index].Phone = repeatedPrimaryPhoneOutcome(rows[index])
					rows[index].AutomationOutcome = repeatedPrimaryAutomationOutcome(rows[index])
					continue
				}
				memberOrdinal++
				rows[index].DestinationRole = repeatedMemberRole(memberOrdinal)
				rows[index].Warning = ""
				rows[index].AttentionReason = ""
				rows[index].AutomationOutcome = repeatedMemberAutomationOutcome(rows[index])
				rows[index].Phone = repeatedMemberPhoneOutcome(rows[index])
				rows[index].ManualActionOwner = ""
				rows[index].ManualActionReason = ""
				rows[index].ResolutionSteps = []string{
					fmt.Sprintf("Keep %s as the primary room owner for the selected primary destination row.", personName),
					fmt.Sprintf("Add %s to the %s room shared line group only.", personName, rows[index].DestinationRoom),
					"Verify Zoom shared-line-group membership during execution.",
				}
				rows[index].ExternalSystems = []string{"Zoom room shared line group", "IncidentIQ room association"}
			}
		default:
			roomLabels := repeatedRoomLabels(rows, activeIndexes)
			warning := fmt.Sprintf("Repeated-user room planning needs primary selection: %s appears in %s with no primary destination role.", personName, strings.Join(roomLabels, ", "))
			for _, index := range activeIndexes {
				rows[index].Warning = warning
				rows[index].AttentionReason = fmt.Sprintf("%s appears in multiple destination rooms, but none is marked as the primary desk-phone owner.", personName)
				rows[index].AutomationOutcome = "Hold primary phone assignment until one destination is marked primary; keep rows available for secondary or tertiary shared-line-group planning."
				rows[index].Phone = "Review required before primary phone assignment"
				rows[index].ResolutionSteps = []string{
					"Mark one row as primary if the person should own a desk phone in that room.",
					"Mark the remaining rows as secondary, tertiary, or member destinations.",
					"Use shared-line-group-only roles when the source data shows the person is not a primary room owner.",
				}
				rows[index].ExternalSystems = []string{"Zoom primary phone assignment", "Zoom room shared line group", "IncidentIQ room association"}
			}
			warnings = appendUniqueString(warnings, warning)
		}
	}
	return rows, warnings
}

// inferRepeatedUserPrimaryIndex applies the source-data branch of the repeated-user planner contract. A retained
// current-room row can become the primary destination only when the source record says that room was a primary owner
// role and no other retained source-room candidate exists; SLG-only source records still require explicit operator
// selection because they do not prove desk-phone ownership.
func inferRepeatedUserPrimaryIndex(rows []roomMoveDraftRow, activeIndexes []int) (int, bool) {
	candidates := []int{}
	for _, index := range activeIndexes {
		row := rows[index]
		if row.CurrentRoomID == "" || row.CurrentRoomID == "none" {
			continue
		}
		if row.DestinationRoomID != row.CurrentRoomID {
			continue
		}
		switch row.SourceRole {
		case "primary", "last_primary":
			candidates = append(candidates, index)
		}
	}
	if len(candidates) != 1 {
		return 0, false
	}
	return candidates[0], true
}

func repeatedRoomLabels(rows []roomMoveDraftRow, indexes []int) []string {
	labels := []string{}
	for _, index := range indexes {
		labels = append(labels, rows[index].DestinationRoom)
	}
	return labels
}

func repeatedMemberRole(ordinal int) string {
	switch ordinal {
	case 1:
		return "secondary"
	case 2:
		return "tertiary"
	default:
		return "member"
	}
}

func repeatedPrimaryPhoneOutcome(row roomMoveDraftRow) string {
	if roomMoveRoomHasCommonAreaPhone(row.DestinationRoomID) {
		return "Assign room phone; replace common-area phone after membership is verified"
	}
	return row.Phone
}

func repeatedPrimaryAutomationOutcome(row roomMoveDraftRow) string {
	if roomMoveRoomHasCommonAreaPhone(row.DestinationRoomID) {
		return fmt.Sprintf("Make %s the primary phone owner for %s, add them to the room shared line group, then retire the common-area phone only after Zoom verifies human coverage.", row.Person, row.DestinationRoom)
	}
	return row.AutomationOutcome
}

func repeatedMemberPhoneOutcome(row roomMoveDraftRow) string {
	if roomMoveRoomHasCommonAreaPhone(row.DestinationRoomID) {
		return "Add to room shared line group; keep common-area phone active"
	}
	return "Add to room shared line group; no desk phone assignment"
}

func repeatedMemberAutomationOutcome(row roomMoveDraftRow) string {
	if roomMoveRoomHasCommonAreaPhone(row.DestinationRoomID) {
		return fmt.Sprintf("Add %s to the %s room shared line group as %s coverage and keep the common-area phone active.", row.Person, row.DestinationRoom, row.DestinationRole)
	}
	return fmt.Sprintf("Add %s to the %s room shared line group as %s coverage without changing the primary desk-phone owner.", row.Person, row.DestinationRoom, row.DestinationRole)
}

// seededBulkDraftFeedbackRows keeps issue #55's browser-verification fixture stable for /room-moves/bulk-draft?draft_id=rm-draft-103.
// The first row demonstrates add semantics with no previous room, and the second row demonstrates removal semantics
// where the destination room is persisted as None after normalization.
func seededBulkDraftFeedbackRows() []roomMoveDraftRow {
	return []roomMoveDraftRow{
		{
			ID:                "row-alex-ramirez-add",
			PersonID:          "alex-ramirez",
			DestinationSiteID: "clover-hs",
			DestinationRoomID: "cla-a104",
			Action:            "add",
		},
		{
			ID:                "row-morgan-lee-removal",
			PersonID:          "morgan-lee",
			DestinationSiteID: "clover-hs",
			DestinationRoomID: "cla-b204",
			Action:            "removal",
		},
	}
}

func roomMoveSummaryCards(rows []roomMoveReviewRow) []summaryCardPayload {
	warnings := 0
	immediate := 0
	batch := 0
	for _, row := range rows {
		if row.Warning != "" {
			warnings++
		}
		if row.MoveType == roomMoveTypeSingle {
			immediate++
		} else {
			batch++
		}
	}
	return []summaryCardPayload{
		{Title: "Draft Moves", Count: fmt.Sprintf("%d", len(rows))},
		{Title: "Warnings", Count: fmt.Sprintf("%d", warnings)},
		{Title: "Immediate", Count: fmt.Sprintf("%d", immediate)},
		{Title: "Batch Cutovers", Count: fmt.Sprintf("%d", batch)},
	}
}

func canManageDistrictRoomMoves(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin"
}

// canMutateRoomMoveDraft separates site-scoped visibility from edit authority
// for DEV room-move drafts. The mutation handlers call it after route and site
// checks so Site Admin and Site Secretary users may see assigned-site drafts
// from IT or another operator but cannot save, apply, cancel, or delete them.
func canMutateRoomMoveDraft(config devPersonaConfig, draft roomMoveDraftPayload) bool {
	if canManageDistrictRoomMoves(config) {
		return true
	}
	return draft.AuthorID != "" && draft.AuthorID == config.Persona.ID
}

// canMutateRoomMoveReviewRow applies the same author-ownership rule to seeded
// review rows that do not yet exist in the in-memory draft map. Seeded rows are
// direct DEV mutation targets, so update and cancel calls must reject them
// before converting the row into a stored draft for another persona.
func canMutateRoomMoveReviewRow(config devPersonaConfig, row roomMoveReviewRow) bool {
	if canManageDistrictRoomMoves(config) {
		return true
	}
	return row.AuthorID != "" && row.AuthorID == config.Persona.ID
}

// roomMoveDraftWithPermissions computes caller-specific edit flags immediately
// before a DEV draft leaves the store. The stored draft keeps its true author,
// while each page/API response reflects whether the active persona may mutate it.
func roomMoveDraftWithPermissions(config devPersonaConfig, draft roomMoveDraftPayload) roomMoveDraftPayload {
	editable := canMutateRoomMoveDraft(config, draft)
	draft.CanEdit = editable
	draft.CanDelete = editable
	draft.CanManageDistrict = canManageDistrictRoomMoves(config)
	return draft
}

func roomMovesScopeSite(config devPersonaConfig) devSiteContext {
	if canManageDistrictRoomMoves(config) {
		return config.DefaultSite
	}
	return config.CurrentSite
}

func resolveRoomMoveScopeSite(config devPersonaConfig, siteID string) devSiteContext {
	if siteID == "" {
		return roomMovesScopeSite(config)
	}
	if site, ok := devSiteCatalog[siteID]; ok {
		return site
	}
	return roomMovesScopeSite(config)
}

func canAccessRoomMoveSite(config devPersonaConfig, siteID string) bool {
	if canManageDistrictRoomMoves(config) {
		return true
	}
	return roomMovesScopeSite(config).ID == siteID
}

func roomMoveVisibleSites(config devPersonaConfig) []devSiteContext {
	if canManageDistrictRoomMoves(config) {
		return sitesByID(devSiteOrder...)
	}
	return []devSiteContext{roomMovesScopeSite(config)}
}

func roomMovePeopleForConfig(config devPersonaConfig) []roomMovePersonOption {
	people := []roomMovePersonOption{}
	for _, person := range roomMovePeopleSeed() {
		if canAccessRoomMoveSite(config, person.SiteID) {
			people = append(people, person)
		}
	}
	return people
}

func roomMovePeopleForSite(siteID string) []roomMovePersonOption {
	people := []roomMovePersonOption{}
	for _, person := range roomMovePeopleSeed() {
		if person.SiteID == siteID {
			people = append(people, person)
		}
	}
	return people
}

func roomMoveRoomsForConfig(config devPersonaConfig) []roomMoveRoomOption {
	rooms := []roomMoveRoomOption{}
	seen := map[string]bool{}
	for _, site := range roomMoveVisibleSites(config) {
		for _, room := range roomMoveRoomsForSite(site.ID) {
			key := room.ID
			if room.ID == "none" {
				key = "none"
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			rooms = append(rooms, room)
		}
	}
	return rooms
}

func roomMoveRoomsForSite(siteID string) []roomMoveRoomOption {
	site := siteByID(siteID)
	rooms := []roomMoveRoomOption{{ID: "none", Label: "None", SiteID: site.ID, Site: site.Name}}
	switch siteID {
	case "clover-hs":
		rooms = append(rooms,
			roomMoveRoomOption{ID: "cla-a104", Label: "A-104", SiteID: site.ID, Site: site.Name},
			roomMoveRoomOption{ID: "cla-a108", Label: "A-108", SiteID: site.ID, Site: site.Name},
			roomMoveRoomOption{ID: "cla-b204", Label: "B-204", SiteID: site.ID, Site: site.Name},
			roomMoveRoomOption{ID: "cla-b210", Label: "B-210", SiteID: site.ID, Site: site.Name},
		)
	case "desert-view":
		rooms = append(rooms,
			roomMoveRoomOption{ID: "dve-c118", Label: "C-118", SiteID: site.ID, Site: site.Name},
			roomMoveRoomOption{ID: "dve-c122", Label: "C-122", SiteID: site.ID, Site: site.Name},
		)
	case "franklin-ms":
		rooms = append(rooms,
			roomMoveRoomOption{ID: "fms-d102", Label: "D-102", SiteID: site.ID, Site: site.Name},
			roomMoveRoomOption{ID: "fms-d112", Label: "D-112", SiteID: site.ID, Site: site.Name},
		)
	default:
		rooms = append(rooms, roomMoveRoomOption{ID: siteID + "-main-office", Label: "Main Office", SiteID: site.ID, Site: site.Name})
	}
	return rooms
}

func roomMovePeopleSeed() []roomMovePersonOption {
	return []roomMovePersonOption{
		{ID: "alex-ramirez", Name: "Alex Ramirez", Email: "alex.ramirez@wusd.org", EmployeeID: "103118", Role: "IT Admin", SiteID: "clover-hs", Site: "Clover High School", CurrentRoomID: "cla-a104", CurrentRoom: "A-104", SourceRole: "primary", Phone: "51042"},
		{ID: "morgan-lee", Name: "Morgan Lee", Email: "morgan.lee@wusd.org", EmployeeID: "103442", Role: "Teacher", SiteID: "clover-hs", Site: "Clover High School", CurrentRoomID: "cla-b210", CurrentRoom: "B-210", SourceRole: "last_primary", Phone: "51017"},
		{ID: "casey-nguyen", Name: "Casey Nguyen", Email: "casey.nguyen@wusd.org", EmployeeID: "105887", Role: "Instructional Specialist", SiteID: "clover-hs", Site: "Clover High School", CurrentRoomID: "cla-a104", CurrentRoom: "A-104", SourceRole: "slg_only", Phone: ""},
		{ID: "taylor-quinn", Name: "Taylor Quinn", Email: "taylor.quinn@wusd.org", EmployeeID: "106103", Role: "Contractor", SiteID: "desert-view", Site: "Desert View Elementary", CurrentRoomID: "none", CurrentRoom: "None", Phone: ""},
		{ID: "jamie-reed", Name: "Jamie Reed", Email: "jamie.reed@wusd.org", EmployeeID: "103772", Role: "Teacher", SiteID: "desert-view", Site: "Desert View Elementary", CurrentRoomID: "dve-c118", CurrentRoom: "C-118", SourceRole: "last_primary", Phone: "52013"},
		{ID: "nia-brooks", Name: "Nia Brooks", Email: "nia.brooks@wusd.org", EmployeeID: "104012", Role: "Site Secretary", SiteID: "franklin-ms", Site: "Franklin Middle School", CurrentRoomID: "fms-d102", CurrentRoom: "D-102", SourceRole: "primary", Phone: "53022"},
	}
}

func roomMovePersonByID(id string) (roomMovePersonOption, bool) {
	for _, person := range roomMovePeopleSeed() {
		if person.ID == id {
			return person, true
		}
	}
	return roomMovePersonOption{}, false
}

// roomMovePersonIDByEmail lets updateDraft turn a seeded review row into an
// editable in-memory draft under the same draft id. The seed payload carries an
// email address for the drawer, while buildRoomMoveDraft requires the stable
// person id before it will normalize and persist the edited row.
func roomMovePersonIDByEmail(email string) string {
	for _, person := range roomMovePeopleSeed() {
		if person.Email == email {
			return person.ID
		}
	}
	return ""
}

// roomMoveIsSameStableRoom is the DEV mock validation predicate shared by draft
// normalization and transition checks. Add, removal, revert, and None
// destinations are real operations, but ordinary change rows must not write a
// future room move when the stable destination id equals the person's current
// room id at the same site.
func roomMoveIsSameStableRoom(person roomMovePersonOption, destinationSiteID string, destinationRoomID string, action string) bool {
	if action == "add" || action == "removal" || action == "revert" || destinationRoomID == "" || destinationRoomID == "none" {
		return false
	}
	return person.SiteID == destinationSiteID && person.CurrentRoomID == destinationRoomID
}

func roomMoveIsNeutralRolloverPlaceholder(person roomMovePersonOption, destinationSiteID string, destinationRoomID string, action string) bool {
	return action == "change" && person.SiteID == destinationSiteID && destinationRoomID == "none" && person.CurrentRoomID != "none"
}

// validateRoomMoveRowsForTransition reruns the same-room guard immediately
// before schedule/apply so stale DEV drafts or manually constructed payloads
// cannot bypass create/update validation and become planned provider work. It
// also keeps untouched site-rollover placeholder rows from turning into
// implicit room-removal work during schedule or apply.
func validateRoomMoveRowsForTransition(rows []roomMoveDraftRow) (int, map[string]string) {
	for index, row := range rows {
		person, ok := roomMovePersonByID(row.PersonID)
		if !ok {
			return http.StatusBadRequest, map[string]string{fmt.Sprintf("rows.%d.person_id", index): "Unknown person."}
		}
		if roomMoveIsNeutralRolloverPlaceholder(person, row.DestinationSiteID, row.DestinationRoomID, row.Action) {
			return http.StatusBadRequest, map[string]string{fmt.Sprintf("rows.%d.destination_room_id", index): fmt.Sprintf("Choose a destination room for %s before scheduling or applying the room move.", person.Name)}
		}
		if roomMoveIsSameStableRoom(person, row.DestinationSiteID, row.DestinationRoomID, row.Action) {
			return http.StatusBadRequest, map[string]string{fmt.Sprintf("rows.%d.destination_room_id", index): fmt.Sprintf("%s is already in %s. Choose a different destination room.", person.Name, person.CurrentRoom)}
		}
	}
	return http.StatusOK, nil
}

func roomMoveRoomByID(roomID string, siteID string) roomMoveRoomOption {
	for _, room := range roomMoveRoomsForSite(siteID) {
		if room.ID == roomID {
			return room
		}
	}
	return roomMoveRoomsForSite(siteID)[0]
}

// roomMoveRoomIDByLabel maps seeded review-row display labels back to DEV room
// option ids so the React drawer can reopen an existing mock row with the same
// target room selected. It is used only for deterministic seed rows; normal
// draft rows already carry their destination room id from the API request.
func roomMoveRoomIDByLabel(siteID string, label string) string {
	for _, room := range roomMoveRoomsForSite(siteID) {
		if room.Label == label {
			return room.ID
		}
	}
	return "none"
}

// roomMoveActivePrimaryOwner models the DEV Zoom inventory condition that makes
// a destination classroom unsafe for primary-phone reassignment. Room Moves
// normalization calls this before returning mock rows so the page can explain
// that the correct automated path is shared-line-group membership, not a vague
// manual ticket.
func roomMoveActivePrimaryOwner(destinationRoomID string, movingPersonID string) (string, bool) {
	if destinationRoomID == "cla-b204" && movingPersonID != "jordan-patel" {
		return "Jordan Patel", true
	}
	return "", false
}

// roomMoveRoomHasCommonAreaPhone marks deterministic DEV fixtures where Zoom
// has CAP/common-area coverage for an otherwise unoccupied room. The repeated-user
// planning pass uses this to show safe CAP-to-human or CAP-preserving outcomes
// without touching the Room Moves UI artboards owned by the active UI branch.
func roomMoveRoomHasCommonAreaPhone(roomID string) bool {
	switch roomID {
	case "cla-a108", "dve-c122", "fms-d112":
		return true
	default:
		return false
	}
}

// primaryConflictWarning is the short warning line surfaced in summary and
// drawer warning areas when a destination room already has an active primary
// room owner. It keeps the table state compact while the structured fields below
// provide the L1-operable explanation.
func primaryConflictWarning(personName string, destinationRoom string, primaryOwner string) string {
	return fmt.Sprintf("%s already has active primary room owner %s; %s will be added to the room shared line group.", destinationRoom, primaryOwner, personName)
}

// primaryConflictAttentionReason explains why a Room Moves row needs operator
// attention without implying that IT must create a manual ticket. The frontend
// renders this in the shared right drawer for seeded and newly created DEV rows.
func primaryConflictAttentionReason(personName string, destinationRoom string, primaryOwner string) string {
	return fmt.Sprintf("%s already has %s as the active primary room owner, so %s should not take over the primary phone assignment.", destinationRoom, primaryOwner, personName)
}

// primaryConflictAutomationOutcome describes the planned automated Zoom outcome
// for active primary-room conflicts. Future live worker code should implement
// this as an idempotent shared-line-group membership update and leave the
// existing primary phone owner unchanged.
func primaryConflictAutomationOutcome(personName string, destinationRoom string) string {
	return fmt.Sprintf("Add %s to the %s room shared line group and leave the room's primary phone owner unchanged.", personName, destinationRoom)
}

// primaryConflictResolutionSteps gives L1 operators enough detail to review and
// release a primary-room conflict row without opening a separate manual ticket
// unless the later Zoom verification step fails.
func primaryConflictResolutionSteps(personName string, destinationRoom string) []string {
	return []string{
		fmt.Sprintf("Confirm %s is the correct destination room.", destinationRoom),
		fmt.Sprintf("Schedule or apply the draft; automation will add %s to the %s room shared line group.", personName, destinationRoom),
		"Escalate to IT only if Zoom cannot verify the shared line group membership during execution.",
	}
}

func draftRowFromPerson(person roomMovePersonOption, destinationSiteID string, destinationRoomID string) roomMoveDraftRow {
	destinationSite := siteByID(destinationSiteID)
	room := roomMoveRoomByID(destinationRoomID, destinationSiteID)
	return roomMoveDraftRow{
		ID:                "row-" + person.ID,
		PersonID:          person.ID,
		Person:            person.Name,
		Email:             person.Email,
		EmployeeID:        person.EmployeeID,
		CurrentSiteID:     person.SiteID,
		CurrentSite:       person.Site,
		CurrentRoomID:     person.CurrentRoomID,
		CurrentRoom:       person.CurrentRoom,
		DestinationSiteID: destinationSiteID,
		DestinationSite:   destinationSite.Name,
		DestinationRoomID: destinationRoomID,
		DestinationRoom:   room.Label,
		Phone:             person.Phone,
		Action:            "change",
	}
}

func draftStatusLabel(status string) string {
	switch status {
	case "scheduled":
		return "Scheduled"
	case "complete":
		return "Complete"
	default:
		return "Ready"
	}
}

func warningLevel(warning string) string {
	if warning == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(warning), "primary") {
		return "review"
	}
	return "warning"
}

func appendUniqueString(values []string, value string) []string {
	if value == "" || slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
