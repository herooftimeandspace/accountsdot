# Accounts Dashboard: Practical Value for District Operations

![Accounts Dashboard day-to-day workflow](docs/sales-assets/accounts-dashboard-day-to-day.svg)

## Executive Summary

The Accounts dashboard gives district teams one staff-only place to see account, room, phone, onboarding, offboarding, sync, and exception work. The goal is not another reporting layer. The goal is a daily operating surface that shows each team what is ready, what is blocked, who owns the next step, and where work has already been completed.

For leadership, this improves confidence that staff access and operational handoffs are being handled consistently. For site admin staff, it reduces the need to chase status through spreadsheets, email, and separate ticket threads.

## What Changes Day To Day

| Audience | How work improves |
| --- | --- |
| Executive leadership | Gets a clearer view of operational readiness, rollout risk, and legacy-process retirement without needing to inspect individual spreadsheets. |
| HR | Can see district-wide onboarding, offboarding, reactivation, and personnel-data blockers in one place, including which records need upstream correction. |
| Site admin staff | Sees only the site-scoped staff, room, phone, onboarding, and offboarding status needed for daily campus work. |
| Site secretaries | Gets targeted queues for source-data fixes, room-move participation, and student invalid-name correction paths. |
| Device wranglers and librarians | Moves Frequent Fliers from a static report into a live site-scoped queue with student, device, and ticket context together. |
| IT | Automates the common path while keeping exceptions, provider delays, and safety controls visible and recoverable. |

## The Short Solution

The dashboard replaces manual side channels with a role-scoped operational view:

- By-person onboarding and offboarding status replaces People Tracker-style spreadsheet chasing.
- Live phone-directory views replace CSV and Google Sheet rebuilds.
- Room moves and phone corrections start from the same context site staff already uses.
- Sync readiness shows whether Aeries, Incident IQ, photos, rooms, and Zoom are ready before work is treated as complete.
- Exceptions are routed to the team that can actually resolve them instead of becoming hidden IT follow-up.

## Why This Matters

Today, the work already happens. The issue is that the status is spread across systems, spreadsheets, exports, tickets, and personal follow-up. The dashboard improves daily work by making the current state visible, making blockers explicit, and keeping routine work out of email whenever possible.

That means:

- New hires are easier to track from intake through account readiness.
- Offboarding has a clearer closeout trail for accounts, devices, phones, access, and related tasks.
- Site staff can answer their own scoped status questions faster.
- Phone and room changes can be planned from current provider-backed data.
- Leadership can support legacy-sheet retirement with evidence instead of anecdotes.

## Implementation In Plain Terms

The repo describes a self-hosted intranet application with a Go service, PostgreSQL-backed workflow state, staff-only access, health checks, sync visibility, and phased rollout controls. The current implemented web surface includes sync dashboard pages, workflow and approval APIs, room-mapping endpoints, health endpoints, and event-stream support. The broader product plan extends that foundation into onboarding, offboarding, phone directory, room moves, student invalid-name handling, Frequent Fliers, and safe legacy cutover.

![Accounts Dashboard connected systems](docs/sales-assets/accounts-dashboard-systems.svg)

![Accounts Dashboard rollout path](docs/sales-assets/accounts-dashboard-rollout.svg)

## Recommended Positioning

Use this as the district's operational account-readiness dashboard, not as a technical automation project. The message to site staff should be simple:

> Go to the dashboard to see what is ready, what needs attention, and what action belongs to your site.

The message to leadership should be:

> This gives us a controlled path to reduce spreadsheet dependency, improve account lifecycle consistency, and retire manual processes only after parity is proven.

## Source Reference Map

This summary is based on the repo's primary Markdown documents:

- [README.md](README.md): project goals, local app scope, documentation policy.
- [PRODUCT_REQUIREMENTS.md](PRODUCT_REQUIREMENTS.md): business-facing product scope, user roles, current-pass boundaries, and core dashboard areas.
- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md): phased implementation details, workflow behavior, provider rules, sync dashboard design, and legacy-process migration notes.
- [TEST_MATRIX.md](TEST_MATRIX.md): named verification scenarios and promotion expectations by phase.
- [ENVIRONMENT_DATA_PLAYBOOK.md](ENVIRONMENT_DATA_PLAYBOOK.md): safe development and staging data strategy.
- [docs/reference-inputs/README.md](docs/reference-inputs/README.md) and [docs/reference-inputs/VENDORED_INVENTORY.md](docs/reference-inputs/VENDORED_INVENTORY.md): provenance for supporting integration, branding, and legacy-process reference material.
