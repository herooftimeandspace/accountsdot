# Issue 48 / 52 Route Performance Browser Summary

Captured: 2026-05-15T19:42Z

Branch: `issue-48-52-route-performance-evidence`

Base: `origin/ui-improvements` at `9bcaec5697e60f90602ac0eb398a5423ad25c62a`

## Scope

This pass targeted GitHub issues #48 and #52. It collected current Browser-backed route timing evidence for historical slow-route candidates without changing route behavior, page artboards, generated artboards, or shared shell UI.

The pass intentionally avoided `scripts/dev_route_performance_matrix.mjs` and `README.md` edits because PR #112 already owns strict route-performance gate changes in that area.

## Environment

- API source: existing local `ui-improvements` DEV API on `localhost:8080`
- Frontend source: this worktree's Vite server on `http://localhost:5175`
- DEV session preflight: `curl -i http://localhost:5175/api/v1/dev/session` returned `200 OK` with development persona JSON
- Browser surface: Codex in-app Browser `iab`
- Harness output directory: `/private/tmp/accountsdot-issue-48-52-route-perf`
- Curated merge output for local review: `artifacts/performance/dev-route-performance-merged-2026-05-15T19-42-08-380Z.md`

The curated merge output is local evidence only. It is not full closure evidence because this was a targeted pass covering 9 of 650 directed transitions and 5 of 52 refresh samples.

## Current Route Plan

`npm run perf:routes:plan` passed on this branch.

- Route targets: 26
- Directed transitions: 650
- Transition budget: warning over 3000 ms, failure over 7000 ms
- Refresh budget: warning over 3000 ms, failure over 7000 ms

The current route plan differs from earlier issue comments that recorded 25 routes / 600 transitions on older branch bases.

## Targeted Transition Results

These transition candidates came from issue #48's original slow list plus later issue #52 historical stale-artifact candidates.

| Index | Edge | Status | Total | Navigation load | Readiness polling | Session fetch | Signal |
| ---: | --- | --- | ---: | ---: | ---: | ---: | --- |
| 85 | `/offboarding` -> `/admin/feature-flags` | ok | 520 ms | 126 ms | 390 ms | 6 ms | `expected_text` |
| 579 | `/dashboard` -> `/departing-seniors` | ok | 494 ms | 118 ms | 373 ms | 6 ms | `expected_text` |
| 439 | `/dashboard/it-admin` -> `/phone-directory/by-room` | ok | 493 ms | 117 ms | 373 ms | 7 ms | `expected_text` |
| 114 | `/admin` -> `/phone-directory/by-room` | ok | 492 ms | 116 ms | 370 ms | 7 ms | `expected_text` |
| 397 | `/offboarding` -> `/data-quality` | ok | 487 ms | 124 ms | 360 ms | 7 ms | `expected_text` |
| 637 | `/dashboard/it-admin` -> `/search` | ok | 481 ms | 118 ms | 359 ms | 7 ms | `expected_text` |
| 175 | `/offboarding` -> `/reports/ticketing-human-work` | ok | 475 ms | 122 ms | 350 ms | 7 ms | `expected_text` |
| 387 | `/room-moves/bulk-draft?draft_id=rm-draft-103` -> `/data-quality` | ok | 472 ms | 119 ms | 351 ms | 7 ms | `expected_text` |
| 221 | `/search?q=alex` -> `/reports/sync-transparency` | ok | 467 ms | 121 ms | 343 ms | 6 ms | `expected_text` |

## Targeted Refresh Results

These refresh candidates came from the stale historical issue #52 summary.

| Route | Status | Total | Setup navigation | Refresh load | Readiness polling | Session fetch | Signal |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `/reports/ticketing-human-work` | ok | 640 ms | 121 ms | 142 ms | 375 ms | 12 ms | `expected_text` |
| `/dashboard/it-admin` | ok | 630 ms | 123 ms | 148 ms | 356 ms | 14 ms | `expected_text` |
| `/admin` | ok | 612 ms | 112 ms | 142 ms | 356 ms | 14 ms | `expected_text` |
| `/phone-directory/by-room` | ok | 584 ms | 115 ms | 114 ms | 353 ms | 14 ms | `expected_text` |
| `/data-quality` | ok | 571 ms | 110 ms | 111 ms | 348 ms | 13 ms | `expected_text` |

## Classification

The current targeted evidence does not reproduce the historical multi-second slow rows.

- Browser measurement overhead / readiness polling: likely dominant for these targeted rows. Each row became ready by the fourth 100 ms readiness poll, so the fixed polling cadence accounts for roughly 343-390 ms of each transition and 348-375 ms of each refresh.
- API fetch: not a current bottleneck in these samples. DEV session fetch timing was 6-14 ms.
- Browser navigation/load: not a current bottleneck in these samples. Navigation/load timing was 111-148 ms.
- Generated artboard import/render: not proven as a current bottleneck by this targeted pass. The frontend generated-artboard import marker is present in the rows, but the marker duration can include warmup or overlapping work and should not be read as a standalone row total when it exceeds the measured route total. Use it as a signal to inspect in a full matrix or focused DevTools trace, not as proof of a current page-local bottleneck.
- React state churn: not indicated by the available evidence. The rows report two route-render commit marks for transitions and four for refreshes, with no console-log captures.

## Findings

1. The historical slow candidates are fast on the current `ui-improvements` branch in this targeted Browser pass. The slowest targeted transition was 520 ms and the slowest targeted refresh was 640 ms.
2. No targeted row exceeded the 3000 ms warning budget or 7000 ms failure budget.
3. The next full-matrix pass should still run before closing #48 or #52 because this pass does not cover every directed transition or both refresh samples for every route.
4. If future full-matrix evidence again shows multi-second rows, the first split should compare `readinessPollingMs` against `navigationLoadMs`; for this pass, readiness polling was the largest measured phase even when the page was otherwise healthy.

## Recommended Next Evidence Pass

Run a clean full Browser matrix from `origin/ui-improvements` or the current issue branch after PR #112 lands or is incorporated:

1. Confirm `curl -i http://localhost:5173/api/v1/dev/session` or the active Vite port returns `200 OK`.
2. Run `npm run perf:routes:plan` and record the route/transition counts.
3. Use the Browser helper from `README.md` until the automatic batch runner reports `"complete": true`.
4. Run `npm run perf:routes:merge:strict -- artifacts/performance`.
5. If the strict merge passes, use the merged Markdown as the closure evidence. If it fails only because the run is partial, keep the merged artifact local and summarize the incomplete coverage in the issue.

Do not open optimization work from stale historical artifacts alone. Open or update a follow-up only when current Browser rows isolate a repeatable bottleneck by route, phase, and likely ownership.
