# DEV Route Performance Matrix

Generated: 2026-05-12T19:19:57.903Z
Base URL: http://localhost:5173
Routes: 24
Directed transitions: 552/552
Refresh samples: 24
Coverage valid: yes

## Slowest Transitions

| From | To | ms | Final URL |
| --- | --- | ---: | --- |
| `/student-data-cleanup` | `/my-profile` | 2997 | `http://localhost:5173/my-profile` |
| `/my-profile` | `/frequent-fliers` | 2405 | `http://localhost:5173/frequent-fliers` |
| `/my-profile` | `/reports` | 2197 | `http://localhost:5173/reports` |
| `/my-profile` | `/admin` | 2091 | `http://localhost:5173/admin` |
| `/admin` | `/my-profile` | 1407 | `http://localhost:5173/my-profile` |
| `/reports` | `/my-profile` | 1375 | `http://localhost:5173/my-profile` |
| `/dashboard` | `/my-profile` | 1309 | `http://localhost:5173/my-profile` |
| `/reports/ticketing-human-work` | `/my-profile` | 1233 | `http://localhost:5173/my-profile` |
| `/my-profile` | `/student-data-cleanup` | 1222 | `http://localhost:5173/student-data-cleanup` |
| `/reports/sync-transparency` | `/my-profile` | 1210 | `http://localhost:5173/my-profile` |

## Slowest Refreshes

| Route | Sample | ms | Final URL |
| --- | ---: | ---: | --- |
| `/admin` | 1 | 4117 | `http://localhost:5173/admin` |
| `/dashboard/it-admin` | 1 | 3801 | `http://localhost:5173/dashboard/it-admin` |
| `/search?q=alex` | 1 | 2848 | `http://localhost:5173/search?q=alex` |
| `/frequent-fliers` | 1 | 2318 | `http://localhost:5173/frequent-fliers` |
| `/phone-directory/by-room` | 1 | 2298 | `http://localhost:5173/phone-directory/by-room` |
| `/student-data-cleanup` | 1 | 2297 | `http://localhost:5173/student-data-cleanup` |
| `/phone-directory/by-person` | 1 | 2218 | `http://localhost:5173/phone-directory/by-person` |
| `/data-quality` | 1 | 2159 | `http://localhost:5173/data-quality` |
| `/reports` | 1 | 2109 | `http://localhost:5173/reports` |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | 1 | 2097 | `http://localhost:5173/room-moves/bulk-draft?draft_id=rm-draft-103` |

## Failures

Transition failures: 1
Refresh failures: 2

### Transition Failures

| From | To | Status | Error |
| --- | --- | --- | --- |
| `/my-profile` | `/reports/ticketing-human-work` | timeout | ready /reports/ticketing-human-work timed out after 10000ms |

### Refresh Failures

| Route | Sample | Status | Error |
| --- | ---: | --- | --- |
| `/dashboard/site-admin` | 1 | timeout | ready after refresh /dashboard/site-admin timed out after 10000ms |
| `/reports/ticketing-human-work` | 1 | timeout | ready after refresh /reports/ticketing-human-work timed out after 10000ms |
