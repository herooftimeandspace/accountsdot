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
