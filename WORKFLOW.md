---
name: accountsdot-ui-improvements-symphony
base_branch: ui-improvements
issue_tracker: github
default_branch_prefix: codex/
worktree_root_env: ACCOUNTSDOT_WORKTREE_ROOT
worktree_root_default: ../
requires_external_lock: true
---

# Agent Workflow Contract

This file is the repo-owned workflow prompt/config contract for Codex agents coordinating issue work in The WIZARD. It adapts the Symphony pattern to this repository by treating GitHub issues as the durable control plane and by keeping all mutation behind repo-documented safety gates.

## Required Prompt Contract

Every agent run that claims or reconciles GitHub issue work must include:

- Repository: `herooftimeandspace/accountsdot`.
- Integration branch: `ui-improvements` for the current UI-improvements track unless the issue or repository docs name a different branch.
- Source of truth order: `AGENTS.md`, `README.md`, `IMPLEMENTATION_PLAN.md`, `PRODUCT_REQUIREMENTS.md`, `TEST_MATRIX.md`, and the specific issue or PR under work.
- Safety boundary: no production/provider writes, credentials, destructive git operations, or latest-code resets unless the task explicitly authorizes that action and the repo docs allow it.
- Output expectation: branch name, PR URL when opened, linked issue numbers, files changed, verification commands, unresolved blockers, and the next handoff state.

## Branch And Worktree Contract

Use one branch per clean unit of issue work.

- Branch names should include the issue number and a short slug, for example `codex/issue-238-symphony-orchestrator`.
- Worktrees should live outside the main checkout. Use `$ACCOUNTSDOT_WORKTREE_ROOT` when it is set; otherwise create sibling worktrees relative to the repository checkout, for example `../accountsdot-issue-<number>-<slug>`.
- Start branches from `origin/ui-improvements` unless the issue documents a different base.
- Do not edit generated build output such as `frontend/dist/`.
- Do not reuse another issue branch unless the issue thread explicitly says the work is intentionally combined.

## Claim And Handoff States

Use GitHub issue comments and PR bodies for durable handoff. Avoid chat-only coordination.

- `unclaimed`: no active branch or PR is linked.
- `claimed`: an agent comment names the branch/worktree and intended scope.
- `in_progress`: commits exist locally or remotely and verification is underway.
- `pr_open`: a PR exists and names the issue scope, checks, and blockers.
- `blocked`: work cannot continue without a decision, permission fix, failing dependency, conflict resolution, or runtime evidence.
- `ready_for_review`: local checks passed and the PR is non-draft.
- `merged`: the PR merged into the integration branch.
- `closed`: the linked issue was closed only after merged work fully satisfied acceptance criteria.

## Retry And Reconciliation Contract

Retry only deterministic, bounded operations.

- Fetch/prune before branch or PR reconciliation.
- Rebase clean branches onto latest `origin/ui-improvements` when the worktree is clean and conflicts are not product, data, auth, security, migration, deployment, or docs decisions.
- Stop and report when git metadata permissions prevent rebase or push.
- Stop and report dirty latest-code worktrees instead of discarding local edits.
- Use draft PRs when acceptance criteria require Browser/runtime evidence that cannot currently be collected.

## Verification Contract

Use the narrowest repo-documented gate that covers the files changed.

- Documentation-only changes: `git diff --check` plus any parser or doc checker that applies to the edited file format.
- Frontend/runtime changes: `npm run build:web`, `npm run a11y:check`, and route-specific Browser evidence when available.
- `.pen` design changes: `npm run pen:sync`, `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check`.
- Go behavior changes: targeted `go test`, then the relevant `make` gate when feasible.
- External-write surfaces: update `docs/external-write-inventory.md` and run `npm run write-inventory:check`.

Do not claim a check passed unless it was run in the current branch or the PR clearly cites equivalent current evidence.
