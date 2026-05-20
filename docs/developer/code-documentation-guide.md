# Code Documentation Guide

This guide explains how The WIZARD documents implemented code so a new contributor can trace data movement, debug call paths, and understand side effects without already knowing the project.

## What To Document

Document every repo-owned runtime and test function under `cmd/`, `internal/`, and `frontend/src/`. Exclude generated artboards, `frontend/dist/`, caches, vendored files, and build output.

Each function comment should answer five questions:

1. Why does this function exist?
2. Who calls it or what route/component/test reaches it?
3. What data enters the function?
4. What data leaves the function?
5. What side effects, errors, permissions, state transitions, or debugging signals matter?

The comment should explain intent and data flow, not every line of syntax. If the function only formats a value, say what upstream data it normalizes and what downstream caller expects. If the function mutates state, name the state and how to verify the mutation.

## Tracing Backend Data Flow

Start with `cmd/provisioner/main.go`. `main` calls `realMain`, which calls `run`, which loads configuration and builds the HTTP handler through `internal/web.NewAppHandler`.

For HTTP requests:

1. Find the route registration in `internal/web/app.go`.
2. Jump to the handler function.
3. Follow helper calls that authenticate the DEV persona, decode JSON, mutate a DEV store, plan workflow jobs, or write a JSON response.
4. Read matching tests in `internal/web/*_test.go` to see the expected status code, response shape, and role behavior.

For workflow planning:

1. Start in `internal/orchestrator/planner.go`.
2. Find the `core.WorkflowType` branch in `PlanWorkflow`.
3. Read the generated provider-operation names as planned work, not necessarily live SDK calls.
4. Check tests in `internal/orchestrator/*_test.go` for the intended job order, approval gates, follow-up workflows, and debounce behavior.

For database safety:

1. Read `internal/db/retry.go` before adding transaction logic.
2. Use `WithRetry` for serializable transaction work that can hit serialization or deadlock failures.
3. Check `internal/db/schema.sql` for authoritative table and constraint names.
4. Document every new database write path in `docs/planning/external-write-inventory.md`.

## Tracing Frontend Data Flow

Start with `frontend/src/app.jsx`. The app reads the current URL, resolves it through `frontend/src/lib/routeRegistry.js`, loads the DEV session, then renders the matching page component.

For a page:

1. Find the route in `routeRegistry.js`.
2. Open the corresponding page under `frontend/src/pages/`.
3. Follow `fetch` calls to `/api/v1/dev/...` routes.
4. Match those routes to handlers in `internal/web/`.
5. Read page-level helpers that transform payload values for display, sorting, filtering, drawers, warnings, or action buttons.

Implemented pages often combine `.pen`-derived static artboards with React overlays. Static geometry lives in generated artboards and `.pen` sources; runtime behavior belongs in React comments and backend handler comments.

## External Write Documentation

Any path that writes to a provider, the database, or a DEV mock store needs extra care. The local app currently contains many mock-only DEV mutations and planner operation names for future provider writes. Treat both as important documentation surfaces.

When documenting a write path, include:

- trigger route or caller,
- authorization/persona requirement,
- target system or store,
- payload or state being changed,
- expected success response,
- retry/idempotency expectation,
- failure signal used during debugging.

Keep `docs/planning/external-write-inventory.md` synchronized whenever a write path changes. If a live SDK call is added later, document the exact provider method, idempotency key, request-log behavior, and staging validation requirement before merging.

## High-Risk Workflow Walkthroughs

Use the walkthroughs in `docs/developer/code-paths/` when debugging or extending high-risk paths that cross frontend UI, route handlers, persona rules, and mutation boundaries:

- [Manual onboarding drafts](code-paths/manual-onboarding-drafts.md)
- [Job lease recovery](code-paths/job-lease-recovery.md)
- [Room moves](code-paths/room-moves.md)
- [Sync dashboard overrides](code-paths/sync-dashboard-overrides.md)
- [Shared shell help content](code-paths/shared-shell-help-content.md)

Each walkthrough names the frontend entrypoint, route, handler/store/helper chain, payload shape, authorization and persona behavior, mutation boundary, tests, and useful breakpoints. Keep those files aligned with `docs/planning/external-write-inventory.md` whenever a route becomes write-capable, stops mutating state, or changes its external-write risk.

## VS Code Debugging

Use the VS Code Dev Containers extension for the most predictable setup:

1. Open this folder in VS Code.
2. Run `Dev Containers: Reopen in Container`.
3. In the container terminal, run `make test` once to confirm dependencies and local paths.

For Go tests:

1. Open the target `*_test.go` file.
2. Set a breakpoint in the test and in the handler/helper it calls.
3. Use the Go extension's `Debug Test` code lens.
4. Step from the test request setup into `NewAppHandler`, the handler, store method, planner, or response writer.

For the Go server:

1. Copy `.env.example` to `.env` and keep real provider writes disabled for local debugging.
2. Run `make up` if Postgres-backed paths are needed.
3. Start the app with the VS Code Go debugger or a terminal command that sets local mock environment variables.
4. Set breakpoints in `cmd/provisioner/main.go`, route handlers, and helpers that decode or write JSON.

For frontend debugging:

1. Run the Vite dev server using the repo's frontend workflow.
2. Open the page in a browser with developer tools.
3. Set breakpoints in `frontend/src/app.jsx`, the target page component, and helper functions that issue `fetch` calls.
4. Use the Network tab to capture the `/api/v1/dev/...` request and then jump to the matching Go handler.

## Keeping Documentation Fresh

Documentation must evolve with code. Before finishing a code change:

1. Search for renamed functions, routes, JSON fields, workflow names, and provider operations.
2. Update direct caller comments when a callee's behavior changes.
3. Remove or rewrite comments for deleted paths.
4. Update this guide if the debugging workflow changes.
5. Update `docs/planning/external-write-inventory.md` when any write-capable path changes.

Stale documentation is dangerous in this project because it can hide provider-write risk. Prefer deleting obsolete detail over leaving a confident but wrong explanation.

## Placeholder Comment Quality Gate

Run the placeholder-comment quality gate whenever adding or editing comments under `cmd/`, `internal/`, or `frontend/src/`:

```bash
npm run docs:comments:check
```

The same check is also available as:

```bash
make docs-comments-check
```

The check scans Go, JavaScript, TypeScript, JSX, TSX, and CSS comments under the implemented-code roots and skips generated output such as `frontend/src/generated/`, `frontend/dist/`, caches, and dependency folders. It looks only at comments, not strings or rendered copy, so placeholder words in user-facing text or test data do not fail the gate.

The gate flags boilerplate phrases from the issue #3 documentation pass, including generic data-flow comments, UI-surface summaries, derived-data summaries, request-path comments, frontend event-handler comments, signature-placeholder wording, and broad side-effect warnings that point to `docs/planning/external-write-inventory.md` without naming the actual caller, state, payload, or failure signal.

Allowed exceptions are intentionally narrow:

- Existing inherited placeholders from the issue #3 documentation branch are listed in `scripts/doc_comment_quality_baseline.json` so this gate can land without rewriting unrelated comments in the issue #13 branch.
- Do not add new baseline entries for new or edited code. Rewrite the comment instead.
- Remove a baseline entry when you replace the corresponding placeholder with a specific comment.
- Regenerate the baseline only as part of a deliberate cleanup pass that removes or rewrites inherited placeholders first.

When the gate fails, rewrite the reported comment so it answers the standard questions in this guide: why the function exists, who calls it, what data enters, what data leaves, and what side effects, errors, permissions, state transitions, or debugging signals matter.
