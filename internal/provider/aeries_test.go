package provider_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

// TestBuildAeriesUploadPayloadIncludesManualNonEscapePersonalPhone verifies the sensitive phone-number boundary for planned Aeries upload work. Provider contract tests call this helper with manual Non-Escape and ESCAPE source inputs so future live workers preserve the documented rule: manual intake may supply the phone value, while ESCAPE-sourced records must not be overwritten by this path.
func TestBuildAeriesUploadPayloadIncludesManualNonEscapePersonalPhone(t *testing.T) {
	payload := provider.BuildAeriesUploadPayload(provider.AeriesUploadInput{
		SourceSystem:    provider.AeriesSourceManualNonEscape,
		EmployeeID:      "6600001",
		FirstName:       "Quincy",
		LastName:        "Zephyr",
		PersonalEmail:   "quincy.zephyr@example.com",
		PersonalPhone:   "7075550134",
		RequestedAccess: "Teacher",
		PreferredDevice: "Mac",
	})

	if got := payload["personal_phone"]; got != "7075550134" {
		t.Fatalf("manual Non-Escape personal_phone = %#v, want canonical 10-digit phone", got)
	}

	escapePayload := provider.BuildAeriesUploadPayload(provider.AeriesUploadInput{
		SourceSystem:    provider.AeriesSourceEscape,
		EmployeeID:      "100241",
		FirstName:       "Jordan",
		LastName:        "Miles",
		PersonalEmail:   "jordan.miles@example.com",
		PersonalPhone:   "7075550199",
		RequestedAccess: "Staff",
		PreferredDevice: "Windows",
	})
	if _, ok := escapePayload["personal_phone"]; ok {
		t.Fatalf("ESCAPE-sourced Aeries payload included personal_phone: %#v", escapePayload)
	}
}
