# Route Performance Profiling Evidence Guide

This guide explains how to turn the DEV route performance harness into useful profiling evidence for GitHub issues such as #48 and #52. It supplements the command reference in `README.md`; the README remains the source of truth for the supported harness commands and strict gate behavior.

## Evidence Boundaries

Use route-performance evidence to answer three separate questions:

1. Did every supported route transition and refresh become ready?
2. Which route rows are slow enough to exceed the current warning or failure budget?
3. Which measured phase explains the slow row closely enough to assign follow-up work?

Keep those answers separate. A partial targeted pass can identify candidates, but it is not closure evidence for route-readiness quality. A stale historical merge can guide the next pass, but it should not be used to justify an optimization unless the current route plan and current Browser rows reproduce the bottleneck.

## Recommended Capture Flow

Start from a clean branch based on the current integration branch for the UI workstream, usually `origin/ui-improvements` for the Pre-Phase 0 hardening track.

1. Start the DEV API with `APP_ENV=development`.
2. Start the Vite frontend.
3. Confirm the active Vite port can reach the DEV session endpoint:

   ```bash
   curl -i http://localhost:5173/api/v1/dev/session
   ```

4. Confirm the route plan:

   ```bash
   npm run perf:routes:plan
   ```

5. Use the Codex Browser skill with the active Browser tab object and run the automatic batch helper from `README.md`.
6. Repeat the helper until it returns `"complete": true`.
7. Merge the artifacts:

   ```bash
   npm run perf:routes:merge -- artifacts/performance
   npm run perf:routes:merge:strict -- artifacts/performance
   ```

Use non-strict merge for handoff context. Use strict merge for closure evidence.

## Targeted Candidate Capture

When the full matrix is too expensive for a first pass, use targeted rows only to decide what to run next. Targeted rows should include:

- the route-plan commit or branch,
- the active base URL,
- the DEV session preflight result,
- transition indexes or refresh route/sample keys,
- `elapsedMs`,
- `navigationLoadMs`,
- `readinessPollingMs`,
- `frontendSessionFetchMs`,
- generated-artboard import/render markers when present,
- console-log capture count or the relevant sanitized messages,
- the local artifact directory.

Targeted evidence should clearly say that it is not full closure evidence unless it covers the whole directed-transition and refresh plan and passes the strict merge gate.

## Phase Interpretation

Classify a slow row by the measured phase that explains the delay.

| Likely source | Evidence pattern | Follow-up owner |
| --- | --- | --- |
| Browser measurement overhead | `readinessPollingMs` dominates and rows become ready after a small fixed number of 100 ms polls | Harness/readiness contract |
| API fetch | `frontendSessionFetchMs` or route-specific API timing dominates | Runtime route/API boundary |
| Lazy chunk load | `navigationLoadMs` or Browser network evidence shows a late JavaScript module request | Runtime route bundling |
| Generated artboard import | `frontendGeneratedArtboardImportMs` repeats as a large current-row phase and Browser network evidence shows generated artboard chunks loading late | Generated artboard loader or prefetch policy |
| Generated artboard render | generated-artboard render marks repeat without matching import delay | `.pen` geometry or generated renderer behavior |
| React state churn | route-render commit counts are high or DevTools profiler shows repeated commits after data is ready | Runtime component state |
| Browser transport failure | logs or row status mention Browser pipe interruption, missing `iab`, or blocked automation | Evidence environment, not app behavior |

Generated-artboard import markers can overlap with prefetch or warmup work. If the marker duration is larger than the route row total, treat it as a signal to inspect rather than as the row's bottleneck by itself.

## Durable Reporting

For a durable issue or PR summary, include:

- current route count and directed-transition count,
- whether the DEV preflight was healthy,
- whether the evidence is full-matrix or targeted,
- the slowest transition rows,
- the slowest refresh rows,
- phase-level classification for each candidate,
- any budget warnings or failures,
- exact local artifact paths for raw JSON/Markdown,
- the reason raw artifacts were or were not committed.

Commit a curated Markdown summary only when it is small, non-sensitive, and useful for future contributors. Keep raw Browser artifacts local unless the issue or PR explicitly asks for a small curated artifact in the repository.
