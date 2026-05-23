# The WIZARD: Windsor Identity Zync, Access, & Retirement Dashboard

The WIZARD: Windsor Identity Zync, Access, & Retirement Dashboard is a self-hosted, mission-critical employee provisioning service designed around PostgreSQL-backed orchestration, resilient provider integrations, and real-time operational visibility.

## Project Goals
1. Integrate and reconcile data from at least 8 separate source systems.
2. Provide reliable workflow-driven provisioning, update, transfer, leave, and deprovisioning operations.
3. Support multiple user types with different access levels, visibility rules, and technical skill levels.
4. Replace spreadsheet-driven onboarding, offboarding, room-move, and phone-directory workflows with a staff-facing dashboard.
5. Surface bad or incomplete data as early as possible to the non-IT parties who can correct it upstream.
6. Automate as much routine IT work as possible while keeping exceptions visible and recoverable.
7. Keep policy decisions and business mappings configurable through the dashboard so IT does not need to change code for ordinary policy updates.
8. Preserve operational safety through auditability, approvals, idempotency, recoverable background workflows, and non-production test environments.
9. Keep documentation as a first-class part of delivery so product goals, implementation constraints, business-rule decisions, and environment-safety procedures remain explicit.
10. Follow DRY design principles and prefer shared canonical records over duplicating the same business data in multiple places.
11. Use inclusive, precise terminology in code and UI. Prefer terms like `allowlist`, `denylist`, `deactivated`, and `deprovisioned`; avoid legacy terms like `whitelist`, `blacklist`, or vague lifecycle wording when a more specific state exists.
12. Keep the product English-only. Localization and translation are permanently out of scope and should not be introduced into the UI, workflow text, or operator-facing configuration surface.
13. Preserve source-system truth in displayed data. Operator-facing labels and controls stay in English, but imported source values should be shown exactly as stored rather than translated or normalized for language.

## Documentation Policy
- [docs/planning/implementation-plan.md](docs/planning/implementation-plan.md) is the authoritative execution plan and decision log for implementation-affecting behavior.
- [docs/product/product-requirements.md](docs/product/product-requirements.md) captures the business-facing product requirements and scope boundaries.
- [docs/operations/environment-data-playbook.md](docs/operations/environment-data-playbook.md) defines the safe process for creating and refreshing mock and staging environments.
- [docs/operations/breakglass-access.md](docs/operations/breakglass-access.md) defines the local non-production emergency access setup, CIDR restrictions, audit expectations, and staging verification notes.
- [docs/operations/health-observability.md](docs/operations/health-observability.md) defines the Phase 0 `/health/live`, `/health/ready`, `/health`, and `/metrics` semantics for dependency, missing-required-check, global-pause, and staging observability evidence.
- [docs/operations/reference-input-snapshot-integrity.md](docs/operations/reference-input-snapshot-integrity.md) defines the Phase 0 startup guard for required repo-local reference inputs and the staging evidence expected for snapshot integrity.
- [docs/operations/provider-readiness.md](docs/operations/provider-readiness.md) defines the Phase 0 provider readiness mock success path, failure diagnostics, staging read-only probe expectations, and credential-redaction rules for readiness evidence.
- [docs/testing/test-matrix.md](docs/testing/test-matrix.md) tracks the named mock scenarios and verification coverage that must stay aligned with the implementation plan during phased delivery. It is a static definition artifact, not a live execution-status tracker; live test tracking and signoff belong in an external IncidentIQ testing ticket.
- [docs/product/permissions-matrix.md](docs/product/permissions-matrix.md) documents the currently implemented DEV route/API permission matrix, field-level visibility, and known authorization gaps for review against the PRD and implementation plan.
- [docs/api/openapi.md](docs/api/openapi.md) documents the generated OpenAPI source of truth, regeneration command, drift check, and surface labels that distinguish DEV mock routes from accepted no-op, planned DB-backed, and callable runtime APIs.
- [docs/operations/promotion-pipeline.md](docs/operations/promotion-pipeline.md) defines the checked-in GitHub Actions branch gates, automated promotion PR behavior, local branch-gate commands, and manual repository settings required for `dev → staging → main` promotion.
- [docs/agent-orchestration/SPEC.md](docs/agent-orchestration/SPEC.md) defines the repo-local Symphony-style Codex orchestration contract for GitHub issue driven agent work. [.agents/WORKFLOW.md](.agents/WORKFLOW.md) is the runner-readable prompt and configuration contract for that workflow. The Go CLI at `cmd/symphony` owns `sync` work-graph planning and `test` orchestration, while `scripts/symphony_runner.mjs` remains the direct entrypoint for Node-backed `report` and `ui-monitor` paths during migration; use `npm run symphony:report`, `npm run symphony:sync -- --dry-run --json`, `npm run symphony:ui-monitor -- --dry-run`, and `npm run symphony:test` for local queue, dispatcher, and monitor validation.
- [docs/reference-inputs/VENDORED_INVENTORY.md](docs/reference-inputs/VENDORED_INVENTORY.md) is the authoritative provenance and refresh ledger for the repo-local reference corpus under `docs/reference-inputs/`.
- The promotion runbook/process also lives outside the repo. It must capture one implementation-signoff entry per workflow bucket, reference the external IncidentIQ testing-ticket evidence, and use the corresponding documented rollback path and rollback trigger conditions for write-capable buckets. Each bucket entry must include exact metadata for release, ticket, phase, bucket, environment, revisions, timestamps, signoff, evidence links, and rollback references as applicable. Each bucket entry must also include a final disposition plus explicit yes/no attestations for scenario cleanliness, evidence review, and write-safety checks. A bucket that was previously `rolled back` may later be updated to `ready` after a new clean pass, and `superseded` means an older attempt was replaced by a newer one in the same release; the replacement current entry must explicitly link back to the superseded one. A `no` attestation does not require a separate explanation field beyond the disposition and any required closure note. If a rollback trigger blocks a bucket in `dev`, the runbook must carry an explicit closure note with links to the replacement evidence before `staging` can begin for that bucket. The external IncidentIQ testing ticket should use one parent ticket per release with evidence organized in `phase → bucket → dev/staging → scenario` order, with `dev` listed before `staging` in every bucket.
- This README must continue to enumerate the product goals at a high level.
- Code should document business decisions where those decisions materially affect implementation, operator behavior, or user-visible outcomes.

## LLM Usage Disclaimer
This repository is an LLM-driven project and was written with heavy use of LLMs. This disclaimer is required project policy and must remain present in the repository.

## Local Development
Local testing is supported through either `docker compose` or the VS Code Dev Containers extension.

### Docker Compose
1. Copy `.env.example` to `.env` and adjust values if needed.
2. Fill the local-only secrets in `.env`; do not commit `.env`.
   ```bash
   openssl rand -hex 32
   ```
   Use one generated value for `POSTGRES_PASSWORD`, set `SESSION_SECRET` to a different generated value, set `ENCRYPTION_KEY` to a key-id plus generated value such as `k1:<generated-value>`, and set `DATABASE_URL` to the local Compose database URL using the same Postgres password. `sslmode=disable` is only acceptable for the local Compose network.
3. Start the local stack:
   ```bash
   make up
   ```
4. Run tests inside the app container:
   ```bash
   make test
   ```
5. Stop the stack:
   ```bash
   make down
   ```

### VS Code Dev Containers
1. Install the Dev Containers extension.
2. Open this folder in VS Code.
3. Run `Dev Containers: Reopen in Container`.
4. Inside the container, run:
   ```bash
   make test
   ```

## Test Commands
- Branch-gate mirrors:
  - `python3 scripts/run_local_ci.py --target dev`
  - `python3 scripts/run_local_ci.py --target staging`
  - `python3 scripts/run_local_ci.py --target main`
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
- `npm run pen:lint`
- `npm run build:web`
- `npm run a11y:check`
- `npm run openapi:generate`
- `npm run openapi:check`
- `npm run perf:routes:plan`
- `npm run perf:routes:batch-plan -- [artifact-input-dir]`
- `npm run perf:routes:merge -- [artifact-input-dir]`
- `npm run perf:routes:merge:strict -- [artifact-input-dir]`
- `npm run perf:routes:self-test`

`make vulncheck` uses a local `govulncheck` binary when available, otherwise it runs `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`. If the host does not have Go installed, it falls back to `make vulncheck-container`, which runs the same scan inside the repo's configured Go Docker image.

Use `make test-container` or `make security-container` when the host Go toolchain is missing or unhealthy. These targets bootstrap through the configured `golang` Docker image and do not require Go to be installed on the host.

Inside the devcontainer, `govulncheck` is installed during `postCreateCommand`.

## Generated Artifact Policy
- `frontend/dist/` is local build output from `npm run build:web`. Do not commit it. A release or deployment build must generate production frontend assets from the approved repository revision during that release process, then package or publish the freshly generated `frontend/dist/` contents through the deployment mechanism for that environment. Do not promote a developer workstation's existing `frontend/dist/` directory as release evidence or a production artifact.
- New raw DEV route performance outputs under `artifacts/performance/` are local by default and are ignored by Git. The currently tracked files in that directory are retained only as historical handoff evidence for the 2026-05-12 DEV performance investigation. Future JSON and Markdown harness output should be copied or linked into the external IncidentIQ testing evidence or promotion runbook when it supports a release decision. Commit new performance artifacts only when a specific issue or PR explicitly requires a small curated evidence file in the repository.
- The repository remains the source of truth for harness code, scenario definitions, command documentation, and artifact policy. Live evidence tracking and retention remain outside the repo in the external IncidentIQ testing ticket and promotion runbook described above.

### DEV Route Performance Harness
Use the DEV route performance harness when route transitions, reload behavior, or Browser-pipe stability needs runtime evidence. Start the Go API and Vite frontend first:

```bash
APP_ENV=development npm run dev:api
npm run dev:web
```

`APP_ENV=development` is required for the DEV-only frontend APIs. The application may otherwise log development-mode configuration defaults while `/api/v1/dev/session` still returns `404`, because the DEV frontend route guard reads `APP_ENV` directly. Before opening Browser or running the route matrix, verify the Vite proxy can reach the DEV session endpoint:

```bash
curl -i http://localhost:5173/api/v1/dev/session
```

If Vite is not running yet, check the API directly on the default Go port instead:

```bash
curl -i http://localhost:8080/api/v1/dev/session
```

The preflight should return `200 OK` with DEV session JSON from either URL. An unauthenticated but correctly started DEV API includes fields such as `"environment":"development"`, `"authenticated":false`, `"authorized":false`, and a non-empty `"personas"` array. A `404` from the direct API URL is a startup/configuration failure; restart the API with `APP_ENV=development` before collecting Browser evidence. A `200 OK` from the direct API URL but a failed Vite URL means the route matrix is still blocked by frontend proxy startup or port wiring, not by Browser or page readiness. A passing preflight followed by lost automation connection, missing `iab` tab access, or interrupted pipe output is a Browser transport failure. A passing preflight with an app-rendered error, timeout after navigation, or missing route content is a page readiness failure for the route being measured.

### DEV Persona Switching From Terminal Tooling
Codex and other terminal-only evidence workflows can switch the active mock persona without clicking the in-browser DEV persona switcher. Start the API with `APP_ENV=development`, keep Vite running when you want the same target the Browser uses, then run:

```bash
npm run dev:persona -- site_admin
```

The helper posts to `/api/v1/dev/login` with `activate_mock_session=true`. That flag is development-only and records a process-local active mock session before returning the same structured session payload the frontend reads from `/api/v1/dev/session`. The JSON output includes `authenticated`, `authorized`, `current_persona`, `landing_path`, `allowed_routes`, default/current site fields, visible sites, shell context, and feature flag availability. Refreshing or navigating the Browser tab after the command makes the frontend consume the selected persona even when an older Browser cookie still exists.

Use the Vite URL for normal Browser evidence:

```bash
npm run dev:persona -- device_wrangler --base-url http://localhost:5173
curl -s http://localhost:5173/api/v1/dev/session
```

Use the API URL when Vite is not running:

```bash
npm run dev:persona -- no_access --base-url http://localhost:8080
curl -s http://localhost:8080/api/v1/dev/session
```

Supported persona ids are the ids returned by `/api/v1/dev/session`, including `it_admin`, `human_resources`, `site_admin`, `site_secretary`, `device_wrangler`, `faculty_staff`, and `no_access`. Site-scoped personas keep their documented default/current site and visible-site context. Invalid persona ids return `400` with `code:"invalid_persona"`, clear the local DEV cookie in the response, and force the shared mock session to anonymous so stale Browser cookies do not silently restore an authorized persona. Missing or non-development `APP_ENV` returns `404`; this tooling is not production authorization and must not be enabled outside local development.

`npm run perf:routes:plan` prints the current route set, directed-transition coverage count, default batch sizes, readiness metadata, and the first transitions without opening a browser. Route variants are content-sensitive by default: `/search?q=alex` must render the expected result text because the query changes the page body. Static generated-page variants may opt in to URL/title readiness only when their variant entry is explicitly annotated with `allowTitleAndUrlReadiness`; the room-move draft routes use this exception because their mock draft body text is not a durable readiness contract. Do not make all variants URL/title-ready, because that would hide regressions on routes where the variant-specific body content is the signal being tested.

`npm run perf:routes:batch-plan -- artifacts/performance` scans local artifacts that match the current route plan and reports the next transition or refresh batch without opening a browser.

`npm run perf:routes:self-test` builds temporary synthetic route-performance artifacts and verifies the local-only batch planner and strict merge gate without opening a browser. It checks that completed transition indexes resume at the next gap, Browser-pipe failures are not counted as completed resume points, a full clean synthetic matrix passes the strict gate, and app-timeout or duplicate-index artifacts fail the strict gate. This command is a harness sanity check, not Browser evidence for issue or PR closure; use it before or after Browser collection when changing the batch runner, merge gate, route plan metadata, or README workflow.

The full measurement run uses the Codex Browser skill because `scripts/dev_route_performance_matrix.mjs` needs the active Browser tab object. Prefer the automatic batch helper for full-matrix evidence so Browser work stays inside bounded calls and the helper resumes from existing local artifacts without manual index bookkeeping:

```js
const { runDevRoutePerformanceBatches } = await import("./scripts/dev_route_performance_matrix.mjs");
await runDevRoutePerformanceBatches({
  tab,
  baseUrl: "http://localhost:5173",
  outputDir: "artifacts/performance",
  transitionBatchSize: 50,
  refreshBatchSize: 12
});
```

By default `runDevRoutePerformanceBatches` executes one bounded Browser batch per call. Re-run the same snippet until it returns `"complete": true`; it measures all directed transitions first, then measures refresh samples across the same route targets. The default batch sizes are 50 directed transitions or 12 route-refresh samples per Browser call. The helper also calls `/api/v1/dev/session` through the configured `baseUrl` before it touches the Browser tab. If that DEV startup guard fails, the helper writes a blocked local artifact with `devServerHealthy: false` and `stopReason: "dev_server_unhealthy"` so strict merge can reject the evidence without confusing startup misconfiguration with page readiness. If a local Browser session is stable and the tool response window allows it, pass `maxBatches` to run more than one bounded batch in the same call.

The harness writes JSON and Markdown summaries to `artifacts/performance/` after every measured row so partial results survive a Browser pipe interruption. Each measured row includes additive performance-budget fields: `budgetStatus`, `budgetWarningMs`, `budgetFailureMs`, and `budgetExceededByMs`. The default transition and refresh budgets warn when readiness time is over `3000 ms` and fail when readiness time is over `7000 ms`; readiness failures still use the existing `status`, `failureClass`, and Browser/app failure sections so slow-but-ready rows are not confused with pages that never became ready.

Override budgets for local investigations by setting environment variables before running the Browser helper or merge command:

```bash
ROUTE_PERF_TRANSITION_WARNING_MS=2500 ROUTE_PERF_TRANSITION_FAILURE_MS=6500 npm run perf:routes:merge -- artifacts/performance
ROUTE_PERF_REFRESH_WARNING_MS=3500 ROUTE_PERF_REFRESH_FAILURE_MS=8000 npm run perf:routes:merge -- artifacts/performance
```

The merge command also accepts one-off threshold flags when reclassifying historical artifacts:

```bash
npm run perf:routes:merge -- artifacts/performance --transition-warning-ms 2500 --transition-failure-ms 6500 --refresh-warning-ms 3500 --refresh-failure-ms 8000
```

Use `--budget-strict` only when the current task explicitly needs a budget-only quality gate. It exits nonzero when budget failure rows exist, but it does not replace the separate strict merge gate work for route coverage, duplicate rows, Browser transport failures, or app readiness failures.

If the Browser pipe fails, restart the Browser automation session and run the same automatic batch helper again. For manual recovery or targeted investigation, the lower-level runner still accepts the reported `resumeFromTransitionIndex` or `nextTransitionIndex`:

```js
const { runDevRoutePerformanceMatrix } = await import("./scripts/dev_route_performance_matrix.mjs");
await runDevRoutePerformanceMatrix({
  tab,
  baseUrl: "http://localhost:5173",
  startTransitionIndex: 372,
  maxTransitions: 50,
  includeRefreshes: false
});
```

Each transition or refresh row keeps the historical `elapsedMs` field and also includes a backward-compatible `phaseTimings` object. The required phase fields are `navigationLoadMs` for the Browser navigation or refresh load-state wait and `readinessPollingMs` for the app-readiness polling loop. Refresh rows may also include `setupNavigationLoadMs` for the initial route open before reload. When the DEV frontend emits sanitized performance markers, rows can include `frontendSessionFetchMs`, `frontendGeneratedArtboardImportMs`, generated-artboard render mark counts, and route-render commit mark counts. Merged JSON artifacts also include `phaseTimingSummary`, which groups those additive fields by phase and records sample counts, min/median/max durations, and the slowest rows for downstream profiling work. These fields are limited to durations, paths, route titles, artboard keys, and non-secret labels; do not add session payloads, provider data, credentials, or raw Browser traces to committed artifacts.

If the Browser skill cannot provide an active in-app browser target, such as when `agent.browsers.get("iab")` reports that `iab` is unavailable and `agent.browsers.list()` returns an empty list, the matrix cannot produce current Browser evidence. Keep any generated blocked or merged summaries local under `artifacts/performance/`, cite their paths in the GitHub issue or PR, and do not close the route-performance issue from historical artifacts alone. Historical merged artifacts are only useful as handoff context when their recorded route count and directed-transition count match the current `npm run perf:routes:plan` output.

After collecting multiple partial runs, merge them with:

```bash
npm run perf:routes:merge -- artifacts/performance
```

Use the non-strict merge command for historical handoff context and interrupted-run diagnosis. For issue or PR closure evidence, use the strict quality gate:

```bash
npm run perf:routes:merge:strict -- artifacts/performance
```

Strict merge still writes the merged Markdown and JSON artifacts, but exits nonzero when the merged evidence is not release-quality. Blocking conditions include transition failures, refresh failures, Browser transport failures, app timeout rows, stale route-plan coverage, missing or duplicate transition indexes, missing, duplicate, extra, or invalid refresh samples, invalid directed-edge coverage, current route-count or directed-transition-count mismatches, and any artifact set that explicitly records `devServerHealthy: false` from the DEV session preflight. The failure message names the blocking counts and points to the merged artifact paths so the Markdown summary can be attached or copied into external evidence.

The merged Markdown file is the human-readable summary to copy into external evidence. The merged JSON file is for debugging and reproducibility; keep it local unless a PR explicitly asks for a curated repository artifact.

## Environment Variables
Required or commonly used local variables:

- `APP_ENV`
- `ENVIRONMENT_ROLE`
- `ENVIRONMENT_DATA_MODE`
- `ENVIRONMENT_DATA_SOURCE`
- `APP_PORT`
- `DATABASE_URL`
- `SESSION_SECRET`
- `ENCRYPTION_KEY`
- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REDIRECT_URL`
- `GOOGLE_ALLOWED_GROUPS_CONFIG`
- `AUTH_ALLOWED_EMAIL_DOMAINS`
- `AUTH_DENIED_EMAIL_DOMAINS`
- `GOOGLE_SAML_ENTITY_ID`
- `GOOGLE_SAML_ACS_URL`
- `GOOGLE_SAML_IDP_METADATA_URL`
- `GOOGLE_SAML_IDP_SSO_URL`
- `GOOGLE_SAML_IDP_CERT_FILE`
- `GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON`
- `GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON`
- `GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON`
- `GOOGLE_SHEETS_SPREADSHEET_ID`
- `GOOGLE_SERVICE_ACCOUNT_JSON`
- `ZOOM_ACCOUNT_ID`
- `ZOOM_CLIENT_ID`
- `ZOOM_CLIENT_SECRET`
- `ZOOM_BASE_URL`
- `ZOOM_OAUTH_URL`
- `AERIES_BASE_URL`
- `AERIES_CLIENT_ID`
- `AERIES_CLIENT_SECRET`
- `SFTP_HOST`
- `SFTP_PORT`
- `SFTP_USERNAME`
- `SFTP_PRIVATE_KEY`
- `SFTP_REMOTE_PATH`
- `USE_MOCK_ZOOM`
- `USE_MOCK_GOOGLE`
- `USE_MOCK_AERIES`
- `USE_MOCK_SFTP`
- `ZOOM_SLG_MAX_MEMBERS`

## Local Testing Notes
- Local mode defaults all `USE_MOCK_*` flags to `true`.
- Real third-party integrations are opt-in and should remain disabled for normal local TDD work.
- The local stack is intentionally lean: app, worker, and postgres are enough for baseline development.
- Compose binds Postgres to `127.0.0.1` only and reads secrets from `.env`, not from committed example values.

## Manual Remote Deployment
The remote deployment stack is for a Proxmox QEMU VM that has Docker Engine,
the Docker Compose plugin, Git, and outbound access to GitHub and public
container registries. It is separate from the local-development `compose.yaml`
so local testing remains app/worker/Postgres focused.

The checked-in deployment files are:

- `docker-compose.deploy.yml`: reverse proxy, three app containers, and
  environment-specific Postgres services.
- `deploy/Caddyfile`: host-based Caddy routing for dev, staging, and main.
- `deploy/Dockerfile`: production-style Go service image for each branch. The
  image copies `docs/reference-inputs/` into `/app/docs/reference-inputs/` and
  sets `WIZARD_REFERENCE_INPUT_ROOT=/app` so the Phase 0 reference-input guard
  can run from the packaged container instead of depending on a checkout.
- `deploy/remote-redeploy.sh`: manual update helper that fetches Git branches,
  refreshes branch worktrees, then runs Docker Compose.
- `deploy/env/*.example`: templates for uncommitted remote environment files.

The stack expects three Git branches to exist on the remote named `dev`,
`staging`, and `main`. The deployment helper creates or refreshes sibling
worktrees under `../accountsdot-deploy-worktrees` by default:

```text
accountsdot/
accountsdot-deploy-worktrees/
  dev/
  staging/
  main/
```

Each app image builds from its matching branch worktree. A redeploy therefore
pulls the latest branch revisions before rebuilding and recreating containers.
The helper keeps those worktrees on detached checkouts of the matching remote
branches; do not make hand edits inside `accountsdot-deploy-worktrees`.

### First-Time Remote Setup
1. Clone the repository on the QEMU VM:
   ```bash
   git clone git@github.com:herooftimeandspace/accountsdot.git
   cd accountsdot
   ```
2. Confirm the deployment branches exist:
   ```bash
   git ls-remote --heads origin dev staging main
   ```
   If `staging` has not been created yet, create it through the documented
   promotion process before deploying the full three-environment stack.
3. Copy the example environment files and fill real remote values:
   ```bash
   cp deploy/env/proxy.env.example deploy/env/proxy.env
   cp deploy/env/dev.db.env.example deploy/env/dev.db.env
   cp deploy/env/dev.app.env.example deploy/env/dev.app.env
   cp deploy/env/staging.db.env.example deploy/env/staging.db.env
   cp deploy/env/staging.app.env.example deploy/env/staging.app.env
   cp deploy/env/main.db.env.example deploy/env/main.db.env
   cp deploy/env/main.app.env.example deploy/env/main.app.env
   ```
4. Replace every placeholder secret with generated values. Use distinct
   database passwords, `SESSION_SECRET` values, and `ENCRYPTION_KEY` values for
   dev, staging, and main. Do not reuse staging credentials in main.
5. Set `ACCOUNTSDOT_DEV_HOST`, `ACCOUNTSDOT_STAGING_HOST`, and
   `ACCOUNTSDOT_MAIN_HOST` in `deploy/env/proxy.env` to the public hostnames
   that should terminate TLS at Caddy.
6. Start the stack:
   ```bash
   chmod +x deploy/remote-redeploy.sh
   ./deploy/remote-redeploy.sh
   ```

### Redeploying
Run the same helper from the root clone whenever one or more deployment branches
has new code:

```bash
./deploy/remote-redeploy.sh
```

The helper runs `git fetch`, updates the `dev`, `staging`, and `main`
deployment worktrees to `origin/dev`, `origin/staging`, and `origin/main`, then
runs:

```bash
docker compose -f docker-compose.deploy.yml up -d --build --remove-orphans
```

Named Docker volumes keep database data across redeploys. Do not use
`docker compose down -v` on this stack unless you intend to destroy the
databases.

### Environment Responsibilities
- Dev runs with `ENVIRONMENT_ROLE=dev`, `APP_ENV=development`,
  `ENVIRONMENT_DATA_MODE=mock`, mock providers enabled, and the long-lived
  `postgres-dev-data` volume. Dev examples must not point at staging or main
  databases and must not assume production-only data exists.
- Staging runs with `ENVIRONMENT_ROLE=staging`, `APP_ENV=staging`,
  `ENVIRONMENT_DATA_MODE=masked-read-only`,
  `ENVIRONMENT_DATA_SOURCE=masked-production-derived`, the long-lived
  `postgres-staging-data` volume, and sandbox or masked/read-only provider
  configuration as defined in `docs/operations/environment-data-playbook.md`.
  After a documented sandbox strategy and safety approval exists, staging may
  instead use `ENVIRONMENT_DATA_MODE=sandbox` and
  `ENVIRONMENT_DATA_SOURCE=documented-sandbox`.
- Main runs with `ENVIRONMENT_ROLE=main`, `APP_ENV=production`,
  `ENVIRONMENT_DATA_MODE=production`, production secrets, and the main database
  only. Production must not reuse dev or staging credentials.

The checked-in examples default staging providers to mocks until the staging
sandbox or masked provider strategy is configured. Flip individual
`USE_MOCK_*` flags only after the corresponding staging credential, data, and
write-safety plan exists.

Run `npm run environment-roles:check` after editing `deploy/env/*.example`,
`docker-compose.deploy.yml`, or environment-role documentation. The check
parses the deployment examples and fails if dev, staging, and main stop using
distinct roles, data modes, databases, Compose env-file wiring, or
mock-provider safety settings.

The current Go service image exposes the backend service on port `8080`. The
React/Vite DEV UI is still a development-time frontend and is not bundled into
the remote image by this deployment stack; production frontend asset packaging
remains a separate deployment decision.
