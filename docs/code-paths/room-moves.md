# Room Moves Walkthrough

Room moves model room, phone, and shared-line-group workflow decisions in DEV before any live Zoom or IncidentIQ write exists. This path is high risk because it stages future room membership changes, handles site-scoped permissions, and supports cancel, schedule, apply, delete, and revert flows with different authorization boundaries.

## Frontend Entrypoint

- Routes: `/room-moves` and `/room-moves/bulk-draft`
- Router registration: `frontend/src/lib/routeRegistry.js`
- Page component: `frontend/src/pages/RoomMovesPage.jsx`
- App dispatch: `frontend/src/app.jsx` renders `RoomMovesPage` for `room-moves` and `room-moves-bulk-draft` route kinds.

`RoomMovesPage` loads either `/api/v1/dev/pages/room-moves` or `/api/v1/dev/pages/room-moves/bulk-draft`. The same component owns the single-move drawer, bulk draft table, schedule/apply transitions, delete, and row cancel behavior.

Key frontend helpers:

- `loadPage` fetches the page payload and handles `401` / `403` by delegating to app-level auth handlers.
- `createDraft` posts a `roomMoveDraftRequest` and navigates to `/room-moves/bulk-draft?draft_id={id}` for bulk flows.
- `SingleMoveDrawer.saveDraft` posts a single-row draft, then optionally posts `/{draft_id}/schedule` or `/{draft_id}/apply`.
- `saveBulkDraft` updates an existing bulk draft through `PUT /api/v1/dev/room-moves/drafts/{id}`.
- `transitionBulkDraft` calls `POST /api/v1/dev/room-moves/drafts/{id}/schedule` or `/apply`.
- `deleteBulkDraft` calls `DELETE /api/v1/dev/room-moves/drafts/{id}`.
- `cancelMove` calls `POST /api/v1/dev/room-moves/drafts/{draft_id}/cancel`.
- `BulkDraftTable.updateRow` keeps client-side row edits aligned with DEV API normalization: `add` rows clear current-room display, and `removal` rows immediately set destination room to `None` before save.

The completed-job revert UI is on static implemented pages, not this primary room-move page. `frontend/src/pages/StaticPenPage.jsx` loads completed jobs and calls `POST /api/v1/dev/room-moves/completed/{id}/revert` for the admin-facing revert action.

## Backend Route And Handler Chain

Routes are registered in `internal/web/app.go`:

- `GET /api/v1/dev/pages/room-moves` -> `handleDevRoomMovesPage`
- `GET /api/v1/dev/pages/room-moves/bulk-draft` -> `handleDevRoomMovesBulkDraftPage`
- `POST /api/v1/dev/room-moves/drafts` -> `handleDevRoomMoveDrafts`
- `PUT /api/v1/dev/room-moves/drafts/{id}` -> `handleDevRoomMoveDraft`
- `POST /api/v1/dev/room-moves/drafts/{id}/cancel` -> `handleDevRoomMoveDraft`
- `POST /api/v1/dev/room-moves/drafts/{id}/schedule` -> `handleDevRoomMoveDraft`
- `POST /api/v1/dev/room-moves/drafts/{id}/apply` -> `handleDevRoomMoveDraft`
- `DELETE /api/v1/dev/room-moves/drafts/{id}` -> `handleDevRoomMoveDraft`
- `GET /api/v1/dev/room-moves/completed` -> `handleDevRoomMoveCompletedJobs`
- `POST /api/v1/dev/room-moves/completed/{id}/revert` -> `handleDevRoomMoveCompletedJob`

The handler/store/helper chain lives in `internal/web/dev_room_moves.go`:

- Page handlers authenticate with `authenticatedRoomMovesPersona`, then read `devRoomMoveStore.page` or `devRoomMoveStore.ensureBulkDraft`.
- Draft mutation handlers decode `roomMoveDraftRequest`, then call `createDraft`, `updateDraft`, `cancelDraft`, `transitionDraft`, or `deleteDraft`.
- Revert handlers authenticate with `authenticatedRoomMoveRevertPersona`, then call `scheduleRevert`.
- Scope and option helpers include `canManageDistrictRoomMoves`, `roomMovesScopeSite`, `roomMoveVisibleSites`, `roomMoveRoomsForConfig`, `roomMovePeopleForConfig`, and `canAccessRoomMoveSite`.
- Draft validation and construction flows through `buildRoomMoveDraft`, mode-specific row builders, and warning helpers.
- `normalizeRoomMoveRows` is the backend guard for bulk-draft action semantics. It canonicalizes `add` and `removal` rows so reloads, saved drafts, and scheduled/applied drafts agree with the browser feedback shown while editing.

## Payload Shape

The page payload has this shape:

```json
{
  "page_id": "room-moves",
  "persona": {},
  "shell": {},
  "generated_at": "2026-05-13T00:00:00Z",
  "page": {
    "can_manage_district": true,
    "scope_site": {},
    "sites": [],
    "rooms": [],
    "people": [],
    "summary_cards": [],
    "rows": [],
    "default_bulk_roster_href": "/room-moves/bulk-draft?mode=bulk_site_roster",
    "default_build_list_href": "/room-moves/bulk-draft?mode=build_move_list"
  }
}
```

Draft mutations send `roomMoveDraftRequest`:

```json
{
  "mode": "mid_year_targeted_move",
  "person_id": "person-clover-alex-ramirez",
  "scope_site_id": "clover-hs",
  "effective_date": "2026-06-01",
  "scheduled_for": "2026-06-01T16:00:00Z",
  "rows": [
    {
      "id": "row-1",
      "person_id": "person-clover-alex-ramirez",
      "destination_site_id": "clover-hs",
      "destination_room_id": "iiq-room-cla-108",
      "action": "move"
    }
  ]
}
```

Successful mutations return `roomMoveDraftResponse`:

```json
{
  "draft": {
    "id": "rm-draft-100",
    "mode": "mid_year_targeted_move",
    "status": "draft",
    "scope_site_id": "clover-hs",
    "scope_site": "Clover High School",
    "effective_date": "2026-06-01",
    "scheduled_for": "2026-06-01T16:00:00Z",
    "author": "IT Admin",
    "warnings": [],
    "rows": [],
    "can_edit": true,
    "can_delete": true,
    "can_manage_district": true
  }
}
```

Bulk draft row actions have persisted room-clearing semantics. `change` is the default and preserves the person's current-room context while planning a destination. `add` represents a person who should be added to the selected destination room without a prior room association in this draft, so the saved payload returns `current_room_id: "none"` and a blank `current_room`. `removal` represents removing the person from room phones, shared line groups, and call queues at the site, so the saved payload always returns `destination_room_id: "none"` and `destination_room: "None"` even if the browser submitted an older destination room value.

## Authorization And Persona Behavior

All routes require DEV mode and an authenticated DEV persona.

`authenticatedRoomMovesPersona` requires `routeAllowed(config, "/room-moves")`. A persona that cannot access Room Moves receives `403`. Site-scoped personas can only see and mutate drafts for visible sites. District managers can operate across district-visible sites through `canManageDistrictRoomMoves`.

Completed-job revert is stricter. `authenticatedRoomMoveRevertPersona` requires both district room-move authority and `routeAllowed(config, "/admin")`; the current handler message says only IT Admin can revert completed room move jobs.

## Mutation Boundary

The mutation boundary is the in-memory `devRoomMoveStore` in `internal/web/dev_room_moves.go`. It owns `drafts`, `completed`, `canceled`, and `jobs` maps behind `devRoomMoveStoreState.mu`.

Mutation effects are DEV-only:

- Create/update changes an in-memory draft.
- Cancel marks pending drafts in `canceled` and removes them from review rows.
- Schedule sets draft status to `scheduled`.
- Apply records the draft as completed and creates a completed-job record.
- Delete removes pending draft state.
- Revert creates a scheduled reversal draft from a completed job.

Keep this aligned with `docs/external-write-inventory.md`: the path models future Zoom room shared-line-group, room membership, site-extension, and common-area-phone effects, but it does not call a live provider SDK or write the database today.

## Tests

Relevant tests live in `internal/web/dev_frontend_test.go`:

- `room moves page scopes site data and admin controls by persona`
- `room moves bulk drafts support roster and manual list lifecycle`
- `room moves cancel pending drafts and schedule IT-only completed-job reversals`

Run targeted tests with:

```bash
go test ./internal/web -run 'TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment/(room moves enforce site scoped drafts and room defaults|room moves bulk drafts support roster and manual list lifecycle|room moves cancel pending drafts and schedule IT-only completed-job reversals)'
```

If the local Go toolchain is unavailable, use `make test-container` from the repo root.

## Debugging Breakpoints

Frontend breakpoints:

- `frontend/src/app.jsx`, where route kinds dispatch to `RoomMovesPage`.
- `frontend/src/pages/RoomMovesPage.jsx` `loadPage`, `createDraft`, `SingleMoveDrawer.saveDraft`, `saveBulkDraft`, `transitionBulkDraft`, `deleteBulkDraft`, and `cancelMove`.
- `frontend/src/pages/StaticPenPage.jsx` `loadCompletedJobs` and `revertJob` for completed-job revert debugging.

Backend breakpoints:

- `internal/web/app.go` `NewAppHandler`, to confirm route registration.
- `internal/web/dev_room_moves.go` `authenticatedRoomMovesPersona` and `authenticatedRoomMoveRevertPersona`.
- `internal/web/dev_room_moves.go` `handleDevRoomMoveDrafts`, `handleDevRoomMoveDraft`, `handleDevRoomMoveCompletedJobs`, and `handleDevRoomMoveCompletedJob`.
- `internal/web/dev_room_moves.go` store methods `createDraft`, `updateDraft`, `transitionDraft`, `cancelDraft`, `deleteDraft`, and `scheduleRevert`.

Useful request symptoms:

- `401` means the DEV session cookie is missing or invalid.
- `403` means the persona lacks the route, site scope, or admin revert authority.
- `400 invalid_json` means the request body could not decode.
- `409` usually means the draft is already completed, canceled, or otherwise in a state that blocks the requested transition.
