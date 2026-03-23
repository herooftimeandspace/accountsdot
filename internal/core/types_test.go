package core_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

func TestUUIDv7Generation(t *testing.T) {
	id, err := core.NewPersonUUID()
	if err != nil {
		t.Fatalf("NewPersonUUID returned error: %v", err)
	}
	if id.Version() != 7 {
		t.Fatalf("expected UUID version 7, got %d", id.Version())
	}
	text := id.String()
	if text != strings.ToLower(text) {
		t.Fatalf("expected lowercase UUID string, got %q", text)
	}
	if _, err := uuid.Parse(text); err != nil {
		t.Fatalf("expected parseable UUID, got %v", err)
	}
}

func TestPersonStatesValidate(t *testing.T) {
	valid := []core.PersonState{
		core.PersonStateIntakePending,
		core.PersonStateProvisionPendingContext,
		core.PersonStateAwaitingReview,
		core.PersonStateActive,
	}
	for _, state := range valid {
		if !state.Valid() {
			t.Fatalf("expected state %q to be valid", state)
		}
	}
	if core.PersonState("bogus").Valid() {
		t.Fatal("expected bogus person state to be invalid")
	}
}

func TestJobStatesValidate(t *testing.T) {
	valid := []core.JobState{
		core.JobStateQueued,
		core.JobStateRunning,
		core.JobStateRecovering,
		core.JobStateBlocked,
		core.JobStateSucceeded,
	}
	for _, state := range valid {
		if !state.Valid() {
			t.Fatalf("expected state %q to be valid", state)
		}
	}
	if core.JobState("bogus").Valid() {
		t.Fatal("expected bogus job state to be invalid")
	}
}

func TestDuplicateReasonCodesValidate(t *testing.T) {
	valid := []core.DuplicateReasonCode{
		core.DuplicateReasonMatchNameNoDOB,
		core.DuplicateReasonMatchNameEmailConflict,
		core.DuplicateReasonMatchSourceIDReuse,
		core.DuplicateReasonContextTimeout,
	}
	for _, code := range valid {
		if !code.Valid() {
			t.Fatalf("expected reason code %q to be valid", code)
		}
	}
	if core.DuplicateReasonCode("bogus").Valid() {
		t.Fatal("expected bogus duplicate reason code to be invalid")
	}
}
