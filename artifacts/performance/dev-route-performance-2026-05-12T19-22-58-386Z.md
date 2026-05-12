# DEV Route Performance Matrix

Generated: 2026-05-12T19:22:58.386Z
Base URL: http://localhost:5173
Routes: 24
Directed transitions: 552/552
Refresh samples: 24
Coverage valid: yes

## Slowest Transitions

| From | To | ms | Final URL |
| --- | --- | ---: | --- |
| `/reports` | `/my-profile` | 4350 | `http://localhost:5173/my-profile` |
| `/my-profile` | `/frequent-fliers` | 2521 | `http://localhost:5173/frequent-fliers` |
| `/my-profile` | `/student-data-cleanup` | 2224 | `http://localhost:5173/student-data-cleanup` |
| `/my-profile` | `/reports` | 1409 | `http://localhost:5173/reports` |
| `/my-profile` | `/reports/sync-transparency` | 1379 | `http://localhost:5173/reports/sync-transparency` |
| `/my-profile` | `/admin` | 1361 | `http://localhost:5173/admin` |
| `/admin` | `/my-profile` | 1273 | `http://localhost:5173/my-profile` |
| `/reports/sync-transparency` | `/my-profile` | 1270 | `http://localhost:5173/my-profile` |
| `/student-data-cleanup` | `/my-profile` | 1236 | `http://localhost:5173/my-profile` |
| `/my-profile` | `/reports/ticketing-human-work` | 1235 | `http://localhost:5173/reports/ticketing-human-work` |

## Slowest Refreshes

| Route | Sample | ms | Final URL |
| --- | ---: | ---: | --- |
| `/phone-directory/by-room` | 1 | 4089 | `http://localhost:5173/phone-directory/by-room` |
| `/frequent-fliers` | 1 | 2970 | `http://localhost:5173/frequent-fliers` |
| `/student-data-cleanup` | 1 | 2946 | `http://localhost:5173/student-data-cleanup` |
| `/room-moves` | 1 | 2546 | `http://localhost:5173/room-moves` |
| `/dashboard/hr-lifecycle` | 1 | 2284 | `http://localhost:5173/dashboard/hr-lifecycle` |
| `/reports/ticketing-human-work` | 1 | 2153 | `http://localhost:5173/reports/ticketing-human-work` |
| `/reports` | 1 | 2151 | `http://localhost:5173/reports` |
| `/onboarding` | 1 | 2111 | `http://localhost:5173/onboarding` |
| `/dashboard/it-admin` | 1 | 2098 | `http://localhost:5173/dashboard/it-admin` |
| `/dashboard` | 1 | 2094 | `http://localhost:5173/dashboard/it-admin` |

## Failures

Transition failures: 0
Refresh failures: 0
