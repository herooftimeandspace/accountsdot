---
name: wizard-ui-hardening
description: Use for The WIZARD/accountsdot frontend UI work involving .pen artboards, generated implemented pages, shared shell/header/sidebar behavior, design annotations, role-filtered navigation, demo polish, or Pre-Phase 0 UI hardening.
---

# WIZARD UI Hardening

Use this skill for implemented-page UI/design work in The WIZARD.

## Source Order

1. Read `README.md` for project goals and documentation policy.
2. Read the relevant implemented-page UI sections in `docs/planning/implementation-plan.md`.
3. Read matching product scope and visibility rules in `docs/product/product-requirements.md`.
4. Check `docs/testing/test-matrix.md` only when the UI change affects named scenario coverage.
5. Use `.agents/AGENTS.md` and `docs/design/mocks/wireframes/implemented-page-design-contract.md` as the compact operating checklist.

## Layer Classification

Classify each issue before editing:

- `pipeline`: sync, generated manifest, renderer, or lint/check tooling.
- `.pen layout`: geometry, spacing, typography, wrapping, static shell/page layout.
- `docs/new behavior`: behavior or affordance not yet defined by PRD/plan.
- `runtime behavior`: documented React routing, data loading, access, state, or interaction.
- `review artifact`: optional SVG/PNG export or human-review output.

Default order is pipeline contract, `.pen layout`, docs for new behavior, runtime behavior, then optional review artifacts. Do not mix layers in one recovery pass unless the user explicitly asks and the risk is documented.

## Implemented-Page Loop

1. Freeze active Codex annotations into a checked-in ledger before implementation.
2. Fix shared shell and shared component issues at the shared pattern level.
3. Update authoritative `.pen` files first for layout defects.
4. Run `npm run pen:sync` after `.pen` updates.
5. Never hand-edit generated artboards, generated presentational components, or generated review exports.
6. Run `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check` after relevant UI changes.

## Primitive-First Feedback Intake

Before editing, convert every Codex annotate item into a ledger row in `docs/design/mocks/wireframes/annotation-ledgers/implemented-pages.md` or a page-specific ledger.

Each row must identify:

- Layer: `pipeline`, `.pen layout`, `docs/new behavior`, `runtime behavior`, or `review artifact`.
- Primitive: `shared shell`, `refresh`, `table`, `wrapper/card/rail`, `helper paragraph`, `status badge`, `action link`, or `page-local`.
- Durable guard: lint rule, shared primitive rule, docs update, generated manifest entry, runtime test, accessibility check, or accepted one-time exception.

Do not close a feedback item as merely "fixed." Closed items must name the durable guard that prevents regression.

## Primitive Escalation

- If the same feedback appears on two or more pages, treat it as shared primitive work.
- Header, sidebar, profile, search, scope, nav, support, notification, help, and platform-status feedback is always `shared shell`.
- Row spacing, baseline alignment, dividers, wrapper borders, overflow, and fragmented text are primitive work first; make them page-local only after the primitive rule is clarified.
- New behavior requests are `docs/new behavior`: stop and update `docs/product/product-requirements.md` plus `docs/planning/implementation-plan.md` before runtime implementation.
- When auth/session loading, route activation, drawer focus, or refresh/sync behavior is touched on more than one page, treat it as shared runtime primitive work and define the invariant before patching a single page.

## Primitive Cleanup Order

1. Shared shell/header/sidebar.
2. Standard refresh and action controls.
3. Wrapper/card/rail spacing.
4. Tables and row-baseline behavior.
5. Helper paragraph and text wrapping.
6. Badges, links, and page-local polish.

## Durable UI Rules

- Shared header/sidebar are canonical logged-in shell surfaces; role filtering must reflow remaining nav items without blank gaps.
- Shared navigation must expose exactly one active row for the current route, even when a parent route resolves to a default child destination after role filtering or redirect logic.
- Standard header `Refresh` is a shared primitive: same Vegas Gold styling, readable black text, and consistent header location wherever exposed.
- A logical paragraph, helper block, or table-cell body should be one wrapping text node unless a documented semantic reason requires separate nodes.
- Dashboard tables keep a shared top baseline, grow rows downward to the tallest cell, and preserve at least `5px` between row text and dividers.
- Bordered cards, rails, tables, notices, and controls keep at least `5px` from neighboring bordered elements unless an intentional shared-border join is documented.
- Live pages must not show shortcut pills, governance labels, mock-policy labels, or validation-process copy unless the PRD defines them as operator-facing product features.

## Runtime Invariants

Before editing shared-shell or auth-adjacent runtime behavior, write down the invariant the change must preserve and encode it in the smallest relevant test.

- Session/auth requests must be race-safe: an older in-flight load must not overwrite a newer login, logout, or refresh result.
- Shared navigation must define active-row precedence explicitly when parent and child destinations can both match.
- Drawer/help primitives must restore focus predictably and share geometry/close behavior across pages.
- Refresh/sync controls must keep labeling, disabled/loading state, and last-refreshed semantics aligned across pages.

Prefer request cancellation, versioning, or stale-response guards over timing assumptions in UI state code.

## Verification Expectations

UI work is not complete when code and generated artboards compile but runtime evidence is missing.

- For shared-shell, auth, routing, drawer, or table-contract changes, capture at least one UI artifact as part of the runtime evidence.
- Use the Browser plugin for local route checks when available. If Browser evaluation is required but unavailable, record the blocker as `needs_browser_evaluation` instead of substituting non-UI probes.
- Verify keyboard focus behavior whenever a drawer, dialog, login flow, or shared action control changes.
- If Browser verification is blocked, state the missing target/tool explicitly and do not silently treat the pass as fully closed.

## Comment Hygiene

This repo enforces comment quality. When touching files that often fail `npm run docs:comments:check`, clean or replace nearby boilerplate comments if they are part of the edited surface.

- Comments should explain business or behavioral intent, not restate syntax.
- When a runtime helper preserves a non-obvious invariant, document that invariant in the code where it is enforced.
- Do not add placeholder comments that defer explanation to PR text alone.

## Loop Guard

If the same annotation set or generated result repeats more than twice without material progress, stop the slice and report the active layer, ledger status, last successful change, stopped processes, and one next action needed to resume safely.

## Feedback Thread Handoff

When reporting progress in a UI feedback thread, include the active page, active primitive, ledger rows touched, layer classification, files expected to change, checks to run, and whether any item was reclassified as behavior.
