# DEV Route Performance Matrix

Generated: 2026-05-12T19:52:38.822Z
Base URL: http://localhost:5173
Routes: 24
Directed transitions: 552/552
Refresh samples: 49
Coverage valid: yes
Transition range: 1-end
Next transition index: 1
Stop reason: browser_pipe_failure
First Browser pipe failure: /dashboard -> refresh
Resume from transition index: 1

## Runbook

Exact failed edge: `/dashboard` -> `refresh`
Last valid edge: none
Next resume index: 1
Dev server healthy: unknown
Browser session restart needed: yes

## Failure Classes

Transition classes: `{}`
Refresh classes: `{"ok":49}`

## Slowest Transitions

| From | To | ms | Final URL |
| --- | --- | ---: | --- |

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

Transition failures: 0
Refresh failures: 3

### Refresh Failures

| Route | Sample | Status | Class | Error |
| --- | ---: | --- | --- | --- |
| `/dashboard/site-admin` | 1 | timeout |  | ready after refresh /dashboard/site-admin timed out after 10000ms |
| `/reports/ticketing-human-work` | 1 | timeout |  | ready after refresh /reports/ticketing-human-work timed out after 10000ms |
| `/dashboard` | 1 | error |  | Browser turn does not belong to this IAB pipe |
