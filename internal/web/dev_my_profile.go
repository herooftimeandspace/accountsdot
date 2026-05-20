package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

const devMyProfileUpdateMaxBodyBytes int64 = 8 * 1024

type devMyProfilePayload struct {
	PageID  string              `json:"page_id"`
	Persona devPersona          `json:"persona"`
	Profile devMyProfileSummary `json:"profile"`
}

type devMyProfileSummary struct {
	LegalFirstName     string `json:"legal_first_name"`
	LegalLastName      string `json:"legal_last_name"`
	LegalName          string `json:"legal_name"`
	PreferredFirstName string `json:"preferred_first_name"`
	PreferredLastName  string `json:"preferred_last_name"`
	DisplayName        string `json:"display_name"`
	Pronouns           string `json:"pronouns"`
	Email              string `json:"email"`
	Site               string `json:"site"`
	Department         string `json:"department"`
	Manager            string `json:"manager"`
	Room               string `json:"room"`
	PhoneExtension     string `json:"phone_extension"`
	Editable           bool   `json:"editable"`
}

type devMyProfileUpdateRequest struct {
	PreferredFirstName string `json:"preferred_first_name"`
	PreferredLastName  string `json:"preferred_last_name"`
	Pronouns           string `json:"pronouns"`
}

var devMyProfileStore = struct {
	sync.Mutex
	updates map[string]devMyProfileUpdateRequest
}{updates: map[string]devMyProfileUpdateRequest{}}

// handleDevMyProfile serves and mutates the DEV-local My Profile mock store for the React drawer.
// The route is available only in development, requires the normal DEV session cookie, and rejects
// student-like personas before any mock state is changed. PUT updates are idempotent overwrites in
// memory only; failures return JSON validation errors and never touch live HR, Google, Zoom, or Aeries
// systems.
func handleDevMyProfile(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return
	}

	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can manage your profile.",
		})
		return
	}
	if !canEditDevMyProfile(config.Persona) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Students cannot update preferred display names through this dashboard.",
			"persona": config.Persona,
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, buildDevMyProfilePayload(config))
	case http.MethodPut:
		updateDevMyProfile(w, r, config)
	default:
		http.NotFound(w, r)
	}
}

// updateDevMyProfile validates drawer-submitted preferred name fields and stores the DEV mock update.
// Repeating the same PUT for a persona leaves the same in-memory state, which mirrors the eventual
// direct self-service write contract without creating audit records or provider requests in this slice.
func updateDevMyProfile(w http.ResponseWriter, r *http.Request, config devPersonaConfig) {
	var request devMyProfileUpdateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, devMyProfileUpdateMaxBodyBytes)).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_request",
			"message": "Request body must include preferred_first_name, preferred_last_name, and pronouns fields.",
		})
		return
	}

	cleaned := devMyProfileUpdateRequest{
		PreferredFirstName: normalizeDevProfileText(request.PreferredFirstName),
		PreferredLastName:  normalizeDevProfileText(request.PreferredLastName),
		Pronouns:           normalizeDevProfileText(request.Pronouns),
	}
	errors := map[string]string{}
	if cleaned.PreferredFirstName == "" {
		errors["preferred_first_name"] = "Preferred first name is required."
	}
	if cleaned.PreferredLastName == "" {
		errors["preferred_last_name"] = "Preferred last name is required."
	}
	if len(cleaned.PreferredFirstName) > 50 {
		errors["preferred_first_name"] = "Preferred first name must be 50 characters or fewer."
	}
	if len(cleaned.PreferredLastName) > 50 {
		errors["preferred_last_name"] = "Preferred last name must be 50 characters or fewer."
	}
	if len(cleaned.Pronouns) > 40 {
		errors["pronouns"] = "Pronouns must be 40 characters or fewer."
	}
	if len(errors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "validation_failed",
			"message": "Profile update could not be saved.",
			"errors":  errors,
		})
		return
	}

	devMyProfileStore.Lock()
	devMyProfileStore.updates[config.Persona.ID] = cleaned
	devMyProfileStore.Unlock()

	writeJSON(w, http.StatusOK, buildDevMyProfilePayload(config))
}

// buildDevMyProfilePayload combines immutable mock source fields with any saved DEV-local display-name
// override for the current persona. Frontend callers use this response to render the read-only profile
// summary and to seed the shared drawer form after a save.
func buildDevMyProfilePayload(config devPersonaConfig) devMyProfilePayload {
	profile := defaultDevMyProfile(config)

	devMyProfileStore.Lock()
	update, ok := devMyProfileStore.updates[config.Persona.ID]
	devMyProfileStore.Unlock()
	if ok {
		profile.PreferredFirstName = update.PreferredFirstName
		profile.PreferredLastName = update.PreferredLastName
		profile.Pronouns = update.Pronouns
	}
	profile.DisplayName = strings.TrimSpace(profile.PreferredFirstName + " " + profile.PreferredLastName)
	profile.Editable = canEditDevMyProfile(config.Persona)

	return devMyProfilePayload{
		PageID:  "my-profile",
		Persona: config.Persona,
		Profile: profile,
	}
}

// defaultDevMyProfile returns non-secret DEV profile fixture data for the logged-in persona. It keeps
// legal-name, preferred/display-name, and directory fields separate so the mock page demonstrates the
// data model without normalizing or overwriting source-system truth.
func defaultDevMyProfile(config devPersonaConfig) devMyProfileSummary {
	firstName, lastName := splitDisplayName(config.Persona.DisplayName)
	emailLocal := strings.ToLower(strings.ReplaceAll(config.Persona.DisplayName, " ", "."))
	departmentByPersona := map[string]string{
		"it_admin":        "Information Technology",
		"human_resources": "Human Resources",
		"site_admin":      "School Administration",
		"site_secretary":  "Site Office",
		"device_wrangler": "Student Devices",
		"faculty_staff":   "Mathematics",
	}

	return devMyProfileSummary{
		LegalFirstName:     firstName,
		LegalLastName:      lastName,
		LegalName:          config.Persona.DisplayName,
		PreferredFirstName: firstName,
		PreferredLastName:  lastName,
		DisplayName:        config.Persona.DisplayName,
		Pronouns:           "She / Her",
		Email:              emailLocal + "@wusd.org",
		Site:               config.CurrentSite.Name,
		Department:         departmentByPersona[config.Persona.ID],
		Manager:            "James Nguyen",
		Room:               "208B",
		PhoneExtension:     "(702) 555-1000 x1200",
		Editable:           canEditDevMyProfile(config.Persona),
	}
}

// canEditDevMyProfile is the server-side guard for preferred/display-name writes. Current DEV personas
// are employee or contractor stand-ins, but the explicit student denial keeps future mock personas from
// inheriting this self-service write route accidentally.
func canEditDevMyProfile(persona devPersona) bool {
	identity := strings.ToLower(persona.ID + " " + persona.Label)
	return !strings.Contains(identity, "student")
}

// splitDisplayName derives editable first and last display-name fields from the DEV persona label without
// changing legal-name source values. A single-token fallback remains editable as first name with blank last
// name until the drawer validates a save.
func splitDisplayName(displayName string) (string, string) {
	parts := strings.Fields(displayName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// normalizeDevProfileText trims and collapses whitespace for drawer-submitted mock profile fields. The
// DEV route stores only display-safe values and returns field-level validation errors instead of logging
// raw submitted names or pronouns.
func normalizeDevProfileText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
