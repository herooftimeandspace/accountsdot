# Symphony-Style Agent Orchestration Specification

## Purpose

The WIZARD uses GitHub issues as the durable control plane for Codex agent work. This specification adapts OpenAI's Symphony orchestration pattern to this repository without adding a background runner. It defines how agents discover eligible work, claim it, isolate implementation, verify results, reconcile conflicts, and hand off status in a way that survives individual chat threads.

## Non-Goals

- This specification does not create a daemon, queue worker, scheduler, or service account.
- This specification does not authorize production writes, provider writes, credential handling, or destructive git operations.
- This specification does not replace `AGENTS.md`, branch protection, code review, promotion gates, or the `dev -> staging -> main` release process.
- This specification does not make `frontend/dist/` or local evidence directories tracked artifacts.

## Source Of Truth

Agents must read the most specific applicable source before changing behavior.

1. `AGENTS.md`
2. `README.md`
3. `IMPLEMENTATION_PLAN.md`
4. `PRODUCT_REQUIREMENTS.md`
5. `TEST_MATRIX.md`
6. Workflow-specific docs such as `docs/permissions-matrix.md`, `docs/external-write-inventory.md`, design ledgers, and code-path guides
7. The GitHub issue and related PR thread

If documentation and code disagree, verify current behavior and update durable docs when the task changes behavior, setup, interfaces, data handling, rollout expectations, or operator workflows.

## Issue Eligibility

An issue is eligible for autonomous agent work when all of the following are true:

- The issue has clear acceptance criteria or a narrow, inferable documentation gap.
- The work can be isolated to one branch and one PR.
- The affected files are not already owned by another active branch unless the issue thread documents shared ownership.
- The work does not require an unmade product, data, auth, security, migration, deployment, or release-policy decision.
- Required credentials, production data, or provider-side write access are not needed.

An issue is not eligible for autonomous implementation when:

- It asks for a product decision that is not documented.
- It requires production/provider writes or real tenant mutation.
- It requires replacing user-authored local changes.
- It would need broad cross-cutting edits across active PRs without a planned integration pass.
- It depends on Browser/runtime evidence and the local latest-code worktree is dirty or dev servers are unhealthy. In that case, a draft PR may be acceptable only if the remaining evidence gap is explicit.

## Claiming Work

Before implementation, an agent should:

- Fetch/prune remotes.
- Inspect open issues and open PRs targeting the integration branch.
- Search for existing branches or PRs for the issue.
- Comment on the issue when beginning material work, naming the branch, expected files, and known overlap.

Do not create duplicate branches when a live PR already owns the issue. If a prior comment claims a branch but no branch exists locally or remotely and no PR exists, a new branch may be created from the current integration branch and the issue comment should explain the reconciliation.

## Branch And Worktree Rules

- Base issue branches on `origin/ui-improvements` for the current UI-improvements track.
- Prefer branch names like `codex/issue-238-symphony-orchestrator`.
- Use one worktree per branch outside the main checkout. Agents should honor `$ACCOUNTSDOT_WORKTREE_ROOT` when it is set; otherwise they should use a sibling path such as `../accountsdot-issue-238-symphony-orchestrator` relative to the repository checkout.
- Keep unrelated issues on separate branches.
- Do not edit the main checkout when it has unrelated local changes.
- Do not hand-edit generated artboards or generated build output.
- Do not force-push over unknown user work.

## Safety Gates

The orchestration contract inherits this repository's environment safety model.

- `dev` and local mocks are the normal place for new write paths to be exercised.
- `staging` is the proving ground before production promotion.
- `main` is production and must not be the first place a write path runs.
- Provider writes must remain idempotent, auditable, and backed by documented external-write inventory.
- Any new live, planned, mock-only, or database write path must update `docs/external-write-inventory.md`.
- Credentials and source-system secrets must never be copied into issues, docs, fixtures, generated artifacts, logs, or evidence.
- Dirty latest-code worktrees must be reported, not reset, unless the dirty files are known generated/server artifacts and the task explicitly authorizes cleanup.

## Verification Policy

Verification must match the changed surface.

- Documentation-only work: run `git diff --check`, plus format or parser checks for structured docs when practical.
- Root workflow docs with YAML front matter: parse the front matter before opening the PR.
- Go behavior: run targeted `go test` packages and the relevant `make` gate when feasible.
- Frontend behavior: run `npm run build:web`, `npm run a11y:check`, and Browser evidence for affected routes when dev servers are healthy.
- `.pen` work: update `.pen` first, run `npm run pen:sync`, then `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check`.
- External-write inventory changes: run `npm run write-inventory:check`.

When checks cannot run, the PR and issue comment must state the exact blocker and whether the PR is draft because of the missing evidence.

## Review Feedback

Agents must use thread-aware PR review data for Codex Review feedback. Flat comments are not enough because resolved and unresolved inline threads have different meanings.

Address review feedback on the existing PR branch. Do not open a duplicate PR for the same feedback. Resolve a review thread only when the fix is demonstrably merged into the branch or the feedback is obsolete.

## Rebase And Conflict Reconciliation

Rebase remaining open PR branches onto latest `origin/ui-improvements` only when:

- The worktree is clean.
- The branch is not known to contain user-only local work.
- The expected conflicts are mechanical or already documented.

Stop and report when:

- Git cannot write shared worktree metadata.
- Conflicts involve product, data, auth, security, migration, deployment, or docs decisions.
- Two branches implement incompatible behavior.
- The latest-code checkout is dirty and cannot be safely refreshed.

## Observability

Every issue and PR handoff should leave enough evidence for the next agent.

Issue comments should include:

- Branch name and PR link.
- Files or modules touched.
- Checks run.
- Browser/runtime evidence or why it is blocked.
- Remaining scope when the issue is only partially addressed.

PR bodies should include:

- Summary of behavior or docs changed.
- Verification commands and outcomes.
- Linked issue numbers.
- Merge-order notes when active PRs overlap.
- Draft reason when acceptance criteria are not fully verified.

## Handoff States

Use these states in issue comments and queue reports:

- `unclaimed`: no active branch or PR exists.
- `claimed`: an agent has named a branch and scope.
- `in_progress`: work exists but no PR is ready.
- `pr_open`: a PR exists and is linked.
- `blocked`: progress needs a decision, permission repair, conflict resolution, dependency, or runtime evidence.
- `ready_for_review`: non-draft PR with relevant checks complete.
- `merged`: PR merged into `ui-improvements`.
- `closed`: issue closed after merged work fully satisfies acceptance criteria.

Do not close an issue for partially completed work. Leave a remaining-scope comment instead.
