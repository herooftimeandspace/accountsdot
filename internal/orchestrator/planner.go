package orchestrator

import (
	"fmt"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

type PlanInput struct {
	WorkflowType              core.WorkflowType
	ChangeReason              core.WorkflowChangeReason
	SubjectKind               core.SubjectKind
	SubjectID                 string
	RoomKnown                 bool
	OldRoomBecomesVacant      bool
	RoomCoverageRequired      bool
	PrimaryAssignmentRequired bool
	Now                       time.Time
}

type FollowUpWorkflow struct {
	WorkflowType core.WorkflowType
	SubjectKind  core.SubjectKind
	SubjectID    string
}

type PlanResult struct {
	WorkflowType core.WorkflowType
	ChangeReason core.WorkflowChangeReason
	SubjectKind  core.SubjectKind
	SubjectID    string
	Jobs         []core.WorkflowJob
	FollowUps    []FollowUpWorkflow
	RunAfter     time.Time
}

type LoopSpec struct {
	Name    string
	Cadence time.Duration
}

type plannedStep struct {
	provider         core.ProviderKind
	operation        string
	approvalRequired bool
}

func PlanWorkflow(input PlanInput) (PlanResult, error) {
	result := PlanResult{
		WorkflowType: input.WorkflowType,
		ChangeReason: input.ChangeReason,
		SubjectKind:  input.SubjectKind,
		SubjectID:    input.SubjectID,
	}

	switch input.WorkflowType {
	case core.WorkflowTypePersonOnboard:
		steps := []plannedStep{
			{provider: core.ProviderKindZoom, operation: "zoom.read_user"},
			{provider: core.ProviderKindZoom, operation: "zoom.create_or_link_user"},
			{provider: core.ProviderKindInternal, operation: "internal.reserve_extension"},
			{provider: core.ProviderKindZoom, operation: "zoom.assign_site_extension"},
			{provider: core.ProviderKindZoom, operation: "zoom.assign_calling_plan"},
		}
		if input.RoomKnown {
			steps = append(steps,
				plannedStep{provider: core.ProviderKindZoom, operation: "zoom.ensure_room_slg"},
				plannedStep{provider: core.ProviderKindZoom, operation: "zoom.add_room_membership"},
			)
		}
		result.Jobs = buildJobs(steps)
	case core.WorkflowTypePersonSameSiteTransfer:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindZoom, operation: "zoom.ensure_room_slg"},
			{provider: core.ProviderKindZoom, operation: "zoom.add_room_membership"},
			{provider: core.ProviderKindZoom, operation: "zoom.remove_room_membership"},
		})
		if input.OldRoomBecomesVacant {
			result.FollowUps = append(result.FollowUps, FollowUpWorkflow{
				WorkflowType: core.WorkflowTypeRoomCoverage,
				SubjectKind:  core.SubjectKindRoom,
				SubjectID:    input.SubjectID,
			})
		}
	case core.WorkflowTypePersonSiteTransfer:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindInternal, operation: "internal.reserve_extension"},
			{provider: core.ProviderKindZoom, operation: "zoom.ensure_room_slg"},
			{provider: core.ProviderKindZoom, operation: "zoom.apply_site_extension_cutover", approvalRequired: true},
			{provider: core.ProviderKindZoom, operation: "zoom.verify_room_membership"},
			{provider: core.ProviderKindInternal, operation: "internal.release_old_extension", approvalRequired: true},
		})
	case core.WorkflowTypePersonLeave, core.WorkflowTypePersonTerminate:
		var steps []plannedStep
		if input.RoomCoverageRequired {
			steps = append(steps,
				plannedStep{provider: core.ProviderKindZoom, operation: "zoom.ensure_room_cap"},
				plannedStep{provider: core.ProviderKindZoom, operation: "zoom.verify_room_cap"},
			)
		}
		steps = append(steps,
			plannedStep{provider: core.ProviderKindZoom, operation: "zoom.remove_room_membership", approvalRequired: true},
			plannedStep{provider: core.ProviderKindZoom, operation: "zoom.remove_phone_assignment", approvalRequired: true},
			plannedStep{provider: core.ProviderKindZoom, operation: "zoom.deprovision_user", approvalRequired: true},
			plannedStep{provider: core.ProviderKindInternal, operation: "internal.release_extension", approvalRequired: true},
		)
		result.Jobs = buildJobs(steps)
	case core.WorkflowTypeRoomCoverage:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindZoom, operation: "zoom.ensure_room_cap"},
			{provider: core.ProviderKindZoom, operation: "zoom.verify_room_cap"},
		})
	case core.WorkflowTypeDirectoryPublish:
		now := input.Now
		if now.IsZero() {
			now = time.Now().UTC()
		}
		result.RunAfter = now.Add(time.Minute)
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindGoogleSheets, operation: "google_sheets.stage_workbook"},
			{provider: core.ProviderKindGoogleSheets, operation: "google_sheets.validate_sentinel"},
			{provider: core.ProviderKindGoogleSheets, operation: "google_sheets.apply_pointers"},
		})
	case core.WorkflowTypePersonUpdate, core.WorkflowTypeContextRefresh:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindInternal, operation: "internal.reconcile_subject"},
		})
	case core.WorkflowTypeStaffSyncDryRun:
		steps := []plannedStep{
			{provider: core.ProviderKindInternal, operation: "internal.sync_ingest_subject"},
			{provider: core.ProviderKindPhoto, operation: "photo.check_delta"},
			{provider: core.ProviderKindIncidentIQ, operation: "incident_iq.resolve_room"},
			{provider: core.ProviderKindIncidentIQ, operation: "incident_iq.resolve_room_asset"},
			{provider: core.ProviderKindZoom, operation: "zoom.validate_room_membership"},
		}
		if input.PrimaryAssignmentRequired {
			steps = append(steps, plannedStep{provider: core.ProviderKindZoom, operation: "zoom.validate_primary_phone_assignment"})
		}
		steps = append(steps, plannedStep{provider: core.ProviderKindInternal, operation: "internal.sync_update_projection"})
		result.Jobs = buildJobs(steps)
	case core.WorkflowTypeStudentSyncDryRun:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindInternal, operation: "internal.sync_ingest_subject"},
			{provider: core.ProviderKindPhoto, operation: "photo.check_delta"},
			{provider: core.ProviderKindIncidentIQ, operation: "incident_iq.match_person"},
			{provider: core.ProviderKindInternal, operation: "internal.sync_update_projection"},
		})
	case core.WorkflowTypeSyncRecheck:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindInternal, operation: "internal.sync_recheck_subject"},
		})
	case core.WorkflowTypeAnnualResetArchive:
		result.Jobs = buildJobs([]plannedStep{
			{provider: core.ProviderKindInternal, operation: "internal.archive_completed_sync_rows"},
			{provider: core.ProviderKindInternal, operation: "internal.clear_sync_exception_overrides"},
		})
	default:
		return PlanResult{}, fmt.Errorf("unsupported workflow type %q", input.WorkflowType)
	}

	return result, nil
}

func DefaultLoopSpecs() []LoopSpec {
	return []LoopSpec{
		{Name: "hr_import_loop", Cadence: 5 * time.Minute},
		{Name: "aeries_sync_loop", Cadence: 5 * time.Minute},
		{Name: "context_watcher_loop", Cadence: 5 * time.Minute},
		{Name: "workflow_planner_loop", Cadence: 30 * time.Second},
		{Name: "recovery_loop", Cadence: 30 * time.Second},
		{Name: "approval_loop", Cadence: 30 * time.Second},
		{Name: "janitor_loop", Cadence: 2 * time.Minute},
	}
}

func buildJobs(steps []plannedStep) []core.WorkflowJob {
	jobs := make([]core.WorkflowJob, 0, len(steps))
	var previousStep string
	for i, step := range steps {
		stepKey := fmt.Sprintf("step_%02d", i+1)
		jobs = append(jobs, core.WorkflowJob{
			StepKey:          stepKey,
			Provider:         step.provider,
			Operation:        step.operation,
			DependsOnStepKey: previousStep,
			ApprovalRequired: step.approvalRequired,
		})
		previousStep = stepKey
	}
	return jobs
}
