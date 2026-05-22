# Documentation Index

This directory holds all repository documentation except the root `README.md` and Codex-facing agent instructions under `.agents/`.

## Product

- [Product requirements](product/product-requirements.md): business-facing product scope, users, workflows, access rules, and out-of-scope boundaries.
- [Permissions matrix](product/permissions-matrix.md): implemented DEV route/API permissions, field visibility, and known authorization gaps.
- [Permissions model](product/permissions-model.md): editable permissions model and future IT Admin access-management behavior.

## Planning

- [Implementation plan](planning/implementation-plan.md): authoritative implementation plan, decision log, phased delivery details, and current behavior contracts.
- [External write inventory](planning/external-write-inventory.md): live, planned, mock-only, and database write paths.
- [Route API authorization inventory](planning/route-api-authorization-inventory.md): route-to-API authorization coverage.

## Testing And Operations

- [Test matrix](testing/test-matrix.md): named scenario definitions and promotion evidence expectations.
- [Route performance evidence guide](testing/route-performance/profiling-evidence-guide.md): route profiling evidence workflow.
- [Environment data playbook](operations/environment-data-playbook.md): dev/staging/main data safety, masking, and refresh strategy.
- [Promotion pipeline](operations/promotion-pipeline.md): GitHub Actions branch gates and promotion PR workflow.

## Developer Guides

- [Code documentation guide](developer/code-documentation-guide.md): code-comment and write-path documentation expectations.
- [Phase 0 database schema review guide](developer/phase-0-database-schema.md): table, field, index, constraint, lifecycle, retry, lease, outbox, sensitive-field, and direct SQL review guidance for Phase 0 database inspection.
- [Code path walkthroughs](developer/code-paths/): high-risk workflow walkthroughs for onboarding, room moves, shared help, and sync overrides.

## Design And Reference Material

- [Design mocks](design/mocks/): `.pen` wireframes, annotation ledgers, fonts, and mock export tooling.
- [Implemented-page design contract](design/mocks/wireframes/implemented-page-design-contract.md): compact UI contract for implemented `.pen` pages.
- [Reference inputs](reference-inputs/): checked-in branding and vendored-input inventory.

## Agent Orchestration

- [Agent orchestration spec](agent-orchestration/SPEC.md): repo-local Symphony-style orchestration contract.
- [Agent orchestration README](agent-orchestration/README.md): overview of the orchestration docs.
- [Agent workflow](../.agents/WORKFLOW.md): runner-readable prompt and configuration contract.

## Stakeholder Materials

- [Accounts dashboard sales summary](stakeholder/accounts-dashboard-sales.md): practical value summary for district operations and leadership audiences.
