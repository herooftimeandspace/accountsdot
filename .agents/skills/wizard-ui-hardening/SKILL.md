---
name: wizard-ui-hardening
description: Use for The WIZARD/accountsdot frontend UI work involving .pen artboards, generated implemented pages, shared shell/header/sidebar behavior, design annotations, role-filtered navigation, demo polish, or Pre-Phase 0 UI hardening.
---

# WIZARD UI Hardening

Use this skill for implemented-page UI/design work in The WIZARD.

## Source Order

1. Read `README.md` for project goals and documentation policy.
2. Read the relevant implemented-page UI sections in `IMPLEMENTATION_PLAN.md`.
3. Read matching product scope and visibility rules in `PRODUCT_REQUIREMENTS.md`.
4. Check `TEST_MATRIX.md` only when the UI change affects named scenario coverage.
5. Use `AGENTS.md` and `docs/mocks/wireframes/implemented-page-design-contract.md` as the compact operating checklist.

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

## Durable UI Rules

- Shared header/sidebar are canonical logged-in shell surfaces; role filtering must reflow remaining nav items without blank gaps.
- Standard header `Refresh` is a shared primitive: same Vegas Gold styling, readable black text, and consistent header location wherever exposed.
- A logical paragraph, helper block, or table-cell body should be one wrapping text node unless a documented semantic reason requires separate nodes.
- Dashboard tables keep a shared top baseline, grow rows downward to the tallest cell, and preserve at least `5px` between row text and dividers.
- Bordered cards, rails, tables, notices, and controls keep at least `5px` from neighboring bordered elements unless an intentional shared-border join is documented.
- Live pages must not show shortcut pills, governance labels, mock-policy labels, or validation-process copy unless the PRD defines them as operator-facing product features.

## Loop Guard

If the same annotation set or generated result repeats more than twice without material progress, stop the slice and report the active layer, ledger status, last successful change, stopped processes, and one next action needed to resume safely.
