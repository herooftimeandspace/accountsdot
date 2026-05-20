# Vendored Reference Input Inventory

This inventory records the reference inputs currently checked into this repository. It should be updated whenever reference assets are added, refreshed, removed, or intentionally narrowed.

The Phase 0 startup guard documented in [reference-input-snapshot-integrity.md](../operations/reference-input-snapshot-integrity.md) treats this inventory, the reference-input README, and the branding files below as the current mandatory startup corpus.

## Checked-In Assets

| Path | Type | Purpose | Notes |
| --- | --- | --- | --- |
| `docs/reference-inputs/README.md` | Reference documentation | Explains the repo-local reference input directory contract | Startup-required documentation file; local links must resolve inside the repo. |
| `docs/reference-inputs/VENDORED_INVENTORY.md` | Reference inventory | Provenance and refresh ledger for checked-in reference inputs | Startup-required documentation file; local links must resolve inside the repo. |
| `docs/reference-inputs/branding/Firefly.png` | Branding image | Mascot mark for DEV UI and design mocks | Non-secret visual asset. |
| `docs/reference-inputs/branding/google-g.png` | Branding image | Google sign-in mark for DEV login mock | Non-secret visual asset. |
| `docs/reference-inputs/branding/Wordmarks/Gold W black outline.png` | Branding image | District wordmark reference | Non-secret visual asset. |

## Referenced But Not Present In This Branch

The implementation plan references additional historical planning inputs under `docs/reference-inputs/`, including provider SDK snapshots, legacy scripts, pipeline notes, data exports, and Zoom reference files. Those artifacts are not present in this branch's current checkout.

Before implementation work relies on one of those external inputs, add a sanitized repo-local snapshot here and update this inventory with provenance, refresh date, source scope, and any intentional omissions.
