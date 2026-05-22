# Environment Data Playbook

## Purpose
This document defines the required process for creating and refreshing safe development and staging environments from production-derived data. The goal is to replace the current high-risk production-first workflow with a repeatable, testable environment strategy.

## Environment Roles
- `dev`: freely breakable developer environment, mock-heavy by default. Remote
  deployment examples identify this role with `ENVIRONMENT_ROLE=dev`,
  `APP_ENV=development`, `ENVIRONMENT_DATA_MODE=mock`, all `USE_MOCK_*` flags
  set to `true`, and a `postgres-dev` database URL.
- `staging`: long-lived test environment for realistic end-to-end validation
  before promotion. Remote deployment examples identify this role with
  `ENVIRONMENT_ROLE=staging`, `APP_ENV=staging`,
  `ENVIRONMENT_DATA_MODE=masked-read-only`,
  `ENVIRONMENT_DATA_SOURCE=masked-production-derived`, and a `postgres-staging`
  database URL. After a provider sandbox strategy and write-safety approval are
  documented, staging may use `ENVIRONMENT_DATA_MODE=sandbox` and
  `ENVIRONMENT_DATA_SOURCE=documented-sandbox` instead.
- `main`: production. Remote deployment examples identify this role with
  `ENVIRONMENT_ROLE=main`, `APP_ENV=production`,
  `ENVIRONMENT_DATA_MODE=production`, all `USE_MOCK_*` flags set to `false`,
  and a `postgres-main` database URL.
- Aeries exception: because no sandbox/staging tenant exists, realistic Aeries integration in non-production uses masked production-backed read-only data, ideally from previous school years.
- Aeries staging should determine the current school year from Aeries School Info and default to `current school year - 1`.
- Example: if Aeries School Info reports `2025-2026`, staging should default to `DatabaseYear=2024`.
- If Aeries School Info disagrees across schools, use the earliest start date and latest end date across all schools to define the district school year before subtracting one year for staging.

## Principles
1. Production is never the first place a new write path is tested.
2. Dev and staging must use separate datasets.
3. Staging should prefer real sandbox tenants when providers offer them.
4. When sandbox tenants do not exist, staging must use masked production-derived data.
5. Sensitive data is masked or omitted before landing in non-production.
6. Refresh steps must be documented, repeatable, and auditable.
7. Provider credentials, tokens, private keys, passwords, and auth headers are never copied into non-production datasets or retained in refresh artifacts.

## Data Classes
- `Safe synthetic`: entirely fake or mocked data for dev.
- `Masked production-derived`: production structure and relationships preserved, but sensitive values transformed.
- `Sandbox provider data`: real third-party sandbox tenants with test identities and resources.
- `Excluded`: fields too sensitive or unnecessary to copy into non-production.

## Required Output Documents and Assets
- updated `docs/reference-inputs/VENDORED_INVENTORY.md` when a refresh changes vendored code or other reference inputs
- passing `P0-0A-001` reference-input startup guard evidence when a refresh or deployment depends on repo-local reference snapshots
- documented masking rules per source system
- repeatable export scripts or jobs
- repeatable masking transform scripts
- repeatable load steps for app database
- per-provider environment configuration
- validation checklist after each refresh
- rollback instructions if the refresh is bad

## Required Steps To Build Staging From Production-Safe Data

### Step 1. Freeze the Refresh Window
- Announce a staging refresh window.
- Confirm no active destructive staging tests are running.
- Record the source snapshot date and time.

### Step 2. Export Production Inputs
- Export the minimum source data needed for realistic staging behavior:
  - application database snapshot
  - provider reference exports required for synchronization behavior
  - directory, room, phone, and group reference data
  - job-title and entitlement mapping tables
- Do not export unnecessary provider fields just because they exist.

### Step 3. Apply Masking and Omission Rules
- Replace or remove direct identifiers and sensitive fields before loading into staging.
- Remove or replace all reusable credential material before data leaves the source environment, including auth headers, bearer tokens, refresh tokens, private keys, client secrets, passwords, and raw service-account JSON.
- Preserve referential integrity and deterministic joins.
- Preserve enough structural truth that:
  - room conflicts still exist
  - title mismatches can still be exercised
  - phone-assignment logic still sees realistic shapes
  - lifecycle state transitions can still be tested
- Use deterministic transforms where repeated refreshes must preserve cross-table linkage.

### Step 4. Build the Staging Database
- Load the masked dataset into a clean staging database.
- Rebuild any derived tables, hashes, projections, and indexes.
- Seed required environment-specific control tables.
- Mark the dataset with:
  - source snapshot timestamp
  - masking rules version
  - refresh operator

### Step 5. Configure Third-Party Integrations
- Prefer true sandbox tenants for:
  - Zoom
  - Google
  - Incident IQ
  - other write-capable providers
- If a sandbox is unavailable:
  - use masked data
  - disable writes until the provider path has a safe non-production strategy
- Record which providers are:
  - real sandbox
  - masked read-only
  - mocked
- Aeries-specific environment rule:
  - use read-only `base URL + certificate`
  - no dedicated sandbox/staging tenant is available
  - prefer previous-year API reads via `DatabaseYear=YYYY` so non-production uses valid but stale records
  - determine current school year from Aeries School Info, then default staging to `current school year - 1`
  - if schools disagree, use the earliest start and latest end dates across schools to define the school year
  - preserve masking and omission rules before any non-production persistence
  - the checked-in staging example disables only the Aeries mock provider and
    keeps `AERIES_READ_ONLY=true`,
    `AERIES_DATABASE_YEAR_MODE=previous_school_year`, and
    `AERIES_MASKED_PREVIOUS_YEAR_ONLY=true`; this is a read-only connectivity
    profile, not permission to write to Aeries
  - startup/config evidence should pass sanitized School Info records into
    `internal/provider.ResolveAeriesPreviousYearStagingConfig`, then record the
    resolved `DatabaseYear` value and the `DatabaseYear=YYYY` query parameter
    in the external IncidentIQ testing ticket or promotion runbook without
    copying credentials, certificates, auth headers, or raw student/staff data
- Provider readiness rule:
  - the app's `/health/ready` provider diagnostics must show either `mocked`
    for intentionally mocked providers or `ok` for live-mode providers whose
    configuration labels pass the local readiness gate
  - live-mode readiness failures must return `503` with a `provider_<name>`
    dependency that names the missing or malformed environment label
  - readiness failure drills must use staging-safe mock, sandbox, or masked
    read-only configuration only; do not reuse production credentials to prove
    a non-production failure path
  - the detailed provider label list and `P0-0D-002` verification commands live
    in `docs/operations/provider-readiness.md`

### Step 6. Validate the Staging Refresh
- Validate record counts and key relationship counts.
- Validate masked data coverage.
- Validate core workflow smoke tests:
  - onboarding
  - offboarding
  - room move
  - name change
  - ticket-safe-to-create detection
- Validate that dangerous writes are pointed only at sandbox or explicitly approved masked endpoints.

### Step 7. Open Staging For Test Use
- Publish the refresh summary.
- Record known gaps.
- Mark staging ready only after validation passes.

## Required Steps To Build Dev

### Default Dev Mode
- Start from mocks by default.
- Seed lightweight synthetic datasets that reflect:
  - multiple sites
  - multiple user roles
  - valid and invalid room assignments
  - common failure states

### Optional Realism Mode
- If a developer needs higher realism:
  - derive dev from a staging-safe masked dataset, not from raw production
  - reduce row counts where possible
  - keep all external writes mocked unless the developer has an explicit sandbox need

## Refresh Playbook

### Remote Compose Database Persistence
- The manual QEMU deployment stack in `docker-compose.deploy.yml` uses named
  Docker volumes for `postgres-dev`, `postgres-staging`, and `postgres-main`.
- `postgres-dev-data` and `postgres-staging-data` are intentionally long-lived
  and must survive ordinary app redeploys.
- Redeploying with `deploy/remote-redeploy.sh` may rebuild app images and
  recreate containers, but it must not delete database volumes.
- Do not run `docker compose -f docker-compose.deploy.yml down -v` unless the
  operator intentionally wants to destroy the remote databases and has captured
  any required backup or refresh evidence first.
- Staging data loaded into the remote compose database must still follow the
  masking, sandbox, read-only, and validation requirements in this playbook.
- Production database storage must use encrypted host storage, encrypted
  managed storage, or another documented encrypted-at-rest mechanism before the
  main environment is treated as production-ready.

### Staging Refresh Cadence
- Refresh on a documented schedule.
- Refresh additionally before major release testing or when reference data becomes too stale.

### Dev Refresh Cadence
- Refresh on demand or on a lightweight schedule when developer test data no longer reflects current workflows.

### Every Refresh Must Record
- vendored inventory manifest update when repo-local reference inputs changed
- reference input snapshot integrity evidence from `go test ./internal/referenceinputs ./internal/config ./cmd/provisioner`
- who ran it
- when it ran
- source snapshot used
- masking version used
- validation outcome
- known follow-up actions

## Promotion Safety Gates
- No new provider write path may promote to `main` unless it has been exercised in:
  - local or mocked tests
  - staging with representative data
- Promotion requires proof that:
  - masking or sandbox strategy exists
  - failure handling is understood
  - rollback behavior is documented
  - ticket fallback behavior is verified where automation cannot complete
  - projection freshness and live action-path verification behavior are both understood for the affected provider

### Environment Role Static Validation
- `npm run environment-roles:check` is the Phase 0 static gate for scenario
  `P0-0A-002` Environment Role Separation.
- The check parses `deploy/env/dev.app.env.example`,
  `deploy/env/staging.app.env.example`, `deploy/env/main.app.env.example`, the
  matching database env examples, and `docker-compose.deploy.yml`.
- The check fails when dev, staging, and main do not declare distinct
  `ENVIRONMENT_ROLE`, `APP_ENV`, `ENVIRONMENT_DATA_MODE`, database names,
  database users, database service URLs, or provider mock flags.
- Dev verification is intentionally strict: the dev app example must stay
  mock-backed and must keep every `USE_MOCK_*` provider flag enabled so a local
  or remote dev run cannot depend on production-only data.
- Staging verification accepts the current
  `ENVIRONMENT_DATA_MODE=masked-read-only` plus
  `ENVIRONMENT_DATA_SOURCE=masked-production-derived` pair, or the future
  `ENVIRONMENT_DATA_MODE=sandbox` plus
  `ENVIRONMENT_DATA_SOURCE=documented-sandbox` pair after a provider has a
  documented sandbox strategy and safety approval. Staging provider mock flags
  must remain explicit booleans so individual `USE_MOCK_*` settings can be
  disabled only when the corresponding sandbox or masked/read-only provider
  plan exists.
- Compose verification checks that every `app-*` and `postgres-*` service loads
  the matching role-specific env file, so a staging service cannot silently load
  dev or main configuration while still passing service-name checks.
- `scripts/run_local_ci.py` includes this check in every branch target so the
  static role contract stays aligned with the checked-in branch gates.

### GitHub Branch Gates
- Checked-in CI/CD branch-gate behavior is defined in `docs/operations/promotion-pipeline.md`.
- `dev` validation is intentionally mock-heavy and local-first. It proves that repository tests, design sync checks, lint checks, and frontend build behavior are clean before work is proposed for `staging`.
- `staging` validation includes the `dev` checks plus security and frontend accessibility checks. Staging remains the required proving ground for representative data, sandbox providers, masked production-derived data, and write-path safety evidence before production promotion.
- `P0-0A-001` staging validation must prove the deployment uses only the checked-out `docs/reference-inputs/` corpus for required references. Workstation paths, cloud-drive paths, missing future snapshots, and links that escape the repository are blockers until the sanitized snapshot and `docs/reference-inputs/VENDORED_INVENTORY.md` entry are added.
- `main` validation includes the `staging` checks plus release-prep static validation. Main promotion PRs must identify the external promotion runbook, the external IncidentIQ testing ticket, and release/deployment metadata before merge.
- Automated promotion PRs require the `PROMOTION_PR_TOKEN` repository or organization secret. The token must be separate from `github.token` so GitHub creates ordinary `pull_request` checks for promotion PRs.
- Workflow secrets and GitHub environments must keep staging and production credentials separate. If a required staging or production credential, environment, release label, deployment manifest, or product decision is missing, document that blocker in the issue or PR rather than guessing or reusing production credentials in staging.

## Provider Access Strategy Validation
- Non-production validation should distinguish clearly between:
  - projection source cadence
  - live action-path reads
  - write-path verification expectations
- API-backed list and queue surfaces should be validated primarily through local projections rather than per-request provider fan-out.
- Selected-record detail, explicit refresh, and write-capable or destructive flows should be validated against live provider reads where the product design depends on fresher provider truth for safety.
- Event-capable providers must still be tested for missed-event recovery through scheduled delta and full reconciliation rather than assuming event delivery is complete or perfect.
- Promotion evidence for API-backed providers should show both:
  - that the local projection can tolerate ordinary provider or API lag without corrupting operator workflows
  - that targeted live verification can still protect action paths when the projection is stale
- Staging validation should confirm that projection freshness targets and live verification expectations are appropriate for each provider before write-capable behavior is promoted.
- Staging readiness review for `P0-0D-004` must reuse the provider capability classification matrix in `docs/planning/implementation-plan.md`. The reviewer should not create a separate staging-only classification; any provider mode or freshness change found during staging must be reflected back into the implementation plan, product requirements, and test matrix before promotion evidence is accepted.

## Provider-Specific Expectations
- Zoom: test assignment, SLG changes, CAP handling, and limited-license reclamation without relying on production users or production devices.
- Zoom: validate the hybrid model directly, with projection-backed operator surfaces, event or delta refresh behavior, and live end-state verification for action paths.
- Incident IQ: verify parent ticket creation first; add subticket and subtask support before declaring the workflow surface complete.
- Incident IQ: validate that list and status projections remain useful without live fan-out while selected-ticket or action-sensitive refresh paths can still query current provider state when needed.
- Google and AD chain: confirm ticket-safe-to-create timing after propagation.
- Google and AD chain: verify that minimal local identity facts are enough for joins and workflow planning, while collision checks, rename verification, and destructive-action confirmation still rely on live reads.
- Aeries: validate read-only integration against the mandatory v1 endpoint families (`staff`, `students`, `schools`, `teachers`, `scheduling`) using masked previous-year data where possible.
- Aeries: verify that staff-related and teacher-related API paths both resolve and can be converged into one internal entity view together with scheduling context.
- Aeries: treat the API as a read-only projection source rather than a live interactive dependency for list views or routine operator navigation.
- Verkada: verify ticket generation timing rather than direct account provisioning.

## Required Tooling TODOs
- export tooling for production-safe source snapshots
- deterministic masking tooling
- environment bootstrap scripts
- validation scripts for refresh correctness
- documentation of provider-by-provider sandbox availability and fallbacks

## Definition Of Done For Environment Safety
The environment strategy is not done until:
- mock, staging, and production roles are clearly separated
- refresh steps are documented and repeatable
- masked or sandbox data paths exist for all critical write workflows
- staging is the required proving ground before production promotion
- the team can refresh staging without improvising from memory
