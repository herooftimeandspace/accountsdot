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
    - human-review
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
  agent_runner_idle_timeout_ms: 120000
daemon:
  enabled: true
  state_dir: /private/tmp/accountsdot-symphony
  default_tick_interval_seconds: 300
  shutdown_mode: drain
  control_transport: file
  status_retention_events: 5000
  watchdog_stale_after_seconds: 900
  watchdog_fallback_sync: true
source_corpus:
  include_markdown:
    - README.md
    - AGENTS.md
    - .agents/**/*.md
    - docs/**/*.md
  priority_roots:
    - .agents/
    - docs/
  exclude_roots:
    - .git/
    - node_modules/
    - frontend/dist/
    - tmp/
    - .vite/
    - .gocache/
    - .gomodcache/
    - artifacts/
    - coverage/
  generated_markers:
    - /generated/
phase_planning:
  issue_chunk_limit: 6
  require_acceptance_criteria: true
  require_safety_context: true
  require_verification_context: true
self_healing:
  enabled: true
  labels:
    - agent-ready
    - bug
    - symphony
  priority: top
  fingerprint_prefix: symphony-self-heal
escalation:
  durable_surface: github_issue
  live_surface: daemon_status_tui
  chat_surface: optional_watchdog_summary
  human_review_labels:
    - human-review
    - agent-blocked
    - symphony
  notify_human_when:
    - blocked_human
    - destructive_cleanup_risk
    - ambiguous_dirty_source_edits
    - repeated_self_healing_failure
review_loops:
  max_review_requests_per_pr: 4
  review_request_comment: "@codex"
  allowed_phase_branch_prefixes:
    - phase-
    - codex/
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
  codex_review_in_progress_reactions:
    - EYES
  codex_review_bot: chatgpt-codex-connector[bot]
  auto_resolve_outdated_codex_review_threads: true
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
9. all comments on currently open GitHub issues that are candidates for starting, continuing, or remediation work

Use the most specific checked-in source that applies. If code and docs disagree, verify current behavior and update the relevant durable docs when the task requires alignment.

## Startup Steps

1. Confirm the current branch and working tree status.
2. Fetch the latest integration branch.
3. Work from `dev` unless the issue or repo docs explicitly require another integration branch.
4. Use branch `codex/issue-<number>-<short-slug>`.
5. Inspect open issues, every comment on those issues, and linked context before implementation.
6. Identify likely file ownership and conflict risk.
7. Do not touch unrelated generated output or user-authored changes.
8. Load applicable repo-local skills from `.agents/skills/*/SKILL.md` before rendering the issue prompt.

## Repo Skill Runtime

The Symphony runner treats checked-in `.agents` skills as runtime guidance. It should discover skill directories automatically, include only the relevant skill summaries in prompts, and report missing expected skills as `agent-blocked`.

- Use `wizard-ui-hardening` for The WIZARD UI, design, `.pen`, generated implemented-page, shared-shell, dashboard, browser-evidence, or route-visual work.
- Use `wizard-code-documentation` for implemented code, route/API, handler, docs, inline-comment, external-write, provider-surface, Symphony orchestration, runtime-contract, or promotion-validator work.
- When both apply, route UI classification first through `wizard-ui-hardening`, then apply code documentation/update obligations through `wizard-code-documentation`.
- Prefer these checked-in repo skills over global memory or chat-only guidance when they apply.
- For Symphony changes, the code documentation skill should push the worker toward five recurring checks before completion: queue-state invariants, workspace-recovery safety, review-thread lifecycle handling, handler-derived API contract fidelity, and exhaustive coverage of safety-critical validation surfaces.

## Markdown Source Corpus Runtime

The Go Symphony `sync` runner at `cmd/symphony` must scan repo-authored Markdown before planning a tick, with `.agents/**/*.md` and `docs/**/*.md` treated as priority sources. The source corpus must include root Markdown such as `README.md` and `AGENTS.md` when present, `.agents/AGENTS.md`, `.agents/WORKFLOW.md`, repo-local skill `SKILL.md` files, and the planning, product, testing, operations, and agent-orchestration docs under `docs/`.

The scanner must exclude vendored, generated, dependency, cache, build, and evidence-output paths such as `.git/`, `node_modules/`, `frontend/dist/`, `.vite/`, `.gocache/`, `.gomodcache/`, `artifacts/`, and generated Markdown unless a checked-in source explicitly marks the path authoritative. The Go runner exposes `source_corpus` in JSON so operators can audit which checked-in docs influenced issue materialization, prompt context, conflict decisions, verification selection, and self-healing bug reports.

If Markdown sources disagree about target branches, safety constraints, verification commands, or phase scope, the runner must follow the documented source-of-truth order and report the conflict in structured output instead of silently choosing. Phase issue materialization must use this indexed Markdown corpus rather than reading only `docs/planning/implementation-plan.md`.

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

## Symphony Remediation Lessons And Non-Regressions

The following rules capture operational failures that previously required manual human prompting. Treat them as durable acceptance criteria for future Synchronizer changes, not as optional implementation notes.

### Codex Review Signals

- Codex Review is complete only when the newest review request has a later explicit clean signal from the configured Codex Review actor. The implemented clean signal is a `THUMBS_UP`/`+1` reaction from `chatgpt-codex-connector[bot]` on the main PR conversation, an `@codex` request comment, or the active review request surface. Explicit top-level clean comments such as "all PR issues have been addressed", "merge is clean", or "no actionable feedback" are desired clean signals tracked by issue #324; until that issue is implemented, the runner must not document or depend on clean-comment text as an implemented merge gate. When all other merge gates pass, the PR should move toward automatic clean merge instead of waiting for manual confirmation.
- An `EYES` reaction from `chatgpt-codex-connector[bot]` on the main PR conversation or an `@codex` review request comment means GitHub Codex Review is currently inspecting the code in the background. It is a pending-review signal, not a pass, failure, remediation instruction, or clean signal. Removing the `EYES` reaction is not approval and must not be interpreted as review completion.
- Within the currently fetched PR conversation window, a newer `@codex` comment with a bot `EYES` reaction, or with no later implemented clean signal yet, supersedes older Codex Review state. Even if an earlier review exists, the PR remains `waiting_for_codex_review` until a later bot thumbs-up reaction or thread-aware review state proves that no current actionable feedback remains for the newest fetched request. Full-history pagination for long PR conversations is not implemented yet; until it is, the runner must describe this as a bounded-window decision rather than proof over the entire PR history. Clean-comment support must follow issue #324 before it can clear this wait state.
- `symphony:sync` review detection must be thread-aware and comment-aware for the data it fetches. Flat PR comments, review bodies, or reaction summaries are not enough to decide whether feedback is resolved. The sync runner must inspect fetched GitHub review threads, fetched top-level PR comments, fetched thread replies, `isResolved`, `isOutdated`, author, path, line, and the request/reaction timeline before deciding merge, remediation, issue materialization, or wait state. `ui-monitor` currently reads review threads only; top-level comment-state support for that path is tracked by issue #326. Any full-history guarantee for high-volume PR comments or review-thread replies requires paginated retrieval before it may be documented as complete.
- Current unresolved Codex Review threads are actionable work. They should start a remediation worker when the branch is safe to modify. They should not permanently block unrelated issue workers from using available capacity.
- Current unresolved Codex Review threads on a merged PR cannot be remediated on the closed PR branch. The target behavior is to create or update deduplicated `agent-ready` GitHub issues that preserve the thread URL, thread id, file path, line, excerpt, severity, acceptance criteria, and verification expectations; implementation is tracked by issue #325. Until the merged-PR intake path exists, a human or watchdog that discovers post-merge Codex feedback should create those issues manually and keep them eligible for normal Symphony dispatch unless they carry a human-only, security, production-write, or other blocking label.
- Outdated unresolved Codex Review threads should be resolved automatically when GitHub reports the thread as obsolete. After resolving them, the runner must refetch thread state before evaluating merge readiness.
- If a remediation commit makes a thread obsolete or directly fixes the feedback, the runner should resolve or reply to that exact conversation before requesting another manual Codex Review round. Do not leave remediated conversations open and then wait for a human to notice.
- If the runner cannot prove that a branch update addressed a thread, it must leave the thread open and report the missing evidence. It must not resolve active feedback just to clear the queue.

### PR Queue And Merge Behavior

- Human-merged PRs are terminal for their referenced issue work. On the next tick, the runner must mark the matching issue workspace state as `merged`, remove or preserve the prepared workspace according to cleanliness, and stop reporting that issue as blocked by the old PR or workspace.
- A clean, non-draft Phase 0 PR that satisfies merge state, label, check, and Codex Review acceptance criteria should be merged automatically with the configured merge method. Waiting for a manual merge after all gates pass is a runner bug.
- Automatic merge should happen only after verification responsibility has been satisfied. If the workflow requires another agent or verification pass before merge, schedule that work with an available worker slot instead of leaving the PR idle.
- Merge-conflicted, dirty, draft, or review-blocked PRs should be remediated when possible, but they must not become global queue stops. Report the blocker, schedule remediation if safe, and continue filling unused worker slots with unrelated eligible issues.
- The top-level status should distinguish `waiting_for_codex_review`, `review_remediation_blocked`, `merge_conflict_blocked`, and `dispatches_started`. Do not let one waiting PR make the whole tick look inactive when issue workers were available.

### Worker Concurrency And Issue Dispatch

- `dispatch.max_concurrent_runs` is a real capacity target. The runner should fill available slots with independent remediation and issue work whenever safety prerequisites allow it.
- Capacity must come from actual outcomes, not pre-dispatch intent. A candidate that ends the tick as `blocked`, `failed`, `waiting_for_codex_review`, or otherwise non-runnable must not be counted as runnable capacity or hide real blockers.
- A PR waiting on Codex Review is not a reason to skip open `agent-ready` issues. If there are free slots and eligible issues, dispatch them.
- Review remediation has priority over new issue work only when the remediation worker can actually be prepared or run. A blocked remediation prerequisite should not consume a worker slot.
- Open issues with `agent-ready`, acceptance criteria, and no blocking labels should not be hidden behind stale state from old PRs, old workspaces, or unrelated review waits. If a candidate is skipped, the queue output must state the concrete reason.
- Workers should be split by issue or PR branch and by non-overlapping file ownership. The dispatcher should maximize independent work while preserving branch isolation and avoiding shared-file conflicts.
- Top-level status must prefer actionable blockers over passive waits and must never report `idle` when only blocked-actionable work remains.

### Workspace Recovery

- Dirty state is not automatically fatal. If a prepared issue or PR remediation workspace is already on the correct branch, local edits are usually previous automation work for that same unit. Pass the dirty file list to the worker prompt and require the worker to inspect, finish, verify, commit, and push the in-scope edits.
- Dirty state is fatal when the workspace is on the wrong branch, detached unexpectedly, tied to an unsafe branch name, missing its remote, or otherwise ambiguous. In those cases the runner must report the exact blocker instead of resetting, deleting, or silently skipping.
- Refreshing a stale remediation workspace must distinguish behind vs ahead state. Do not hard-reset away clean local-only commits or preserved recovery evidence just because the local and remote OIDs differ.
- A stale issue workspace must not permanently consume a worker slot. Missing `state.json`, unreadable state, succeeded state without an open PR, or branch/workspace slug drift should be handled explicitly.
- Issue title changes can make old slug-derived workspace paths stale. Reuse a resolved old-slug workspace only when its repo or state still matches the current issue branch. If the branch slug changed, create or use the current branch/workspace instead of repeatedly blocking on the old checkout.
- Worker prompts must render the actual prepared workspace path, repo path, target branch, and working branch. They must not mix a resolved legacy workspace with a newly derived slug path.
- Malformed legacy state must fail closed. Invalid `attempts`, unreadable state, or inconsistent state should not create infinite retry loops.
- Shared git stash state is global to the repository. When preserving dirty edits before refresh, record a stable stash identifier or equivalent evidence rather than a reflog slot like `stash@{0}`.
- Merged-workspace teardown should normalize generated cache permissions before declaring cleanup blocked. If a clean merged workspace cannot be removed because `.gomodcache`, `.gocache`, or `node_modules/.cache` contains permission-protected files, make those generated cache trees user-writable and retry once.
- Clean teardown recovery is limited to generated cache directories. Do not chmod arbitrary source paths, do not delete dirty workspaces, and do not discard local edits.

### Worker Runtime Environment

- `dispatch.agent_runner_command` must launch Codex with a writable per-workspace `CODEX_HOME`. Automation-launched workers must not share or write the desktop app's `~/.codex` state database. Before dispatching a worker, the runner must resolve the configured executable through `PATH` and known local Codex app fallback locations. If the executable cannot be found, the issue or PR remediation must become an actionable runner-configuration blocker instead of failing with a low-level spawn error.
- The per-workspace Codex home should reuse authentication and configuration by symlink or equivalent safe reference, but writable runtime state such as SQLite databases must live under `dispatch.agent_runner_codex_home_root`.
- Worker stdout and stderr must be streamed and persisted under the workspace `logs/` directory so failures can be debugged without rerunning the task.
- Idle workers must be reaped according to `dispatch.agent_runner_idle_timeout_ms`. A worker that has already emitted a durable `turn.completed` event may be recorded as completed even if the process needs cleanup.

### Scheduler And Wrapper Cadence

- The scheduled wrapper should run every 15 minutes on the configured cadence and must not sleep or poll long enough to overlap the next tick.
- The wrapper should not serialize independent work. It should fetch/prune, prepare the clean runner worktree, invoke `npm run symphony:sync -- --json --max-runs <capacity>`, summarize JSON, and exit.
- The wrapper must not duplicate PR merge policy, review signal interpretation, issue-dispatch eligibility, remediation policy, or worker concurrency rules. Those belong in this checked-in workflow and runner code.
- If the runner reports no active dispatches while eligible `agent-ready` issues exist and worker slots are free, that is a bug to investigate, not an acceptable idle state.

### Local Daemon And Watchdog

- The preferred continuous runner is the local Go daemon, invoked with `npm run symphony:daemon -- --phase <phase-id> --phase-branch <branch>`. The daemon owns repeated ticks, local state, the singleton lock, pause/resume/drain/stop control, and worker-capacity changes.
- The daemon writes operator-readable state under `daemon.state_dir`, including `controller.json`, `status.json`, `status.md`, `runs.jsonl`, and worker state. `npm run symphony:status -- --watch` and `npm run symphony:tui` must read that state instead of rebuilding queue policy.
- The terminal TUI is only a client. It may queue pause, resume, drain, stop, cancel-worker, cancel as a backwards-compatible alias, and concurrency commands, but it must not duplicate issue ranking, review-thread interpretation, merge policy, workspace recovery, or self-healing classification. It must expose enough diagnostics for a local operator to understand each active worker without opening multiple files: worker identifier, issue or PR reference, current lane/status, elapsed runtime, retry count, latest event, branch, workspace, log pointer, and best-effort CPU when a local worker PID is available. If Symphony has not recorded a PID for the worker, display CPU as unavailable.
- Codex automations are optional watchdog/backstop jobs for this path. A watchdog should check whether `controller.json` and the daemon lock are fresh. If the daemon is active and healthy, it reports status and exits. If the daemon is inactive or stale and `daemon.watchdog_fallback_sync` is true, it may run one non-overlapping `npm run symphony:sync -- --json --max-runs <capacity>` tick. It must not start a second daemon or reimplement scheduling decisions in the automation prompt.

### Escalation Surfaces

- GitHub issues are the durable escalation surface for Symphony failures. The runner should create or update one deduplicated issue per stable self-healing fingerprint, include the exact blocker, affected workspace, related issue or PR numbers, dry-run evidence, acceptance criteria, safety constraints, and the next safe action.
- Automatable self-healing issues use `self_healing.labels`, keep `agent-ready`, and run ahead of ordinary phase work. They should not interrupt the operator unless the same fingerprint repeats, exhausts retry budget, or becomes non-automatable.
- Human decisions use `escalation.human_review_labels`. When cleanup could destroy source edits, the branch/workspace state is ambiguous, a destructive action is required, secrets or production writes are involved, or retries are exhausted, the runner must mark the work `blocked_human`, add a concise GitHub issue comment explaining the decision needed, and remove it from runnable capacity.
- The daemon status files and TUI are the live awareness surface. `status.json`, `status.md`, `runs.jsonl`, and `npm run symphony:tui` must make `blocked_human` and `human-review` items visible with issue links and the required operator decision.
- Codex chat is not the control plane. A watchdog may summarize active `blocked_human` items in chat, but it must link back to the GitHub issue and daemon status instead of treating the chat as durable state.

## Phase 0 Pull Request Queue Runtime

The Synchronizer owns the Phase 0 pull request queue before it starts more issue work. The automation wrapper should not carry this policy in a long heartbeat prompt; it should invoke the repo command and summarize the structured result.

For every open non-draft PR targeting `phase-0-platform-foundation`, the Synchronizer must inspect merge state, labels, checks, thread-aware Codex Review data, and issue-comment reactions before selecting more work. Merge readiness is intentionally evidence-based:

- The PR must target `phase-0-platform-foundation`, be non-draft, have a clean GitHub merge state, and have no blocking labels from `tracker.blocked_labels`.
- Required checks must be passing, or GitHub must report no required check rollup for the PR.
- Current unresolved Codex Review threads from configured `codex_review_authors` are hard blockers until a branch update makes the feedback fixed or obsolete.
- Outdated unresolved Codex Review threads from configured `codex_review_authors` should be resolved at the start of each Synchronizer tick, then thread state must be fetched again before merge readiness is evaluated. A stale conversation that GitHub marks obsolete must not continue blocking an otherwise clean PR.
- Requested-changes reviews from configured Codex Review authors are hard blockers until a later review or thread state makes them non-actionable.
- If there is no Codex Review response yet, a thumbs-up reaction from `chatgpt-codex-connector[bot]` on the PR conversation or review-request comment is the implemented explicit clean signal. Later top-level Codex Review comments that explicitly say all PR issues have been addressed, the merge is clean, or no actionable feedback remains are desired clean signals tracked by issue #324, but they are not an implemented merge-clearing signal until the synchronizer reads and configures them. In the implemented path, the PR is safe for merge when the reaction-based clean signal, other merge gates, and current thread-aware review state are clear.
- An eyes reaction from `chatgpt-codex-connector[bot]` on the PR conversation or review-request comment means GitHub Codex Review is actively looking at the PR. It is not a clean signal and not a remediation signal by itself; the Synchronizer should record `waiting_for_codex_review` with an in-progress note and move on to the next tick. If a newer `@codex` comment has a bot eyes reaction, the PR remains pending review even when older Codex Review comments already exist. Removing the eyes reaction is still not approval; the runner must wait for a thumbs-up reaction or current review-thread state that proves there is no actionable feedback for the newest request.
- A missing reaction or pending review request is not a clean signal. The Synchronizer should record `waiting_for_codex_review`, keep the PR out of the merge lane, and use any remaining worker slots for unrelated eligible issues instead of sleeping inside the current tick. If a newer `@codex` request exists and no newer bot response has arrived yet, keep waiting even when the bot has not reacted at all.
- Clean PR merging is approved only for PRs targeting `phase-0-platform-foundation`; use the configured GitHub merge method and do not merge PRs for `dev`, `main`, or `ui-improvements` from this dispatcher.

The previous chat-level behavior of sleeping for five minutes inside the automation tick is not allowed. Review waiting is stateful and non-blocking: record when a review was requested or observed, report the wait reason, and let the next scheduled tick re-evaluate. This keeps the `:00`, `:15`, `:30`, and `:45` automation windows available instead of letting one run occupy the next one.

If a PR is merge-conflicted, draft, or has actionable Codex Review feedback, the Synchronizer should prioritize remediation before ordinary new issue dispatch. Remediation consumes the same bounded worker slots as issue dispatch only when a remediation worker can actually be prepared or run; a blocked remediation prerequisite must be reported without consuming a worker slot. After scheduling the available PR-remediation workers, the runner should continue selecting unrelated eligible issues for any remaining slots in the same tick unless a clean PR is ready to merge first. It should reuse the prepared issue workspace when one exists, implement in-scope fixes, run the relevant checks, push with `--force-with-lease`, and leave concise review-thread replies or resolution only when the code/docs evidence makes the comment non-actionable. Review-remediation prompts must include thread identifiers when available so the worker can resolve or reply to the exact GitHub conversation after the fixing commit is pushed.

Actionable Codex Review feedback is agent work, not a terminal blocker. When the Synchronizer sees current unresolved Codex Review threads on an open Phase 0 PR, it must launch a review-remediation agent in the PR's existing workspace before considering unrelated new issue dispatch. The linked issue may already be closed, so remediation must not depend on the issue still appearing in the open-issue queue. The Synchronizer should match issues only from explicit PR references such as `#123` or `issue-123`, reuse a stable prepared workspace even if the issue title slug changed, preserve the PR's actual target branch in prompts and state, and verify that the prepared repo is checked out on the PR head branch before launching the agent. That agent receives the PR URL, target branch, working branch, review-thread URLs, comment excerpts, file paths, and any available original issue context. The agent must inspect the feedback, plan and implement the in-scope fix, run relevant verification, commit and push the PR branch, and reply to or resolve threads only when the branch update makes the feedback fixed or obsolete. If the PR workspace is missing, dirty, on the wrong branch, or ambiguous, the Synchronizer records a concrete remediation blocker instead of silently pausing the queue. Dry-run output must report those same live-run blockers without writing prompts, state, branches, or comments.

When a review-remediation worker succeeds and advances the PR branch, the Synchronizer should resolve the Codex Review threads that it handed to that worker before requesting or waiting on another Codex Review round. If the worker reports success without changing the branch, the runner must leave the threads open and report that no branch update was available to justify resolution.

When Codex Review leaves current actionable feedback after a PR has already merged, the Synchronizer cannot safely push fixes back to that merged PR. The implemented fallback is to create follow-up issues manually or from a watchdog; the automated merged-PR intake path is tracked by issue #325. Once that path exists, it must materialize each actionable thread as a deduplicated GitHub issue with `agent-ready`, the relevant phase label, source PR, thread id, thread URL, file path, line, severity, acceptance criteria, implementation notes, and verification expectations. These issues are ordinary runnable work unless another configured blocking label or safety rule applies.

If a PR does not have a prepared issue workspace but its branch uses an approved automation prefix, the runner may create a dedicated PR remediation workspace under `dispatch.workspace_root` from the remote PR branch. This is intended for manual or out-of-band PRs that still need Codex Review, draft, or merge-conflict remediation. Dry-run and live runs must evaluate the same safety prerequisites. The runner must still refuse unsafe branch names and missing remote branches. If the PR branch is already checked out elsewhere but the dedicated workspace is missing, the runner may create a detached worktree from `origin/<branch>` and require the worker to push with `git push --force-with-lease origin HEAD:<branch>`.

Dirty prepared PR-remediation workspaces are not terminal blockers when the workspace is already on the PR head branch. They are usually previous automation work for that same PR. The Synchronizer should pass the dirty file list into the remediation prompt, launch the remediation worker, and require the worker to inspect, finish, verify, commit, and push the in-scope edits. It must still block wrong-branch workspaces, unsafe branch names, missing remotes, and ambiguous worktrees that are not tied to the PR branch. The runner should report the dirty file list in JSON so humans can audit what was handed to the worker.

## Issue Dispatcher Runtime

The repo-owned issue dispatcher is `npm run symphony:sync`, backed by the Go CLI in `cmd/symphony`. It is the checked-in entrypoint for turning `agent-ready` GitHub issues into deterministic workspaces and branch prompts. During the Go migration, Node-backed `report` and `ui-monitor` remain direct `scripts/symphony_runner.mjs` commands, and the Go sync CLI may call that script as a legacy adapter for side-effect paths that have not yet been ported. Source scanning, work-graph construction, capacity accounting, and top-level sync status decisions belong in Go. It reads the same `.agents/WORKFLOW.md` front matter as the monitor, uses `tracker.active_labels` and `tracker.blocked_labels` for eligibility, derives each issue target branch from the issue body, and honors the `phase-0-platform-foundation` target branch when that line is present in Phase 0 issues.

The dispatcher is conservative by design:

- `npm run symphony:sync -- --dry-run --json` is non-mutating and should be the first command in any automation tick.
- A non-dry-run tick creates or reuses one workspace per selected issue under `dispatch.workspace_root`, creates a branch using `branching.branch_template`, and writes `prompt.md` plus `state.json` for the assigned run.
- A failed issue workspace may be retried automatically until `dispatch.max_attempts` is reached, provided the worktree is clean. This prevents a transient agent crash from making the issue permanently inactive.
- The dispatcher skips issues with blocked labels, missing acceptance criteria, missing `agent-ready`, or an already-open PR that references the issue.
- Skipped issues must not consume worker slots. The dispatcher must keep scanning after non-`agent-ready` candidates and must fetch explicitly labeled `agent-ready` issues in addition to the broad open-issue page so older ready work cannot be hidden behind newer unready issues.
- Before starting, continuing, or remediating issue work, the dispatcher must gather full open-issue context by reading every comment on each open issue it is evaluating. Issue comments often contain manual decisions, prior remediation attempts, branch or PR references, verification results, and changed acceptance criteria; a worker prompt is incomplete if it includes only the issue body and title. If comment retrieval fails for an otherwise eligible issue, record a concrete blocker instead of launching work with partial context.
- Open-issue discovery must paginate until the configured search scope is exhausted. Do not let a fixed page limit hide older eligible issues or make the queue appear empty.
- A stale issue workspace must not permanently consume an issue slot. If the workspace is already checked out on the issue branch but has no readable `state.json`, or the last state is `succeeded` but no PR exists yet, the dispatcher should re-enter that same workspace instead of skipping the issue forever. If the same-issue worktree has local edits, pass the dirty file list to the worker prompt as previous automation work for that issue so the worker can inspect, finish, commit, push, and open or update the PR.
- When a human manually merges a PR that references an issue, the next dispatcher tick treats that PR as resolved and merged. It must not keep the issue blocked by the old prepared workspace; instead it records `status: merged` in the matching workspace `state.json` when the file exists, removes the clean prepared `repo` worktree checkout, and reports the issue queue entry as `merged`. If the prepared worktree is dirty, it must report the dirty files instead of deleting anything. If `git worktree remove` fails only because generated cache directories such as `.gomodcache`, `.gocache`, or `node_modules/.cache` contain permission-protected files, the dispatcher should make those generated cache trees user-writable and retry teardown once before reporting a blocker.
- The dispatcher refuses to reset an existing branch, overwrite dirty worktrees, or create a workspace from a missing target branch.
- The dispatcher treats only a clean PR that is ready to merge as a reason to pause ordinary issue dispatch. Blocked, dirty, draft, or waiting-for-review PRs must be reported and remediated when possible, but they should not prevent unrelated eligible issues from using remaining worker capacity. If issue work is dispatched while a PR remains in `waiting_for_codex_review`, the top-level sync status should report the issue dispatch result instead of making the PR wait look like a global queue stop.
- `dispatch.agent_runner_command` is approved for this repo and launches `codex exec` from the prepared worktree with the rendered issue prompt on stdin. The dispatcher sets `CODEX_HOME` to a per-workspace directory under `dispatch.agent_runner_codex_home_root`, symlinks the operator's existing Codex auth/config/plugin references into that writable home, and keeps worker state databases out of the desktop app's `~/.codex` directory. This prevents automation-launched workers from failing on readonly `state_5.sqlite` or app-server state writes while still using the authenticated Codex installation. The dispatcher writes runner stdout and stderr to the issue workspace `logs/` directory and records completion or failure in `state.json`.
- The dispatcher streams worker stdout and stderr while the worker is running. If a worker produces no output for `dispatch.agent_runner_idle_timeout_ms`, the dispatcher terminates it and records a bounded failure instead of occupying the next scheduled tick. If the worker has already emitted a Codex `turn.completed` event, the dispatcher may reap the still-running process and record the run as completed with a reaper note because the agent's final response is already durable in `logs/agent-stdout.log`.

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
