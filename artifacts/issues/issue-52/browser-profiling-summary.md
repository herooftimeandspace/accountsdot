# Issue 52 Browser Profiling Summary

Captured: 2026-05-14T19:26:12Z

## Scope

This pass targeted GitHub issue #52, "Capture Browser profiling evidence for slowest pages." The goal was to collect non-sensitive route-performance evidence for the current slowest implemented pages without making optimization code changes.

## Current Route And Build Evidence

`npm run perf:routes:plan` passed on `origin/dev` at `2e42d95b2e29c0a2f7eb345dedc02416bb4baf93`.

- Current route count: 25
- Current directed-transition count: 600
- Default transition budget: warning over 3000 ms, failure over 7000 ms
- Default refresh budget: warning over 3000 ms, failure over 7000 ms

`npm run build:web` passed. The largest emitted production chunks were generated/static artboard modules and the main application bundle:

| Asset | Size | Gzip |
| --- | ---: | ---: |
| `index-C29pLrhf.js` | 198.91 kB | 62.43 kB |
| `dashboard-site-admin.artboard-DuMLQ2F3.js` | 53.26 kB | 5.78 kB |
| `dashboard-it-admin.artboard-BjU-2Auj.js` | 44.97 kB | 5.00 kB |
| `admin.artboard-COw-IxfQ.js` | 42.86 kB | 5.31 kB |
| `dashboard-hr-lifecycle.artboard-DG4BifgQ.js` | 41.58 kB | 4.76 kB |
| `reports.artboard-DQXNP6U9.js` | 38.44 kB | 4.54 kB |
| `phone-directory-by-department.artboard-CHhYM8Hc.js` | 37.30 kB | 4.64 kB |
| `frequent-fliers.artboard-BMUsT7-X.js` | 36.39 kB | 4.61 kB |

## Browser Availability

Local DEV startup and documented preflight succeeded:

- API command: `APP_ENV=development GOCACHE=.gocache GOMODCACHE=.gomodcache npm run dev:api`
- Web command: `npm run dev:web`
- Preflight: `curl -i http://localhost:5173/api/v1/dev/session`
- Preflight result: `200 OK` with DEV session JSON and persona list

The Codex Browser plugin was installed, but no in-app Browser target was available in this session. `agent.browsers.get("iab")` failed with `Browser is not available: iab`. Because the Browser tab object could not be acquired, the route matrix Browser helper could not produce fresh current transition rows, screenshots, network timing, render timing, console marks, or readiness-polling phase data.

## Documented Fallback Evidence

`npm run perf:routes:batch-plan -- artifacts/performance` confirmed there are no current-plan local Browser artifacts to resume:

- Matching artifact count: 0
- Remaining transitions: 600 of 600
- Remaining refresh samples: 50 of 50
- Next transition batch: start index 1, max 50

`npm run perf:routes:merge -- artifacts/performance` merged the historical tracked artifacts for context only. The merged artifact is not current closure evidence because it records 24 routes and 552 directed transitions, while the current route plan records 25 routes and 600 directed transitions. The merge reported `staleCurrentRoutePlan: true`, `routeCountMismatch: true`, and `directedEdgeCountMismatch: true`.

Historical context from the stale merge still identifies these slow rows:

| Classification | Route / Edge | Timing |
| --- | --- | ---: |
| historical budget failure | `/offboarding` -> `/data-quality` | 7643 ms |
| historical budget failure | `/offboarding` -> `/reports/ticketing-human-work` | 7268 ms |
| historical budget warning | `/room-moves/bulk-draft?draft_id=rm-draft-103` -> `/data-quality` | 4524 ms |
| historical refresh warning | `/admin` refresh | 4117 ms |
| historical refresh warning | `/phone-directory/by-room` refresh | 4089 ms |
| historical refresh warning | `/dashboard/it-admin` refresh | 3801 ms |

Those rows should be treated as target candidates for the next Browser-enabled pass, not as current proof.

## Adjacent Issue Evidence

- Issue #49 / PR #66 already captured fallback runtime evidence for generated-artboard prefetching. After authorized-route prefetch, `/reports/sync-transparency` was ready in 101 ms, `/admin/feature-flags` in 315 ms, and `/frequent-fliers` in 321 ms. The validation also confirmed Site Secretary did not prefetch unauthorized admin/report/dashboard/frequent-flier artboards and still received 403 for direct `/admin/feature-flags`.
- Issue #50 / PR #67 added performance-budget reporting. The current issue #52 run consumed those default budget thresholds from `npm run perf:routes:plan`.
- Issue #51 / PR #68 added phase-timing fields and summaries. No fresh Browser rows were captured in this issue #52 pass, so there are no current phase rows to classify.

## Findings And Recommended Path

No optimization code change is warranted from this issue #52 pass because the required Browser target was unavailable and current DevTools-style evidence could not be captured.

Recommended next Browser-enabled profiling targets:

1. Run the full Browser route-performance helper from a clean `artifacts/performance` directory against the current 25-route / 600-transition plan.
2. Prioritize edges involving `/data-quality`, `/reports/ticketing-human-work`, `/admin`, `/phone-directory/by-room`, and `/dashboard/it-admin`, because stale historical rows and current bundle output both point at generated/static artboard plus runtime-page combinations.
3. Use the issue #51 phase fields to distinguish `navigationLoadMs`, `readinessPollingMs`, `frontendSessionFetchMs`, and `frontendGeneratedArtboardImportMs` before opening any new optimization work.
4. Treat issue #49 / PR #66 as the current generated-artboard prefetch evidence source. Do not add another prefetch optimization from issue #52 unless fresh Browser rows show a remaining generated-artboard import bottleneck after that PR's behavior is on the tested branch.
