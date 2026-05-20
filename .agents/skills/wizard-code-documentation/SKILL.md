---
name: wizard-code-documentation
description: Use for The WIZARD/accountsdot implemented code changes, documentation maintenance, call-path tracing, function comments, external write inventory updates, and pruning stale code documentation. Trigger whenever cmd/, internal/, frontend/src/, tests, routes, schemas, provider operations, or external-write behavior changes.
---

# WIZARD Code Documentation

Use this skill whenever implemented code changes in The WIZARD. The goal is to keep the codebase understandable to a junior engineer who needs to trace data flow, debug call paths, and identify write-capable provider boundaries.

## Source Order

1. Read `README.md` for documentation policy, local setup, and test commands.
2. Read `.agents/AGENTS.md` for repo safety, architecture, access, frontend, and verification rules.
3. Read the code being changed and every direct caller/callee affected by the change.
4. Read `docs/code-documentation-guide.md` for the repo's call-path and debugging documentation standard.
5. Read `docs/planning/external-write-inventory.md` before changing any provider, database, or DEV mock mutation path.

## Required Documentation Loop

1. Identify the changed functions, routes, handlers, components, test helpers, schemas, and provider-operation names.
2. Update inline documentation in the same patch as the code change.
3. Update documentation on direct callers when the callee's behavior, output, side effects, or failure mode changes.
4. Update `docs/planning/external-write-inventory.md` before adding, removing, or changing any live, planned, mock-only, or database write path.
5. Prune comments and guide sections that describe removed functions, obsolete routes, renamed symbols, superseded workflow behavior, or stale debugging steps.
6. Search for old symbol names, route paths, provider-operation names, JSON fields, and workflow labels before finishing.
7. Run `npm run docs:comments:check` after adding or editing comments under `cmd/`, `internal/`, or `frontend/src/`.

## Inline Documentation Standard

- Document why the function exists, who calls it, what data enters, what data leaves, and what errors or state changes matter.
- Prefer precise call-path language: HTTP route, React page/component, test case, planner operation, provider contract, or store method.
- Explain side effects clearly. Name database writes, in-memory DEV store mutations, external SDK/API writes, and generated provider-operation plans.
- Keep comments accurate and maintainable. Do not narrate obvious syntax.
- Do not introduce placeholder templates such as generic data-flow summaries, UI-surface summaries, request-path summaries, frontend event-handler summaries, signature-placeholder wording, or vague side-effect warnings. The repo-local quality gate flags these phrases unless they are inherited entries in `scripts/doc_comment_quality_baseline.json`.
- Keep generated files, dist output, caches, vendored assets, and `.pen`-generated artboards out of manual documentation edits.

## External Write Rule

Any code that can write to Zoom, IncidentIQ, Google, the local database, a DEV mock store, or another provider-backed system needs a nearby comment that names:

- the triggering caller or route,
- the system or store being mutated,
- the expected success result,
- the idempotency or retry expectation,
- the failure/debugging signal.

If the path is only planned or mock-only today, document that explicitly so future live SDK work does not inherit ambiguous safety assumptions.

## Verification

- Run the narrow tests for the touched code, then the repo's normal relevant checks.
- Run `npm run docs:comments:check` for comment-only changes or any patch that touches implemented-code comments.
- For Go comments, run `make test-unit` or `make test` when practical.
- For frontend comments, run `npm run build:web` when practical.
- Manually verify that comments still match code after formatting or refactors.
