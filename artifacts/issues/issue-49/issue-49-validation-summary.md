# Issue 49 Runtime Evidence

Captured: 2026-05-14T01:06:21.022Z

Browser fallback: Headless Chrome via Chrome DevTools Protocol because the in-app Browser endpoint listed no available browsers.

## Before Evidence Used

Issue #48 recorded slow generated-route transitions from the clean issue #9 route matrix, including /dashboard -> /departing-seniors at 6894 ms, /search?q=alex -> /reports/sync-transparency at 5825 ms, /offboarding -> /admin/feature-flags at 5043 ms, and additional generated static page transitions over 3000 ms. Baseline production build chunk output before this patch showed individual generated artboard chunks from 16.05 kB to 53.26 kB uncompressed, with reports-sync-transparency at 28.12 kB, admin-feature-flags at 16.05 kB, and frequent-fliers at 36.39 kB.

## After Evidence

IT Admin authorized artboard requests warmed after session load: 19.

| Route | After prefetch readiness |
| --- | ---: |
| /reports/sync-transparency | 101 ms |
| /admin/feature-flags | 315 ms |
| /frequent-fliers | 321 ms |

## Unauthorized Prefetch Check

Site Secretary allowed routes: /my-profile, /search, /student-data-cleanup, /room-moves, /room-moves/bulk-draft, /phone-directory/by-person, /phone-directory/by-room, /phone-directory/by-department

Site Secretary prefetched artboards: http://127.0.0.1:5179/src/generated/my-profile.artboard.json?import, http://127.0.0.1:5179/src/generated/phone-directory-by-department.artboard.json?import, http://127.0.0.1:5179/src/generated/phone-directory-by-person.artboard.json?import, http://127.0.0.1:5179/src/generated/phone-directory-by-room.artboard.json?import, http://127.0.0.1:5179/src/generated/room-moves-bulk-draft.artboard.json?import, http://127.0.0.1:5179/src/generated/room-moves.artboard.json?import, http://127.0.0.1:5179/src/generated/student-data-cleanup.artboard.json?import

Unauthorized admin/report/dashboard/frequent-flier artboard prefetches: 0

Direct /admin/feature-flags as Site Secretary returned 403: true

Screenshot: /Users/lcampbell/code.internal/accountsdot/.worktrees/issue-49-artboard-prefetch/artifacts/issues/issue-49/issue-49-it-admin-prefetched-route.png
