package provider_test

import (
	"testing"
	"time"

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

// TestP000D003AeriesPreviousYearStagingConfig documents the Phase 0 staging
// safety contract for Aeries before a live client is wired in. Provider
// readiness code will feed sanitized School Info metadata into this resolver,
// and the returned query parameters prove that staging uses read-only masked
// previous-year reads with `DatabaseYear=current school year - 1`.
func TestP000D003AeriesPreviousYearStagingConfig(t *testing.T) {
	config, err := provider.ResolveAeriesPreviousYearStagingConfig(provider.AeriesPreviousYearStagingInput{
		AppEnv:                 "staging",
		EnvironmentDataMode:    provider.AeriesEnvironmentDataModeMaskedReadOnly,
		EnvironmentDataSource:  provider.AeriesEnvironmentSourceMaskedProduction,
		UseMockAeries:          false,
		ReadOnly:               true,
		DatabaseYearMode:       provider.AeriesDatabaseYearModePreviousSchoolYear,
		MaskedPreviousYearOnly: true,
		SchoolInfo: []provider.AeriesSchoolInfo{
			{
				SchoolCode: "101",
				SchoolYear: "2025-2026",
				StartDate:  time.Date(2025, time.August, 12, 0, 0, 0, 0, time.UTC),
				EndDate:    time.Date(2026, time.June, 4, 0, 0, 0, 0, time.UTC),
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAeriesPreviousYearStagingConfig returned error: %v", err)
	}
	if config.DatabaseYear != 2024 {
		t.Fatalf("DatabaseYear = %d, want 2024", config.DatabaseYear)
	}
	if got := config.QueryParameters["DatabaseYear"]; got != "2024" {
		t.Fatalf("DatabaseYear query parameter = %q, want 2024", got)
	}
	if !config.ReadOnly || config.AccessMode != "masked_previous_year_read_only" {
		t.Fatalf("unexpected access flags: %#v", config)
	}
}

// TestResolveAeriesPreviousDatabaseYearUsesEarliestSchoolStart verifies the
// disagreement rule from the environment playbook. Sanitized School Info can
// contain different school calendars; the resolver uses the earliest start
// date as the district current-year boundary before subtracting one year.
func TestResolveAeriesPreviousDatabaseYearUsesEarliestSchoolStart(t *testing.T) {
	databaseYear, err := provider.ResolveAeriesPreviousDatabaseYear([]provider.AeriesSchoolInfo{
		{
			SchoolCode: "201",
			StartDate:  time.Date(2025, time.August, 12, 0, 0, 0, 0, time.UTC),
			EndDate:    time.Date(2026, time.June, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			SchoolCode: "301",
			StartDate:  time.Date(2025, time.July, 29, 0, 0, 0, 0, time.UTC),
			EndDate:    time.Date(2026, time.June, 12, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("ResolveAeriesPreviousDatabaseYear returned error: %v", err)
	}
	if databaseYear != 2024 {
		t.Fatalf("DatabaseYear = %d, want 2024", databaseYear)
	}
}

// TestResolveAeriesPreviousDatabaseYearFallsBackToSchoolYearString keeps DEV
// mock evidence usable when sanitized School Info records omit date fields but
// still expose a `YYYY-YYYY` school-year label from Aeries.
func TestResolveAeriesPreviousDatabaseYearFallsBackToSchoolYearString(t *testing.T) {
	databaseYear, err := provider.ResolveAeriesPreviousDatabaseYear([]provider.AeriesSchoolInfo{
		{SchoolCode: "101", SchoolYear: "2025-2026"},
	})
	if err != nil {
		t.Fatalf("ResolveAeriesPreviousDatabaseYear returned error: %v", err)
	}
	if databaseYear != 2024 {
		t.Fatalf("DatabaseYear = %d, want 2024", databaseYear)
	}
}

// TestResolveAeriesPreviousYearStagingConfigFailsClosed verifies that the
// resolver refuses staging evidence when any safety marker would allow live
// current-year or writable Aeries access.
func TestResolveAeriesPreviousYearStagingConfigFailsClosed(t *testing.T) {
	input := provider.AeriesPreviousYearStagingInput{
		AppEnv:                 "staging",
		EnvironmentDataMode:    provider.AeriesEnvironmentDataModeMaskedReadOnly,
		EnvironmentDataSource:  provider.AeriesEnvironmentSourceMaskedProduction,
		UseMockAeries:          false,
		ReadOnly:               true,
		DatabaseYearMode:       provider.AeriesDatabaseYearModePreviousSchoolYear,
		MaskedPreviousYearOnly: true,
		SchoolInfo: []provider.AeriesSchoolInfo{
			{SchoolCode: "101", SchoolYear: "2025-2026"},
		},
	}

	tests := []struct {
		name   string
		mutate func(*provider.AeriesPreviousYearStagingInput)
	}{
		{name: "not staging", mutate: func(in *provider.AeriesPreviousYearStagingInput) { in.AppEnv = "production" }},
		{name: "unmasked data", mutate: func(in *provider.AeriesPreviousYearStagingInput) { in.EnvironmentDataMode = "production" }},
		{name: "mock still enabled", mutate: func(in *provider.AeriesPreviousYearStagingInput) { in.UseMockAeries = true }},
		{name: "write capable", mutate: func(in *provider.AeriesPreviousYearStagingInput) { in.ReadOnly = false }},
		{name: "current year mode", mutate: func(in *provider.AeriesPreviousYearStagingInput) { in.DatabaseYearMode = "current_school_year" }},
		{name: "not previous-year-only", mutate: func(in *provider.AeriesPreviousYearStagingInput) { in.MaskedPreviousYearOnly = false }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			candidate := input
			tc.mutate(&candidate)
			if _, err := provider.ResolveAeriesPreviousYearStagingConfig(candidate); err == nil {
				t.Fatal("expected safety validation error")
			}
		})
	}
}
