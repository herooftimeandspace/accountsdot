# .agents/AGENTS.md

This file applies to the whole repository. It condenses durable repo-specific instructions from prior workspace sessions and the repo markdown so agents can work here without rereading every planning document first.

## Project Identity

- The product is **The WIZARD: Windsor Identity Zync, Access, & Retirement Dashboard**.
- It is a staff-only, self-hosted district operations dashboard for account, identity, room, phone, onboarding, offboarding, sync, and exception workflows.
- The main service is Go with PostgreSQL-backed orchestration, JSON APIs, health checks, metrics, and resilient provider integrations.
- The frontend is React + Vite for the current DEV UI track, with `.pen` files as authoritative design sources for implemented pages.
- The system is mission-critical. Favor safety, auditability, idempotency, recoverability, explicit documentation, and staged rollout over speed.

## Core Source Of Truth

- Read `README.md` first for project goals, documentation policy, local setup, test commands, and environment variables.
- Use `docs/planning/implementation-plan.md` as the authoritative execution plan and implementation decision log.
- Use `docs/product/product-requirements.md` for business-facing product scope, users, workflows, visibility rules, and current-pass boundaries.
- Use `docs/testing/test-matrix.md` for named mock scenarios, verification buckets, and promotion expectations.
- Use `docs/operations/environment-data-playbook.md` for dev/staging/main environment strategy, masking, refresh, and promotion safety.
- Keep these documents aligned when behavior, access scope, provider behavior, data handling, UI affordances, rollout gates, or operator-facing expectations change.
- If a user rejects a design or process recommendation, document the recommendation, override decision, and user-provided reason in the repo docs.

## Implementation Posture

- Work TDD-first for implementation-affecting code. Tests should reflect documented behavior and decision history.
- Do not change tests merely to hide broken behavior. Update tests when the documented intended behavior changes.
- Prefer standard library first. Keep dependencies minimal and respect the repo's dependency allowlist tests.
- Surface any new dependency that materially reduces risk or complexity for explicit approval before adding it.
- The primary application codebase must remain Go. Use narrowly scoped Python only when it clearly improves correctness, clarity, or implementation speed for a specific task.
- Do not duplicate tests already covered by provider SDKs. Test repo-owned normalization, orchestration, payload contracts, safety behavior, UI routing, and reporting.
- Preserve source-system truth in displayed imported data. Operator labels and controls stay English-only, but imported source values should be shown exactly as stored.
- Use inclusive, precise terminology such as `allowlist`, `denylist`, `deactivated`, and `deprovisioned`.

## Repo Map

- `cmd/provisioner`: application entrypoint.
- `cmd/symphony`: Symphony CLI, daemon, status, and orchestration entrypoints.
- `internal/core`: workflow, sync, and domain types.
- `internal/db`: schema and database retry behavior.
- `internal/provider`: provider contracts and provider-specific serialization.
- `internal/orchestrator`: workflow and sync planning.
- `internal/symphony`: Symphony queueing, work graph, source corpus, GitHub, daemon, and TUI logic.
- `internal/web`: HTTP handlers, health, DEV frontend API, and server-rendered legacy surfaces.
- `frontend/src`: React DEV frontend.
- `frontend/src/generated`: generated `.pen`-derived artboard files. Do not hand-edit these.
- `docs/design/mocks/wireframes`: authoritative `.pen` wireframes and explicit review exports.
- `docs/agent-orchestration`: Symphony service contract and runner rules.
- `scripts/symphony_runner.mjs`: legacy Node adapter for remaining Symphony side-effect paths.
- `scripts`: frontend design sync, accessibility checks, and mock/wireframe tooling.

## Data And Environment Safety

- Production is never the first place a new write path is tested.
- Treat `dev`, `staging`, and `main` as separate long-lived datasets.
- `dev` uses mocks by default and should be freely breakable.
- `staging` is the required proving ground before production promotion.
- Prefer real sandbox tenants in staging. If no sandbox exists, use masked production-derived data and disable unsafe writes.
- Aeries has no dedicated sandbox/staging tenant. Non-production realism should use read-only masked previous-year data, defaulting to `current school year - 1`.
- Never copy provider credentials, tokens, auth headers, private keys, passwords, client secrets, or raw service-account JSON into non-production datasets, logs, tickets, generated assets, mock fixtures, or reference corpora.
- Persist only fields needed for workflows, audits, and UI. Omit or encrypt sensitive subfields in raw payloads, request/response summaries, audit diffs, backups, and diagnostics.
- Local Compose secrets belong in `.env`, not committed files. Local Postgres should bind only to `127.0.0.1`.

## Security Expectations

- Treat all inbound data as untrusted, including CSV, SFTP payloads, API responses, SAML claims, and operator-entered form data.
- Protect FERPA, COPPA, GDPR-sensitive, personnel, and district-cautioned data conservatively.
- Use field-level encryption or omission for sensitive fields, especially manual intake fields and raw provider payload fragments.
- Diagnostics should use non-secret credential labels, service-account ownership, key ids, token metadata that is not sensitive, and one-way fingerprints.
- Re-run security review after major frontend changes, especially for XSS, unsafe HTML injection, storage of persona/session data, auth bypasses, and mock-only controls leaking outside development.
- Run `npm audit --omit=dev` and full `npm audit` after dependency changes.
- Run Go tests and `govulncheck` in a Go-capable environment or container before promotion.

## Architecture Rules

- FSM transitions, extension mutations, and room-scoped mutations must use `SERIALIZABLE` transactions.
- Strict ordering must use sequence-backed `global_tick`, never UUID sort order.
- Database retry behavior belongs in `internal/db.WithRetry(ctx, pool, fn)` with jittered exponential backoff for serialization and deadlock-class failures.
- Provider writes must be idempotent and backed by deterministic idempotency keys plus `external_request_log`.
- Resource allocation is two-phase where applicable, such as `available → reserved_for_job_id → assigned_to_person_id`.
- Long-running sync jobs may finish even if the next cadence window arrives. Prevent conflicting overlapping runs for the same provider or job family.
- IncidentIQ tickets are the standard fallback for work automation cannot complete directly. Ticket descriptions must be complete enough for the assignee to act without reopening the parent workflow.

## Access And Product Scope

- The dashboard is staff-only and must explicitly deny student access.
- Domain gate before role authorization: allow `@wusd.org`, `@it.wusd.org`, `@staff.wusd.org`; deny `@stu.wusd.org`; local breakglass accounts are exempt.
- Unauthorized users hitting the same URL should receive access-denied responses rather than soft-reduced content.
- Keep `Student Data Cleanup` as the user-facing page, route, and sidebar label. Do not resurrect `Invalid Student Names` as operator-facing copy.
- Localization and translation are permanently out of scope. Operator-facing labels, workflow text, controls, and configurable UI text remain English-only.
- Fixed defensive fallback text such as `Unknown` should remain English and render as muted/system placeholder styling unless explicitly documented otherwise.

## Frontend And Design

- For implemented pages, `.pen` files own geometry, spacing, typography, text blocks, and static shell/page layout.
- React runtime owns documented interaction, routing, data loading, access rules, and behavior.
- Before UI/design work, read the implemented-page UI sections of `docs/planning/implementation-plan.md` and `docs/product/product-requirements.md`, then use `docs/design/mocks/wireframes/implemented-page-design-contract.md` as the compact working contract.
- Classify every UI issue before editing as exactly one of `pipeline`, `.pen layout`, `docs/new behavior`, `runtime behavior`, or `review artifact`.
- Do not add new shell/page behavior until `docs/product/product-requirements.md` and `docs/planning/implementation-plan.md` define it.
- For implemented pages, update the `.pen` source first, then run `npm run pen:sync`. Do not hand-edit generated artboards, generated presentational components, or generated review artifacts.
- For annotation-driven work, freeze Codex annotate feedback into the relevant checked-in ledger before implementation. Do not start from loose annotation memory.
- Export SVG/PNG review artifacts only when human review, signoff, archival comparison, or an explicit request needs them.
- Shared shell issues must be fixed at the shared pattern level, not one page at a time.
- Standard controls such as the shared-header `Refresh` action must be treated as reusable primitives with shared geometry and styling, not page-local copies.
- Live implemented pages must not surface shortcut pills, governance labels, validation-process text, mock-policy labels, or other shell adornments unless they are documented operator-facing product behavior.
- When a card, rail, helper block, or table cell expresses one logical paragraph, use one wrapping text node in the authoritative `.pen`; do not split it into stacked fragments unless a documented semantic reason requires separate nodes.
- Preserve table layout rules: shared top baseline across row cells, rows grow downward to the tallest cell, and text must not collide with dividers.
- Preserve at least a `5px` visual gap between row text and horizontal dividers unless a documented shared-border join is intentional.
- Preserve at least a `5px` buffer between bordered wrapper elements unless the design contract documents an intentional shared-border join.
- Generated dashboard assets must keep borders, icons, text, badges, and controls from visually colliding or overflowing.
- Use Pencil local-app or interactive workflows when a task requires Pencil-authored `.pen` work. Validate one pilot screen before batch exporting a wireframe set.
- After relevant UI/design changes, run `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check` unless a narrower check is justified and stated.

## Visual System

- Follow district brand guidance. The UI should feel district-specific rather than generic admin software.
- Use Atkinson Hyperlegible for primary UI/body typography.
- Varsity and Reset fonts are display/accent fonts only and must not compromise readability or WCAG compliance.
- Confirmed core palette: text `#01161E`, background `#FFFFFF`, Vegas Gold highlight `#CEB770`, neutral canvas/border `#DDE2E3`.
- Confirmed accents include `#FCD9E9`, `#D73533`, `#FE5E41`, `#00A878`, `#A6B0B5`, `#CA8BB9`, `#6E7E85`, `#774936`, `#89B6E7`, `#5797DB`, `#2D7ED2`, `#194676`, and `#0E2843`.
- Enforce WCAG contrast. Use black text on Vegas Gold, canvas, pink, coral, green, gray, mauve, light blue, and mid blue. Use white text on red, brown, navy, and deep navy.
- Use gradients only for decorative accent bands, chart fills, or non-text backgrounds unless every text overlay passes contrast.
- Prefer official PNG assets for web/UI use. Do not invent new logo treatments when official marks exist.

## Testing And Verification

- Common test commands:
  - `make test-unit`
  - `make test-contract`
  - `make test-integration`
  - `make test`
  - `make test-container`
  - `make vulncheck`
  - `make vulncheck-container`
  - `make security`
  - `make security-container`
  - `npm run pen:check`
  - `npm run build:web`
  - `npm run a11y:check`
- Use `make test-container` or `make security-container` when the host Go toolchain is missing or unhealthy.
- UI-heavy workflow buckets need runtime evidence, including at least one UI artifact by default.
- For frontend changes, verify persona routing, access denial, keyboard focus, mobile/reflow readability, and WCAG behavior.
- Named mock scenarios in `docs/testing/test-matrix.md` must stay synchronized with `docs/planning/implementation-plan.md`.
- Live execution tracking and promotion signoff live outside the repo in IncidentIQ, not as rolling status fields in `docs/testing/test-matrix.md`.

## Documentation And Communication Outputs

- Executive or site-admin-facing material should be concise, practical, and focused on day-to-day operational improvement rather than technical depth.
- Sales or leadership summaries should explain what becomes easier, what blockers become visible, and how spreadsheet/email side channels are reduced.
- Technical docs should capture business decisions where they materially affect implementation, operator behavior, access scope, rollout gates, or user-visible outcomes.
- Do not add speculative feature controls, queues, dashboard affordances, or mock states that are not cleanly supported by the PRD and implementation plan.
- Keep inline code documentation current with implemented behavior. When code surfaces change, update the function comments, caller comments, debugging docs, and external-write inventory in the same branch.
- Prune stale documentation aggressively. Remove or rewrite comments and guides that describe deleted functions, renamed routes, obsolete workflow behavior, superseded provider operations, or old debugging paths.
- Any live, planned, mock-only, or database write path must stay documented in `docs/planning/external-write-inventory.md`, including Zoom, IncidentIQ, Google, local database, and DEV mock store mutations.

## Custom Skill Extraction Guidance

- The repo-local Codex UI hardening skill lives at `.agents/skills/wizard-ui-hardening/SKILL.md`.
- The repo-local code documentation skill lives at `.agents/skills/wizard-code-documentation/SKILL.md`.
- Use the code documentation skill for implemented code changes under `cmd/`, `internal/`, `frontend/src/`, tests, route handlers, provider-operation planning, database behavior, and external-write surfaces.
- Use the code documentation skill for Symphony queue invariants, workspace recovery policy, review-thread gating, OpenAPI/runtime contract work, readiness surfacing, and promotion-validator coverage checks.
- Keep repo skill bodies lean. Point to repo docs as references instead of copying large sections into `SKILL.md`.
- Recommended UI trigger: use the UI hardening skill for work on The WIZARD, accountsdot, district account lifecycle, `.pen`-derived dashboard pages, provider sync/orchestration, or dev/staging/main promotion safety when the task is frontend/UI/design oriented.
- Recommended code documentation trigger: use the code documentation skill whenever implemented code changes, documentation may have gone stale, or an external-write path is introduced, renamed, removed, mocked, or made live.
- Put long reference detail into skill `references/` files only when it is needed outside the repo or cannot be reliably discovered from the checked-out docs.
- Preserve the source-of-truth hierarchy: `docs/planning/implementation-plan.md` for implementation decisions, `docs/product/product-requirements.md` for product scope, `docs/testing/test-matrix.md` for scenario coverage, `docs/operations/environment-data-playbook.md` for environment safety, and `README.md` for project overview and commands.
