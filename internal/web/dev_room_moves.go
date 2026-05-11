package web

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	ID              string `json:"id"`
	DraftID         string `json:"draft_id"`
	MoveType        string `json:"move_type"`
	Person          string `json:"person"`
	Email           string `json:"email"`
	EmployeeID      string `json:"employee_id"`
	CurrentSiteID   string `json:"current_site_id"`
	CurrentSite     string `json:"current_site"`
	CurrentRoom     string `json:"current_room"`
	DestinationSite string `json:"destination_site"`
	DestinationRoom string `json:"destination_room"`
	Phone           string `json:"phone"`
	Author          string `json:"author"`
	State           string `json:"state"`
	Warning         string `json:"warning,omitempty"`
	WarningLevel    string `json:"warning_level,omitempty"`
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
	ID                string `json:"id"`
	PersonID          string `json:"person_id"`
	Person            string `json:"person"`
	Email             string `json:"email"`
	EmployeeID        string `json:"employee_id"`
	CurrentSiteID     string `json:"current_site_id"`
	CurrentSite       string `json:"current_site"`
	CurrentRoomID     string `json:"current_room_id"`
	CurrentRoom       string `json:"current_room"`
	DestinationSiteID string `json:"destination_site_id"`
	DestinationSite   string `json:"destination_site"`
	DestinationRoomID string `json:"destination_room_id"`
	DestinationRoom   string `json:"destination_room"`
	Phone             string `json:"phone"`
	Action            string `json:"action"`
	Warning           string `json:"warning,omitempty"`
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
		nextID:    100,
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
	if !devModeEnabled() || r.Method != http.MethodGet {
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
	if !devModeEnabled() || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	config, ok := authenticatedRoomMovesPersona(w, r)
	if !ok {
		return
	}
	draftID := strings.TrimSpace(r.URL.Query().Get("draft_id"))
	draft := devRoomMoveStore.ensureBulkDraft(config, draftID, roomMoveTypeBulkRoster)
	writeJSON(w, http.StatusOK, roomMovesBulkDraftPayload{
		PageID:      "room-moves-bulk-draft",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Page: roomMovesBulkDraftContent{
			Title:             "Bulk Room Move Draft",
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

func handleDevRoomMoveDrafts(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodPost {
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

func handleDevRoomMoveDraft(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() {
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
	if !devModeEnabled() || r.Method != http.MethodGet {
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
	if !devModeEnabled() {
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
	if !routeAllowed(config, "/room-moves") {
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
	if !canManageDistrictRoomMoves(config) || !routeAllowed(config, "/admin") {
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
		seedRoomMoveReviewRow("single-alex-ramirez", "single-alex-ramirez", roomMoveTypeSingle, "Alex Ramirez", "alex.ramirez@wusd.org", "103118", "clover-hs", "A-104", "clover-hs", "A-108", "Move ext 51042", "Alex Ramirez", "Ready", ""),
		seedRoomMoveReviewRow("single-morgan-lee", "single-morgan-lee", roomMoveTypeSingle, "Morgan Lee", "morgan.lee@wusd.org", "103442", "clover-hs", "B-210", "clover-hs", "B-204", "Manual ticket", "Avery Shah", "Review", "Primary conflict"),
		seedRoomMoveReviewRow("bulk-clover-summer", "rm-draft-103", roomMoveTypeBulkRoster, "Bulk Move", "", "", "clover-hs", "Multiple", "clover-hs", "Multiple", "Batch cutover", "Alex Ramirez", "Scheduled", "Two rows need review before scheduling"),
		seedRoomMoveReviewRow("single-jamie-reed", "single-jamie-reed", roomMoveTypeSingle, "Jamie Reed", "jamie.reed@wusd.org", "103772", "desert-view", "C-118", "desert-view", "None", "Remove phone", "Alex Ramirez", "Review", "Null-room outcome"),
		seedRoomMoveReviewRow("single-nia-brooks", "single-nia-brooks", roomMoveTypeSingle, "Nia Brooks", "nia.brooks@wusd.org", "104012", "franklin-ms", "D-102", "franklin-ms", "D-112", "Assign line", "Avery Shah", "Ready", ""),
	}
	baseDraftIDs := map[string]bool{}
	for _, row := range base {
		baseDraftIDs[row.DraftID] = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, draft := range s.drafts {
		if baseDraftIDs[draft.ID] {
			continue
		}
		if s.completed[draft.ID] || s.canceled[draft.ID] {
			continue
		}
		if !canAccessRoomMoveSite(config, draft.ScopeSiteID) {
			continue
		}
		if len(draft.Rows) == 1 && draft.Mode == roomMoveTypeSingle {
			row := draft.Rows[0]
			base = append(base, roomMoveReviewRow{
				ID:              row.ID,
				DraftID:         draft.ID,
				MoveType:        roomMoveTypeSingle,
				Person:          row.Person,
				Email:           row.Email,
				EmployeeID:      row.EmployeeID,
				CurrentSiteID:   row.CurrentSiteID,
				CurrentSite:     row.CurrentSite,
				CurrentRoom:     row.CurrentRoom,
				DestinationSite: row.DestinationSite,
				DestinationRoom: row.DestinationRoom,
				Phone:           row.Phone,
				Author:          draft.Author,
				State:           draftStatusLabel(draft.Status),
				Warning:         row.Warning,
				WarningLevel:    warningLevel(row.Warning),
			})
			continue
		}
		base = append(base, roomMoveReviewRow{
			ID:              "bulk-" + draft.ID,
			DraftID:         draft.ID,
			MoveType:        draft.Mode,
			Person:          "Bulk Move",
			Email:           "",
			EmployeeID:      "",
			CurrentSiteID:   draft.ScopeSiteID,
			CurrentSite:     draft.ScopeSite,
			CurrentRoom:     "Multiple",
			DestinationSite: draft.ScopeSite,
			DestinationRoom: "Multiple",
			Phone:           fmt.Sprintf("%d rows", len(draft.Rows)),
			Author:          draft.Author,
			State:           draftStatusLabel(draft.Status),
			Warning:         strings.Join(draft.Warnings, " "),
			WarningLevel:    warningLevel(strings.Join(draft.Warnings, " ")),
		})
	}

	filtered := base[:0]
	for _, row := range base {
		if canAccessRoomMoveSite(config, row.CurrentSiteID) && !s.canceled[row.DraftID] {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func seedRoomMoveReviewRow(id string, draftID string, moveType string, person string, email string, employeeID string, currentSiteID string, currentRoom string, destinationSiteID string, destinationRoom string, phone string, author string, state string, warning string) roomMoveReviewRow {
	currentSite := siteByID(currentSiteID)
	destinationSite := siteByID(destinationSiteID)
	return roomMoveReviewRow{
		ID:              id,
		DraftID:         draftID,
		MoveType:        moveType,
		Person:          person,
		Email:           email,
		EmployeeID:      employeeID,
		CurrentSiteID:   currentSiteID,
		CurrentSite:     currentSite.Name,
		CurrentRoom:     currentRoom,
		DestinationSite: destinationSite.Name,
		DestinationRoom: destinationRoom,
		Phone:           phone,
		Author:          author,
		State:           state,
		Warning:         warning,
		WarningLevel:    warningLevel(warning),
	}
}

func seedRoomMoveReviewRowByDraftID(draftID string) (roomMoveReviewRow, bool) {
	for _, row := range []roomMoveReviewRow{
		seedRoomMoveReviewRow("single-alex-ramirez", "single-alex-ramirez", roomMoveTypeSingle, "Alex Ramirez", "alex.ramirez@wusd.org", "103118", "clover-hs", "A-104", "clover-hs", "A-108", "Move ext 51042", "Alex Ramirez", "Ready", ""),
		seedRoomMoveReviewRow("single-morgan-lee", "single-morgan-lee", roomMoveTypeSingle, "Morgan Lee", "morgan.lee@wusd.org", "103442", "clover-hs", "B-210", "clover-hs", "B-204", "Manual ticket", "Avery Shah", "Review", "Primary conflict"),
		seedRoomMoveReviewRow("bulk-clover-summer", "rm-draft-103", roomMoveTypeBulkRoster, "Bulk Move", "", "", "clover-hs", "Multiple", "clover-hs", "Multiple", "Batch cutover", "Alex Ramirez", "Scheduled", "Two rows need review before scheduling"),
		seedRoomMoveReviewRow("single-jamie-reed", "single-jamie-reed", roomMoveTypeSingle, "Jamie Reed", "jamie.reed@wusd.org", "103772", "desert-view", "C-118", "desert-view", "None", "Remove phone", "Alex Ramirez", "Review", "Null-room outcome"),
		seedRoomMoveReviewRow("single-nia-brooks", "single-nia-brooks", roomMoveTypeSingle, "Nia Brooks", "nia.brooks@wusd.org", "104012", "franklin-ms", "D-102", "franklin-ms", "D-112", "Assign line", "Avery Shah", "Ready", ""),
	} {
		if row.DraftID == draftID {
			return row, true
		}
	}
	return roomMoveReviewRow{}, false
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

func (s *devRoomMoveStoreState) updateDraft(config devPersonaConfig, draftID string, request roomMoveDraftRequest) (roomMoveDraftPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.drafts[draftID]
	if !ok {
		return roomMoveDraftPayload{}, http.StatusNotFound, map[string]string{"draft": "Draft not found."}
	}
	if !canAccessRoomMoveSite(config, existing.ScopeSiteID) {
		return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot update another site's room move draft."}
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
	draft.Status = existing.Status
	s.drafts[draft.ID] = draft
	return draft, http.StatusOK, nil
}

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
	if s.canceled[draft.ID] {
		return roomMoveDraftPayload{}, http.StatusConflict, map[string]string{"draft": "Canceled drafts cannot be scheduled or applied."}
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
	return draft, http.StatusOK, nil
}

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
		draft.Status = "canceled"
		s.drafts[draft.ID] = draft
		s.canceled[draft.ID] = true
		return draft, http.StatusOK, nil
	}
	if seed, ok := seedRoomMoveReviewRowByDraftID(draftID); ok {
		if !canAccessRoomMoveSite(config, seed.CurrentSiteID) {
			return roomMoveDraftPayload{}, http.StatusForbidden, map[string]string{"scope": "This persona cannot cancel another site's room move draft."}
		}
		s.canceled[draftID] = true
		return roomMoveDraftPayload{
			ID:          draftID,
			Mode:        seed.MoveType,
			Status:      "canceled",
			ScopeSiteID: seed.CurrentSiteID,
			ScopeSite:   seed.CurrentSite,
			Rows: []roomMoveDraftRow{{
				ID:              seed.ID,
				Person:          seed.Person,
				Email:           seed.Email,
				EmployeeID:      seed.EmployeeID,
				CurrentSiteID:   seed.CurrentSiteID,
				CurrentSite:     seed.CurrentSite,
				CurrentRoom:     seed.CurrentRoom,
				DestinationSite: seed.DestinationSite,
				DestinationRoom: seed.DestinationRoom,
				Phone:           seed.Phone,
			}},
			CanManageDistrict: canManageDistrictRoomMoves(config),
		}, http.StatusOK, nil
	}
	return roomMoveDraftPayload{}, http.StatusNotFound, map[string]string{"draft": "Draft not found."}
}

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
		return draft
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
	draft, _, _ := buildRoomMoveDraft(config, newID, request)
	s.drafts[draft.ID] = draft
	return draft
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
			rows = append(rows, draftRowFromPerson(person, person.SiteID, person.CurrentRoomID))
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
		Warnings:          warnings,
		Rows:              normalizedRows,
		CanEdit:           true,
		CanDelete:         true,
		CanManageDistrict: canManageDistrictRoomMoves(config),
	}, http.StatusOK, nil
}

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
		destinationRoomID := row.DestinationRoomID
		if destinationSiteID != person.SiteID {
			destinationRoomID = "none"
		} else if destinationRoomID == "" {
			destinationRoomID = person.CurrentRoomID
		}
		room := roomMoveRoomByID(destinationRoomID, destinationSiteID)
		warning := row.Warning
		if destinationRoomID == "none" && person.CurrentRoomID != "none" {
			warning = "Destination room is none; phone and room assignments will be removed."
			warnings = appendUniqueString(warnings, warning)
		}
		if destinationSiteID != person.SiteID {
			warning = "Inter-site move: destination room is set to none until the destination site confirms the room."
			warnings = appendUniqueString(warnings, warning)
		}
		action := row.Action
		if action == "" {
			action = "change"
		}
		normalized = append(normalized, roomMoveDraftRow{
			ID:                firstNonEmpty(row.ID, fmt.Sprintf("row-%02d-%s", index+1, person.ID)),
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
			Action:            action,
			Warning:           warning,
		})
	}
	return normalized, warnings, http.StatusOK, nil
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
		{ID: "alex-ramirez", Name: "Alex Ramirez", Email: "alex.ramirez@wusd.org", EmployeeID: "103118", Role: "IT Admin", SiteID: "clover-hs", Site: "Clover High School", CurrentRoomID: "cla-a104", CurrentRoom: "A-104", Phone: "51042"},
		{ID: "morgan-lee", Name: "Morgan Lee", Email: "morgan.lee@wusd.org", EmployeeID: "103442", Role: "Teacher", SiteID: "clover-hs", Site: "Clover High School", CurrentRoomID: "cla-b210", CurrentRoom: "B-210", Phone: "51017"},
		{ID: "taylor-quinn", Name: "Taylor Quinn", Email: "taylor.quinn@wusd.org", EmployeeID: "106103", Role: "Contractor", SiteID: "desert-view", Site: "Desert View Elementary", CurrentRoomID: "none", CurrentRoom: "None", Phone: ""},
		{ID: "jamie-reed", Name: "Jamie Reed", Email: "jamie.reed@wusd.org", EmployeeID: "103772", Role: "Teacher", SiteID: "desert-view", Site: "Desert View Elementary", CurrentRoomID: "dve-c118", CurrentRoom: "C-118", Phone: "52013"},
		{ID: "nia-brooks", Name: "Nia Brooks", Email: "nia.brooks@wusd.org", EmployeeID: "104012", Role: "Site Secretary", SiteID: "franklin-ms", Site: "Franklin Middle School", CurrentRoomID: "fms-d102", CurrentRoom: "D-102", Phone: "53022"},
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

func roomMoveRoomByID(roomID string, siteID string) roomMoveRoomOption {
	for _, room := range roomMoveRoomsForSite(siteID) {
		if room.ID == roomID {
			return room
		}
	}
	return roomMoveRoomsForSite(siteID)[0]
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
