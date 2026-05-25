# Manual Onboarding Drafts Walkthrough

Manual onboarding drafts let Human Resources and IT Admin model a Non-Escape onboarding record in DEV without writing to Escape, Google, Zoom, IncidentIQ, Aeries, Verkada, or any live provider. This path is high risk because it validates sensitive identity intake fields, detects collisions with Escape-backed employees, and demonstrates where future production onboarding writes will need approval, audit, and idempotency protections.

## Frontend Entrypoint

- Route: `/onboarding`
- Router registration: `frontend/src/lib/routeRegistry.js`
- Page component: `frontend/src/pages/OnboardingPage.jsx`
- App dispatch: `frontend/src/app.jsx` renders `OnboardingPage` when `currentRoute.kind === "onboarding"`.

`OnboardingPage` loads `/api/v1/dev/pages/onboarding`, renders the page and shared-shell overlay, and exposes the `Add Non-Escape Record` hotspot when the page payload sets `page.can_manage_manual` to `true`. Manual-draft user actions flow through these frontend helpers:

- `handleAddManual` calls `POST /api/v1/dev/onboarding/manual-drafts`.
- `saveDraft` calls `PUT /api/v1/dev/onboarding/manual-drafts/{id}` with the current form.
- `handleSaveDraft` saves first, then calls `POST /api/v1/dev/onboarding/manual-drafts/{id}/finalize` when the saved draft has no missing fields and is not invalid.
- `handleDeleteDraft` calls `DELETE /api/v1/dev/onboarding/manual-drafts/{id}`.
- `RoomOverrideForm` calls `PUT /api/v1/dev/onboarding/rows/{id}/room` from row drawers when the selected row is visible to the active persona and can accept a DEV room override.
- The autosave effect calls `saveDraft` every 60 seconds when `dirtyRef.current` is set.

## Backend Route And Handler Chain

Routes are registered in `internal/web/app.go`:

- `GET /api/v1/dev/pages/onboarding` -> `handleDevOnboardingPage`
- `POST /api/v1/dev/onboarding/manual-drafts` -> `handleDevOnboardingManualDrafts`
- `PUT /api/v1/dev/onboarding/manual-drafts/{id}` -> `handleDevOnboardingManualDraft`
- `POST /api/v1/dev/onboarding/manual-drafts/{id}/finalize` -> `handleDevOnboardingManualDraft`
- `DELETE /api/v1/dev/onboarding/manual-drafts/{id}` -> `handleDevOnboardingManualDraft`
- `PUT /api/v1/dev/onboarding/rows/{id}/room` -> `handleDevOnboardingRoomUpdate`

The handler/store/helper chain lives in `internal/web/dev_onboarding.go`:

- `handleDevOnboardingPage` builds `onboardingPagePayload` from the authenticated DEV persona, `devOnboardingStore.rows`, `devOnboardingStore.draftPayloads`, and `devOnboardingFormOptions`.
- `handleDevOnboardingRoomUpdate` validates route access, active site scope, site-level field restrictions, and room options before it mutates the in-memory DEV room override state.
- `handleDevOnboardingManualDrafts` requires a manual-onboarding manager, then calls `devOnboardingStore.create`.
- `handleDevOnboardingManualDraft` parses the draft id and optional `finalize` action, then calls `devOnboardingStore.update`, `devOnboardingStore.finalize`, or `devOnboardingStore.softDelete`.
- `sanitizeManualDraftRequest` normalizes form input and rejects invalid dates, last-4 SSN values, email addresses, personal phone numbers, sites, rooms, titles, employee types, classifications, devices, Aeries access choices, or replacement employees.
- `applyDerivedDraftStateLocked` computes missing-field state, Escape collision state, reactivation data, generated email, generated employee id, and late-start scheduling.

## Payload Shape

The page payload has this shape:

```json
{
  "page_id": "onboarding",
  "persona": {},
  "shell": {},
  "generated_at": "2026-05-13T00:00:00Z",
  "page": {
    "can_manage_manual": true,
    "rows": [],
    "drafts": [],
    "manual_draft_retention": "30 days",
    "manual_autosave_seconds": 60
  },
  "form": {
    "employee_types": [],
    "classifications": [],
    "job_titles": [],
    "sites": [],
    "preferred_devices": [],
    "requested_aeries_access": [],
    "replacing_employees": [],
    "rooms": []
  },
  "hotspots": {
    "add_manual": { "node_id": "f109", "label": "Add Non-Escape Record" }
  }
}
```

Draft create/finalize/delete responses use `onboardingManualDraftResponse`:

```json
{
  "draft": {
    "id": "manual-draft-1",
    "status": "Incomplete Data",
    "start_date": "2026-05-20",
    "ssn_last4": "1234",
    "employee_type": "Contractor",
    "classification": "Contractor",
    "first_name": "Sam",
    "last_name": "Taylor",
    "job_title": "Counselor",
    "site_id": "district-office",
    "personal_email": "sam@example.invalid",
    "personal_phone": "7075550134",
    "preferred_device": "Mac",
    "requested_aeries_access": "Staff",
    "missing_fields": [],
    "created_at": "May 13, 2026 9:00 AM PDT",
    "updated_at": "May 13, 2026 9:00 AM PDT"
  },
  "rows": []
}
```

`PUT` accepts `onboardingManualDraftRequest` fields: `start_date`, `ssn_last4`, `employee_type`, `classification`, `first_name`, `last_name`, `job_title`, `site_id`, `personal_email`, `personal_phone`, `preferred_device`, `requested_aeries_access`, `replacing_employee_id`, `room_id`, and `notes`. Personal phone is required for manual Non-Escape drafts, accepted only as canonical `10` digits or the drawer-submitted `(NNN) NNN-NNNN` display format, canonicalized to a `10`-digit United States number, and used only by planned Aeries upload serialization for `manual_non_escape` source rows. District email override fields such as `generated_email`, `district_email`, `assigned_email`, and `work_email` are intentionally unsupported; if a direct API caller submits one, the update is rejected so unauthorized `@wusd.org` contractor patterns cannot be silently accepted or ignored.

## Authorization And Persona Behavior

All routes require DEV mode. Page load calls `resolveAuthenticatedDevPersona` and then `routeAllowed(config, "/onboarding")`; unauthenticated requests return `401`, and disallowed personas return `403`.

Manual draft mutations go through `requireManualOnboardingManager`. Only `it_admin` and `human_resources` can create, update, finalize, or delete manual onboarding drafts. Other personas may be allowed to view onboarding rows but receive `403` for manual-draft mutations. Manual managers receive district-wide site options; other visible-site behavior is scoped by the persona configuration returned from `internal/web/dev_frontend.go`.

Site Admin and Site Secretary receive onboarding rows, drawer details, global-search onboarding results, and room options only for the active assigned site. Site Admin is stricter than generic site-scoped roles: the DEV fixture exposes exactly one assigned site, and any future production or grant-management input that would create multiple active Site Admin sites must fail closed outside this DEV mock path. Their row-drawer mutation boundary is `Room` only: `PUT /api/v1/dev/onboarding/rows/{id}/room` rejects any non-`room_id` request key with `403`, rejects out-of-scope rows with `404`, and rejects rooms outside the effective site options with `400`.

## Mutation Boundary

The mutation boundary is the in-memory `devOnboardingStore` in `internal/web/dev_onboarding.go`. Store methods lock `devOnboardingStoreState.mu`, mutate `drafts` or DEV-only `roomOverrides`, purge expired non-finalized drafts, and return cloned payloads. New manual Non-Escape contractor draft generation uses the approved external contractor email domain `@ext.wusd.org`; inactive Escape identity reuse preserves the existing assigned email on the linked identity. There are no live provider writes and no database writes in this path.

Keep this aligned with `docs/planning/external-write-inventory.md`: manual draft create, update, finalize, soft-delete, and room override are DEV mock mutations only. The personal phone value is sensitive workflow data; tests may use deterministic reserved `555-01xx` values, but diagnostics, audit summaries, and generated artifacts should redact or omit raw values. Future production onboarding writes must add provider-specific idempotency keys, request logging, staging validation, sanitized diagnostics, and rollback expectations before merging.

## Tests

Relevant tests live in `internal/web/dev_frontend_test.go`:

- `onboarding page exposes manual intake options only to hr and it`
- `onboarding scopes site persona rows search and room-only drawer updates`
- `manual onboarding draft validates sanitizes and finalizes into mock row`
- `inactive escape contractor reactivation reuses existing identity`
- `active escape contractor collision saves invalid draft and allows soft delete`
- `past-dated manual entry shows warning fields and schedules next cycle`
- `escape-backed past-date row preserves source date and exposes next-cycle schedule`
- `manual onboarding generated email falls through collision order`
- `createAndFinalizeManualOnboarding`, the test helper used by later scenarios

Run targeted tests with:

```bash
go test ./internal/web -run 'TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment/(onboarding page exposes manual intake options only to hr and it|onboarding scopes site persona rows search and room-only drawer updates|manual onboarding draft validates sanitizes and finalizes into mock row|inactive escape contractor reactivation reuses existing identity|active escape contractor collision saves invalid draft and allows soft delete|past-dated manual entry shows warning fields and schedules next cycle|escape-backed past-date row preserves source date and exposes next-cycle schedule|manual onboarding generated email falls through collision order)'
```

If the local Go toolchain is unavailable, use the repo container test path described in `README.md`.

## Debugging Breakpoints

Frontend breakpoints:

- `frontend/src/app.jsx`, where `/onboarding` dispatches to `OnboardingPage`.
- `frontend/src/pages/OnboardingPage.jsx`, especially `loadPage`, `handleAddManual`, `saveDraft`, `handleSaveDraft`, and `handleDeleteDraft`.
- `frontend/src/pages/OnboardingPage.jsx` `ManualDraftDrawer`, when field state or collision UI is wrong.

Backend breakpoints:

- `internal/web/app.go` `NewAppHandler`, to confirm route registration.
- `internal/web/dev_onboarding.go` `handleDevOnboardingPage`, `handleDevOnboardingManualDrafts`, `handleDevOnboardingManualDraft`, `handleDevOnboardingRoomUpdate`, and `requireManualOnboardingManager`.
- `internal/web/dev_onboarding.go` `sanitizeManualDraftRequest`, `applyDerivedDraftStateLocked`, `finalize`, and `softDelete`.

Useful request symptoms:

- `401` means the DEV session cookie did not resolve to a persona.
- `403` means the persona cannot access `/onboarding`, is not `it_admin` / `human_resources` for manual-draft mutations, or submitted a non-Room field to the site-level room update route.
- `400 validation_failed` means `sanitizeManualDraftRequest` rejected field content.
- `409 unsupported_overlap` means a manual contractor draft matched an active Escape employee and must be deleted instead of finalized.
