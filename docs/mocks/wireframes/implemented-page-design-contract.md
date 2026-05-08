# Implemented-Page Design Contract

This contract makes the implemented-page UI rules from `IMPLEMENTATION_PLAN.md` and `PRODUCT_REQUIREMENTS.md` operational for rapid Pre-Phase 0 design work. It does not replace those documents. When this file and the source-of-truth documents disagree, update the source-of-truth documents first, then revise this contract.

## Authority Split

- Authoritative `.pen` files own static geometry, spacing, typography, text blocks, shell layout, page layout, and visual grouping.
- Generated artboard JSON and generated presentational components are derived outputs. Do not hand-edit them.
- React owns documented runtime behavior: routing, role authorization, data loading, persona state, form state, sorting, links, access-denied handling, and interactive controls already defined by `PRODUCT_REQUIREMENTS.md` and `IMPLEMENTATION_PLAN.md`.
- New behavior starts as `docs/new behavior`. Do not infer new behavior from a visual mock until the PRD and implementation plan define it.

## Shared Logged-In Shell

- The logged-in shell uses `docs/mocks/wireframes/wireframe-shared-shell.pen` as the canonical sidebar/header source.
- Sidebar/header shell defects must be fixed at the shared-shell pattern level, not by patching individual pages.
- Role-based navigation filtering remains runtime behavior. Hidden nav groups must reflow upward in canonical order with no blank gaps.
- The active nav highlight must sit behind the active icon and label so the destination remains readable.
- The scope selector, search field, notification/help controls, account box, profile image/initials, support affordance, and platform-status row must be sized for the longest supported persona labels and approved branding assets.
- Live shell surfaces must not show shortcut hints, governance labels, mock-policy text, validation-process labels, or runbook/evidence copy unless the PRD defines that copy as operator-facing product behavior.

## Shared Primitives

- `Refresh`: the standard header refresh action is a Vegas Gold primary action with black text, `8px` radius, and the canonical header location declared in the generated implemented-page design manifest. Pages that expose header refresh must use this primitive.
- `Table`: tables use a shared top baseline across row cells. Multi-line cells grow the row downward; sparse cells and badges remain top-aligned.
- `Wrapper/Card/Rail`: bordered containers keep clean separation from neighboring bordered elements and reserve space for titles, badges, icons, and actions.
- `Helper Paragraph`: a card, rail, notice, or table cell that conveys one logical paragraph uses one wrapping text node.
- `Status Badge`: badges must fit their text without colliding with card headers, table content, or action controls.
- `Drawer`: row-selected detail/context surfaces use the shared right-hand runtime drawer. The drawer is closed by default, opens only after an explicit row selection, updates in place when another row is selected, and closes through its upper-right `X`.
- `Action Link`: links that lead to external systems must be defined by product behavior, not created solely because a mock contains link-like text.

## Primitive Feedback Matrix

| Feedback touches | Primitive | Default layer | Durable guard |
| --- | --- | --- | --- |
| Header, sidebar, profile, search, scope, nav, support, notification, help, platform status | `shared shell` | `.pen layout` or `runtime behavior` | Shared shell manifest, lint rule, runtime access/navigation test, or docs update |
| Header refresh or repeated action placement/style | `refresh` or `action link` | `.pen layout` or `runtime behavior` | Standard primitive manifest, lint rule, runtime interaction test, or docs update |
| Row spacing, row baseline, dividers, table overflow | `table` | `.pen layout` | `npm run pen:lint` table diagnostics or promoted failure rule |
| Card, rail, notice, panel, bordered control spacing | `wrapper/card/rail` | `.pen layout` | Spacing lint diagnostic, shared primitive rule, or accepted shared-border exception |
| Split helper copy, paragraph fragments, table-cell body fragments | `helper paragraph` | `.pen layout` | Fragmented-paragraph lint diagnostic or explicit semantic split note |
| Badge sizing, label fit, collision with row/card controls | `status badge` | `.pen layout` | Shared primitive rule, lint diagnostic, or accessibility check |
| Row-selected detail panels, selected item context, or directory detail overlays | `drawer` | `docs/new behavior`, `.pen layout`, or `runtime behavior` | Shared drawer primitive, row-click browser verification, accessibility check, or docs update |
| Page-specific visual issue with no shared primitive match | `page-local` | `.pen layout` | Page ledger row plus accepted one-time fix note or later primitive promotion |

## Spacing And Text Rules

- Preserve at least `5px` between row text and horizontal dividers.
- Preserve at least `5px` between neighboring bordered wrappers by default.
- Exception: when two bordered elements intentionally share one edge, such as a table header-to-body transition or row-to-row divider, collapse the join to one border with no gap and no double-width.
- Text must wrap or truncate inside its container rather than overflowing into adjacent content or off canvas.
- One logical paragraph should not be split into multiple stacked text boxes. Split text only when the pieces have distinct semantics, independent runtime slots, or intentionally different styling.
- Fields such as `Last refreshed` may wrap across multiple lines when needed to avoid collisions with adjacent controls.

## Annotation Ledger Workflow

- Before an annotation-driven pass starts, copy active Codex annotate feedback into a checked-in ledger under `docs/mocks/wireframes/annotation-ledgers/` or the page-specific existing ledger.
- Each ledger row must include: id, page, source, layer, expected fix location, status, and durable guard.
- Valid layers are `pipeline`, `.pen layout`, `docs/new behavior`, `runtime behavior`, and `review artifact`.
- Valid statuses are `open`, `closed`, `reclassified as behavior`, `accepted exception`, and `still failing`.
- A closed annotation must remain in the ledger until it is protected by a lint rule, shared primitive rule, docs update, or explicit one-time-fix note.

## Checks

- `npm run pen:check` verifies generated outputs are current with authoritative `.pen` sources.
- `npm run pen:lint` enforces high-confidence design contract checks and reports warning-only visual risks for the first hardening pass.
- `npm run build:web` verifies the generated/runtime frontend builds.
- `npm run a11y:check` verifies current accessibility guardrails.
- Warning-only lint findings should be reviewed during cleanup and promoted to failures after false positives are resolved.
