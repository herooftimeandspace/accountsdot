# Health And Observability Semantics

This document defines the Phase 0 health and metrics contract used by local
development and staging promotion checks.

## Endpoints

- `GET /health/live` verifies that the HTTP process is alive and can return a
  diagnostic response. It remains `200 OK` during global pause so operators can
  inspect diagnostics while workers are prevented from claiming new work. It
  does not execute database-backed dependency or control checks.
- `GET /health/ready` verifies whether the process should receive normal
  readiness traffic. It returns `503 Service Unavailable` when a configured
  dependency fails or when global pause is active.
- `GET /health` is the legacy readiness alias and uses the same status code and
  JSON payload as `/health/ready`.
- `GET /metrics` emits Prometheus-compatible gauges for liveness, readiness,
  global pause, and each bounded dependency name.

## Readiness Payload

Readiness JSON uses three top-level fields:

- `status`: `ok`, `paused`, or `degraded`.
- `dependencies`: a map with bounded dependency names. Current names are `db`,
  `sequence`, `import_path`, `sftp`, and `google`.
- `controls`: a map for operator controls. Current control name is
  `global_pause`.

Dependency values are intentionally plain diagnostic states:

- `ok` means the callback passed.
- `not_configured` means the current process has no callback for an optional dependency.
  In Phase 0 this is valid for unwired SFTP before integration-mode configuration
  provides a concrete reachability check.
- `missing_required_check` means the current process has no callback for required
  DB, sequence, import-staging storage, or Google service-account readiness. This
  fails readiness closed so missing wiring cannot report false safety.
- `unavailable` means the callback failed and makes readiness fail. The
  response deliberately does not expose raw driver, provider, hostname,
  credential, SQL, or context error text.

Global pause values are:

- `ok` when the control can be read and pause is inactive.
- `paused` when `system_controls.global_pause` is active.
- `not_configured` when no pause callback is wired.
- `unavailable` when the control check fails. Raw database or provider error
  text is not copied into the response.

When only global pause is active, `/health/ready` and `/health` return
`503 Service Unavailable` with `status:"paused"`. When global pause is active and
any dependency also fails, the top-level status is `degraded` so the pause signal
does not hide the dependency outage.

## Metrics

`/metrics` emits these gauges:

- `app_up`: always `1` when the HTTP process can serve the metrics request.
- `app_ready`: `1` only when readiness would return `ok`; `0` when paused or
  degraded.
- `app_global_pause`: `1` when global pause is active; `0` otherwise.
- `app_dependency_ready{name="<dependency>"}`: `1` for `ok` or optional
  `not_configured`; `0` for `missing_required_check` and concrete dependency
  failures.

Metric labels are bounded and non-secret. Do not add provider URLs, tenant names,
credential labels, email addresses, user IDs, raw error text, or service-account
JSON to metric labels.

## Database-Backed Checks

When `DATABASE_URL` is present at startup, `cmd/provisioner` wires read-only
health callbacks:

- DB readiness calls `Ping`.
- Sequence readiness checks usage permission on `global_tick_seq` without
  advancing the sequence.
- Global pause reads `system_controls.enabled` for
  `control_name = 'global_pause'`; a missing row is treated as not paused.

Database-backed checks derive their probe timeout from the incoming HTTP request
context, so client cancellation cancels the DB probe instead of letting a
background health query continue after the request is gone. `/health/live` does
not call these callbacks.

If `DATABASE_URL` cannot be parsed, the server still starts so `/health/live`,
`/health/ready`, and `/metrics` can expose a sanitized readiness failure. The raw
database URL, password, username, host-specific credential material, and service
account data must never appear in health JSON, metrics, logs, issues, or
promotion evidence.

## Verification

For the Phase 0 `P0-0E-001` missing-dependency scenario, use:

```bash
go test ./internal/web -run 'TestHealth(ReadyFailsClosedOnMissingRequiredDependency|ReadyFailsDependency|ReadyAllowsMissingOptionalSFTPCheck|Routes)$'
git diff --check
```

For the Phase 0 `P0-0E-002` pause and observability scenario, use:

```bash
go test ./internal/web ./cmd/provisioner -run 'TestHealth|TestMetrics|TestNewServerReportsInvalidHealthDatabaseConfig|TestHealthProbeErrorsAreBounded'
git diff --check
```

Staging verification should capture the same endpoint and metric semantics from
the deployed staging revision:

1. Confirm `/health/live` remains `200 OK` while global pause is active.
2. Confirm `/health/ready` and `/health` return `503` with `status:"paused"`
   when global pause is active and dependencies are otherwise healthy.
3. Confirm a missing required dependency drill returns `status:"degraded"`,
   `missing_required_check`, `app_ready 0`, and `app_dependency_ready 0` for the
   affected required dependency.
4. Confirm a configured dependency failure drill returns `status:"degraded"`
   and clears the affected `app_dependency_ready` gauge without exposing raw
   provider or driver error text.
5. Confirm `/metrics` includes `app_global_pause 1` while paused.

Store live dev or staging evidence in the external IncidentIQ testing ticket or
promotion runbook. Do not commit transient curl output, screenshots, raw
Prometheus scrapes, database URLs, credentials, tokens, or service-account JSON.
