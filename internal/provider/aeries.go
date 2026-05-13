package provider

const (
	AeriesSourceEscape          = "escape"
	AeriesSourceManualNonEscape = "manual_non_escape"
)

type AeriesUploadInput struct {
	SourceSystem    string
	EmployeeID      string
	FirstName       string
	LastName        string
	PersonalEmail   string
	PersonalPhone   string
	RequestedAccess string
	PreferredDevice string
	ChangeReason    string
}

// BuildAeriesUploadPayload prepares the planned Aeries upload data that future provider-write workers will send after manual Non-Escape onboarding completes. DEV onboarding workflow tests and provider contract tests reach this helper; callers pass already-normalized intake fields and receive a serializable map for planning only, with no live Aeries mutation. Personal phone is intentionally copied only for manual Non-Escape source rows so ESCAPE-sourced phone data remains authoritative and cannot be overwritten through this path.
func BuildAeriesUploadPayload(input AeriesUploadInput) map[string]any {
	payload := map[string]any{
		"source_system":    input.SourceSystem,
		"employee_id":      input.EmployeeID,
		"first_name":       input.FirstName,
		"last_name":        input.LastName,
		"personal_email":   input.PersonalEmail,
		"requested_access": input.RequestedAccess,
		"preferred_device": input.PreferredDevice,
		"change_reason":    input.ChangeReason,
	}
	if input.SourceSystem == AeriesSourceManualNonEscape && input.PersonalPhone != "" {
		payload["personal_phone"] = input.PersonalPhone
	}
	return payload
}
