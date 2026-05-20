# WIZARD Symphony Service Specification

Status: Draft v1 for issue #238

Purpose: Define a workspace-specific service that orchestrates Codex agents against GitHub Issues while preserving The WIZARD repository's safety, documentation, branch, and verification rules.

Reference: OpenAI's "An open-source spec for Codex orchestration: Symphony" describes a spec-first orchestrator that turns an issue tracker into a control plane for coding agents. This repository adapts that pattern to GitHub Issues, `dev`-based branches, local worktrees, and district-data safety rules.

## 1. Problem Statement

The repository has many small, well-scoped implementation, documentation, UI-hardening, and verification issues. Humans currently have to choose an issue, create the branch, reread the workflow rules, start a Codex session, monitor progress, run checks, and remember handoff details. That context switching slows the project down and increases the chance that safety rules are applied inconsistently.

The WIZARD Symphony service should make GitHub Issues the durable control plane for routine agent work. It should continuously inspect eligible issues, create one isolated workspace per issue, run a Codex agent with the repo-owned workflow prompt, and keep enough state and logs for humans to review the result.

The service is a scheduler, workspace manager, prompt builder, and runner. It is not the source of truth for product behavior. Product behavior remains governed by `README.md`, `docs/planning/implementation-plan.md`, `docs/product/product-requirements.md`, `docs/testing/test-matrix.md`, `docs/operations/environment-data-playbook.md`, checked-in code, and issue acceptance criteria.

## 2. Goals

- Poll GitHub Issues on a fixed cadence with bounded concurrency.
- Dispatch only issues that are unblocked, eligible, and safe for unattended agent work.
- Create deterministic per-issue workspaces from the repository integration branch.
- Load runtime behavior from the repository-owned `.agents/WORKFLOW.md` file.
- Start one Codex agent run per eligible issue and pass a complete, issue-specific prompt.
- Preserve workspaces and logs across restarts so humans can inspect partial work.
- Stop or mark runs stale when issue state, labels, blockers, or branch state changes.
- Recover transient failures with bounded exponential backoff and a maximum retry budget.
- Emit structured logs and a compact status file that explain what the service did and why.
- Keep production provider writes, credential handling, and promotion gates outside unattended automation.

## 3. Non-Goals

- A rich web UI or multi-tenant hosted control plane.
- A general-purpose workflow engine.
- Directly editing, closing, or merging GitHub issues and pull requests without an agent or human review step.
- Replacing the repo's existing GitHub issue and branch workflow.
- Running real production provider writes, staging refreshes, destructive git operations, or secret-management actions.
- Deciding product scope that is not already documented in the PRD, implementation plan, issue, or reviewed human instruction.

## 4. Source-of-Truth Order

The runner and each dispatched agent must apply sources in this order:

1. Platform and safety policy from the active Codex environment.
2. Repository `.agents/AGENTS.md` and global AGENTS instructions.
3. `README.md`.
4. `docs/planning/implementation-plan.md`.
5. `docs/product/product-requirements.md`.
6. `docs/testing/test-matrix.md` when the change affects named scenarios or promotion coverage.
7. `docs/operations/environment-data-playbook.md` when the change touches environment data, masking, staging, or promotion safety.
8. `docs/agent-orchestration/SPEC.md`.
9. `.agents/WORKFLOW.md`.
10. The GitHub issue body, comments, linked PRs, and acceptance criteria.

If a lower-priority source conflicts with a higher-priority source, the agent should follow the higher-priority source and report the conflict in the issue or PR handoff.

## 5. Main Components

### 5.1 Workflow Loader

The workflow loader reads `.agents/WORKFLOW.md`, parses the YAML front matter, and returns two values:

- `config`: typed runner settings such as poll cadence, branch prefix, active labels, terminal labels, concurrency, and verification policy.
- `prompt_template`: the Markdown body used to build an issue-specific agent prompt.

The loader must reject a missing or invalid `.agents/WORKFLOW.md` rather than falling back to hard-coded behavior.

### 5.2 GitHub Tracker Client

The tracker client reads GitHub issue data through the GitHub app or authenticated `gh` CLI. It normalizes each issue into the domain model in section 6.

The client may read:

- open issues
- labels
- assignees
- issue body and comments
- linked pull requests when GitHub exposes them
- branch references matching the configured issue branch pattern
- recent CI or PR status when needed for reconciliation

The client should not require write access for the first implementation. If write access exists, writes should be limited to clear, factual issue comments and PR metadata requested by the agent workflow.

### 5.3 Eligibility Engine

The eligibility engine decides which issues may be dispatched. It should be conservative.

An issue is eligible when all of these are true:

- state is open
- issue is not marked blocked, waiting, needs-decision, security-sensitive, production-write, or human-only
- issue has acceptance criteria or a clear task description
- issue is not already represented by an active workspace run
- issue does not already have an open non-stale branch or PR unless the workflow explicitly asks for continuation
- issue does not require credentials, provider writes, production data access, or external approval
- issue file ownership does not conflict with another active issue workspace

The first local version should allow an explicit opt-in label such as `agent-ready` before unattended dispatch. A human can still run the same workflow manually for any issue.

### 5.4 Workspace Manager

The workspace manager creates deterministic workspaces under a runner-owned directory outside the main checkout.

Recommended root selection:

- Use `$ACCOUNTSDOT_WORKTREE_ROOT` when it is set.
- Otherwise use sibling worktrees relative to the repository checkout, represented in `.agents/WORKFLOW.md` as `../`.

Recommended per-issue layout:

```text
<workspace-root>/
  issue-238-add-repo-local-symphony-agent-orchestrator-contract/
    repo/
    logs/
    state.json
    prompt.md
    handoff.md
```

The `repo/` directory should be a git worktree or clone based on the configured integration branch. For this repository, the default integration branch is `dev`.

The workspace manager must not delete workspaces automatically on success. Retention is useful for review, diff inspection, test reruns, and recovery. Cleanup should be an explicit operator action or a documented retention pass.

### 5.5 Orchestrator

The orchestrator owns the poll loop, runtime state, dispatch queue, retry queue, and reconciliation logic.

On each tick, it should:

1. Load and validate `.agents/WORKFLOW.md`.
2. Read open candidate issues.
3. Reconcile active runs with current issue state, branch state, and process state.
4. Stop runs whose issues became ineligible.
5. Select new eligible issues up to the concurrency limit.
6. Prepare or reuse each issue workspace.
7. Render the issue-specific prompt.
8. Start the Codex runner.
9. Record state and structured logs.

The orchestrator should keep a single authoritative local state file. A database is not required for the first version.

### 5.6 Agent Runner

The agent runner launches Codex in the prepared workspace. A future implementation may use Codex App Server mode when available, or a CLI-compatible runner if that is the available local interface.

The runner passes:

- the rendered issue prompt
- the issue URL and number
- the target branch name
- the expected files or modules when known
- the required checks from `.agents/WORKFLOW.md`
- explicit safety limits

The runner streams output to the workspace log directory and updates the active run status.

### 5.7 Status Surface

The first status surface can be file-based:

- `state.json` per workspace
- one global `runs.jsonl` event stream
- one human-readable `status.md`

The status surface should answer:

- which issue is running
- which branch and workspace are assigned
- current phase
- latest significant event
- retry count and next retry time
- required human action, if any

## 6. Domain Model

### 6.1 Issue

Normalized issue fields:

- `id`: GitHub node id or stable REST id
- `number`: GitHub issue number
- `title`: issue title
- `body`: issue body
- `url`: browser URL
- `state`: open or closed
- `labels`: lowercase label names
- `assignees`: GitHub logins
- `created_at`: timestamp
- `updated_at`: timestamp
- `comments`: optional recent comments used for prompt context
- `linked_prs`: optional PR numbers and states
- `blocked_by`: issue numbers or textual blockers, when detected
- `file_hints`: file paths or modules mentioned by the issue
- `priority`: optional derived sort value

### 6.2 Run

Runtime fields:

- `run_id`: stable local id
- `issue_number`
- `issue_url`
- `branch_name`
- `workspace_path`
- `status`: pending, preparing, running, waiting_retry, human_review, succeeded, failed, canceled, stale
- `phase`: tracker_read, eligibility, workspace_prepare, prompt_render, agent_run, verification, handoff, reconciliation
- `attempt`
- `started_at`
- `updated_at`
- `last_event`
- `last_error`
- `required_checks`
- `changed_files`
- `handoff_path`

### 6.3 Branch

Branch fields:

- `base_branch`: usually `dev`
- `name`: `codex/issue-<number>-<short-slug>`
- `owner`: runner id or human login
- `workspace_path`
- `created_at`
- `pushed`: boolean
- `pull_request`: optional URL

The branch name should include the issue number so humans and future agents can connect Git history back to the durable tracker.

## 7. State Policy

### 7.1 Issue State Mapping

GitHub Issues do not have custom workflow states unless the repository uses labels or projects. The local runner should treat labels as state hints.

Suggested labels:

- `agent-ready`: eligible for unattended routine work.
- `agent-running`: a runner has claimed the issue.
- `agent-review`: an agent produced a branch or handoff for human review.
- `agent-blocked`: the runner or agent needs a human decision.
- `human-only`: never dispatch unattended.
- `blocked`: do not dispatch until blockers are resolved.
- `security-sensitive`: do not dispatch unattended.
- `production-write`: do not dispatch unattended.

The labels are advisory for the first version. If the repository does not yet use them, a human may still run the workflow manually.

### 7.2 Run State Transitions

Allowed transitions:

- `pending -> preparing`
- `preparing -> running`
- `running -> human_review`
- `running -> succeeded`
- `running -> waiting_retry`
- `running -> failed`
- `waiting_retry -> preparing`
- `preparing -> failed`
- `human_review -> succeeded`
- any active state -> `canceled`
- any active state -> `stale`

The runner should prefer `human_review` over `succeeded` when work produces code, docs, UI artifacts, or PRs. `succeeded` is appropriate for pure analysis or issue triage that ends with a complete handoff.

## 8. Dispatch Sorting

Default sort order:

1. Issues with explicit `agent-ready`.
2. Higher-priority labels if the repository defines them.
3. Older open issues before newer issues.
4. Issues with clearer acceptance criteria before ambiguous issues.
5. Issues with smaller file ownership and lower conflict risk before broad refactors.

The runner must not optimize only for oldest issue age. Safety and conflict risk come first.

## 9. Workspace Preparation

For each issue:

1. Fetch the latest remote refs.
2. Create or reuse a worktree under the configured workspace root from `origin/dev` unless the issue or repo docs specify another integration branch.
3. Create or switch to `codex/issue-<number>-<slug>` inside that workspace.
4. Read `README.md`, `.agents/AGENTS.md`, and relevant repo docs before edits.
5. Preserve existing user-authored changes.
6. Keep generated build output such as `frontend/dist/` out of the issue branch unless the issue explicitly changes artifact policy.

If the workspace cannot be prepared cleanly, the run should move to `agent-blocked` or equivalent handoff with the exact git error.

## 10. Prompt Rendering

The rendered prompt should include:

- issue title, number, URL, body, labels, and acceptance criteria
- current branch name and base branch
- source-of-truth order
- known safety rules
- expected verification commands
- file ownership or conflict notes
- explicit instruction to update docs when behavior changes
- explicit instruction to avoid production writes and secrets
- required final report format

Prompt rendering must not inject secrets or raw provider credentials. Environment variable names are acceptable; values are not.

## 11. Verification Policy

Agents should run the narrowest meaningful verification for the touched area, then escalate to broader checks when risk warrants it.

Default checks by change type:

- Docs-only: review rendered Markdown and run no code tests unless links, generated docs, or examples need validation.
- Go behavior: `make test-unit` or the narrower package test, then broader `make test` when shared behavior changes.
- Provider contracts: relevant contract tests plus safety review of fixture and secret handling.
- Frontend runtime: `npm run build:web` and targeted accessibility or route checks when UI behavior changes.
- `.pen` implemented-page layout: `npm run pen:check`, `npm run pen:lint`, `npm run build:web`, and `npm run a11y:check` unless the issue justifies a narrower set.
- Dependency changes: relevant tests plus `npm audit --omit=dev`, full `npm audit`, `make vulncheck`, or container equivalents as applicable.

Agents must report checks they did not run.

## 12. Safety Policy

The runner and agents must honor these safety limits:

- No production provider writes.
- No real downstream write paths from unattended runs.
- No staging promotion or main promotion without the documented external runbook evidence and human signoff.
- No secrets in code, docs, fixtures, logs, issue comments, PR descriptions, generated prompts, or artifacts.
- No raw service-account JSON, tokens, passwords, private keys, auth headers, or provider credentials in prompts.
- No destructive git commands unless a human explicitly requested them in the active session.
- No editing generated `.pen`-derived artifacts by hand.
- No committing `frontend/dist/` unless an issue explicitly changes generated artifact policy.
- No product behavior that is not supported by the PRD and implementation plan.
- No closing GitHub issues until related work is merged into the integration branch.

If a task appears to require any forbidden action, the agent should stop and produce a handoff explaining the needed human decision.

## 13. Reconciliation

On startup and each poll tick, the orchestrator should reconcile:

- active process status
- current GitHub issue state and labels
- branch existence
- worktree existence
- PR state, when present
- retry timers
- stale workspaces

A run becomes stale when:

- the issue is closed
- the issue receives `blocked`, `human-only`, `security-sensitive`, or `production-write`
- the branch was deleted or replaced by a human
- another PR already resolves the same issue
- the workspace files conflict with newer integration branch changes and automated recovery is unsafe

Stale runs should stop cleanly and keep their logs.

## 14. Retry Policy

Retry only transient failures:

- GitHub API timeout
- temporary network failure
- Codex process crash before making changes
- dependency download failure that is known to be temporary
- test infrastructure flake with evidence

Do not retry:

- failing tests caused by the agent's code
- policy violations
- missing secrets
- permission denial
- merge conflicts requiring product judgment
- ambiguous issue scope

Default retry schedule:

- attempt 1 immediately
- attempt 2 after 1 minute
- attempt 3 after 5 minutes
- attempt 4 after 15 minutes

After four attempts, move to `agent-blocked` or the local equivalent and require human review.

## 15. Observability

Structured events should include:

- `tracker.poll.started`
- `tracker.poll.completed`
- `issue.eligible`
- `issue.skipped`
- `workspace.prepared`
- `prompt.rendered`
- `agent.started`
- `agent.output`
- `agent.completed`
- `agent.failed`
- `run.retry_scheduled`
- `run.human_review`
- `run.stale`
- `run.canceled`

Each event should include timestamp, run id, issue number, branch, workspace path, phase, message, and optional error detail. Error details must be sanitized before writing logs.

## 16. Human Handoff

Each run should produce `handoff.md` with:

- issue number and URL
- branch name
- changed files
- summary of changes
- tests and checks run
- checks not run
- docs updated
- safety notes
- known risks
- suggested PR title and body
- follow-up issues created or recommended

For UI-heavy work, the handoff should link screenshots or runtime evidence when available.

## 17. First Implementation Slice

The first runnable implementation should stay small:

1. Read `.agents/WORKFLOW.md`.
2. List open GitHub issues with `agent-ready`.
3. Print an eligibility report without dispatching.
4. Create workspaces only when invoked with an explicit `--dispatch` flag.
5. Start at concurrency `1`.
6. Write local logs and state files.
7. Require a human to push branches or open PRs until the review packet is trusted.

This keeps the service useful while avoiding a premature always-on daemon.

## 18. Acceptance Criteria For This Contract

- The repository contains this workspace-specific Symphony specification.
- The repository contains `.agents/WORKFLOW.md` with machine-readable front matter and a human-readable agent prompt.
- The implementation plan references the contract as the source for future agent orchestration.
- The contract clearly defines GitHub issue eligibility, workspace isolation, branch naming, retries, reconciliation, observability, handoff, and safety boundaries.
