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
  max_concurrent_runs: 6
  max_attempts: 4
  require_explicit_agent_ready_label: true
  workspace_root: /private/tmp/accountsdot-symphony
  agent_runner_command: codex --ask-for-approval never exec --json --sandbox workspace-write --cd {repo} -
  agent_runner_codex_home_root: /private/tmp/accountsdot-symphony/.codex-agent-homes
  agent_runner_timeout_ms: 21600000
pull_requests:
  target_branch: phase-0-platform-foundation
  inspect_before_dispatch: true
  auto_merge_clean_prs: true
  merge_method: squash
  codex_review_authors:
    - chatgpt-codex-connector
    - chatgpt-codex-connector[bot]
    - github-copilot
    - codex-review
  codex_review_success_reactions:
    - THUMBS_UP
    - +1
  codex_review_bot: chatgpt-codex-connector[bot]
  no_review_with_bot_thumbs_up_is_clean: true
  remediate_blocked_prs: true
  max_review_remediations_per_tick: 6
  review_wait_policy: non_blocking_stateful
  review_grace_period_seconds: 300
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
maintenance:
  ui_improvements_monitor:
    target_branch: ui-improvements
    lock_path: /private/tmp/accountsdot-ui-improvements-github-scan.lock
    lock_max_age_ms: 7200000
    latest_code_worktree: /Users/lcampbell/code.internal/accountsdot-latest-ui
    reconcile_worktree_root: /private/tmp/accountsdot-symphony-prs
    reconcile_pr_branches: true
    safe_branch_prefixes:
      - codex/
      - issue-
    codex_review_authors:
      - chatgpt-codex-connector
      - github-copilot
      - codex-review
    auto_resolve_outdated_codex_review_threads: true
    latest_code_allowed_dirty:
      - frontend/dist/
      - tmp/
      - .vite/
    browser_default_url: http://localhost:5173/dashboard/it-admin
    browser_screenshot_required: false
    health_urls:
      - http://localhost:8080/health
      - http://localhost:5173/api/v1/dev/session
      - http://localhost:5173/dashboard/it-admin
    dev_servers:
      - api
      - vite
    browser_required: true
    safe_rebase: true
    dirty_worktree_policy: stop
skill_routing:
  discover_root: .agents/skills
  precedence:
    - wizard-ui-hardening
    - wizard-code-documentation
  missing_skill_policy: agent-blocked
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
8. Load applicable repo-local skills from `.agents/skills/*/SKILL.md` before rendering the issue prompt.

## Repo Skill Runtime

The Symphony runner treats checked-in `.agents` skills as runtime guidance. It should discover skill directories automatically, include only the relevant skill summaries in prompts, and report missing expected skills as `agent-blocked`.

- Use `wizard-ui-hardening` for The WIZARD UI, design, `.pen`, generated implemented-page, shared-shell, dashboard, browser-evidence, or route-visual work.
- Use `wizard-code-documentation` for implemented code, route/API, handler, docs, inline-comment, external-write, or provider-surface work.
- When both apply, route UI classification first through `wizard-ui-hardening`, then apply code documentation/update obligations through `wizard-code-documentation`.
- Prefer these checked-in repo skills over global memory or chat-only guidance when they apply.

## UI Monitor And Browser Evaluation Runtime

The `ui_improvements_monitor` maintenance runner replaces the old long prompt-based heartbeat. It owns GitHub queue reporting, safe branch/worktree reconciliation, and latest-code health checks. It must emit `browser_evaluations[]` when UI/runtime validation needs the Codex in-app Browser.

When `safe_rebase` and `reconcile_pr_branches` are enabled, the monitor may automatically rebase and push known agent-owned PR branches onto `origin/ui-improvements`. This remediation is intentionally narrow:

- It only considers non-draft PR branches whose names start with an allowed `safe_branch_prefixes` entry.
- It requires a clean PR worktree and a local branch head that still matches `origin/<branch>` before rebase.
- It pushes only with `--force-with-lease`.
- It records dirty worktrees, divergent local branches, missing remote branches, and rebase conflicts as blocked reconciliation results instead of overwriting local work.
- It does not manually merge PRs, close issues, or choose product, data, auth, security, migration, deployment, or documentation conflict behavior.

The monitor also owns stale Codex Review cleanup for PRs in the queue. It must use thread-aware GitHub review data, not flat comments. Outdated unresolved Codex Review threads from configured `codex_review_authors` may be resolved automatically because the reviewed diff line is obsolete. Current unresolved Codex Review threads must remain blocking until the branch contains an in-scope code/docs fix and verification evidence; the automation should report them as review-remediation work instead of hiding them from the queue.

Repo-local Node code does not import the Browser plugin directly. The Codex automation wrapper is responsible for executing `browser_evaluations[]` through the Browser skill bridge, then passing structured `browser_results[]` back to the runner. Missing Browser results must be reported as `needs_browser_evaluation`, not as passed verification.

The Browser skill bridge is the in-app Browser access path. It uses the `node_repl` JavaScript tool to import the plugin's `scripts/browser-client.mjs`, call `setupBrowserRuntime({ globals: globalThis })`, select `agent.browsers.get("iab")`, and drive that tab with the Browser skill API. The wrapper should not look for a direct `browser` tool namespace.

Browser screenshots are preferred evidence but are not required in the current monitor config because the in-app Browser bridge can load and inspect local DOM state even when CDP screenshot capture is unavailable. If a future environment supports reliable screenshot capture, set `browser_screenshot_required: true` and treat missing screenshots as a blocking Browser result again.

The Codex automation wrapper should stay small:

1. Invoke `npm run symphony:ui-monitor -- --json`.
2. If the status contains `browser_evaluations[]`, activate the requested DEV persona with the emitted `persona_setup` command before navigation. The runner derives that command from the configured browser URL origin.
3. Preserve the current local app URL when available, otherwise open or reload the requested URL through the Browser skill bridge.
4. Capture DOM and screenshot evidence, saving screenshots under `/private/tmp` or another non-committed evidence path.
5. Record each Browser result with URL, status, evidence, findings, and checked timestamp.
6. Write results back with `node scripts/symphony_runner.mjs record-browser-results --browser-results <path> --json`.
7. Summarize the queue, blockers, Browser evidence, and next recommended PR without manually merging PRs.

## Phase 0 Pull Request Queue Runtime

The Synchronizer owns the Phase 0 pull request queue before it starts more issue work. The automation wrapper should not carry this policy in a long heartbeat prompt; it should invoke the repo command and summarize the structured result.

For every open non-draft PR targeting `phase-0-platform-foundation`, the Synchronizer must inspect merge state, labels, checks, thread-aware Codex Review data, and issue-comment reactions before selecting more work. Merge readiness is intentionally evidence-based:

- The PR must target `phase-0-platform-foundation`, be non-draft, have a clean GitHub merge state, and have no blocking labels from `tracker.blocked_labels`.
- Required checks must be passing, or GitHub must report no required check rollup for the PR.
- Current unresolved Codex Review threads from configured `codex_review_authors` are hard blockers until a branch update makes the feedback fixed or obsolete.
- Requested-changes reviews from configured Codex Review authors are hard blockers until a later review or thread state makes them non-actionable.
- If there is no Codex Review response yet, a thumbs-up reaction from `chatgpt-codex-connector[bot]` on the PR conversation or review-request comment is an explicit clean signal. In that case the PR is safe for merge when the other merge gates pass.
- An eyes reaction, missing reaction, or pending review request is not a clean signal. The Synchronizer should record `waiting_for_codex_review` and move on to the next tick instead of sleeping inside the current tick.
- Clean PR merging is approved only for PRs targeting `phase-0-platform-foundation`; use the configured GitHub merge method and do not merge PRs for `dev`, `main`, or `ui-improvements` from this dispatcher.

The previous chat-level behavior of sleeping for five minutes inside the automation tick is not allowed. Review waiting is stateful and non-blocking: record when a review was requested or observed, report the wait reason, and let the next scheduled tick re-evaluate. This keeps the `:00`, `:15`, `:30`, and `:45` automation windows available instead of letting one run occupy the next one.

If a PR is merge-conflicted, draft, or has actionable Codex Review feedback, the Synchronizer should prioritize remediation before ordinary new issue dispatch. Remediation consumes the same bounded worker slots as issue dispatch only when a remediation worker can actually be prepared or run; a blocked remediation prerequisite must be reported without consuming a worker slot. After scheduling the available PR-remediation workers, the runner should continue selecting unrelated eligible issues for any remaining slots in the same tick unless a clean PR is ready to merge first. It should reuse the prepared issue workspace when one exists, implement in-scope fixes, run the relevant checks, push with `--force-with-lease`, and leave concise review-thread replies or resolution only when the code/docs evidence makes the comment non-actionable.

Actionable Codex Review feedback is agent work, not a terminal blocker. When the Synchronizer sees current unresolved Codex Review threads on an open Phase 0 PR, it must launch a review-remediation agent in the PR's existing workspace before considering unrelated new issue dispatch. The linked issue may already be closed, so remediation must not depend on the issue still appearing in the open-issue queue. The Synchronizer should match issues only from explicit PR references such as `#123` or `issue-123`, reuse a stable prepared workspace even if the issue title slug changed, preserve the PR's actual target branch in prompts and state, and verify that the prepared repo is checked out on the PR head branch before launching the agent. That agent receives the PR URL, target branch, working branch, review-thread URLs, comment excerpts, file paths, and any available original issue context. The agent must inspect the feedback, plan and implement the in-scope fix, run relevant verification, commit and push the PR branch, and reply to or resolve threads only when the branch update makes the feedback fixed or obsolete. If the PR workspace is missing, dirty, on the wrong branch, or ambiguous, the Synchronizer records a concrete remediation blocker instead of silently pausing the queue. Dry-run output must report those same live-run blockers without writing prompts, state, branches, or comments.

If a PR does not have a prepared issue workspace but its branch uses an approved automation prefix, the runner may create a dedicated PR remediation workspace under `dispatch.workspace_root` from the remote PR branch. This is intended for manual or out-of-band PRs that still need Codex Review, draft, or merge-conflict remediation. Dry-run and live runs must evaluate the same safety prerequisites. The runner must still refuse unsafe branch names, missing remote branches, dirty worktrees, or branches already checked out elsewhere.

## Issue Dispatcher Runtime

The repo-owned issue dispatcher is `npm run symphony:sync`. It is the checked-in entrypoint for turning `agent-ready` GitHub issues into deterministic workspaces and branch prompts. It reads the same `.agents/WORKFLOW.md` front matter as the monitor, uses `tracker.active_labels` and `tracker.blocked_labels` for eligibility, derives each issue target branch from the issue body, and honors the `phase-0-platform-foundation` target branch when that line is present in Phase 0 issues.

The dispatcher is conservative by design:

- `npm run symphony:sync -- --dry-run --json` is non-mutating and should be the first command in any automation tick.
- A non-dry-run tick creates or reuses one workspace per selected issue under `dispatch.workspace_root`, creates a branch using `branching.branch_template`, and writes `prompt.md` plus `state.json` for the assigned run.
- A failed issue workspace may be retried automatically until `dispatch.max_attempts` is reached, provided the worktree is clean. This prevents a transient agent crash from making the issue permanently inactive.
- The dispatcher skips issues with blocked labels, missing acceptance criteria, missing `agent-ready`, or an already-open PR that references the issue.
- When a human manually merges a PR that references an issue, the next dispatcher tick treats that PR as resolved and merged. It must not keep the issue blocked by the old prepared workspace; instead it records `status: merged` in the matching workspace `state.json` when the file exists, removes the clean prepared `repo` worktree checkout, and reports the issue queue entry as `merged`. If the prepared worktree is dirty, it must report the dirty files instead of deleting anything.
- The dispatcher refuses to reset an existing branch, overwrite dirty worktrees, or create a workspace from a missing target branch.
- The dispatcher treats only a clean PR that is ready to merge as a reason to pause ordinary issue dispatch. Blocked, dirty, draft, or waiting-for-review PRs must be reported and remediated, but they should not prevent unrelated eligible issues from using remaining worker capacity.
- `dispatch.agent_runner_command` is approved for this repo and launches `codex exec` from the prepared worktree with the rendered issue prompt on stdin. The dispatcher sets `CODEX_HOME` to a per-workspace directory under `dispatch.agent_runner_codex_home_root`, symlinks the operator's existing Codex auth/config/plugin references into that writable home, and keeps worker state databases out of the desktop app's `~/.codex` directory. This prevents automation-launched workers from failing on readonly `state_5.sqlite` or app-server state writes while still using the authenticated Codex installation. The dispatcher writes runner stdout and stderr to the issue workspace `logs/` directory and records completion or failure in `state.json`.

The Synchronizer wrapper should now call `npm run symphony:sync -- --json` for pull-request queue handling and issue dispatch state instead of maintaining a chat-only issue queue. The wrapper remains responsible for making sure the runner worktree is clean and current, invoking the repo-owned command, and surfacing runner failures. It should not duplicate merge policy, run long sleeps, or decide review status from flat comments when the runner can inspect thread-aware review state.

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
