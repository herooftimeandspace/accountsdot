# Data Quality Annotation Recovery Ledger

This ledger freezes the current Data Quality recovery cycle so the implemented page can be corrected without mixing layout, runtime, docs, and review-export changes in one loop.

## Recovery Scope
- Page: `docs/mocks/wireframes/wireframe-data-quality-dashboard.pen`
- Runtime target: `http://localhost:5173/data-quality`
- Authority rule:
  - `.pen` owns static geometry, spacing, typography, and page-shell layout
  - runtime code owns only documented behavior already present in the implemented page
  - new behavior requires PRD + implementation-plan updates before runtime changes begin

## Current Recovery Set
| ID | Annotation | Class | Current Resolution Layer | Status |
| --- | --- | --- | --- | --- |
| A1 | Shrink the header account box to remove the visible gap without clipping the longest current persona label. | layout-only | `.pen` | closed |
| A2 | Account dropdown should open a menu with `My Profile` and `Sign Out`. | new behavior requiring doc lock | docs first | reclassified as behavior |
| A3 | Refresh button should read clearly as a Vegas Gold primary action and continue to re-collect Data Quality issues when clicked. | existing-behavior verification | live DEV verification | closed |
| A4 | `Next Action` cells should link to the destination page or fall back to `{Action} in {system}` with a top-level system link when no in-app route exists. | new behavior requiring doc lock | docs first | reclassified as behavior |
| A5 | Wrapped queue-cell text should push row dividers down instead of overlapping them. | layout-only | `.pen` | closed |
| A6 | `Support` question mark and label should align together and open IncidentIQ ticket creation. | new behavior requiring doc lock | docs first | reclassified as behavior |
| A7 | Summary cards should use denser vertical centering with larger count typography and less dead space. | layout-only | `.pen` | closed |
| A8 | Scope selector should be wide enough to keep the current site label on one line before dropdown behavior is added. | layout-only | `.pen` | closed |
| A9 | Remove the redundant page-scope badge because the shell selector already conveys scope. | layout-only | `.pen` | closed |
| A10 | Move the bottom-left support/status cluster upward so it is not pinned too close to the page edge. | layout-only | `.pen` | closed |
| A11 | Move the tagline down, make it full sidebar width, and keep it clear of the branding block above. | layout-only | `.pen` | closed |

## Recovery Rules
- Do not edit derived SVG or PNG review artifacts during the recovery cycle.
- Do not mix layout, docs, and runtime behavior changes in one validation pass.
- Close layout-only items against the live DEV page first.
- If a behavior request is still undefined after layout is stable, stop and update `PRODUCT_REQUIREMENTS.md` and `IMPLEMENTATION_PLAN.md` before touching runtime code.
