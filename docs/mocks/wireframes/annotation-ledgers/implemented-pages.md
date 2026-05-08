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
