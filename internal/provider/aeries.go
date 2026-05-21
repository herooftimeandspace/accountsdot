package provider

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	AeriesSourceEscape          = "escape"
	AeriesSourceManualNonEscape = "manual_non_escape"

	AeriesDatabaseYearModePreviousSchoolYear = "previous_school_year"
	AeriesEnvironmentDataModeMaskedReadOnly  = "masked-read-only"
	AeriesEnvironmentSourceMaskedProduction  = "masked-production-derived"
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

type AeriesSchoolInfo struct {
	SchoolCode string
	SchoolYear string
	StartDate  time.Time
	EndDate    time.Time
}

type AeriesPreviousYearStagingInput struct {
	AppEnv                 string
	EnvironmentDataMode    string
	EnvironmentDataSource  string
	UseMockAeries          bool
	ReadOnly               bool
	DatabaseYearMode       string
	MaskedPreviousYearOnly bool
	SchoolInfo             []AeriesSchoolInfo
}

type AeriesPreviousYearStagingConfig struct {
	DatabaseYear    int
	QueryParameters map[string]string
	ReadOnly        bool
	AccessMode      string
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

// ResolveAeriesPreviousYearStagingConfig turns staging environment markers plus
// Aeries School Info into the read-only query shape used for Phase 0 evidence.
// Config loading and provider contract tests can call this before any Aeries
// client exists; it accepts only non-secret mode flags and School Info metadata,
// returns the resolved `DatabaseYear=YYYY` parameter, and fails closed when the
// environment is not masked, read-only, previous-year-only staging.
func ResolveAeriesPreviousYearStagingConfig(input AeriesPreviousYearStagingInput) (AeriesPreviousYearStagingConfig, error) {
	if input.AppEnv != "staging" {
		return AeriesPreviousYearStagingConfig{}, fmt.Errorf("aeries previous-year staging config requires APP_ENV=staging, got %q", input.AppEnv)
	}
	if input.EnvironmentDataMode != AeriesEnvironmentDataModeMaskedReadOnly {
		return AeriesPreviousYearStagingConfig{}, fmt.Errorf("aeries staging config requires ENVIRONMENT_DATA_MODE=%s, got %q", AeriesEnvironmentDataModeMaskedReadOnly, input.EnvironmentDataMode)
	}
	if input.EnvironmentDataSource != AeriesEnvironmentSourceMaskedProduction {
		return AeriesPreviousYearStagingConfig{}, fmt.Errorf("aeries staging config requires ENVIRONMENT_DATA_SOURCE=%s, got %q", AeriesEnvironmentSourceMaskedProduction, input.EnvironmentDataSource)
	}
	if input.UseMockAeries {
		return AeriesPreviousYearStagingConfig{}, errors.New("aeries staging config cannot query masked previous-year data while USE_MOCK_AERIES=true")
	}
	if !input.ReadOnly {
		return AeriesPreviousYearStagingConfig{}, errors.New("aeries staging config requires read-only provider access")
	}
	if input.DatabaseYearMode != AeriesDatabaseYearModePreviousSchoolYear {
		return AeriesPreviousYearStagingConfig{}, fmt.Errorf("aeries staging config requires AERIES_DATABASE_YEAR_MODE=%s, got %q", AeriesDatabaseYearModePreviousSchoolYear, input.DatabaseYearMode)
	}
	if !input.MaskedPreviousYearOnly {
		return AeriesPreviousYearStagingConfig{}, errors.New("aeries staging config requires masked previous-year-only reads")
	}

	databaseYear, err := ResolveAeriesPreviousDatabaseYear(input.SchoolInfo)
	if err != nil {
		return AeriesPreviousYearStagingConfig{}, err
	}
	return AeriesPreviousYearStagingConfig{
		DatabaseYear: databaseYear,
		QueryParameters: map[string]string{
			"DatabaseYear": strconv.Itoa(databaseYear),
		},
		ReadOnly:   true,
		AccessMode: "masked_previous_year_read_only",
	}, nil
}

// ResolveAeriesPreviousDatabaseYear derives the Aeries `DatabaseYear` for
// staging from School Info records. Provider readiness code will pass the
// read-only School Info response here; when schools disagree on dates, the
// earliest start date defines the district current year before subtracting one,
// matching the Phase 0 environment playbook without relying on a local calendar.
func ResolveAeriesPreviousDatabaseYear(schools []AeriesSchoolInfo) (int, error) {
	if len(schools) == 0 {
		return 0, errors.New("aeries school info is required to resolve the previous DatabaseYear")
	}

	var earliestStart time.Time
	for _, school := range schools {
		if school.StartDate.IsZero() {
			continue
		}
		if earliestStart.IsZero() || school.StartDate.Before(earliestStart) {
			earliestStart = school.StartDate
		}
	}
	if !earliestStart.IsZero() {
		return earliestStart.Year() - 1, nil
	}

	currentStartYear := 0
	for _, school := range schools {
		startYear, err := parseAeriesSchoolYearStart(school.SchoolYear)
		if err != nil {
			return 0, fmt.Errorf("school %q: %w", school.SchoolCode, err)
		}
		if currentStartYear == 0 || startYear < currentStartYear {
			currentStartYear = startYear
		}
	}
	return currentStartYear - 1, nil
}

// parseAeriesSchoolYearStart accepts the human-readable School Info year string
// used in Aeries evidence, such as `2025-2026`, and returns the first calendar
// year so the staging resolver can calculate the previous `DatabaseYear` when
// date fields are unavailable in a mock or sanitized response.
func parseAeriesSchoolYearStart(schoolYear string) (int, error) {
	parts := strings.Split(strings.TrimSpace(schoolYear), "-")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid Aeries school year %q", schoolYear)
	}
	startYear, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, fmt.Errorf("invalid Aeries school year start %q", parts[0])
	}
	endYear, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, fmt.Errorf("invalid Aeries school year end %q", parts[1])
	}
	if endYear != startYear+1 {
		return 0, fmt.Errorf("Aeries school year %q must span consecutive years", schoolYear)
	}
	return startYear, nil
}
