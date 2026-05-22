# Sync Dashboard Overrides Walkthrough

Sync dashboard overrides represent high-risk manual exception handling for sync dry runs. The current checkout does not persist override records or call live providers; it exposes route stubs, sync projection logic, and planner operation names that document how future write-capable sync override work must be guarded.

## Frontend Entrypoint

There are two visible entrypoints:

- Static sync dashboard route: `/sync-dashboard`, served by Go HTML in `internal/web/app.go`.
- DEV implemented-page route: `/data-quality`, registered in `frontend/src/lib/routeRegistry.js` and rendered by `frontend/src/pages/DataQualityPage.jsx`.

`DataQualityPage` loads `/api/v1/dev/pages/data-quality` and presents the inline Data Quality queue. The implemented `/data-quality` page intentionally does not link to `/sync-dashboard/mappings`: the current PRD and implementation plan do not define a supported mapping-dashboard workflow from Data Quality, so routing guidance lives in the shared help drawer until a documented IT Admin mapping surface replaces the legacy stub. The page payload includes a `Site mismatch` queue row whose next action is `Apply temporary override`, but the implemented DEV page does not currently post an override.

The legacy or API-level sync override route is:

- `POST /api/v1/sync-status/{user_type}/{user_id}/override`

No React component calls that route in this checkout. Use it as the backend stub and future integration target when debugging override behavior.

## Backend Route And Handler Chain

Routes are registered in `internal/web/app.go`:

- `GET /sync-dashboard` -> `handleSyncDashboard`
- `GET /sync-dashboard/mappings` -> `handleSyncDashboardMappings`
- `GET /api/v1/sync-status/pending` -> `handleSyncStatusRoutes`
- `GET /api/v1/sync-status/in-progress` -> `handleSyncStatusRoutes`
- `GET /api/v1/sync-status/completed` -> `handleSyncStatusRoutes` -> `writeSyncTabResponse`
- `GET /api/v1/sync-status/history` -> `handleSyncStatusRoutes` -> `writeSyncTabResponse`
- `POST /api/v1/sync-status/{user_type}/{user_id}/override` -> `handleSyncStatusRoutes`
- `GET /api/v1/dev/pages/data-quality` -> `handleDevDataQualityPage` in `internal/web/dev_frontend.go`

Domain and planner helpers:

- `internal/core/sync.go` defines sync subject types, phases, issue codes, `ProjectSyncProgress`, and `AnnualResetDisposition`.
- `internal/orchestrator/planner.go` plans `staff_sync_dry_run`, `student_sync_dry_run`, `sync_recheck`, and `annual_reset_archive` operation names.
- `internal/orchestrator/planner.go` annual reset includes `internal.clear_sync_exception_overrides`, matching the external write inventory.

## Payload Shape

The current override route accepts the route path as the payload source and does not decode a request body:

```http
POST /api/v1/sync-status/staff/12345/override
```

Successful response:

```json
{
  "status": "accepted",
  "user_type": "staff",
  "user_id": "12345"
}
```

Tab list responses use:

```json
{
  "tab": "completed",
  "filters": {
    "site_code": "clover-hs",
    "user_type": "staff",
    "school_year": "2025-2026"
  },
  "items": []
}
```

The DEV data-quality page payload includes queue rows and mapping-dashboard hotspots:

```json
{
  "page_id": "data-quality",
  "page": {
    "queue": {
      "rows": [
        {
          "issue": "Site mismatch",
          "source": "Escape / Aeries",
          "owner": "HR",
          "impact": "Blocks baseline site selection",
          "next_action": "Apply temporary override"
        }
      ]
    }
  }
}
```

## Authorization And Persona Behavior

The legacy `/sync-dashboard`, `/sync-dashboard/mappings`, and `/api/v1/sync-status/...` routes do not currently authenticate or check persona. They are route stubs and HTML/JSON scaffolding.

The implemented DEV data-quality page does enforce persona behavior. `handleDevDataQualityPage` requires DEV mode, `resolveAuthenticatedDevPersona`, and `routeAllowed(config, "/data-quality")`; unauthenticated requests receive `401`, and disallowed personas receive `403`. The frontend reads the page through `DataQualityPage`'s React Query call to `fetchDevApiJSON`; `handleDevApiAuthError` receives `401` and `403` failures and calls the app-level unauthorized or forbidden handler.

Any future production override mutation must add explicit authorization before it becomes a real write path. Do not rely on the current stub behavior for production.

## Mutation Boundary

The current `POST /api/v1/sync-status/{user_type}/{user_id}/override` handler only returns `202 Accepted`; it does not mutate an in-memory store, database row, or provider. `handleRoomMappings` similarly returns accepted status for `POST /api/v1/room-mappings` without persisted state.

The planned write boundary is documented in `docs/planning/external-write-inventory.md` under internal sync operations:

- `internal.sync_ingest_subject`
- `internal.sync_update_projection`
- `internal.sync_recheck_subject`
- `internal.archive_completed_sync_rows`
- `internal.clear_sync_exception_overrides`
- `internal.reconcile_subject`

Those are planner operation names or internal projection/archive concepts today, not live provider SDK calls. Future override persistence must document table names, transaction isolation, `internal/db.WithRetry` behavior, audit fields, and annual-reset clearing behavior.

## Tests

Relevant tests:

- `internal/web/sync_dashboard_test.go` covers the HTML routes, JSON tab routes, allowed override route shape, and malformed override routes.
- `internal/core/sync_test.go` covers enum validity, projection behavior, manual-action status, rollover wait behavior, and annual reset clearing overrides.
- `internal/orchestrator/sync_planner_test.go` covers staff sync dry-run, student sync dry-run, sync recheck, and annual reset operation names.
- `internal/web/dev_frontend_test.go` covers `/api/v1/dev/pages/data-quality` session and route behavior.

Run targeted tests with:

```bash
go test ./internal/web -run 'TestSyncDashboard|TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment/(it admin login sets session and can load data quality|unauthenticated data quality is 401|human resources data quality is 403)'
go test ./internal/core ./internal/orchestrator -run 'Sync|AnnualReset'
```

If the local Go toolchain is unavailable, use `make test-container`.

## Debugging Breakpoints

Frontend breakpoints:

- `frontend/src/app.jsx`, where `/data-quality` dispatches to `DataQualityPage`.
- `frontend/src/pages/DataQualityPage.jsx` `fetchDevApiJSON` query setup, `buildDataQualityTextOverrides`, and `DataQualitySemanticContent`.
- Browser Network tab on `/api/v1/dev/pages/data-quality`; the Data Quality page should not expose a mapping-dashboard navigation control unless a future PRD/plan update documents that route.

Backend breakpoints:

- `internal/web/app.go` `handleSyncDashboard`, `handleSyncDashboardMappings`, `handleSyncStatusRoutes`, `handleRoomMappings`, and `writeSyncTabResponse`.
- `internal/web/dev_frontend.go` `handleDevDataQualityPage`.
- `internal/core/sync.go` `ProjectSyncProgress` and `AnnualResetDisposition`.
- `internal/orchestrator/planner.go` `PlanWorkflow`, especially the sync and annual reset workflow cases.

Useful request symptoms:

- `404` on `/api/v1/sync-status/...` usually means the method or path shape does not match the handler's tab or `/{user_type}/{user_id}/override` patterns.
- `202` on an override means the stub accepted the route shape only; it does not prove any state changed.
- `401` / `403` on `/api/v1/dev/pages/data-quality` comes from DEV persona resolution and route visibility.
