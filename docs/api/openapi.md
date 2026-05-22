# OpenAPI Contract

Issue #248 uses this directory as the Phase 0 API contract source of truth.

## Source Of Truth

`docs/api/openapi-source.json` is the editable source catalog. It lists every
registered `/api/v1` mux path and every public operation that belongs in the
OpenAPI contract. `scripts/generate_openapi_spec.mjs` reads that catalog and
generates:

- `docs/api/openapi.json`, the checked-in OpenAPI 3.1 document for clients and
  review.
- `internal/web/openapi_spec_gen.go`, the generated Go constant served by
  `GET /api/v1/openapi.json`.

Do not edit generated files by hand. Update `openapi-source.json`, then run:

```bash
npm run openapi:generate
```

Validate drift before review:

```bash
npm run openapi:check
```

`make test` also runs `openapi:check`, so CI fails when a `/api/v1` route is
added, removed, or renamed without updating the catalog and generated spec.

Every non-`GET` operation in `openapi-source.json` must declare
`successStatus`. The generator intentionally refuses to infer write success
from the HTTP method because current handlers include accepted no-op routes
that return `202`, DEV mutation routes that return `200` or `201`, and delete
routes that return `204` with no body. Operations should also list the concrete
`errorStatuses` that the handler can return for invalid JSON, validation
failure, missing resources, state conflicts, oversized payloads, or unavailable
backing services. Do not add a generic `422` unless the handler really emits
that status.

The generated `security` section is derived from `x-wizard-auth` only for
cookie-backed route families. Public discovery, edge-auth-planned
introspection, local breakglass token bootstrap, anonymous DEV session reads,
and development-only login/logout routes intentionally do not advertise a
required `wizard_session` cookie.

## Surface Labels

Each operation carries repo-specific OpenAPI extensions:

- `x-wizard-surface`: `dev-mock`, `accepted-no-op`,
  `db-backed-runtime-planned`, `db-backed-runtime-conditional`, or
  `db-backed-runtime`.
- `x-wizard-phase`: the phase that owns the current behavior.
- `x-wizard-auth`: the current or planned authorization boundary.
- `x-wizard-write-boundary`: read-only, DEV mock mutation, local database
  write, session/audit write, or planned durable write boundary.

These labels are intentionally conservative. DEV mock endpoints are not
production/staging callable APIs. Accepted no-op endpoints acknowledge legacy
operator intent but do not write durable workflow state in this checkout.
Future DB-backed implementations must update this source catalog, add
structured schemas, add route tests for auth/site/feature/field behavior, and
update `docs/planning/external-write-inventory.md` before any new durable write
path lands.

## Database And Write Expectations

The first DB-backed callable surfaces approved for Phase 0 are contract and
introspection oriented: OpenAPI discovery, health/readiness, session/auth
introspection, and planned workflow/approval/status reads. Mutating workflow,
approval, sync override, room mapping, annual reset, onboarding, offboarding,
departing-senior, and room-move operations remain either DEV mock mutations or
accepted no-op/planned write boundaries until a later issue implements the
durable runtime.

When a planned boundary becomes a database write path:

1. Use typed request/response structs in the handler package instead of
   ad hoc JSON maps where practical.
2. Enforce authentication, role/persona, site scope, feature flag, and
   field-level permissions server-side before reading or mutating data.
3. Run serializable transaction work through `internal/db.WithRetry` when the
   operation can encounter serialization or deadlock-class failures.
4. Persist audit rows and deterministic idempotency keys before exposing a
   retryable mutation to frontend or automation clients.
5. Update `docs/planning/external-write-inventory.md` with the exact table,
   idempotency, audit, retry, rollback, staging-validation, and provider
   writeback behavior.
6. Keep provider writeback blocked unless the implementation plan phase,
   what-if validation, live-write pilot allowlist, and explicit approval gates
   are all satisfied.
