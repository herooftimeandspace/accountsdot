# DEV Browser Hang Report
Source artifact: artifacts/performance/dev-route-performance-2026-05-12T19-36-40-604Z.json
Generated: 2026-05-12T19:40:39.188Z
## Browser Break Point
The Browser pipe first failed at directed transition index 372: `/phone-directory/by-person` -> `/room-moves/bulk-draft?draft_id=rm-draft-103`.
Error: Browser turn does not belong to this IAB pipe
## Coverage Before Pipe Failure
Routes in source artifact: 24
Expected directed transitions: 552
Transitions attempted: 552
Valid transition rows before/around failure: 371
Invalid transition rows: 181
Browser pipe transition rows: 181
Browser pipe refresh rows: 24
## Slowest Valid Transitions Before Failure
| From | To | ms |
| --- | --- | ---: |
| `/offboarding` | `/data-quality` | 7643 |
| `/offboarding` | `/reports/ticketing-human-work` | 7268 |
| `/room-moves/bulk-draft?draft_id=rm-draft-103` | `/data-quality` | 4524 |
| `/reports/ticketing-human-work` | `/admin` | 3597 |
| `/dashboard/it-admin` | `/reports/sync-transparency` | 3297 |
| `/reports/sync-transparency` | `/room-moves` | 3175 |
| `/student-data-cleanup` | `/search?q=alex` | 3034 |
| `/departing-seniors` | `/admin` | 3000 |
| `/search?q=alex` | `/data-quality` | 2667 |
| `/my-profile` | `/room-moves/bulk-draft` | 2505 |
## Resume Instruction
Restart the Browser automation session, then resume at transition index 372 with `startTransitionIndex: 372`. The resumable harness now stops immediately on Browser pipe failure instead of continuing to pollute the artifact.
