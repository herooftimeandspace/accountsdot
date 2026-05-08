# Implemented Pages Annotation Ledger

This ledger is the default landing place for annotation-driven hardening work across implemented `.pen` pages. Move a page section into its own file when the active annotation set becomes large enough to need a dedicated recovery ledger.

| ID | Page | Source | Layer | Expected fix location | Status | Durable guard |
| --- | --- | --- | --- | --- | --- | --- |
| DQ-A1 through DQ-A11 | Data Quality | `docs/mocks/wireframes/data-quality-annotation-ledger.md` | `.pen layout` | `docs/mocks/wireframes/wireframe-data-quality-dashboard.pen` | closed | Existing page ledger plus `npm run pen:lint` shared spacing/primitive checks |
| UIH-001 | All implemented pages | Design/UI hardening plan | pipeline | `scripts/sync_implemented_pages.mjs`, generated manifest, `scripts/lint_implemented_pages.mjs` | open | `npm run pen:check`, `npm run pen:lint` |
| UIH-002 | All logged-in pages | Design/UI hardening plan | `.pen layout` | shared shell/header/sidebar pattern | open | Shared shell manifest and standard refresh lint checks |
| UIH-003 | All implemented table pages | Design/UI hardening plan | `.pen layout` | authoritative `.pen` table layouts | open | Warning-only table baseline and divider-gap lint diagnostics |
| UIH-004 | All implemented pages | Design/UI hardening plan | `.pen layout` | authoritative `.pen` helper text/card/table text nodes | open | Warning-only fragmented-paragraph lint diagnostics |
| UIH-005 | All implemented pages | Design/UI hardening plan | review artifact | runtime screenshot/accessibility evidence | open | `npm run build:web`, `npm run a11y:check`, page screenshots when requested |
| UIH-006 | All logged-in implemented pages | User feedback: persona switcher feels broken after role change | docs/new behavior, runtime behavior | `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, `frontend/src/app.jsx` | closed | Docs update plus `npm run build:web`, `npm run a11y:check`, and in-app browser verification of DEV persona-switch routing and strict direct-link `403` behavior |
| UIH-007 | All logged-in implemented pages | User-edited Data Quality shell should become standalone shared shell authority | pipeline, `.pen layout` | `wireframe-shared-shell.pen`, `scripts/sync_implemented_pages.mjs`, generated artboards | closed | `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, `npm run a11y:check`, Pencil layout snapshots, and browser verification on `/data-quality`, `/phone-directory/by-person`, and logged-in `/error/403` |
| UIH-008 | All logged-in implemented pages | User feedback: shared-shell tagline should stay Vegas Gold even when contrast tooling objects | docs/new behavior, `.pen layout`, runtime behavior | `wireframe-shared-shell.pen`, generated artboards, `scripts/check_frontend_accessibility.mjs` | closed | PRD/plan override record plus narrow accessibility-check exception for exact tagline and browser verification |
| UIH-009 | IT Admin Dashboard | User feedback: dashboard overview should use shared shell/refresh primitives and reduce split table text boxes | pipeline, `.pen layout` | `wireframe-it-admin-overview.pen`, generated dashboard artboard | closed | `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, `npm run a11y:check`, Pencil layout snapshot, and browser verification on `/dashboard/it-admin` |
| UIH-010 | IT Admin Dashboard | User feedback: Schedules Running Next status badges overlap the card border | `.pen layout` | `wireframe-it-admin-overview.pen`, generated dashboard artboard | closed | Status badge column resize in the authoritative PEN plus `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, `npm run a11y:check`, and in-app browser verification on `/dashboard/it-admin` |
| UIH-011 | Onboarding | User feedback: cleanup pass should use shared shell/refresh primitives and reduce fragmented table text, spacing collisions, and badge overflow | pipeline, `.pen layout` | `wireframe-onboarding-dashboard.pen`, generated onboarding artboard | closed | Page-pane cleanup in the authoritative PEN plus `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, `npm run a11y:check`, and in-app browser verification on `/onboarding` |
| UIH-012 | Onboarding | User feedback: replace fixed Selected Workflow overlay with a row-click right-hand drawer, delete helper callout, and expand the table into the freed pane space | docs/new behavior, runtime behavior, `.pen layout` | `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, `StaticPenPage.jsx`, `wireframe-onboarding-dashboard.pen`, generated onboarding artboard | closed | PRD/plan behavior note plus row-click/update/close browser verification, `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check` |
| UIH-013 | All row-detail implemented pages | User feedback: right-hand drawer should become a reusable runtime primitive and replace static row detail/context panels | docs/new behavior, runtime behavior, pipeline, `.pen layout` | `implemented-page-design-contract.md`, `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, shared drawer component, affected `.pen` files, generated artboards | completed | Shared drawer primitive plus row-click/update/close browser verification, `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check` |
| UIH-014 | Onboarding and shared shell | User feedback: add Non-Escape manual intake drawer, rename sidebar item to Onboarding, remove Current Rules/helper note, pin table footer, and move Start to first table column | docs/new behavior, runtime behavior, DEV mock API behavior, `.pen layout`, pipeline | `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, `wireframe-shared-shell.pen`, `wireframe-onboarding-dashboard.pen`, generated artboards, Onboarding runtime page, DEV onboarding APIs | completed | Docs update plus `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, Go handler tests, `npm run build:web`, `npm run a11y:check`, and browser verification on `/onboarding` |
| UIH-015 | Onboarding | User feedback: add Date Added as the first onboarding table column, make Add Non-Escape reliably open the drawer, compact drawer field rows, move missing required fields into the status bubble, use brand red missing-field borders, and show workflow action details | docs/new behavior, runtime behavior, DEV mock API behavior, `.pen layout` | `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, `wireframe-onboarding-dashboard.pen`, generated onboarding artboard, Onboarding runtime page, DEV onboarding APIs | completed | Docs update plus `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, Go handler tests, `npm run build:web`, `npm run a11y:check`, and browser verification on `/onboarding` |
| UIH-016 | Onboarding | Regression feedback: Add Non-Escape visual button could appear without a dependable runtime-owned click target | runtime behavior, `.pen layout` | Onboarding runtime page and runtime button styling | completed | Single visible runtime-owned Vegas Gold button, static PEN button hidden in live page, `npm run build:web`, `npm run a11y:check`, and browser verification on `/onboarding` |

## Page Index

- `admin`: no additional active annotations captured yet.
- `dashboard-hr-lifecycle`: no additional active annotations captured yet.
- `dashboard-it-admin`: no additional active annotations captured yet.
- `dashboard-site-admin`: no additional active annotations captured yet.
- `data-quality`: see `docs/mocks/wireframes/data-quality-annotation-ledger.md`.
- `frequent-fliers`: no additional active annotations captured yet.
- `login`: no additional active annotations captured yet.
- `my-profile`: no additional active annotations captured yet.
- `offboarding`: no additional active annotations captured yet.
- `onboarding`: no additional active annotations captured yet.
- `phone-directory-by-department`: no additional active annotations captured yet.
- `phone-directory-by-person`: no additional active annotations captured yet.
- `phone-directory-by-room`: no additional active annotations captured yet.
- `reports`: no additional active annotations captured yet.
- `reports-sync-transparency`: no additional active annotations captured yet.
- `reports-ticketing-human-work`: no additional active annotations captured yet.
- `room-moves`: no additional active annotations captured yet.
- `student-data-cleanup`: no additional active annotations captured yet.
