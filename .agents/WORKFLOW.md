---
name: wizard-symphony-workflow
version: 1
tracker:
  kind: github
  repository: herooftimeandspace/accountsdot
  active_labels:
    - agent-ready
  blocked_labels:
    - blocked
    - agent-blocked
    - human-only
    - security-sensitive
    - production-write
  review_label: agent-review
branching:
  integration_branch: dev
  branch_prefix: codex/
  branch_template: codex/issue-{number}-{slug}
workspace:
  root_env: ACCOUNTSDOT_WORKTREE_ROOT
  root_default: ../
  preserve_after_success: true
dispatch:
  poll_interval_seconds: 300
  max_concurrent_runs: 1
  max_attempts: 4
  require_explicit_agent_ready_label: true
verification:
  docs_only:
    - markdown review
  go_behavior:
    - make test-unit
  frontend_runtime:
    - npm run build:web
  implemented_page_layout:
    - npm run pen:check
    - npm run pen:lint
    - npm run build:web
    - npm run a11y:check
safety:
  allow_production_writes: false
  allow_secret_values_in_prompts: false
  allow_destructive_git_without_human_instruction: false
  allow_frontend_dist_commits_by_default: false
---

# WIZARD Symphony Agent Workflow

You are working as a Codex agent on The WIZARD repository. Your task is to complete the assigned GitHub issue in an isolated workspace, produce a reviewable branch or handoff, and preserve the repository's safety and documentation standards.

## Source Order

Before changing files, read the repository sources in this order:

1. `README.md`
2. `.agents/AGENTS.md`
3. `docs/planning/implementation-plan.md`
4. `docs/product/product-requirements.md`
5. `docs/testing/test-matrix.md` when scenarios, verification, or promotion evidence are affected
6. `docs/operations/environment-data-playbook.md` when environment data, masking, staging, production data, or promotion safety are affected
7. `docs/agent-orchestration/SPEC.md`
8. the assigned GitHub issue, its comments, linked PRs, and acceptance criteria

Use the most specific checked-in source that applies. If code and docs disagree, verify current behavior and update the relevant durable docs when the task requires alignment.

## Startup Steps

1. Confirm the current branch and working tree status.
2. Fetch the latest integration branch.
3. Work from `dev` unless the issue or repo docs explicitly require another integration branch.
4. Use branch `codex/issue-<number>-<short-slug>`.
5. Inspect open issues and linked context before implementation.
6. Identify likely file ownership and conflict risk.
7. Do not touch unrelated generated output or user-authored changes.

## Implementation Rules

- Prefer small, reversible changes that match existing patterns.
- Work test-first for behavior changes when feasible.
- Update `docs/planning/implementation-plan.md`, `docs/product/product-requirements.md`, `docs/testing/test-matrix.md`, `docs/operations/environment-data-playbook.md`, or README when behavior, access scope, data handling, rollout expectations, setup, or operator workflows change.
- Keep product scope grounded in checked-in docs and issue acceptance criteria.
- Do not invent new UI affordances, queues, controls, workflow states, or provider behavior without documentation support.
- For implemented-page UI work, update authoritative `.pen` files first and run the `.pen` sync/check workflow. Do not hand-edit generated artboards, generated presentational components, or generated review exports.
- Preserve source-system truth in imported data.
- Use inclusive, precise terminology.
- Keep operator-facing text English-only.

## Safety Rules

- Never copy secrets, tokens, auth headers, private keys, passwords, client secrets, or raw service-account JSON into code, docs, prompts, logs, fixtures, issues, PRs, or generated artifacts.
- Do not perform production provider writes.
- Do not exercise a new write path first in production.
- Do not bypass dev/staging/main promotion gates.
- Do not run destructive git commands unless a human explicitly requested them in the active session.
- Do not commit `frontend/dist/` unless the assigned issue explicitly changes generated artifact policy.
- Stop and ask for human direction if the issue requires credentials, production data, security-sensitive judgment, incompatible product decisions, or destructive actions.

## Verification Rules

Run the narrowest meaningful checks for the files and behavior you changed. Broaden checks when shared behavior, provider contracts, frontend access, or promotion risk is involved.

Use these defaults:

- Docs-only: inspect the Markdown diff and links. State that code tests were not run because only documentation changed.
- Go behavior: run targeted package tests or `make test-unit`; run broader tests when shared orchestration, provider contracts, or web behavior changes.
- Frontend runtime: run `npm run build:web` and targeted accessibility or route checks when UI behavior changes.
- Implemented-page `.pen` work: run `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check` unless the issue justifies a narrower set.
- Dependency changes: run the relevant tests plus dependency health and security checks.

Do not claim verification you did not run.

## Handoff Requirements

Finish each run with a concise handoff that includes:

- issue number and title
- branch name
- changed files
- summary of the implementation
- docs updated
- verification run and results
- verification not run, with reason
- safety notes or known risks
- follow-up issues created or recommended

If you opened or updated a PR, include the PR URL. If the work is incomplete, mark the issue blocked and explain exactly what decision or access is needed.
