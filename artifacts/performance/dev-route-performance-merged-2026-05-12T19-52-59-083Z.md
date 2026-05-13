# DEV Route Performance Matrix

Generated: 2026-05-12T19:52:59.083Z
Base URL: http://localhost:5173
Routes: 24
Directed transitions: 552/552
Refresh samples: 49
Coverage valid: yes
Transition range: 1-372
Next transition index: 372
Stop reason: browser_pipe_failure
First Browser pipe failure: /phone-directory/by-person -> /room-moves/bulk-draft?draft_id=rm-draft-103
Resume from transition index: 372

## Runbook

Exact failed edge: `/phone-directory/by-person` -> `/room-moves/bulk-draft?draft_id=rm-draft-103`
Last valid edge: `/dashboard` -> `/phone-directory/by-person`
Next resume index: 372
Dev server healthy: unknown
Browser session restart needed: yes

## Failure Classes

Transition classes: `{"ok":370,"app_timeout":1,"browser_pipe_failure":1}`
Refresh classes: `{"ok":46,"app_timeout":2,"browser_pipe_failure":1}`

## Slowest Transitions

| From | To | ms | Final URL |
| --- | --- | ---: | --- |
| `/offboarding` | `/data-quality` | 7643 | `http://localhost:5173/data-quality` |
| `/offboarding` | `/reports/ticketing-human-work` | 7268 | `http://localhost:5173/reports/ticketing-human-work` |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/data-quality` | 4524 | `http://localhost:5173/data-quality` |
| `/reports/ticketing-human-work` | `/admin` | 3597 | `http://localhost:5173/admin` |
| `/dashboard/it-admin` | `/reports/sync-transparency` | 3297 | `http://localhost:5173/reports/sync-transparency` |
| `/reports/sync-transparency` | `/room-moves` | 3175 | `http://localhost:5173/room-moves` |
| `/student-data-cleanup` | `/search?q=alex` | 3034 | `http://localhost:5173/search?q=alex` |
| `/departing-seniors` | `/admin` | 3000 | `http://localhost:5173/admin` |
| `/student-data-cleanup` | `/my-profile` | 2997 | `http://localhost:5173/my-profile` |
| `/search?q=alex` | `/data-quality` | 2667 | `http://localhost:5173/data-quality` |

## Slowest Refreshes

| Route | Sample | ms | Final URL |
| --- | ---: | ---: | --- |
| `/admin` | 1 | 4117 | `http://localhost:5173/admin` |
| `/phone-directory/by-room` | 1 | 4089 | `http://localhost:5173/phone-directory/by-room` |
| `/dashboard/it-admin` | 1 | 3801 | `http://localhost:5173/dashboard/it-admin` |
| `/frequent-fliers` | 1 | 2970 | `http://localhost:5173/frequent-fliers` |
| `/student-data-cleanup` | 1 | 2946 | `http://localhost:5173/student-data-cleanup` |
| `/search?q=alex` | 1 | 2848 | `http://localhost:5173/search?q=alex` |
| `/room-moves` | 1 | 2546 | `http://localhost:5173/room-moves` |
| `/frequent-fliers` | 1 | 2318 | `http://localhost:5173/frequent-fliers` |
| `/phone-directory/by-room` | 1 | 2298 | `http://localhost:5173/phone-directory/by-room` |
| `/student-data-cleanup` | 1 | 2297 | `http://localhost:5173/student-data-cleanup` |

## Failures

Transition failures: 2
Refresh failures: 3

### Transition Failures

| Index | From | To | Status | Class | Error |
| ---: | --- | --- | --- | --- | --- |
| 4 | `/my-profile` | `/reports/ticketing-human-work` | timeout | app_timeout | ready /reports/ticketing-human-work timed out after 10000ms |
| 372 | `/phone-directory/by-person` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | browser_pipe_failure | Browser turn does not belong to this IAB pipe |

### Refresh Failures

| Route | Sample | Status | Class | Error |
| --- | ---: | --- | --- | --- |
| `/dashboard/site-admin` | 1 | timeout | app_timeout | ready after refresh /dashboard/site-admin timed out after 10000ms |
| `/reports/ticketing-human-work` | 1 | timeout | app_timeout | ready after refresh /reports/ticketing-human-work timed out after 10000ms |
| `/dashboard` | 1 | error | browser_pipe_failure | Browser turn does not belong to this IAB pipe |
