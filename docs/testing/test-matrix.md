# Test Matrix

## Purpose
This document tracks the named mock scenarios and verification coverage required to promote each phase safely through `dev → staging → main`.

## Rules
- The scenario inventory here must stay synchronized with `docs/planning/implementation-plan.md`.
- Every workflow introduced in a phase must have named mock scenarios in `dev` before any real third-party integration or downstream write is attempted.
- Scenario grouping in this document should follow the per-phase delivery buckets defined in the implementation plan.
- Features within a phase should be validated individually rather than waiting for one large end-of-phase test pass.
- Scenario names should be stable and human-readable so both implementers and operators can refer to the same test case during review.
- A phase is not ready for promotion if its required named mock scenarios are missing, stale, or not passing.
- Staging verification expectations must be tracked alongside dev/mock scenarios for every workflow bucket.
- Promotion evidence is passing scenario evidence plus implementation signoff for each workflow bucket in the phase.
- Implementation signoff means the responsible implementer confirms code, configuration, docs, and scenario definitions remain aligned with the implementation plan and that required evidence has been reviewed. That signoff is captured in the external promotion runbook/process, not in this document.
- Review-only queue scenarios should name the expected operational owner when ownership is part of safe rollout.
- For now, one passing named scenario per workflow is sufficient minimum scenario evidence unless a later phase/bucket section raises the bar.
- Every named scenario in a phase must be clean/passing before promotion; written acceptance is not a substitute for an unresolved scenario.
- Any rollback trigger that fires for a workflow bucket blocks promotion for that bucket until the trigger condition is resolved and the bucket is clean again.
- If a rollback trigger fires for a workflow bucket in `dev`, `staging` verification for that same bucket is prohibited until the `dev` trigger condition is resolved and `dev` is clean again.
- Required proof artifacts should be defined at the workflow-bucket level rather than repeated on every scenario row.
- Runtime evidence is sufficient unless a later phase/bucket section explicitly requires a retained repository artifact.
- Runtime evidence means real `dev` or `staging` execution evidence such as logs, API traces, database state checks, UI captures, ticket output, or provider-side reads that can be tied back to the workflow bucket and scenario under review.
- UI-heavy workflow buckets must include at least one UI artifact in their runtime evidence set.
- A screenshot is sufficient by default for required UI artifacts unless a later bucket-specific rule raises the bar.
- When a UI artifact is required, a qualifying screenshot from either `dev` or `staging` is sufficient by default as long as both environments still have passing runtime evidence.
- When UI artifacts are filed under environment-specific evidence sections, the screenshot itself does not need a separate explicit environment label.
- A workflow bucket is UI-heavy when acceptance depends on rendered or interactive operator-facing behavior that cannot be proven from backend state alone.
- This document is a static definition artifact, not a live execution tracker. Do not add rolling status, evidence-link, or last-verified fields here.
- Live execution tracking, evidence collection, and signoff capture live outside the repo in an external IncidentIQ testing ticket.
- That external IncidentIQ testing ticket should use one parent ticket per release and organize evidence inside it in `phase → bucket → dev/staging → scenario` order, with `dev` listed before `staging` in every bucket.
- The CI/CD promotion-pipeline branch gates are defined in `docs/operations/promotion-pipeline.md`. Those gates enforce repository checks and promotion PR shape; they do not replace named scenario evidence, external IncidentIQ evidence, or the external promotion runbook.
- Promotion evidence should be retained for 90 days after the relevant phase promotion.
- The external promotion runbook/process must capture one implementation-signoff entry per workflow bucket and reference the corresponding IncidentIQ testing-ticket evidence; it does not need to name a rollback owner.
- Each write-capable workflow bucket must have its concrete rollback path documented in a dedicated per-phase rollback subsection of the implementation plan and referenced during external promotion review.
- Each write-capable workflow bucket must also have rollback-trigger conditions documented in that dedicated per-phase rollback subsection of the implementation plan.
- Each external runbook bucket entry must include exact metadata fields for release id, parent IncidentIQ ticket link, phase, bucket, environment, application/config revision ids, timestamps, signoff actor/time, environment evidence links, and rollback references when applicable.
- Each external runbook bucket entry must include a final disposition (`ready`, `blocked`, `rolled back`, or `superseded`) plus explicit yes/no attestations for scenario cleanliness, evidence review, and required write-safety checks.
- A bucket entry that was previously `rolled back` may later be updated to `ready` after a new clean verification pass for that same bucket.
- `Superseded` means the recorded attempt is no longer the active promotion candidate because a newer verification attempt or revision replaced it under the same release.
- When a bucket entry is superseded, the replacement current entry must explicitly link back to the superseded entry.
- A `no` attestation does not require a separate explanation field beyond the bucket disposition and any required rollback closure note.
- If a rollback trigger blocks a bucket in `dev`, the external runbook/process must carry an explicit closure note, including links to the replacement evidence, before `staging` can begin for that bucket.

## Phase 0
- Bucket-level evidence requirements for this phase:
  - `0A` repo-local safety artifacts and environment playbooks
    - evidence that required reference inputs and linked documents resolve from repo-local paths
    - evidence that missing required snapshots fail clearly
    - environment-role evidence for `dev`, `staging`, and `main` separation
  - `0B` core database schema, workflow engine, and recovery primitives
    - worker-crash recovery trace
    - ordering evidence showing `global_tick` drives sequencing
    - overlap-prevention evidence for duplicate job-family suppression
  - `0C` auth gate, breakglass, global pause, and emergency controls
    - allow/deny auth-gate evidence for staff and student domains
    - breakglass access evidence
    - global-pause runtime evidence showing new claims stop without bringing down diagnostics
  - `0D` provider configuration, read-only connectivity, and mock scaffolding
    - provider readiness success evidence against mocks
    - provider readiness failure evidence for missing or bad credentials/config
    - safe Aeries previous-year staging configuration evidence
    - provider access-mode evidence showing batch, projection-backed, live-detail, and live-write-verification boundaries are documented before implementation
  - `0E` health checks, metrics, and promotion plumbing
    - `/health/live` and `/health/ready` evidence under healthy and degraded conditions
    - observability evidence for pause/dependency state
    - promotion-gate evidence showing required scenario checks are enforced
- `0A` repo-local safety artifacts and environment playbooks
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P0-0A-001` | Reference Input Snapshot Integrity | Assert required vendored reference inputs are present, linked docs resolve, and startup fails clearly if required snapshots are missing. | Confirm staging deploy uses repo-local references only and does not depend on workstation paths or missing artifacts. |
  | `P0-0A-002` | Environment Role Separation | Verify mock configuration distinguishes `dev`, `staging`, and `main` roles and blocks production-only assumptions in `dev`. | Confirm staging config is distinct from dev/main and uses the documented masked-data path. |
- `0B` core database schema, workflow engine, and recovery primitives
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P0-0B-001` | Worker Crash Lease Recovery | Simulate a worker crash after job claim and verify recovery loop reclaims or reconciles the job without duplicate execution. | Verify the same recovery behavior works against staging infrastructure without manual DB repair. |
  | `P0-0B-002` | Global Tick Ordering Integrity | Create ordered events/jobs in mocks and verify ordering logic always uses `global_tick`, never UUID ordering. | Confirm staging job/event ordering matches dev expectations under concurrent inserts. |
  | `P0-0B-003` | Overlap Protection Prevents Duplicate Job Family Runs | Simulate a second scheduled run starting while the first is still active and verify the second run is deferred without clobbering work. | Verify staging prevents duplicate scheduled family execution and records overlap state. |
- `0C` auth gate, breakglass, global pause, and emergency controls
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P0-0C-001` | Staff Domain Allowlist Gate | Verify allowed staff-domain identities can reach the app and unauthorized domains are blocked before role logic. | Confirm staging auth gate behavior matches dev for staff domains. |
  | `P0-0C-002` | Student Domain Deny Gate | Verify `@stu.wusd.org` identities are denied access even when otherwise authenticated. | Confirm explicit student denial in staging. |
  | `P0-0C-003` | Google Group And Attribute Role Mapping | Verify a staff-domain identity receives only roles derived from current Google group and SAML attribute inputs, and that a known staff identity with no role mapping receives access denied instead of partial content. | Confirm staging uses the approved Google Workspace groups/attributes and denies same-URL access for users without matching role assignments. |
  | `P0-0C-004` | Site Scope Recalculation | Verify site scope is recalculated from current group/attribute mapping input when a user's group or site assignment changes. | Confirm staging reflects changed Google group/site assignments without stale cross-site access. |
  | `P0-0C-005` | Breakglass Access Bypass | Verify a configured named local breakglass account can create an emergency IT Admin session through `/api/v1/breakglass/login`, denied source addresses and unknown accounts do not receive a session, sanitized audit events are recorded, untrusted `X-Forwarded-For` values cannot bypass CIDR checks, and the DEV persona switcher remains a separate development-only flow. | Confirm staging has named account configuration with no sanitized env-key collisions, per-account token hashes supplied outside the repo, default or explicitly approved CIDR restrictions, explicit `BREAKGLASS_TRUSTED_PROXY_CIDRS` when forwarded client IPs are required, Secure breakglass cookies, `audit_log` records for allowed and denied attempts, explicit sign-out audit evidence, and a documented decision for whether cookie expiration requires a durable session table before promotion. |
  | `P0-0C-006` | Global Pause Stops New Claims | Trigger global pause and verify workers stop claiming new jobs while UI and diagnostics remain online. | Confirm staging pause can be used as an emergency cutoff without requiring deployment rollback. |
  | `P0-0C-007` | DEV Persona Tooling Switch | Verify `APP_ENV=development` terminal tooling can set every DEV mock persona through `/api/v1/dev/login` with `activate_mock_session=true`, including No Access and site-scoped personas, that `/api/v1/dev/session` reads back the selected persona for the frontend after refresh/navigation, invalid persona ids force anonymous readback, and non-development environments return `404`. | Not a staging promotion path; staging must continue to deny normal DEV persona switching and use only the separate breakglass flow where documented. |
- `0D` provider configuration, read-only connectivity, and mock scaffolding
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P0-0D-001` | Provider Readiness Mock Success Path | Verify provider clients initialize and pass readiness against mocks without real outbound writes. | Confirm staging read-only/provider connectivity checks succeed with configured non-prod credentials. |
  | `P0-0D-002` | Provider Readiness Failure Surfacing | Simulate missing credential, bad URL, or bad certificate and verify readiness fails closed with actionable diagnostics. | Confirm the same failures surface clearly in staging without partial startup ambiguity. |
  | `P0-0D-003` | Aeries Previous-Year Staging Configuration | Verify Aeries staging config resolves `current school year - 1` and uses masked previous-year reads only. | Confirm staging can successfully query masked previous-year Aeries data with `DatabaseYear=YYYY`. |
  | `P0-0D-004` | Provider Access Modes Classified Before Implementation | Verify the implementation plan documents each provider as batch-only, projection-backed, live-detail, or live-write-verified where appropriate before code locks in a full-mirror design. | Confirm staging readiness review uses the same provider capability classification and freshness expectations. |
- `0E` health checks, metrics, and promotion plumbing
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P0-0E-001` | Readiness Fails Closed on Missing Dependency | Simulate DB/service-account/storage dependency loss and verify `/health/ready` fails while `/health/live` remains meaningful. | Confirm staging health endpoints behave consistently under dependency failure drills. |
  | `P0-0E-002` | Health Endpoints Reflect Pause and Dependency State | Verify health endpoints and metrics expose paused/degraded states without implying false readiness. | Confirm staging observability reflects the same readiness/pause semantics. |
  | `P0-0E-003` | Promotion Gate Requires Named Scenario Passes | Verify promotion tooling/checklist blocks advancement when required scenario evidence is missing, and verify `docs/operations/promotion-pipeline.md` maps local branch gates to the checked-in GitHub workflows. | Confirm staging promotion checklist enforces named scenario completion before main promotion and that the main promotion PR carries external runbook, IncidentIQ testing-ticket, and release/deployment metadata before merge. |

## Phase 1
- Bucket-level evidence requirements for this phase:
  - `1A` canonical read model ingestion and dashboard projections
    - ingest/projection runtime evidence from masked or mock source inputs
    - conflict-surfacing evidence showing no silent normalization
    - projection-refresh evidence after successful import
    - evidence that projection-backed read surfaces expose freshness context without requiring live provider fan-out for list rendering
  - `1B` onboarding/offboarding visibility and review-only mismatch queues
    - scope evidence for HR, Site Admin, and IT review surfaces
    - queue ownership evidence for review-only operational handling
    - review-only evidence showing no write controls are available in this phase
  - `1C` phone directory visibility surfaces
    - runtime evidence for person-centric, room-centric, and department-centric synced views
    - evidence that site default and cross-site lookup behavior matches scope rules
    - evidence that the view is provider-synced rather than CSV-driven
    - evidence that `By Person`, `By Room`, and `By Department` alternate a single primary view rather than co-rendering side by side
    - evidence that all logged-in user types can reach their authorized directory surface
    - evidence that directory state labels use explicit documented values and do not surface undefined abbreviations
    - evidence that department-view classification labels and call-queue retrieval behave as documented
    - evidence that each phone-directory mode restricts visible rows to its documented type families rather than re-ranking one mixed result pool
    - evidence that DEV directory seeds keep numeric-only 4-, 5-, and 6-digit extensions valid while preferring 6-digit examples by default
    - evidence that multi-line dashboard tables use a shared top baseline across all cells and do not vertically center sparse cells
    - evidence that selected-record detail refresh remains scoped to the chosen row rather than turning the whole directory into a live provider table
- `1D` student invalid-name visibility
    - site-secretary scope evidence
    - invalid-name detail evidence including suggested correction and Aeries link
    - student-denial evidence for this dashboard surface
    - evidence that invalid-name review is a dedicated screen and does not co-render Frequent Fliers
- `1E` Frequent Fliers visibility
    - runtime evidence of threshold/lookback behavior
    - site-scope evidence for Device Wranglers and Site Admins
    - aggregation evidence for student, device, and ticket context
    - evidence that Frequent Fliers is a dedicated screen and does not co-render student invalid-name review
- `1F` room-move draft planning and validation
    - site-scoped draft-create evidence
    - district-wide IT draft-create evidence
    - draft-validation evidence proving no execution side effects occur in this phase
    - evidence that authorized directory detail actions can open a one-person targeted draft with current context prefilled
- `1A` canonical read model ingestion and dashboard projections
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P1-1A-001` | Escape Ingest Creates Canonical Person Projection | Mock Escape ingest and verify a canonical person/workflow projection is created with expected identity links. | Confirm staging ingest produces the same projection shape from masked data. |
  | `P1-1A-002` | Source Conflict Surfaces Without Silent Normalization | Feed conflicting upstream values and verify the dashboard surfaces the conflict rather than normalizing it away. | Confirm staging conflict handling produces owned review surfaces, not silent drift. |
  | `P1-1A-003` | Projection Refresh After Successful Import | Verify successful import refreshes the visible read model without manual rebuild steps. | Confirm staging projections refresh after completed sync/import runs. |
  | `P1-1A-004` | Projection Backed Lists Expose Freshness Context | Verify projection-backed dashboard and queue surfaces show last-sync or equivalent freshness context without requiring live provider calls for the list view itself. | Confirm staging list and queue surfaces expose the same freshness context while remaining projection-backed. |
- `1B` onboarding/offboarding visibility and review-only mismatch queues
  | Scenario ID | Scenario Name | Operational Owner | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- | --- |
  | `P1-1B-001` | HR District-Wide Onboarding Visibility | `HR` | Verify HR can see district-wide onboarding/offboarding status including sensitive fields. | Confirm staging HR visibility and redaction boundaries match spec. |
  | `P1-1B-002` | Site-Scoped Administrative Visibility | `Site Admin / Site Secretary` | Verify site-scoped onboarding users see only active-site records, counts, search results, drawer details, and approved fields; drawer edits are limited to Room for Site Admin and Site Secretary. | Confirm staging site scoping prevents cross-site leakage and enforces the same Room-only boundary server-side. |
  | `P1-1B-003` | Review-Only Google Active Aeries Inactive Queue | `IT Admin` | Verify the queue is visible in review-only mode with no write actions enabled in Phase 1. | Confirm staging queue ownership and review-only behavior match the phase boundary. |
  | `P1-1B-004` | HR/IT Manual Offboarding Action Authorization | `HR / IT Admin` | Verify only HR and IT Admin can see or call Emergency Offboarding and Offboard Contractor candidate/search schedule APIs, and that Site Admin, Site Secretary, Device Wrangler, Faculty and Staff, and signed-out requests receive denial before candidate data or mutation results are returned. | Confirm staging preserves the same authorization boundary before any provider-backed offboarding candidate data or write planning is exposed. |
- `1C` phone directory visibility surfaces
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P1-1C-001` | Phone Directory Person View Uses Synced Provider Data | Verify person-centric phone directory renders from synchronized provider data, not CSV/manual uploads, and that `By Person` shows only people plus common area phones. | Confirm staging person view matches real synced data shape and preserves the same person/common-area filtering. |
  | `P1-1C-002` | Phone Directory Room View Uses Synced Provider Data | Verify room-centric phone directory renders from synchronized provider data and that `By Room` shows only common area phones plus classroom shared line groups in the documented `[sitecode]+30000` range. | Confirm staging room view matches real synced data shape and preserves the same room/common-area filtering. |
  | `P1-1C-003` | Site Default With Cross-Site Lookup | Verify users land on their site by default and can still search across sites as allowed. | Confirm staging honors scoped defaults while permitting authorized cross-site lookup. |
  | `P1-1C-004` | Directory Toggle Alternates Single Primary View | Verify `By Person`, `By Room`, and `By Department` swap the same primary directory surface instead of rendering multiple directory views side by side. | Confirm staging preserves the same alternate-view behavior in the rendered UI. |
  | `P1-1C-005` | Directory State Labels Use Explicit Documented Values | Verify phone-directory assignment state values render only documented labels such as `Assigned`, `Unassigned`, `Common Area`, `Conflict`, or `Pending Change`, with no undefined abbreviations like `NV`. | Confirm staging uses the same explicit assignment-state vocabulary in rendered directory views. |
  | `P1-1C-006` | Department View Uses Department Extensions And Call Queues | Verify the `By Department` view includes only `{site code}+50+{extension}` department shared-line rows plus Zoom call queues retrieved from `GET /phones/call_queues`, excludes people, common area phones, classroom shared line groups, and auto attendants, uses documented classification labels such as `Department`, `Main Line`, and `Call Queue`, shows all queue or SLG members in `Assigned To / Destination`, and renders displayed extension values as numeric-only strings. | Confirm staging department view uses the same inclusion rules, exclusions, membership rendering, classification vocabulary, and numeric-only extension formatting against synchronized provider data. |
  | `P1-1C-007` | Phone Directory Available To All Logged-In User Types | Verify every logged-in authorized role can access the phone directory while still seeing only the scope and fields allowed for that role. | Confirm staging role access and scoping match the documented directory-availability rules. |
  | `P1-1C-008` | Dashboard Tables Preserve Shared Top Baseline | Verify representative dashboard tables with multi-line cells keep the first visible line aligned horizontally across all columns, with row height expanding downward and no vertical centering of sparse cells or state badges. | Confirm staging renders representative dashboard tables with the same shared top baseline and top-aligned badges across directory and non-directory table surfaces. |
  | `P1-1C-009` | DEV Directory Seeds Preserve Reference Extension Patterns | Verify DEV phone-directory seed data uses sanitized names and emails, preserves real extension values and type families from the read-only reference HTML, treats 4-, 5-, and 6-digit numeric extensions as valid, and prefers 6-digit examples as the default visible mock pattern. | Confirm staging continues to render real synchronized extension values without introducing non-numeric formatting artifacts or invalid type-family mixing. |
  | `P1-1C-010` | Directory Detail Live Refresh Stays Scoped To Selected Record | Verify selected-record phone-directory detail or drawer refresh can fetch fresher provider state for the active row without turning the surrounding table into a live provider fan-out surface. | Confirm staging keeps selected-record detail refresh targeted while the directory list remains projection-backed. |
- `1D` student invalid-name visibility
  | Scenario ID | Scenario Name | Operational Owner | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- | --- |
  | `P1-1D-001` | Secretary Sees Site-Scoped Active Invalid Names Only | `Site Secretary` | Verify site secretaries see only active unresolved invalid-name rows for their own site. | Confirm staging site scoping and unresolved-only behavior. |
  | `P1-1D-002` | Suggested Corrected Name And Aeries Link | `Site Secretary` | Verify invalid-name details include current Aeries first-name and last-name values with visible leading/trailing whitespace markers, separate suggested corrections only when the displayed suggestion differs, and the base `Open Aeries` link. | Confirm staging renders the same separate-field suggestion, visible-whitespace, suggestion-suppression, and base-link behavior from masked data. |
  | `P1-1D-003` | Student Login Denied For Invalid-Name Dashboard | `Site Secretary` | Verify student identities cannot access the dashboard surface. | Confirm staging enforces staff-only access against this workflow. |
  | `P1-1D-004` | Invalid-Name Review Stays On Dedicated Screen | `Site Secretary` | Verify the invalid-name dashboard does not co-render Frequent Fliers content on the same screen. | Confirm staging keeps invalid-name review as its own dedicated surface. |
  | `P1-1D-005` | Invalid-Name Queue Uses Separate Name Fields And Student-ID Sort | `Site Secretary` | Verify the queue validates separate Aeries `FirstName` and `LastName` fields, never flags missing-comma conditions, renders the human-readable name as `FirstName LastName`, and sorts rows ascending by `Student ID`. | Confirm staging preserves separate-field validation semantics and ascending `Student ID` ordering. |
- `1E` Frequent Fliers visibility
  | Scenario ID | Scenario Name | Operational Owner | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- | --- |
  | `P1-1E-001` | Frequent Fliers Threshold And Lookback Default | `Device Wrangler / Site Admin` | Verify Frequent Fliers defaults to `>=2` assignments in `90` days and shows live queue data. | Confirm staging uses the same default thresholds against masked/live-synced inputs. |
  | `P1-1E-002` | Device Wrangler Site Scope | `Device Wrangler / Site Admin` | Verify Device Wranglers and site admins see only their authorized site-scoped Frequent Fliers data. | Confirm staging scope and role inheritance behave as documented. |
  | `P1-1E-003` | Student Device Ticket Aggregation | `Device Wrangler / Site Admin` | Verify student, device, and all related student-device tickets appear in one combined view. | Confirm staging aggregation works against masked IncidentIQ-linked data. |
  | `P1-1E-004` | Frequent Fliers Stays On Dedicated Screen | `Device Wrangler / Site Admin` | Verify the Frequent Fliers dashboard does not co-render student invalid-name content on the same screen. | Confirm staging keeps Frequent Fliers as its own dedicated surface. |
  | `P1-1E-005` | Frequent Fliers Filters Auto-Apply Without Apply Button | `Device Wrangler / Site Admin` | Verify changing threshold, metric, or lookback refreshes the table immediately, the filter bar has no `Apply` button, and the fixed greater-than-or-equal operator renders as unboxed inline `≥`. | Confirm staging keeps the same immediate filter behavior and operator rendering against masked/live-synced inputs. |
  | `P1-1E-006` | Frequent Fliers Mock Views Differ Across Supported Dropdown Combinations | `Device Wrangler / Site Admin` | Verify representative threshold, metric, and lookback combinations produce visibly different DEV/mock table rows and trends rather than only changing labels. | Confirm staging filter combinations produce materially different result sets when the underlying data supports them, and document any data-limited combinations in the external evidence. |
- `1F` room-move draft planning and validation
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P1-1F-001` | Site User Saves Room-Move Draft | Verify site-scoped users can save valid drafts for their authorized site only. | Confirm staging persists site-scoped drafts correctly. |
  | `P1-1F-002` | IT Creates District-Wide Room-Move Draft | Verify IT can create district-wide drafts without site-scope restriction. | Confirm staging IT draft scope behaves as documented. |
  | `P1-1F-003` | Draft Validation For Add Change Removal Rows | Verify add/change/removal rows validate correctly without triggering execution. | Confirm staging draft validation logic matches dev for representative masked data. |
  | `P1-1F-004` | Directory Detail Opens Prefilled One-Person Move Draft | Verify an authorized site-scoped directory detail action opens a targeted one-person room-move draft with the selected person, current room, phone context, and site prefilled. | Confirm staging preserves the same prefilled corrective-draft behavior without execution side effects. |

## Phase 2
- Bucket-level evidence requirements for this phase:
  - `2A` provisioning-profile and baseline bundle foundation
    - profile-edit audit evidence
    - workflow snapshot/log evidence
    - database state check for unmapped-title blocking scope
  - `2B` common-path onboarding and baseline update/deprovision lifecycle
    - end-to-end workflow timeline or log excerpt
    - database state check before and after lifecycle execution
    - downstream action summary
    - `IncidentIQ` workflow-status evidence showing hourly-bounded user/ticket polling and dashboard linkage
    - evidence that live provider disagreement on a write path refreshes the local projection and prevents unsafe writes
  - `2C` reactivation and AD → Entra propagation warning handling
    - warning visibility evidence
    - resume/cancel/replan execution trace
    - baseline-restoration database state check
  - `2D` preferred-name self-service and downstream sync
    - request audit evidence
    - downstream sync evidence
    - Zoom `/users/{userId}` and `/phone/users/{userId}` naming evidence
    - authorization evidence that students cannot submit preferred-name edits
  - `2E` actionable Google-active / Aeries-inactive controls
    - individual and bulk action audit evidence
    - resulting queue-state database check
    - downstream workflow summary
- `2A` provisioning-profile and baseline bundle foundation
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P2-2A-001` | Profile Save Applies To Not-Yet-Started Work | Verify a profile edit changes the effective plan for imported users whose workflows have not started yet. | Confirm staging uses the latest saved profile for queued but not-yet-started work. |
  | `P2-2A-002` | Workflow Snapshot Freezes At Start | Verify a started workflow keeps its profile snapshot even if the profile changes mid-run. | Confirm staging provider steps use the workflow-start snapshot instead of rereading live edits. |
  | `P2-2A-003` | Unmapped Title Blocks Only Affected Person | Verify unmapped job titles block only those people while mapped titles continue normally. | Confirm staging blocker behavior isolates only the affected records. |
- `2B` common-path onboarding and baseline update/deprovision lifecycle
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P2-2B-001` | New Staff Common Path Onboarding | Verify a new staff member reaches the baseline provisioned state end to end in mocks. | Confirm staging common-path onboarding produces auditable state transitions without manual spreadsheet steps. |
  | `P2-2B-002` | Staff Offboarding Baseline Deprovision | Verify offboarding removes baseline access and starts the expected reclaim/ticket side effects. | Confirm staging offboarding reaches the documented baseline deprovisioned state. |
  | `P2-2B-003` | Highest Category Re-Evaluation After Assignment Change | Verify a change in active Escape assignments recalculates the highest effective category correctly. | Confirm staging assignment changes preserve or reset baseline access according to category outcome. |
  | `P2-2B-004` | IncidentIQ User Poll And External Ticket Status Linkage | Verify the workflow polls `IncidentIQ` for the user by email no more than once per hour and links only the earliest created matching externally created `Aeries` and `Verkada` tickets once the user exists there, where matching means requestor email plus ticket category using `Aeries (Asset Tag: AERIES) → User Rights → Add User` and `Security Systems → Alarm Codes → Add Alarm Code`. Verify the earliest match remains authoritative even if a later matching ticket is still open, that no linked ticket is shown if the earliest match later disappears or becomes inaccessible, and that this absence is silent with no warning text. Confirm linked results show the full raw ticket number as link plus current status value with no truncation inside the selected Onboarding person drawer, not in a standalone ticketing report route. | Confirm staging surfaces linked `IncidentIQ` user and external ticket status without taking ownership of ticket creation. |
  | `P2-2B-005` | Live Provider Disagreement Refreshes Projection Before Write | Verify a write-capable lifecycle path re-reads live provider state, detects disagreement with the stored projection, refreshes the projection, and refuses the unsafe mutation until the planner has current source truth. | Confirm staging uses live provider disagreement as a safety stop rather than applying stale projected state blindly. |
  | `P2-2B-006` | Manual Non-Escape Aeries Personal Phone Capture | Verify manual Non-Escape onboarding requires and normalizes a personal phone number, includes it in the planned Aeries upload payload only for `manual_non_escape`, and omits it for ESCAPE-sourced payloads. | Confirm staging uses masked or synthetic phone evidence and does not overwrite ESCAPE-sourced phone data. |
  | `P2-2B-007` | DEV Emergency And Contractor Offboarding Scheduling | Verify Emergency Offboarding records an immediate DEV mock deprovision only after HR/IT selects an active employee or contractor, and Offboard Contractor records a dated DEV mock deprovision only after HR/IT selects an active manual contractor and clicks Schedule Offboarding. | Confirm staging treats the workflow as what-if/pilot-gated before any provider-backed offboarding mutation runs. |
- `2C` reactivation and AD → Entra propagation warning handling
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P2-2C-001` | Reactivation Restores Baseline Not Extras | Verify reactivation restores only baseline profile access and surfaces prior extras as deltas. | Confirm staging reactivation behavior matches the documented baseline-first rule. |
  | `P2-2C-002` | Entra Propagation Warning After One Hour | Verify the workflow continues with warning when Entra convergence is still incomplete after one hour. | Confirm staging emits the warning to all workflow viewers and the IT Admin warning surface. |
  | `P2-2C-003` | Resume Cancels And Replans With Latest Profile | Verify paused workflow resume cancels the stale run and replans using the latest profile state. | Confirm staging treats cancel-and-replan as normal behavior and preserves workflow continuity. |
  | `P2-2C-004` | Active Escape Contractor Collision Blocks Workflow | Verify a manual contractor entry for an already-active Escape employee is saved, marked invalid, linked to the Escape record, and never enqueues onboarding or provider work. Confirm the drawer shows the soft-delete remediation action. | Confirm staging preserves the invalid audit record, shows the linked Escape record, and blocks workflow creation for the overlap case. |
  | `P2-2C-005` | Inactive Escape Contractor Reactivation Reuses Identity | Verify a former Escape employee rehired as a manual Non-Escape contractor reuses the existing identity instead of creating a second account. | Confirm staging reactivates the same identity under the contractor/manual baseline. |
  | `P2-2C-006` | Past-Dated Start Preserves Date And Schedules Catch-Up | Verify both Escape-backed and manual past-date starts preserve the source date, show the standard late-start warning, and schedule the next available workflow cycle. | Confirm staging treats past-date starts as late-but-valid work without rewriting the source/requested date. |
- `2D` preferred-name self-service and downstream sync
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P2-2D-001` | Preferred Name Submission For Employee Or Contractor Applies Without Review | Verify an employee or manual contractor can submit a preferred-name change and the system applies it without an HR approval gate. | Confirm staging writes the preferred-name change directly for eligible non-student identities. |
  | `P2-2D-002` | Preferred Name Sync Updates Zoom User And Phone Profiles | Verify preferred-name propagation updates the documented downstream display-name targets, including Zoom `/users/{userId}` and `/phone/users/{userId}`. | Confirm staging updates both Zoom naming surfaces in addition to the other documented downstream targets. |
  | `P2-2D-003` | Students Cannot Submit Preferred Name Through Dashboard | Verify students cannot access preferred-name self-service, and server-side writes are rejected if a student somehow reaches an authenticated dashboard state. | Confirm staging keeps student preferred-name edits blocked at both UI and write layers. |
- `2F` staff and manual-contractor legal-name rename handling
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P2-2F-001` | Escape Staff Legal-Name Change Creates Rename Job After Provisioning Exists | Verify an Escape-backed legal-name change on an already-provisioned staff identity creates a downstream rename job rather than mutating the account inline during sync. | Confirm staging keeps the source date/name authoritative while running the rename as its own audited job. |
  | `P2-2F-002` | Manual Contractor Legal-Name Change Reuses Existing Identity | Verify a legal-name change on an already-provisioned manual contractor reuses the same identity and creates the downstream rename job. | Confirm staging preserves identity continuity for contractor rename work. |
  | `P2-2F-003` | Pre-Provisioning Staff Or Contractor Name Correction Does Not Create Rename Job | Verify a source-name correction that happens before the first AD/downstream account creation produces no rename job and only affects the later initial provisioning outcome. | Confirm staging does not enqueue rename work for pre-provisioning corrections. |
  | `P2-2F-004` | Username Collision Falls Through Fallback Order | Verify an unrelated existing account holding the preferred username/email forces the rename job to continue through the documented fallback order until a unique result is found. | Confirm staging never overwrites another person's username or alias ownership when resolving a legal-name change. |
  | `P2-2F-005` | Rename Completion Preserves Google Alias And Sends Notification | Verify successful rename keeps the old primary username/email as a receive-only Google alias and emits the completion-notification email to the affected person. | Confirm staging preserves Gmail alias behavior and sends the documented completion notice. |
- `2E` actionable Google-active / Aeries-inactive controls
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P2-2E-001` | Individual Queue Action Changes Outcome | Verify IT Admin can take an allowed individual action and the row moves to the correct next state. | Confirm staging individual actions change queue state and downstream workflow state correctly. |
  | `P2-2E-002` | Bulk Queue Actions On Day One | Verify bulk selection supports the documented actions from day one without partial-state corruption. | Confirm staging bulk actions behave consistently across realistic record counts. |
  | `P2-2E-003` | Graduated Senior Suppression And Override | Verify qualified seniors appear as suppressed, can be bulk-overridden, and move to `Pending Deprovision`. | Confirm staging suppression and override behavior matches the documented grace-period rules. |

## Phase 3
- Bucket-level evidence requirements for this phase:
  - current evidence bar for this phase uses ordinary runtime evidence only; no separate manual-fallback drill is required unless a later revision raises that requirement
- `3A` room-move execution path and final review
    - runtime evidence of draft-to-final-review progression
    - scheduled execution timeline or log excerpt
    - runtime evidence that non-IT batches of more than `5` moves require explicit off-hours cutover scheduling
    - runtime evidence that a one-person targeted corrective move can execute from the directory-initiated workflow path once final review exists
    - recovery trace proving duplicate move commit prevention
  - `3B` same-site room and IncidentIQ assignment changes
    - runtime evidence of same-site room update behavior
    - authorization/scope evidence for room overrides
    - downstream IncidentIQ state check
  - `3C` Zoom phone, SLG, extension, and CAP orchestration
    - runtime evidence of phone baseline outcome
    - runtime evidence of attendance-driven CAP assignment behavior
    - the default attendance-driven CAP path is sufficient at the current evidence bar; a separate manual-exception CAP proof case is not yet required
    - conflict/recovery evidence showing no silent overwrite
  - `3D` inter-site transfer and cutover handling
    - runtime evidence of unchanged-category transfer preservation
    - runtime evidence of downgrade/reset behavior
    - cutover evidence showing destination-site scope was applied
  - `3E` bulk summer rollover and recovery
    - runtime evidence of bulk path exercised successfully
    - state check confirming non-moving users were untouched
    - fallback-ticket evidence for unresolved conflicts
- `3A` room-move execution path and final review
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P3-3A-001` | Final Review Promotes Draft To Scheduled Move | Verify an approved draft enters scheduled move state only after final review exists in this phase. | Confirm staging final review gates move execution correctly. |
  | `P3-3A-002` | Scheduled Move Executes On Effective Date | Verify scheduled moves execute at the intended effective date/time and update workflow state, including explicit off-hours scheduling for non-IT batches of more than `5` moves. | Confirm staging executes dated moves at the configured cadence boundary and enforces the non-IT off-hours rule. |
  | `P3-3A-003` | Recovery Prevents Duplicate Move Commit | Simulate interruption during move execution and verify recovery avoids duplicate commit behavior. | Confirm staging recovery protects against duplicate room-move execution. |
  | `P3-3A-004` | Directory-Initiated One-Person Correction Executes | Verify a directory-initiated one-person targeted move can progress through final review and execute immediately under the documented `5 or fewer` rule. | Confirm staging preserves the same targeted correction path from directory context into execution. |
- `3B` same-site room and IncidentIQ assignment changes
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P3-3B-001` | Same-Site Room Move Updates IncidentIQ Assignment | Verify a same-site move updates room-linked IncidentIQ assignment data correctly. | Confirm staging same-site room updates are reflected in IncidentIQ-linked state. |
  | `P3-3B-002` | Manual Room Override Unblocks Teaching Workflow | Verify an authorized room override unblocks room-dependent teaching workflow steps. | Confirm staging room override permissions and downstream effects match spec. |
  | `P3-3B-003` | Same-Site Move Avoids Cross-Site Leakage | Verify same-site execution does not expose or alter data outside the authorized site scope. | Confirm staging preserves site scoping during room-change execution. |
- `3C` Zoom phone, SLG, extension, and CAP orchestration
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P3-3C-001` | Eligible Staff Receives Zoom Phone Baseline | Verify eligible staff receive the documented Zoom phone baseline outcome during phone execution. | Confirm staging phone provisioning matches category and room/site inputs. |
  | `P3-3C-002` | Attendance Queue Drives Customer Engagement Pack | Verify attendance-linked users receive the Customer Engagement Pack outcome under the documented rule. | Confirm staging CAP assignment honors attendance and manual-exception rules. |
  | `P3-3C-003` | Phone Conflict Falls Into Recovery Instead Of Overwrite | Verify conflicting phone state is diverted into recovery/ticket handling instead of silent overwrite. | Confirm staging conflict handling protects active phone users. |
  | `P3-3C-004` | Pending Zoom Desk Phone Rename Report | Verify the IT Admin-only `/reports/zoom-desk-phone-renames` report lists only pending manual adjustment and error rows, excludes healthy/completed/non-actionable phones, renders serial number, MAC address, current name, new name, and IncidentIQ asset links, and directs IT Admins to update the IncidentIQ asset location to force the Zoom rename. | Confirm staging report eligibility and IncidentIQ asset links match masked provider projections before relying on the report for manual rename cleanup. |
- `3D` inter-site transfer and cutover handling
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P3-3D-001` | Same Category Site Transfer Preserves Baseline Access | Verify a transfer that keeps the same highest category preserves baseline entitlements. | Confirm staging site transfer preserves baseline state when category is unchanged. |
  | `P3-3D-002` | Category Downgrade Transfer Resets Baseline Access | Verify a transfer that changes the highest category resets baseline access appropriately. | Confirm staging baseline reset follows the documented category-precedence rule. |
  | `P3-3D-003` | Inter-Site Phone Cutover Uses New Site Scope | Verify phone/site execution uses the new site context during inter-site cutover. | Confirm staging inter-site phone outcomes align with the destination site scope. |
- `3E` bulk summer rollover and recovery
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P3-3E-001` | Bulk Summer Move Draft To Commit | Verify bulk summer moves can progress from draft through scheduled execution without manual data reshaping. | Confirm staging supports bulk seasonal move execution with the documented workflow states. |
  | `P3-3E-002` | Non-Moving Users Are Not Clobbered During Rollover | Verify bulk rollover logic does not modify users not present in the approved move set. | Confirm staging protects non-moving users during high-volume move execution. |
  | `P3-3E-003` | Fallback Ticket Raised For Unresolved Move Conflict | Verify unresolved move conflicts create sufficient fallback tickets for manual completion and surface the ticket in the Room Moves review drawer with owner, reason, resolution steps, linked external systems, full raw ticket number link, status, and technical-verification outcome. Verify fallback rows resolve only after the linked ticket is closed and the room/phone technical outcome is verified. | Confirm staging fallback ticket content is sufficient for IT/manual resolution and that ticket closure alone does not clear the row. |
  | `P3-3E-004` | Repeated User Added To Two Common-Area Rooms | Verify a manual bulk draft can keep the same person in two destination rows where both destination rooms have common-area/CAP coverage, selects one primary destination when explicitly provided, and keeps CAP active for the secondary shared-line-group-only row. | Confirm staging preserves CAP coverage for secondary repeated-user destinations and retires CAP only for the verified primary human assignment. |
  | `P3-3E-005` | Repeated User Added To Occupied Room As Secondary | Verify a repeated-user draft can leave the person primary in one room while adding them to an occupied destination room as secondary without raising a primary-phone overwrite plan. | Confirm staging leaves the occupied room primary phone owner unchanged and adds only SLG/IIQ membership for the secondary row. |
  | `P3-3E-006` | Repeated User Added To Occupied Rooms Without Primary Ownership | Verify repeated-user rows where the person is not primary in one or more occupied destination rooms produce shared-line-group-only outcomes and do not assign more than one desk phone. | Confirm staging protects all occupied-room primary owners while preserving repeated-user membership rows. |
  | `P3-3E-007` | Repeated User Source Is SLG-Only | Verify a person whose previous room relationship is shared-line-group-only can move into multiple destination rows without unassigning the source desk phone or converting the source room to CAP. | Confirm staging removes only the source classroom SLG membership required by the approved move and leaves source phone/CAP state intact. |
  | `P3-3E-008` | Repeated User Was Last Primary Source | Verify a person who was the last primary source for the previous room triggers safe common-area/CAP routing for that previous room unless a replacement primary is present in the same batch. | Confirm staging retains safe call routing for the vacated source room and does not leave the room without SLG-callable coverage. |
  | `P3-3E-009` | Repeated User N-Room Primary Secondary Tertiary Plan | Verify a manual bulk draft can keep one person across three or more destination rows, with one resolved primary row and secondary, tertiary, or later-order SLG-only rows. | Confirm staging supports N-room repeated-user groups without truncation, duplicate collapse, or row-order-only primary selection. |
  | `P3-3E-010` | Repeated User Ambiguous Multi-Primary Warning | Verify a repeated-user group with multiple primary rows or no deterministic primary emits actionable review warnings and holds primary phone assignment rather than silently choosing a room. | Confirm staging review output names the person, candidate rooms, affected systems, and resolution steps before any phone write proceeds. |

## Phase 4
- Bucket-level evidence requirements for this phase:
  - current evidence bar for this phase uses ordinary runtime evidence only; no retained repository artifact is required
  - `4A` orphaned Zoom cleanup queue and verification
    - runtime evidence of unresolved-to-resolved queue behavior
    - runtime evidence that generic queue rows do not expose lingering entitlement detail inline
    - verification evidence showing technical end-state polling drives resolution
  - `4B` orphaned cleanup ticket automation and auto-resolution
    - runtime evidence of ticket creation after threshold
    - runtime evidence of automation close attempt with the standard comment body
    - runtime evidence of ticket-sync warning behavior when close fails
  - `4C` sync-overlap/cadence hardening
    - runtime overlap-count evidence across the defined 7-day window
    - ticket evidence for create/update behavior and material-cadence-change notes
    - runtime evidence that no-op saves do not reset overlap state
    - evidence that missed provider events are repaired by delta or full reconciliation rather than leaving drift unresolved
  - `4D` legacy Google Sheets retirement and cutover
    - runtime or audit evidence of the 90-day no-end-user-edit gate
    - evidence distinguishing automation writes from end-user edits
    - observed parity plus the 90-day gate is sufficient at the current evidence bar; a separate staging dry-run retirement rehearsal is not yet required
    - cutover evidence showing the dashboard is the primary control surface before retirement
    - runtime evidence that the retirement clock begins only after the required `IT Admin` runbook signoff exists
- `4A` orphaned Zoom cleanup queue and verification
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P4-4A-001` | Zoom Cleanup Stays Unresolved Until Full End-State | Verify a row remains unresolved until account state and all targeted Zoom entitlements match the documented end-state. | Confirm staging cleanup rows do not resolve early on partial Zoom cleanup. |
  | `P4-4A-002` | Generic Queue Row With Zoom Cleanup Subtype | Verify the queue shows generic unresolved state plus subtype badge without exposing inline entitlement detail. | Confirm staging queue presentation matches the documented generic/specific split. |
  | `P4-4A-003` | Auto-Resolve After Verified Zoom End-State | Verify polling resolves the row automatically once Zoom reflects the full desired end-state. | Confirm staging auto-resolution waits for actual Zoom state verification. |
- `4B` orphaned cleanup ticket automation and auto-resolution
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P4-4B-001` | Ticket Created After Thirty Days Unresolved | Verify an unresolved row creates the correct IncidentIQ ticket after the 30-day threshold. | Confirm staging timer, routing, and affected-user ownership are correct. |
  | `P4-4B-002` | Automation Closes Ticket After Verified Cleanup | Verify verified cleanup triggers ticket-close attempt with the standard automation comment. | Confirm staging close behavior and comment body match the documented template. |
  | `P4-4B-003` | Ticket Close Failure Becomes Informational Warning | Verify failed ticket close leaves the row resolved and emits an IT-Admin-only informational warning. | Confirm staging does not reopen cleanup rows on ticket-sync failure. |
- `4C` sync-overlap/cadence hardening
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P4-4C-001` | Five Overlaps In Seven Days Opens Or Updates Ticket | Verify five overlaps inside seven days create or update the cadence-adjustment ticket. | Confirm staging overlap counting and ticket update behavior match the threshold rule. |
  | `P4-4C-002` | Material Cadence Change Adds Old And New Values To Ticket | Verify interval/lookback changes append the old/new values to the open overlap ticket. | Confirm staging ticket notes record the expected cadence delta details. |
  | `P4-4C-003` | Overlap Counter Persists Until Material Change | Verify overlap counting does not reset on mere ticket creation or no-op config save. | Confirm staging counter reset requires a material cadence change only. |
  | `P4-4C-004` | Missed Provider Event Repaired By Reconciliation | Verify a missed or delayed provider event is corrected by the documented delta or full reconciliation pass and does not leave the projection permanently stale. | Confirm staging reconciliation backstops can repair event loss without manual data surgery. |
- `4D` legacy Google Sheets retirement and cutover
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P4-4D-001` | Ninety Day No End-User Edit Retirement Gate | Verify the retirement gate passes only when end-user edits have been absent for 90+ days. | Confirm staging retirement evidence distinguishes end-user edits from automation writes. |
  | `P4-4D-002` | Automation Writes Do Not Block Retirement Gate | Verify automation-only sheet updates do not count as continued end-user dependence. | Confirm staging gate logic ignores system automation modifications correctly. |
  | `P4-4D-003` | Cutover Runbook Leaves Dashboard As Primary Control Surface | Verify the cutover runbook can retire legacy sheet dependence without losing operator visibility, and that the retirement clock does not begin before the required `IT Admin` runbook signoff exists. | Confirm staging cutover evidence proves dashboard parity before retirement and includes the required runbook signoff. |

## Phase 5
- Bucket-level evidence requirements for this phase:
  - `5A` student lifecycle and student-access automation
    - runtime evidence of base student profile creation from school and grade
    - runtime evidence of end-of-day recalculation after enrollment, schedule, or course change
    - state evidence showing highest-privilege resolution with same-tier app union behavior
    - evidence that student lifecycle list and queue surfaces remain projection-backed while targeted live verification is used only where safety requires it
  - `5B` multi-application orphaned-permission cleanup
    - runtime evidence of person-row badge aggregation and subtype drill-in behavior
    - runtime evidence that bulk actions remain constrained to a single subtype
    - session-state evidence for the documented view, sort, filter, and breadcrumb behavior
  - `5C` GitHub lifecycle automation
    - runtime evidence of school-specific STEM course mapping outcomes
    - runtime evidence of teacher-profile persistent access behavior independent of active course membership
    - runtime evidence of graduation-grace-period timing and reminder behavior before removal
    - runtime evidence that the offboarding reminder is delivered before GitHub removal
  - `5D` deferred app-specific entitlements
    - runtime evidence of default mapping behavior for explicitly defined deferred-app rules
    - runtime evidence that users without matching rules receive no deferred-app entitlement
    - runtime evidence that IT Admin mapping changes take effect without redeploy
- `5A` student lifecycle and student-access automation
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P5-5A-001` | Student Base Profile Created From School And Grade | Verify a new student account receives the default school/grade profile from the future student provisioning module. | Confirm staging student base provisioning follows the documented school/grade profile rules. |
  | `P5-5A-002` | Course Change Triggers End-Of-Day Access Recalculation | Verify enrollment/schedule/course changes trigger end-of-day recalculation of extended student access. | Confirm staging recalculation waits until cutoff and reflects current course participation. |
  | `P5-5A-003` | Highest Privilege Profile Wins With Same-Tier Union | Verify student access chooses the highest privilege profile and unions app access within the same tier. | Confirm staging student-access resolution follows the documented precedence model. |
  | `P5-5A-004` | Student Lifecycle Lists Stay Projection Backed With Targeted Verification | Verify student lifecycle tables, queues, and search surfaces stay projection-backed while targeted live verification is reserved for write-sensitive or destructive action paths only. | Confirm staging preserves the same projection-backed default while allowing targeted live verification where the workflow requires it. |
  | `P5-5A-005` | Aeries Student Legal-Name Change Creates Rename Job Only After Provisioning Exists | Verify an Aeries legal-name change for a student who already has an AD/downstream identity creates the downstream rename job rather than silently mutating source-linked account fields. | Confirm staging runs the student rename as an audited job only when the student account already exists. |
  | `P5-5A-006` | Pre-Provisioning Student Name Correction Does Not Trigger Rename | Verify a student name correction made in Aeries before the student's first account provisioning produces no rename job and only affects the later initial provisioning run. | Confirm staging keeps pre-provisioning student corrections out of the rename pipeline. |
- `5B` multi-application orphaned-permission cleanup
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P5-5B-001` | Person View Shows Multiple Cleanup Badges | Verify a person-centric row can show multiple orphaned-permission subtype badges in Title Case. | Confirm staging mixed-app cleanup view renders person rows with multiple subtype badges correctly. |
  | `P5-5B-002` | Mixed-App Bulk Action Restricted To Single Subtype | Verify bulk actions cannot span mixed app subtypes in the person-centric view. | Confirm staging enforces single-subtype bulk action boundaries. |
  | `P5-5B-003` | Subtype Drill-In Preserves Session View State | Verify subtype drill-in uses same-table context, breadcrumb back, and session-persisted view state as documented. | Confirm staging drill-in/back behavior matches the future multi-app queue spec. |
- `5C` GitHub lifecycle automation
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P5-5C-001` | School-Specific STEM Course Mapping Grants Student GitHub Access | Verify school-specific STEM course mappings can grant GitHub access to qualifying students. | Confirm staging GitHub eligibility respects school-specific course-number rules. |
  | `P5-5C-002` | Teacher Profile Grants Persistent GitHub Access | Verify a teacher profile can grant GitHub access independent of active course membership. | Confirm staging teacher GitHub access persists until employee-type change or district exit. |
  | `P5-5C-003` | Graduated Senior Grace Period And Reminder Before GitHub Removal | Verify graduating students keep GitHub access through the grace period and receive the personal-email reminder before removal. | Confirm staging GitHub removal timing follows the same senior grace-period rule as Google access. |
  | `P5-5C-004` | Student Offboarding Reminder Delivered Before GitHub Removal | Verify the student offboarding reminder is delivered before GitHub access is removed at the end of the grace period. | Confirm staging reminder delivery can be evidenced before GitHub removal completes. |
- `5D` deferred app-specific entitlements
  | Scenario ID | Scenario Name | Dev Mock Verification | Staging Verification |
  | --- | --- | --- | --- |
  | `P5-5D-001` | SASI Default Meraki OrgAdmin Mapping | Verify the default deferred-app mapping grants `OrgAdmin` only to `SASI` where the Meraki rule is enabled. | Confirm staging deferred Meraki mapping follows the documented initial rule. |
  | `P5-5D-002` | No Deferred App Attribute Without Matching Rule | Verify users without a matching deferred-app rule receive no attribute/access by default. | Confirm staging does not emit deferred-app entitlements in the absence of an explicit rule. |
  | `P5-5D-003` | IT Admin Changes Deferred App Mapping Without Redeploy | Verify an IT Admin can change deferred-app entitlement mapping through dashboard levers without redeploying. | Confirm staging reflects app-specific mapping changes through config, not code rollout. |
