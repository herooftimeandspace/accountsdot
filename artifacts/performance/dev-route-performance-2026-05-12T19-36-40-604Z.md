# DEV Route Performance Matrix

Generated: 2026-05-12T19:36:40.604Z
Base URL: http://localhost:5173
Routes: 24
Directed transitions: 552/552
Refresh samples: 24
Coverage valid: yes

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
| `/search?q=alex` | `/data-quality` | 2667 | `http://localhost:5173/data-quality` |
| `/my-profile` | `/room-moves/bulk-draft` | 2505 | `http://localhost:5173/room-moves/bulk-draft` |

## Slowest Refreshes

| Route | Sample | ms | Final URL |
| --- | ---: | ---: | --- |

## Failures

Transition failures: 181
Refresh failures: 24

### Transition Failures

| From | To | Status | Error |
| --- | --- | --- | --- |
| `/phone-directory/by-person` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/phone-directory/by-person` | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/room-moves/bulk-draft?draft_id=rm-draft-103` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/room-moves/bulk-draft?draft_id=rm-draft-101` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/room-moves/bulk-draft` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/room-moves` | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/departing-seniors` | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/offboarding` | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/onboarding` | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/search?q=alex` | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/search` | error | Browser turn does not belong to this IAB pipe |
| `/search` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/dashboard/site-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/dashboard/hr-lifecycle` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard` | `/dashboard/it-admin` | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | `/dashboard` | error | Browser turn does not belong to this IAB pipe |

### Refresh Failures

| Route | Sample | Status | Error |
| --- | ---: | --- | --- |
| `/dashboard` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/it-admin` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/hr-lifecycle` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/dashboard/site-admin` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/search` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/search?q=alex` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/onboarding` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/offboarding` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/departing-seniors` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/room-moves` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-101` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-person` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-department` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/phone-directory/by-room` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/data-quality` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/frequent-fliers` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/student-data-cleanup` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/reports` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/reports/sync-transparency` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/reports/ticketing-human-work` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/admin` | 1 | error | Browser turn does not belong to this IAB pipe |
| `/my-profile` | 1 | error | Browser turn does not belong to this IAB pipe |
