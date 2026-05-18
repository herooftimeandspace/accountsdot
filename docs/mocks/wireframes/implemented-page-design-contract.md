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
- Logged-in implemented pages use one shared inner-page scroll container. The generated page body scrolls inside that container while the shared header and left sidebar stay anchored to the viewport through the shared renderer/CSS primitive.
- The shared header owns the top edge of the logged-in shell. Runtime drawers, help drawers, search, scope, notification/help controls, account controls, and page refresh controls must use the header's bottom edge as their top offset rather than page-local offsets.
- Role-based navigation filtering remains runtime behavior. Hidden nav groups must reflow upward in canonical order with no blank gaps.
- The active nav highlight must sit behind the active icon and label so the destination remains readable.
- Documented nested route buttons, such as IT Admin's `/admin/feature-flags` child under Admin, must render visually subordinate to their parent row, use the same role-filtered no-gap compaction as top-level rows, and align their active dot, label, focus target, and highlight to one row center.
- The scope selector, search field, notification/help controls, account box, profile image/initials, support affordance, and platform-status row must be sized for the longest supported persona labels and approved branding assets.
- The DEV-only persona switcher must render inside the shared sidebar bounds below Platform Status instead of as a top-right viewport toolbar; it may expand as an overlay for switching, but its collapsed control must not change the app frame width or right-drawer anchor.
- Live shell surfaces must not show shortcut hints, governance labels, mock-policy text, validation-process labels, or runbook/evidence copy unless the PRD defines that copy as operator-facing product behavior.

## Shared Primitives

- `Refresh`: the standard header refresh action is a Vegas Gold primary action with black text, `8px` radius, and the canonical header location declared in the generated implemented-page design manifest. Pages that expose header refresh must use this primitive.
- `Page Sync/Refresh`: pages with last-refreshed metadata, next-sync metadata, manual refresh, or manual sync behavior use the shared runtime primitive aligned to the header refresh bounds. Use `Refresh` for targeted page rereads and `Sync now` for source reconciliation or DEV mock source-sync simulation. The primitive owns disabled/loading copy and accessible button names; page-local refresh buttons should only remain when a product requirement explicitly places an action outside the shared header.
- `Header Scope Dropdown`: implemented pages use one shared runtime dropdown primitive for the header scope field. The primitive owns the visible white control and focus ring so static `.pen` scope text cannot show behind it; pages with documented scope behavior, such as Phone Directory directory focus, provide their own options and change handler while preserving the shared style.
- `Table`: tables use a shared top baseline across row cells. Multi-line cells grow the row downward; sparse cells and badges remain top-aligned.
- `Table Controls`: runtime tables expose a shared table search field plus three-way sortable headers. Header sort cycles `None → Ascending → Descending → None`; each page defines its own default sort column. The search field filters against data available in that table and must not include values hidden from users who lack permission.
- `Summary Info Box`: summary/stat cards and metric boxes must use a shared primitive that centers its text, renders the numeric value very large, and color-codes the numeric value using an explicit good-to-bad scale appropriate to the metric. Every retained info box must lead to a clear user action (navigation, filter, drawer, or decision); passive decoration should be removed or redesigned instead of persisting as a non-actionable status tile.
- `Wrapper/Card/Rail`: bordered containers keep clean separation from neighboring bordered elements and reserve space for titles, badges, icons, and actions.
- `Helper Paragraph`: a card, rail, notice, or table cell that conveys one logical paragraph uses one wrapping text node.
- `Status Badge`: badges must fit their text without colliding with card headers, table content, or action controls. Reused status bubbles/buttons must use the canonical severity palette below rather than page-local colors.
- `Drawer`: row-selected detail/context surfaces use the shared right-hand runtime drawer. The drawer is closed by default, opens only after an explicit row selection, updates in place when another row is selected, and closes through its upper-right `X`. The drawer anchors to the right edge directly below the shared header, avoids normal internal drawer scrolling, and relies on the shared page scroll range plus trailing space when drawer content is taller than the visible page.
- `Page Help`: every page that renders the shared-shell help icon opens the shared right-hand runtime drawer with end-user documentation for the current page. The copy explains what the page does and how a non-technical operator should use it; it is not implementation help.
- `Action Link`: links that lead to external systems must be defined by product behavior, not created solely because a mock contains link-like text.
- `Varsity Display Text`: any UI text rendered with the Varsity font must be authored in all lowercase. The product name remains `The WIZARD` in prose, metadata, and non-Varsity UI, but Varsity-rendered display text uses lowercase source copy rather than CSS-only transformation.

## Status Badge And Button Color Inventory

This inventory records the currently defined status bubbles/buttons and the proposed canonical color treatment for review. The goal is to make repeated badge states a shared primitive instead of page-local styling. Text colors are chosen from the existing brand palette for readable contrast.

| Labels / states | Primitive role | Current implementation notes | Canonical background | Canonical text | Severity intent |
| --- | --- | --- | --- | --- | --- |
| `Blocked`, `Invalid`, `Failed`, `Error` | Critical status badge / destructive state | Static `.pen` badges already use brand red for some `Blocked` states; runtime badges must not downgrade these labels to warning colors. | Brand red `#D73533` / `var(--color-accent-red)` | White `#FFFFFF` | Work cannot proceed; user attention is required before automation should continue. |
| `Needs Review`, `Review`, `Manual action`, `External action` | Review/action-needed badge | Static `.pen` uses coral for several `Review` states. | Coral `#FE5E41` / `var(--color-accent-coral)` | The WIZARD text `#01161E` | Human review is required, but the row is not necessarily hard-blocked. |
| `Incomplete Data`, `Warning`, start-date warning icon | Warning / incomplete badge | Incomplete data and warnings are high-attention states in the operator queue. | Brand red `#D73533` / `var(--color-accent-red)` | White `#FFFFFF` | Data is missing or a timing risk exists; record can remain visible while operators complete it. |
| `In Progress`, `Running` | Active/in-flight badge | Runtime default active status uses light blue. | Light blue `#89B6E7` / `var(--color-accent-light-blue)` | The WIZARD text `#01161E` | Automation is actively underway. |
| `Queued`, `Scheduled`, `Waiting` | Waiting/scheduled badge | Waiting and scheduled states should read as pending, not active. | Pink `#FCD9E9` / `var(--color-accent-pink)` | The WIZARD text `#01161E` | Work is queued, scheduled, or waiting on expected timing. |
| `Ready`, `Ready to Provision`, `Healthy`, `Complete`, `Allowed` | Success / ready badge | Static and runtime ready states use green for several badges; some dashboard `Healthy` and KPI values are plain text rather than badges. | Green `#00A878` / `var(--color-accent-green)` | The WIZARD text `#01161E` | Workflow is healthy, complete, ready, or explicitly allowed. |
| Neutral/default status, unknown non-error state | Neutral badge | Several generated mock labels are plain text on white and should stay plain unless they become reusable status badges. | Canvas `#DDE2E3` / `var(--color-canvas)` | The WIZARD text `#01161E` | Informational state with no severity. |
| Primary command buttons: `Refresh`, `Save`, `Add Non-Escape Record`, `Return to Dashboard` | Primary action button | Shared header refresh, onboarding save/add, and error recovery buttons use Vegas Gold. | Vegas Gold `#CEB770` / `var(--color-highlight)` | The WIZARD text `#01161E` | Main affirmative action for the current surface. |
| Destructive command buttons: `Delete Manual Entry` and future destructive actions | Destructive action button | Documented for manual-entry remediation; implementation should not reuse generic browser red. | Brand red `#D73533` / `var(--color-accent-red)` | White `#FFFFFF` | User action deletes, rejects, or removes a record. |

Known cleanup target: migrate status rendering to a shared badge primitive so runtime pages and `.pen`-generated pages use the same label-to-severity mapping. `Blocked` should not render as a Vegas Gold warning badge once that migration is applied.

## Primitive Feedback Matrix

| Feedback touches | Primitive | Default layer | Durable guard |
| --- | --- | --- | --- |
| Header, sidebar, profile, search, scope, nav, support, notification, help, platform status | `shared shell` | `.pen layout` or `runtime behavior` | Shared shell manifest, lint rule, runtime access/navigation test, or docs update |
| Header refresh, sync, freshness metadata, or repeated action placement/style | `refresh` or `action link` | `.pen layout` or `runtime behavior` | Standard primitive manifest, shared page sync/refresh primitive, lint rule, runtime interaction test, or docs update |
| Row spacing, row baseline, dividers, table overflow | `table` | `.pen layout` | `npm run pen:lint` table diagnostics or promoted failure rule |
| Summary boxes, stat cards, or metric tiles that should be actionable | `summary info box` | `docs/new behavior`, `.pen layout`, or `runtime behavior` | Shared primitive rule, `.pen` primitive, metric-to-action mapping docs, browser verification, or docs update |
| Card, rail, notice, panel, bordered control spacing | `wrapper/card/rail` | `.pen layout` | Spacing lint diagnostic, shared primitive rule, or accepted shared-border exception |
| Split helper copy, paragraph fragments, table-cell body fragments | `helper paragraph` | `.pen layout` | Fragmented-paragraph lint diagnostic or explicit semantic split note |
| Badge sizing, label fit, collision with row/card controls | `status badge` | `.pen layout` | Shared primitive rule, lint diagnostic, or accessibility check |
| Row-selected detail panels, selected item context, or directory detail overlays | `drawer` | `docs/new behavior`, `.pen layout`, or `runtime behavior` | Shared drawer primitive, row-click browser verification, accessibility check, or docs update |
| Page-specific visual issue with no shared primitive match | `page-local` | `.pen layout` | Page ledger row plus accepted one-time fix note or later primitive promotion |

## Spacing And Text Rules

- Preserve at least `5px` between row text and horizontal dividers.
- Preserve at least `5px` between neighboring bordered wrappers by default.
- Exception: when two bordered elements intentionally share one edge, such as a table header-to-body transition or row-to-row divider, collapse the join to one border with no gap and no double-width.
- Logged-in page frames stay left-aligned to the viewport-fixed shared shell. Do not center a generated logged-in artboard separately from fixed sidebar/header nodes.
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
