#!/usr/bin/env node
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");
const workflowPath = path.join(repoRoot, ".agents", "WORKFLOW.md");
const skillsRoot = path.join(repoRoot, ".agents", "skills");

const DEFAULT_MONITOR = {
  targetBranch: "ui-improvements",
  lockPath: "/private/tmp/accountsdot-ui-improvements-github-scan.lock",
  latestCodeWorktree: "/Users/lcampbell/code.internal/accountsdot-latest-ui",
  reconcileWorktreeRoot: "/private/tmp/accountsdot-symphony-prs",
  reconcilePrBranches: true,
  safeBranchPrefixes: ["codex/", "issue-"],
  codexReviewAuthors: ["chatgpt-codex-connector", "github-copilot", "codex-review"],
  autoResolveOutdatedCodexReviewThreads: true,
  latestCodeAllowedDirty: ["frontend/dist/", "tmp/", ".vite/"],
  browserDefaultUrl: "http://localhost:5173/dashboard/it-admin",
  browserScreenshotRequired: false,
  healthUrls: [
    "http://localhost:8080/health",
    "http://localhost:5173/api/v1/dev/session",
    "http://localhost:5173/dashboard/it-admin",
  ],
  devServers: [
    {
      name: "api",
      port: 8080,
      command:
        "APP_ENV=development GOCACHE=/Users/lcampbell/code.internal/accountsdot/.gocache GOMODCACHE=/Users/lcampbell/code.internal/accountsdot/.gomodcache npm run dev:api",
    },
    {
      name: "vite",
      port: 5173,
      command: "APP_ENV=development npm run dev:web",
    },
  ],
  lockMaxAgeMs: 2 * 60 * 60 * 1000,
};

const DEFAULT_PULL_REQUESTS = {
  targetBranch: "phase-0-platform-foundation",
  inspectBeforeDispatch: true,
  autoMergeCleanPrs: false,
  mergeMethod: "squash",
  codexReviewAuthors: ["chatgpt-codex-connector", "chatgpt-codex-connector[bot]", "github-copilot", "codex-review"],
  codexReviewBot: "chatgpt-codex-connector[bot]",
  codexReviewSuccessReactions: ["THUMBS_UP", "+1"],
  noReviewWithBotThumbsUpIsClean: true,
  remediateBlockedPrs: false,
  maxReviewRemediationsPerTick: 1,
  reviewWaitPolicy: "non_blocking_stateful",
  reviewGracePeriodSeconds: 300,
};

const SKILL_RULES = [
  {
    skill: "wizard-ui-hardening",
    reason:
      "UI, design, .pen, shared shell, implemented page, route visual, or browser-evidence work needs the repo UI hardening workflow.",
    patterns: [
      /\bui\b/i,
      /\bdesign\b/i,
      /\.pen\b/i,
      /\bartboard\b/i,
      /\bwireframe\b/i,
      /\bshared shell\b/i,
      /\bbrowser evidence\b/i,
      /\bimplemented page\b/i,
      /\bdashboard\b/i,
    ],
  },
  {
    skill: "wizard-code-documentation",
    reason:
      "Implemented code, route/API, docs, comments, handler, external-write, or provider-surface work needs code documentation checks.",
    patterns: [
      /\bcode\b/i,
      /\broute\b/i,
      /\bapi\b/i,
      /\bhandler\b/i,
      /\binternal\/web\b/i,
      /\bfrontend\/src\b/i,
      /\bdocs?\b/i,
      /\bcomment/i,
      /\bexternal write\b/i,
      /\bprovider\b/i,
    ],
  },
];

function usage() {
  return [
    "Usage: node scripts/symphony_runner.mjs <command> [options]",
    "",
    "Commands:",
    "  report                 Print a read-only issue/PR queue report.",
    "  sync                   Dispatch eligible GitHub issues into repo-owned workspaces.",
    "  ui-monitor             Run the lock-protected ui-improvements monitor.",
    "  record-browser-results Record Browser plugin results into runner state.",
    "  test                   Run runner self-tests.",
    "",
    "Options:",
    "  --dry-run              Do not mutate refs, branches, issues, PRs, dev servers, Browser, or runner state.",
    "  --json                 Print JSON only.",
    "  --max-runs N           Maximum issues to dispatch in one sync tick.",
    "  --browser-results PATH JSON file for record-browser-results.",
  ].join("\n");
}

function parseArgs(argv) {
  const [command, ...rest] = argv;
  const options = { dryRun: false, json: false, browserResultsPath: "", maxRuns: null };
  const remaining = [...rest];
  while (remaining.length > 0) {
    const arg = remaining.shift();
    if (arg === "--dry-run") {
      options.dryRun = true;
    } else if (arg === "--json") {
      options.json = true;
    } else if (arg === "--browser-results") {
      options.browserResultsPath = remaining.shift() || "";
    } else if (arg === "--max-runs") {
      const rawValue = remaining.shift() || "";
      const parsed = Number(rawValue);
      if (!Number.isInteger(parsed) || parsed < 1) {
        throw new Error(`--max-runs must be a positive integer, got ${rawValue}`);
      }
      options.maxRuns = parsed;
    } else if (arg === "--help" || arg === "-h") {
      options.help = true;
    } else {
      throw new Error(`Unexpected argument: ${arg}`);
    }
  }
  return { command, options };
}

function readWorkflow(filePath = workflowPath) {
  const text = fs.readFileSync(filePath, "utf8");
  const match = text.match(/^---\n([\s\S]*?)\n---\n?/);
  if (!match) {
    throw new Error(`${filePath} is missing YAML front matter`);
  }
  return {
    config: parseSimpleYaml(match[1]),
    promptTemplate: text.slice(match[0].length),
    rawFrontMatter: match[1],
  };
}

function parseSimpleYaml(source) {
  const root = {};
  const stack = [{ indent: -1, value: root }];
  const lines = source.split(/\r?\n/);

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex += 1) {
    const rawLine = lines[lineIndex];
    if (!rawLine.trim() || rawLine.trim().startsWith("#")) continue;
    const indent = rawLine.match(/^ */)[0].length;
    const line = rawLine.trim();
    while (stack.length > 1 && indent <= stack[stack.length - 1].indent) {
      stack.pop();
    }
    const parent = stack[stack.length - 1].value;
    if (line.startsWith("- ")) {
      if (!Array.isArray(parent)) {
        throw new Error(`List item without list parent: ${line}`);
      }
      parent.push(parseScalar(line.slice(2)));
      continue;
    }
    const keyMatch = line.match(/^([^:]+):(.*)$/);
    if (!keyMatch) {
      throw new Error(`Unsupported YAML line: ${line}`);
    }
    const key = keyMatch[1].trim();
    const rest = keyMatch[2].trim();
    if (rest) {
      parent[key] = parseScalar(rest);
      continue;
    }
    const nextLine = nextContentLine(lines, lineIndex);
    const nextTrimmed = nextLine ? nextLine.trim() : "";
    const child = nextTrimmed.startsWith("- ") ? [] : {};
    parent[key] = child;
    stack.push({ indent, value: child });
  }
  return root;
}

function nextContentLine(lines, currentIndex) {
  for (let i = currentIndex + 1; i < lines.length; i += 1) {
    if (lines[i].trim() && !lines[i].trim().startsWith("#")) return lines[i];
  }
  return "";
}

function parseScalar(value) {
  if (value === "true") return true;
  if (value === "false") return false;
  if (/^-?\d+$/.test(value)) return Number(value);
  return value.replace(/^["']|["']$/g, "");
}

function discoverSkills(root = skillsRoot) {
  if (!fs.existsSync(root)) return [];
  return fs
    .readdirSync(root, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((entry) => {
      const skillPath = path.join(root, entry.name, "SKILL.md");
      if (!fs.existsSync(skillPath)) {
        return {
          skill_name: entry.name,
          skill_path: skillPath,
          reason_selected: "",
          instructions_included: false,
          missing_or_blocked: `Missing ${skillPath}`,
          summary: "",
        };
      }
      const body = fs.readFileSync(skillPath, "utf8");
      return {
        skill_name: entry.name,
        skill_path: skillPath,
        reason_selected: "",
        instructions_included: false,
        missing_or_blocked: "",
        summary: summarizeSkill(body),
      };
    });
}

function summarizeSkill(markdown) {
  const lines = markdown
    .split(/\r?\n/)
    .filter((line) => line.trim() && !line.startsWith("---"))
    .slice(0, 24);
  return lines.join("\n");
}

function routeSkills(text, skills = discoverSkills()) {
  const selected = [];
  const lowered = text || "";
  for (const rule of SKILL_RULES) {
    if (!rule.patterns.some((pattern) => pattern.test(lowered))) continue;
    const skill = skills.find((candidate) => candidate.skill_name === rule.skill);
    if (!skill) {
      selected.push({
        skill_name: rule.skill,
        skill_path: path.join(skillsRoot, rule.skill, "SKILL.md"),
        reason_selected: rule.reason,
        instructions_included: false,
        missing_or_blocked: "Skill was expected but not discovered",
      });
      continue;
    }
    selected.push({
      ...skill,
      reason_selected: rule.reason,
      instructions_included: !skill.missing_or_blocked,
    });
  }
  return selected;
}

function run(cmd, args, { cwd = repoRoot, allowFailure = false } = {}) {
  try {
    return execFileSync(cmd, args, {
      cwd,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
  } catch (error) {
    if (allowFailure) {
      return {
        failed: true,
        status: error.status,
        stdout: String(error.stdout || "").trim(),
        stderr: String(error.stderr || error.message || "").trim(),
      };
    }
    throw error;
  }
}

function ghJson(args) {
  let lastError = null;
  for (let attempt = 1; attempt <= 3; attempt += 1) {
    try {
      const output = run("gh", args);
      return output ? JSON.parse(output) : null;
    } catch (error) {
      lastError = error;
      if (attempt === 3 || !isTransientGhError(error)) {
        throw error;
      }
      sleepMs(250 * attempt);
    }
  }
  throw lastError;
}

function isTransientGhError(error) {
  const text = `${error?.stdout || ""}\n${error?.stderr || ""}\n${error?.message || ""}`;
  return /\b(502|503|504)\b/i.test(text) || /Bad Gateway|Service Unavailable|Gateway Timeout|timed out/i.test(text);
}

function sleepMs(milliseconds) {
  Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, milliseconds);
}

function listOpenPullRequests(baseRef) {
  return ghJson([
    "pr",
    "list",
    "--state",
    "open",
    "--base",
    baseRef,
    "--json",
    "number,title,url,headRefName,isDraft,mergeStateStatus,reviewDecision,headRefOid,statusCheckRollup,updatedAt,labels",
    "--limit",
    "100",
  ]);
}

function listOpenPullRequestsForBases(baseRefs) {
  const seen = new Set();
  const prs = [];
  for (const baseRef of [...new Set(baseRefs.filter(Boolean))]) {
    for (const pr of listOpenPullRequests(baseRef)) {
      if (seen.has(pr.number)) continue;
      seen.add(pr.number);
      prs.push({ ...pr, baseRefName: baseRef });
    }
  }
  return prs;
}

function listMergedPullRequests(baseRef) {
  return ghJson([
    "pr",
    "list",
    "--state",
    "merged",
    "--base",
    baseRef,
    "--json",
    "number,title,url,headRefName,mergedAt,labels",
    "--limit",
    "200",
  ]);
}

function listMergedPullRequestsForBases(baseRefs) {
  const seen = new Set();
  const prs = [];
  for (const baseRef of [...new Set(baseRefs.filter(Boolean))]) {
    for (const pr of listMergedPullRequests(baseRef)) {
      if (seen.has(pr.number)) continue;
      seen.add(pr.number);
      prs.push({ ...pr, baseRefName: baseRef });
    }
  }
  return prs;
}

function listOpenIssues() {
  return ghJson([
    "issue",
    "list",
    "--state",
    "open",
    "--json",
    "number,title,body,labels,url,updatedAt,assignees",
    "--limit",
    "200",
  ]);
}

function fetchReviewThreads(baseRef) {
  const query =
    'query($owner:String!,$repo:String!,$base:String!){ repository(owner:$owner,name:$repo){ pullRequests(first:100, states:OPEN, baseRefName:$base) { nodes { number reviewThreads(first:100) { nodes { id isResolved isOutdated comments(first:10){ nodes { author { login } body createdAt path line originalLine url } } } } } } } }';
  const result = ghJson([
    "api",
    "graphql",
    "-f",
    `query=${query}`,
    "-F",
    "owner=herooftimeandspace",
    "-F",
    "repo=accountsdot",
    "-F",
    `base=${baseRef}`,
  ]);
  return result.data.repository.pullRequests.nodes.map((pr) => ({
    number: pr.number,
    review_threads: pr.reviewThreads.nodes,
    unresolved_threads: pr.reviewThreads.nodes.filter((thread) => !thread.isResolved && !thread.isOutdated),
    outdated_unresolved_threads: pr.reviewThreads.nodes.filter((thread) => !thread.isResolved && thread.isOutdated),
  }));
}

function fetchPullRequestReviewSignals(baseRef) {
  const query =
    'query($owner:String!,$repo:String!,$base:String!){ repository(owner:$owner,name:$repo){ pullRequests(first:100, states:OPEN, baseRefName:$base) { nodes { number reactionGroups { content users(first:20){ nodes { login } } } reviews(first:50){ nodes { author { login } state submittedAt } } comments(first:50){ nodes { author { login } body createdAt reactionGroups { content users(first:20){ nodes { login } } } } } } } } }';
  const result = ghJson([
    "api",
    "graphql",
    "-f",
    `query=${query}`,
    "-F",
    "owner=herooftimeandspace",
    "-F",
    "repo=accountsdot",
    "-F",
    `base=${baseRef}`,
  ]);
  return new Map(
    result.data.repository.pullRequests.nodes.map((pr) => [
      pr.number,
      {
        reviews: pr.reviews.nodes,
        comments: pr.comments.nodes,
        reactionGroups: pr.reactionGroups || [],
      },
    ]),
  );
}

function queuePullRequests(prs) {
  return [...prs].sort((a, b) => {
    if (a.isDraft !== b.isDraft) return a.isDraft ? 1 : -1;
    const cleanA = a.mergeStateStatus === "CLEAN";
    const cleanB = b.mergeStateStatus === "CLEAN";
    if (cleanA !== cleanB) return cleanA ? -1 : 1;
    return a.number - b.number;
  });
}

function queueReason(pr) {
  if (pr.isDraft) return "draft PR; keep behind ready non-draft work";
  if (pr.mergeStateStatus !== "CLEAN") return `merge state ${pr.mergeStateStatus}; resolve before merge`;
  if (!pr.statusCheckRollup || pr.statusCheckRollup.length === 0) {
    return "merge-clean but no required check rollup is present";
  }
  return "merge-clean non-draft PR";
}

function resolveReviewThread(threadId) {
  const query = "mutation($threadId:ID!){ resolveReviewThread(input:{threadId:$threadId}) { thread { id isResolved } } }";
  return ghJson(["api", "graphql", "-f", `query=${query}`, "-F", `threadId=${threadId}`]);
}

function commentAuthor(comment) {
  return comment?.author?.login || "";
}

function isCodexReviewThread(thread, config = DEFAULT_MONITOR) {
  const authors = new Set((config.codexReviewAuthors || []).map((author) => String(author).toLowerCase()));
  return thread.comments.nodes.some((comment) => authors.has(commentAuthor(comment).toLowerCase()));
}

function summarizeReviewThread(thread) {
  const firstComment = thread.comments.nodes[0] || {};
  return {
    thread_id: thread.id,
    author: commentAuthor(firstComment),
    path: firstComment.path || "",
    line: firstComment.line || firstComment.originalLine || null,
    url: firstComment.url || "",
    is_outdated: Boolean(thread.isOutdated),
    body_excerpt: String(firstComment.body || "").replace(/\s+/g, " ").slice(0, 240),
  };
}

function remediateReviewThreads({ reviewThreads, config, dryRun }) {
  const results = [];
  for (const pr of reviewThreads) {
    for (const thread of pr.review_threads || []) {
      if (thread.isResolved || !isCodexReviewThread(thread, config)) continue;
      const result = {
        number: pr.number,
        action: "skipped",
        status: "skipped",
        reason: "",
        ...summarizeReviewThread(thread),
      };
      results.push(result);

      if (thread.isOutdated) {
        result.action = dryRun ? "would-resolve-outdated-thread" : "resolve-outdated-thread";
        if (dryRun) {
          result.status = "dry-run";
          result.reason = "dry-run performs no review-thread mutations";
          continue;
        }
        if (!config.autoResolveOutdatedCodexReviewThreads) {
          result.status = "blocked";
          result.reason = "auto_resolve_outdated_codex_review_threads is disabled";
          continue;
        }
        try {
          resolveReviewThread(thread.id);
          result.status = "resolved";
          result.reason = "outdated Codex Review thread resolved after branch changes made the comment obsolete";
        } catch (error) {
          result.status = "blocked";
          result.reason = error.stderr || error.message || String(error);
        }
        continue;
      }

      result.action = "requires-code-remediation";
      result.status = "blocked";
      result.reason =
        "active Codex Review thread still needs an in-scope code/docs fix before the automation may resolve it";
    }
  }
  return results;
}

function normalizedLogin(login) {
  return String(login || "")
    .toLowerCase()
    .replace(/\[bot\]$/i, "");
}

function authorMatches(login, authors) {
  const normalized = normalizedLogin(login);
  return authors.some((author) => normalizedLogin(author) === normalized);
}

function statusRollupState(rollup) {
  if (!rollup || (Array.isArray(rollup) && rollup.length === 0)) return "none";
  const entries = Array.isArray(rollup) ? rollup : rollup.nodes || [];
  if (entries.length === 0) return "none";
  const failing = entries.filter((entry) => {
    const value = String(entry.conclusion || entry.state || entry.status || "").toUpperCase();
    return value && !["SUCCESS", "SKIPPED", "NEUTRAL", "COMPLETED"].includes(value);
  });
  return failing.length > 0 ? "failing" : "passing";
}

function reviewThreadsForPr(reviewThreads, number) {
  return reviewThreads.find((entry) => entry.number === number) || {
    number,
    review_threads: [],
    unresolved_threads: [],
    outdated_unresolved_threads: [],
  };
}

function hasBotSuccessReaction(signals, config) {
  const successReactions = new Set((config.codexReviewSuccessReactions || []).map((reaction) => String(reaction).toUpperCase()));
  const botLogin = config.codexReviewBot || "";
  const hasSuccessReaction = (reactionGroups) =>
    (reactionGroups || []).some((group) => {
      if (!successReactions.has(String(group.content || "").toUpperCase())) return false;
      return (group.users?.nodes || []).some((user) => authorMatches(user.login, [botLogin]));
    });
  if (hasSuccessReaction(signals.reactionGroups)) return true;
  return (signals.comments || []).some((comment) => hasSuccessReaction(comment.reactionGroups));
}

function hasCodexReviewResponse({ signals, threadEntry, config }) {
  const authors = config.codexReviewAuthors || [];
  if ((signals.reviews || []).some((review) => authorMatches(review.author?.login, authors))) return true;
  return (threadEntry.review_threads || []).some((thread) => isCodexReviewThread(thread, { codexReviewAuthors: authors }));
}

function hasRequestedChanges(signals, config) {
  const authors = config.codexReviewAuthors || [];
  return (signals.reviews || []).some(
    (review) => authorMatches(review.author?.login, authors) && String(review.state || "").toUpperCase() === "CHANGES_REQUESTED",
  );
}

function evaluatePullRequestForMerge({ pr, reviewThreads, signals, config }) {
  const threadEntry = reviewThreadsForPr(reviewThreads, pr.number);
  const blockers = [];
  const warnings = [];
  const labelBlockers = prLabelNames(pr).filter((label) => (config.blockedLabels || []).includes(label));
  if (pr.isDraft) blockers.push("draft PR");
  if (pr.mergeStateStatus !== "CLEAN") blockers.push(`merge state ${pr.mergeStateStatus}`);
  if (labelBlockers.length > 0) blockers.push(`blocked labels: ${labelBlockers.join(", ")}`);
  const checkState = statusRollupState(pr.statusCheckRollup);
  if (checkState === "failing") blockers.push("status checks are failing or pending");
  if (threadEntry.unresolved_threads.some((thread) => isCodexReviewThread(thread, { codexReviewAuthors: config.codexReviewAuthors }))) {
    blockers.push("unresolved current Codex Review thread");
  }
  if (hasRequestedChanges(signals, config)) blockers.push("Codex Review requested changes");

  const codexReviewResponse = hasCodexReviewResponse({ signals, threadEntry, config });
  const botThumbsUp = hasBotSuccessReaction(signals, config);
  if (!codexReviewResponse && !botThumbsUp) {
    warnings.push("waiting for Codex Review response or bot thumbs-up reaction");
  }
  if (!codexReviewResponse && botThumbsUp && config.noReviewWithBotThumbsUpIsClean) {
    warnings.push("no Codex Review response, but chatgpt-codex-connector bot thumbs-up is configured as clean evidence");
  }

  const ready = blockers.length === 0 && (codexReviewResponse || (botThumbsUp && config.noReviewWithBotThumbsUpIsClean));
  return {
    number: pr.number,
    title: pr.title,
    url: pr.url || "",
    head_ref: pr.headRefName,
    target_branch: config.targetBranch,
    merge_state: pr.mergeStateStatus,
    check_state: checkState,
    codex_review_response: codexReviewResponse,
    bot_thumbs_up: botThumbsUp,
    unresolved_codex_review_threads: threadEntry.unresolved_threads.filter((thread) =>
      isCodexReviewThread(thread, { codexReviewAuthors: config.codexReviewAuthors }),
    ).length,
    unresolved_codex_review_thread_summaries: threadEntry.unresolved_threads
      .filter((thread) => isCodexReviewThread(thread, { codexReviewAuthors: config.codexReviewAuthors }))
      .map(summarizeReviewThread),
    status: ready ? "ready_to_merge" : blockers.length > 0 ? "blocked" : "waiting_for_codex_review",
    blockers,
    notes: warnings,
  };
}

function mergePullRequest({ evaluation, config, dryRun }) {
  const result = {
    number: evaluation.number,
    action: dryRun ? "would-merge" : "merge",
    status: dryRun ? "dry-run" : "skipped",
    merge_method: config.mergeMethod,
    reason: "",
  };
  if (evaluation.status !== "ready_to_merge") {
    result.status = "blocked";
    result.reason = evaluation.blockers.concat(evaluation.notes).join("; ");
    return result;
  }
  if (!config.autoMergeCleanPrs) {
    result.status = "blocked";
    result.reason = "auto_merge_clean_prs is disabled";
    return result;
  }
  if (dryRun) {
    result.reason = "dry-run performs no PR merge";
    return result;
  }
  const mergeFlag = config.mergeMethod === "merge" ? "--merge" : config.mergeMethod === "rebase" ? "--rebase" : "--squash";
  const output = run("gh", ["pr", "merge", String(evaluation.number), mergeFlag], { allowFailure: true });
  if (output.failed) {
    result.status = "blocked";
    result.reason = output.stderr || output.stdout || `gh pr merge failed with status ${output.status}`;
    return result;
  }
  result.status = "merged";
  result.reason = output || `PR #${evaluation.number} merged with ${config.mergeMethod}`;
  return result;
}

function openIssuesWithoutOpenPr(issues, prs) {
  const prText = prs.map((pr) => `${pr.number} ${pr.title} ${pr.headRefName}`).join("\n").toLowerCase();
  return issues.filter((issue) => !prText.includes(`#${issue.number}`) && !prText.includes(`issue-${issue.number}`));
}

function issueLabelNames(issue) {
  return (issue.labels || []).map((label) => String(label.name || label).toLowerCase());
}

function prLabelNames(pr) {
  return (pr.labels || []).map((label) => String(label.name || label).toLowerCase());
}

function readDispatchConfig(workflowConfig) {
  const dispatch = workflowConfig.dispatch || {};
  const tracker = workflowConfig.tracker || {};
  const branching = workflowConfig.branching || {};
  const workspaceRoot = dispatch.workspace_root || path.join(os.tmpdir(), "accountsdot-symphony");
  return {
    activeLabels: (tracker.active_labels || ["agent-ready"]).map((label) => String(label).toLowerCase()),
    blockedLabels: (tracker.blocked_labels || []).map((label) => String(label).toLowerCase()),
    defaultTargetBranch: branching.integration_branch || "dev",
    branchPrefix: branching.branch_prefix || "codex/",
    branchTemplate: branching.branch_template || "codex/issue-{number}-{slug}",
    maxConcurrentRuns: Number(dispatch.max_concurrent_runs || 1),
    maxAttempts: Number(dispatch.max_attempts || 1),
    requireExplicitAgentReadyLabel: dispatch.require_explicit_agent_ready_label !== false,
    workspaceRoot,
    agentRunnerCommand: dispatch.agent_runner_command || "",
    agentRunnerCodexHomeRoot:
      dispatch.agent_runner_codex_home_root || path.join(workspaceRoot, ".codex-agent-homes"),
    agentRunnerTimeoutMs: Number(dispatch.agent_runner_timeout_ms || 6 * 60 * 60 * 1000),
  };
}

function readPullRequestConfig(workflowConfig, dispatchConfig) {
  const configured = workflowConfig.pull_requests || {};
  return {
    ...DEFAULT_PULL_REQUESTS,
    targetBranch: configured.target_branch || DEFAULT_PULL_REQUESTS.targetBranch || dispatchConfig.defaultTargetBranch,
    inspectBeforeDispatch:
      configured.inspect_before_dispatch === undefined
        ? DEFAULT_PULL_REQUESTS.inspectBeforeDispatch
        : Boolean(configured.inspect_before_dispatch),
    autoMergeCleanPrs:
      configured.auto_merge_clean_prs === undefined
        ? DEFAULT_PULL_REQUESTS.autoMergeCleanPrs
        : Boolean(configured.auto_merge_clean_prs),
    mergeMethod: configured.merge_method || DEFAULT_PULL_REQUESTS.mergeMethod,
    codexReviewAuthors: configured.codex_review_authors || DEFAULT_PULL_REQUESTS.codexReviewAuthors,
    codexReviewBot: configured.codex_review_bot || DEFAULT_PULL_REQUESTS.codexReviewBot,
    codexReviewSuccessReactions:
      configured.codex_review_success_reactions || DEFAULT_PULL_REQUESTS.codexReviewSuccessReactions,
    noReviewWithBotThumbsUpIsClean:
      configured.no_review_with_bot_thumbs_up_is_clean === undefined
        ? DEFAULT_PULL_REQUESTS.noReviewWithBotThumbsUpIsClean
        : Boolean(configured.no_review_with_bot_thumbs_up_is_clean),
    remediateBlockedPrs:
      configured.remediate_blocked_prs === undefined
        ? DEFAULT_PULL_REQUESTS.remediateBlockedPrs
        : Boolean(configured.remediate_blocked_prs),
    maxReviewRemediationsPerTick: Number(
      configured.max_review_remediations_per_tick || DEFAULT_PULL_REQUESTS.maxReviewRemediationsPerTick,
    ),
    reviewWaitPolicy: configured.review_wait_policy || DEFAULT_PULL_REQUESTS.reviewWaitPolicy,
    reviewGracePeriodSeconds: Number(
      configured.review_grace_period_seconds || DEFAULT_PULL_REQUESTS.reviewGracePeriodSeconds,
    ),
    blockedLabels: (workflowConfig.tracker?.blocked_labels || []).map((label) => String(label).toLowerCase()),
  };
}

function issueHasAcceptanceCriteria(issue) {
  const body = String(issue.body || "");
  return /acceptance criteria/i.test(body) || /-\s+\[[ xX]\]/.test(body);
}

function issueTargetBranch(issue, dispatchConfig) {
  const body = String(issue.body || "");
  for (const line of body.split(/\r?\n/)) {
    const match = line.match(/^\s*Target branch:\s*`?([A-Za-z0-9._/-]+)`?(?=[.,;)]|\s|$)/i);
    if (match) return match[1].replace(/[.`]+$/g, "");
  }
  return dispatchConfig.defaultTargetBranch;
}

function issueSlug(issue) {
  return safePathSegment(
    String(issue.title || `issue-${issue.number}`)
      .toLowerCase()
      .replace(/^p0-[0-9a-z-]+:\s*/i, "")
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 48),
  );
}

function issueBranchName(issue, dispatchConfig) {
  return dispatchConfig.branchTemplate
    .replace("{number}", String(issue.number))
    .replace("{slug}", issueSlug(issue));
}

function issuePriorityScore(issue, dispatchConfig) {
  const labels = new Set(issueLabelNames(issue));
  let score = 0;
  if (labels.has("phase-0")) score -= 1000;
  if (labels.has("documentation")) score -= 80;
  if (labels.has("enhancement")) score -= 40;
  if (String(issue.title || "").startsWith("P0-")) score -= 200;
  if (issueTargetBranch(issue, dispatchConfig) !== dispatchConfig.defaultTargetBranch) score -= 20;
  return score + Number(issue.number || 0);
}

function prReferencesIssue(pr, issue) {
  const text = `${pr.title || ""} ${pr.headRefName || ""}`.toLowerCase();
  return text.includes(`#${issue.number}`) || text.includes(`issue-${issue.number}-`);
}

function readIssueWorkspaceState(issue, dispatchConfig) {
  const statePath = path.join(findIssueWorkspace(issue, dispatchConfig), "state.json");
  if (!fs.existsSync(statePath)) return null;
  try {
    return JSON.parse(fs.readFileSync(statePath, "utf8"));
  } catch {
    return null;
  }
}

function mergedPullRequestForIssueWithWorkspace(issue, mergedPrs, targetBranch, dispatchConfig) {
  const state = readIssueWorkspaceState(issue, dispatchConfig);
  return (
    mergedPrs
      .filter((pr) => prReferencesIssue(pr, issue) && (!targetBranch || pr.baseRefName === targetBranch))
      .filter((pr) => {
        if (issue.updatedAt && pr.mergedAt && new Date(issue.updatedAt) > new Date(pr.mergedAt)) return false;
        if (!state) return true;
        const stateBranch = String(state.branch || "");
        return stateBranch === pr.headRefName || issueBranchName(issue, dispatchConfig) === pr.headRefName;
      })
      .sort((a, b) => String(b.mergedAt || "").localeCompare(String(a.mergedAt || "")))[0] || null
  );
}

function activeWorkspaceForIssue(issue, dispatchConfig) {
  const workspacePath = issueWorkspacePath(issue, dispatchConfig);
  const statePath = path.join(workspacePath, "state.json");
  const promptPath = path.join(workspacePath, "prompt.md");
  if (fs.existsSync(statePath)) {
    try {
      const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
      const activeStatuses = new Set(["prepared", "running", "waiting_retry", "human_review", "succeeded", "failed"]);
      if (activeStatuses.has(state.status)) {
        const attempts = Number(state.attempts || 1);
        return {
          workspace: workspacePath,
          prompt_path: promptPath,
          state_path: statePath,
          status: state.status,
          attempts,
          retryable: state.status === "failed" && attempts < dispatchConfig.maxAttempts,
        };
      }
    } catch (error) {
      return { workspace: workspacePath, prompt_path: promptPath, state_path: statePath, status: "unreadable_state", blocker: error.message };
    }
  }
  const branchName = issueBranchName(issue, dispatchConfig);
  const existingWorktree = worktreeForBranch(branchName);
  if (existingWorktree) {
    return { workspace: path.dirname(existingWorktree.path), prompt_path: promptPath, state_path: statePath, status: "branch_checked_out" };
  }
  return null;
}

function classifyIssueForDispatch(issue, prs, mergedPrs, dispatchConfig) {
  const labels = new Set(issueLabelNames(issue));
  const targetBranch = issueTargetBranch(issue, dispatchConfig);
  const branchName = issueBranchName(issue, dispatchConfig);
  const openPr = prs.find((pr) => prReferencesIssue(pr, issue));
  const mergedPr = mergedPullRequestForIssueWithWorkspace(issue, mergedPrs, targetBranch, dispatchConfig);
  const activeWorkspace = mergedPr ? null : activeWorkspaceForIssue(issue, dispatchConfig);
  const reasons = [];
  const blockedBy = [...labels].filter((label) => dispatchConfig.blockedLabels.includes(label));

  if (mergedPr) {
    return {
      number: issue.number,
      title: issue.title,
      url: issue.url,
      labels: [...labels].sort(),
      target_branch: targetBranch,
      branch: branchName,
      workspace: "",
      prompt_path: "",
      state_path: "",
      open_pr: null,
      merged_pr: {
        number: mergedPr.number,
        head: mergedPr.headRefName,
        base: mergedPr.baseRefName || "",
        url: mergedPr.url,
        merged_at: mergedPr.mergedAt || "",
      },
      eligible: false,
      status: "merged",
      reason: `merged PR #${mergedPr.number}`,
      priority_score: Number.MAX_SAFE_INTEGER - Number(issue.number || 0),
    };
  }

  if (dispatchConfig.requireExplicitAgentReadyLabel) {
    const hasActiveLabel = dispatchConfig.activeLabels.some((label) => labels.has(label));
    if (!hasActiveLabel) reasons.push(`missing active label (${dispatchConfig.activeLabels.join(", ")})`);
  }
  if (blockedBy.length > 0) reasons.push(`blocked by label: ${blockedBy.join(", ")}`);
  if (!issueHasAcceptanceCriteria(issue)) reasons.push("missing acceptance criteria");
  if (openPr) reasons.push(`open PR already references issue (#${openPr.number})`);
  if (activeWorkspace && !activeWorkspace.retryable) reasons.push(`active workspace already exists (${activeWorkspace.status})`);
  if (!targetBranch) reasons.push("missing target branch");

  return {
    number: issue.number,
    title: issue.title,
    url: issue.url,
    labels: [...labels].sort(),
    target_branch: targetBranch,
    branch: branchName,
    workspace: activeWorkspace?.workspace || "",
    prompt_path: activeWorkspace?.prompt_path || "",
    state_path: activeWorkspace?.state_path || "",
    open_pr: openPr ? { number: openPr.number, head: openPr.headRefName, base: openPr.baseRefName || "" } : null,
    merged_pr: null,
    eligible: reasons.length === 0,
    status: reasons.length === 0 ? "eligible" : "skipped",
    reason: reasons.join("; "),
    priority_score: issuePriorityScore(issue, dispatchConfig),
  };
}

function rankedDispatchQueue(issues, prs, mergedPrs, dispatchConfig) {
  return issues
    .map((issue) => classifyIssueForDispatch(issue, prs, mergedPrs, dispatchConfig))
    .sort((a, b) => {
      if (a.eligible !== b.eligible) return a.eligible ? -1 : 1;
      if (a.priority_score !== b.priority_score) return a.priority_score - b.priority_score;
      return a.number - b.number;
    })
    .map((entry, index) => ({ rank: index + 1, ...entry }));
}

function issueWorkspacePath(issue, dispatchConfig) {
  return path.join(dispatchConfig.workspaceRoot, `issue-${issue.number}-${issueSlug(issue)}`);
}

function markMergedIssueWorkspaceStates({ issues, mergedPrs, dispatchConfig, dryRun }) {
  const results = [];
  for (const issue of issues) {
    const workspacePath = findIssueWorkspace(issue, dispatchConfig);
    const statePath = path.join(workspacePath, "state.json");
    if (!fs.existsSync(statePath)) continue;
    const baseResult = {
      issue_number: issue.number,
      workspace: workspacePath,
      state_path: statePath,
      status: dryRun ? "would-mark-merged" : "merged",
      teardown_status: "not-needed",
    };
    try {
      const current = JSON.parse(fs.readFileSync(statePath, "utf8"));
      const mergedPr =
        mergedPrs
          .filter((pr) => prReferencesIssue(pr, issue) && pr.baseRefName === issueTargetBranch(issue, dispatchConfig))
          .filter((pr) => !issue.updatedAt || !pr.mergedAt || new Date(issue.updatedAt) <= new Date(pr.mergedAt))
          .filter((pr) => String(current.branch || "") === pr.headRefName || issueBranchName(issue, dispatchConfig) === pr.headRefName)
          .sort((a, b) => String(b.mergedAt || "").localeCompare(String(a.mergedAt || "")))[0] || null;
      if (!mergedPr) continue;
      const result = {
        ...baseResult,
        pr_number: mergedPr.number,
      };
      const repoPath = path.join(workspacePath, "repo");
      if (fs.existsSync(repoPath)) {
        const status = cleanStatus(repoPath);
        if (!status.clean) {
          results.push({ ...result, status: "blocked", teardown_status: "blocked", reason: status.blocker, dirty_files: status.dirty_files });
          continue;
        }
        if (!dryRun) {
          const removed = run("git", ["worktree", "remove", repoPath], { cwd: repoRoot, allowFailure: true });
          if (removed.failed) {
            results.push({
              ...result,
              status: "blocked",
              teardown_status: "blocked",
              reason: removed.stderr || removed.stdout || `git worktree remove failed with status ${removed.status}`,
            });
            continue;
          }
        }
        result.teardown_status = dryRun ? "would-remove-clean-worktree" : "removed-clean-worktree";
      }
      if (current.status === "merged" && current.merged_pr === mergedPr.number) {
        result.status = "already_merged";
        results.push(result);
        continue;
      }
      if (!dryRun) {
        fs.writeFileSync(
          statePath,
          `${JSON.stringify(
            {
              ...current,
              status: "merged",
              merged_pr: mergedPr.number,
              merged_pr_url: mergedPr.url,
              merged_at: mergedPr.mergedAt || new Date().toISOString(),
              resolved_by: "manual-or-external-pr-merge",
            },
            null,
            2,
          )}\n`,
        );
      }
      results.push(result);
    } catch (error) {
      results.push({ ...baseResult, status: "blocked", reason: error.message });
    }
  }
  return results;
}

function shouldRemediateBlockedPullRequest(entry) {
  if (entry.status !== "blocked" || !entry.head_ref) return false;
  if (entry.unresolved_codex_review_threads > 0 && entry.unresolved_codex_review_thread_summaries.length > 0) return true;
  return entry.blockers.some((blocker) => /^merge state /i.test(blocker) || blocker === "draft PR");
}

function remediationConsumesDispatchSlot(remediation) {
  return ["would-remediate", "prepared", "succeeded", "failed"].includes(remediation.status);
}

function renderIssuePrompt({ issue, dispatchConfig, workflow, skills }) {
  const routedSkills = routeSkills(`${issue.title}\n${issue.body || ""}`, skills);
  const skillSection =
    routedSkills.length === 0
      ? "No repo-local skill matched this issue. Apply the base workflow and source-of-truth order."
      : routedSkills
          .map(
            (skill) =>
              `## ${skill.skill_name}\nPath: ${skill.skill_path}\nReason: ${skill.reason_selected}\nStatus: ${
                skill.missing_or_blocked || "included"
              }\n\n${skill.summary || ""}`,
          )
          .join("\n\n");
  const targetBranch = issueTargetBranch(issue, dispatchConfig);
  const branchName = issueBranchName(issue, dispatchConfig);
  return [
    "# Symphony Issue Dispatch Prompt",
    "",
    `Issue: #${issue.number} ${issue.title}`,
    `Issue URL: ${issue.url}`,
    `Target branch: ${targetBranch}`,
    `Working branch: ${branchName}`,
    `Workspace root: ${issueWorkspacePath(issue, dispatchConfig)}`,
    "",
    "## Issue Body",
    "",
    issue.body || "(No issue body was returned by GitHub.)",
    "",
    "## Repo Workflow Prompt",
    "",
    workflow.promptTemplate.trim(),
    "",
    "## Applicable Repo Skills",
    "",
    skillSection,
    "",
    "## Dispatcher Contract",
    "",
    "- Work only in the prepared issue worktree and branch.",
    "- Keep PRs for this issue targeted to the target branch above.",
    "- Do not perform production writes, provider writeback, secret disclosure, destructive git, or manual merges.",
    "- Update `/docs` for any implemented behavior, setup, schema, API, database access, or operator workflow change.",
    "- Commit completed work on the prepared branch, push it, and open a PR against the target branch when verification passes.",
    "- Finish with changed files, verification evidence, safety notes, and PR URL if one is opened.",
    "",
  ].join("\n");
}

function splitCommandLine(commandLine) {
  const tokens = [];
  let current = "";
  let quote = "";
  let escaped = false;
  for (const char of String(commandLine)) {
    if (escaped) {
      current += char;
      escaped = false;
      continue;
    }
    if (char === "\\") {
      escaped = true;
      continue;
    }
    if (quote) {
      if (char === quote) {
        quote = "";
      } else {
        current += char;
      }
      continue;
    }
    if (char === "'" || char === '"') {
      quote = char;
      continue;
    }
    if (/\s/.test(char)) {
      if (current) {
        tokens.push(current);
        current = "";
      }
      continue;
    }
    current += char;
  }
  if (escaped) current += "\\";
  if (quote) throw new Error(`Unclosed quote in command: ${commandLine}`);
  if (current) tokens.push(current);
  return tokens;
}

function renderCommandToken(token, replacements) {
  return String(token).replace(/\{([a-z_]+)\}/g, (match, key) => {
    if (!(key in replacements)) return match;
    return replacements[key];
  });
}

function ensureSymlink(target, linkPath) {
  try {
    const existing = fs.lstatSync(linkPath);
    if (existing.isSymbolicLink() && fs.readlinkSync(linkPath) === target) return;
    if (existing.isSymbolicLink()) {
      fs.unlinkSync(linkPath);
    } else {
      return;
    }
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
  fs.symlinkSync(target, linkPath);
}

function prepareAgentCodexHome({ codexHomeRoot, workspacePath }) {
  if (!codexHomeRoot) return null;
  const workspaceName = safePathSegment(path.basename(workspacePath || "workspace")) || "workspace";
  const codexHome = path.join(codexHomeRoot, workspaceName);
  fs.mkdirSync(codexHome, { recursive: true, mode: 0o700 });
  const sourceHome = path.resolve(process.env.CODEX_HOME || path.join(os.homedir(), ".codex"));
  for (const entry of ["auth.json", "config.toml", "installation_id", "AGENTS.md"]) {
    const sourcePath = path.join(sourceHome, entry);
    if (fs.existsSync(sourcePath)) ensureSymlink(sourcePath, path.join(codexHome, entry));
  }
  for (const entry of ["plugins", "skills"]) {
    const sourcePath = path.join(sourceHome, entry);
    if (fs.existsSync(sourcePath)) ensureSymlink(sourcePath, path.join(codexHome, entry));
  }
  return codexHome;
}

function runAgentRunner({
  commandLine,
  repoPath,
  promptPath,
  prompt,
  issue,
  branchName,
  targetBranch,
  logsPath,
  timeoutMs,
  codexHomeRoot,
  workspacePath,
}) {
  const tokens = splitCommandLine(commandLine);
  if (tokens.length === 0) {
    return { status: "skipped", runner_status: "needs_agent_runner", reason: "agent runner command is empty" };
  }
  const replacements = {
    repo: repoPath,
    prompt_path: promptPath,
    issue_number: issue ? String(issue.number) : "",
    issue_url: issue?.url || "",
    branch: branchName,
    target_branch: targetBranch,
  };
  const [cmd, ...args] = tokens.map((token) => renderCommandToken(token, replacements));
  const startedAt = new Date().toISOString();
  let codexHome = "";
  try {
    codexHome = prepareAgentCodexHome({ codexHomeRoot, workspacePath }) || "";
    const stdout = execFileSync(cmd, args, {
      cwd: repoPath,
      input: prompt,
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
      timeout: timeoutMs,
      maxBuffer: 50 * 1024 * 1024,
      env: codexHome ? { ...process.env, CODEX_HOME: codexHome } : process.env,
    });
    const completedAt = new Date().toISOString();
    const stdoutPath = path.join(logsPath, "agent-stdout.log");
    const stderrPath = path.join(logsPath, "agent-stderr.log");
    fs.writeFileSync(stdoutPath, stdout || "");
    fs.writeFileSync(stderrPath, "");
    return {
      status: "succeeded",
      runner_status: "completed",
      command: [cmd, ...args],
      codex_home_path: codexHome,
      started_at: startedAt,
      completed_at: completedAt,
      stdout_path: stdoutPath,
      stderr_path: stderrPath,
      reason: "agent runner completed successfully",
    };
  } catch (error) {
    const completedAt = new Date().toISOString();
    const stdoutPath = path.join(logsPath, "agent-stdout.log");
    const stderrPath = path.join(logsPath, "agent-stderr.log");
    fs.writeFileSync(stdoutPath, String(error.stdout || ""));
    fs.writeFileSync(stderrPath, String(error.stderr || error.message || ""));
    return {
      status: "failed",
      runner_status: "failed",
      command: [cmd, ...args],
      codex_home_path: codexHome,
      started_at: startedAt,
      completed_at: completedAt,
      stdout_path: stdoutPath,
      stderr_path: stderrPath,
      exit_status: error.status ?? null,
      reason: String(error.stderr || error.message || "agent runner failed").trim(),
    };
  }
}

// Review remediation may run after the source issue is closed, so this parser
// only trusts explicit PR references and never falls back to arbitrary digits.
function explicitIssueNumberForPullRequest(pr) {
  const text = `${pr.title || ""}\n${pr.body || ""}\n${pr.head_ref || pr.headRefName || ""}`;
  const matches = [
    ...text.matchAll(/(?:^|[^\w-])#(\d+)(?=$|[^\w-])/gi),
    ...text.matchAll(/(?:^|[^\w-])issue-(\d+)(?=$|[-_\s/.]|[^\w-])/gi),
  ].map((match) => Number(match[1]));
  return matches.find((number) => Number.isInteger(number) && number > 0) || null;
}

function issueForPullRequest(pr, issues) {
  const issueNumber = explicitIssueNumberForPullRequest(pr);
  if (!issueNumber) return null;
  return issues.find((issue) => issue.number === issueNumber) || null;
}

// Issue titles can change after a workspace is prepared. Reuse any existing
// issue-number workspace instead of treating slug drift as a missing workspace.
function findIssueWorkspace(issue, dispatchConfig) {
  if (!issue) return "";
  const currentPath = issueWorkspacePath(issue, dispatchConfig);
  if (fs.existsSync(currentPath)) return currentPath;
  if (!fs.existsSync(dispatchConfig.workspaceRoot)) return currentPath;
  const prefix = `issue-${issue.number}-`;
  const candidates = fs
    .readdirSync(dispatchConfig.workspaceRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && entry.name.startsWith(prefix))
    .map((entry) => path.join(dispatchConfig.workspaceRoot, entry.name))
    .sort();
  return candidates[0] || currentPath;
}

function workspaceRepoMatchesBranch(workspacePath, branchName) {
  if (!branchName) return false;
  const repoPath = path.join(workspacePath, "repo");
  if (!fs.existsSync(repoPath)) return false;
  const branch = currentBranch(repoPath);
  return branch.ok && branch.branch === branchName;
}

// Review remediation belongs to the PR branch that already exists. Prefer the
// prepared branch workspace, then recorded state, then issue or PR fallbacks.
function workspaceForReviewRemediation({ pr, issue, dispatchConfig }) {
  const branchName = pr.head_ref || pr.headRefName || "";
  const existingWorktree = branchName ? worktreeForBranch(branchName) : null;
  if (existingWorktree && path.basename(existingWorktree.path) === "repo") return path.dirname(existingWorktree.path);

  if (fs.existsSync(dispatchConfig.workspaceRoot)) {
    const stateMatches = fs
      .readdirSync(dispatchConfig.workspaceRoot, { withFileTypes: true })
      .filter((entry) => entry.isDirectory())
      .map((entry) => path.join(dispatchConfig.workspaceRoot, entry.name))
      .map((workspacePath) => {
        const statePath = path.join(workspacePath, "state.json");
        if (!fs.existsSync(statePath)) return null;
        try {
          const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
          const matchesReviewPr = state.review_remediation_pr === pr.number;
          const matchesBranch = state.branch === branchName;
          const matchesIssue = state.issue_number === issue?.number;
          if (!matchesReviewPr && !matchesBranch && !matchesIssue) return null;
          return { workspacePath, matchesReviewPr, matchesBranch, matchesIssue, repoMatchesBranch: workspaceRepoMatchesBranch(workspacePath, branchName) };
        } catch {
          return null;
        }
      })
      .filter(Boolean)
      .sort((a, b) => {
        const score = (candidate) =>
          (candidate.repoMatchesBranch ? 1000 : 0) +
          (candidate.matchesReviewPr ? 100 : 0) +
          (candidate.matchesBranch ? 10 : 0) +
          (candidate.matchesIssue ? 1 : 0);
        if (score(a) !== score(b)) return score(b) - score(a);
        return a.workspacePath.localeCompare(b.workspacePath);
      });
    if (stateMatches.length > 0) return stateMatches[0].workspacePath;
  }

  if (issue) return findIssueWorkspace(issue, dispatchConfig);
  return path.join(dispatchConfig.workspaceRoot, `pr-${pr.number}`);
}

function renderReviewRemediationPrompt({ pr, issue, dispatchConfig, workflow, skills }) {
  const targetBranch = pr.target_branch || "phase-0-platform-foundation";
  const branchName = pr.head_ref;
  const blockerList =
    (pr.blockers || []).length === 0
      ? "- No merge/draft/check blockers were returned; inspect the PR before changing code."
      : pr.blockers.map((blocker) => `- ${blocker}`).join("\n");
  const threadList =
    pr.unresolved_codex_review_thread_summaries.length === 0
      ? "- No thread details were returned; inspect the PR review threads directly before changing code."
      : pr.unresolved_codex_review_thread_summaries
          .map((thread, index) =>
            [
              `### Thread ${index + 1}`,
              `- URL: ${thread.url || "(missing)"}`,
              `- File: ${thread.path || "(missing)"}`,
              `- Line: ${thread.line || "(unknown)"}`,
              `- Author: ${thread.author || "(unknown)"}`,
              `- Excerpt: ${thread.body_excerpt || "(empty)"}`,
            ].join("\n"),
          )
          .join("\n\n");
  const routedSkills = routeSkills(`${pr.title}\n${issue?.body || ""}\nCodex Review remediation`, skills);
  const skillSection =
    routedSkills.length === 0
      ? "No repo-local skill matched this PR remediation. Apply the base workflow and source-of-truth order."
      : routedSkills
          .map(
            (skill) =>
              `## ${skill.skill_name}\nPath: ${skill.skill_path}\nReason: ${skill.reason_selected}\nStatus: ${
                skill.missing_or_blocked || "included"
              }\n\n${skill.summary || ""}`,
          )
          .join("\n\n");
  return [
    "# Symphony PR Review Remediation Prompt",
    "",
    `Pull request: #${pr.number} ${pr.title}`,
    `PR URL: ${pr.url}`,
    `Linked issue: ${issue ? `#${issue.number} ${issue.title}` : "(not resolved)"}`,
    `Target branch: ${targetBranch}`,
    `Working branch: ${branchName}`,
    "",
    "## PR Blockers To Resolve",
    "",
    blockerList,
    "",
    "## Codex Review Threads To Address",
    "",
    threadList,
    "",
    "## Linked Issue Body",
    "",
    issue?.body || "(No linked issue body was resolved. Use the PR and review thread context.)",
    "",
    "## Repo Workflow Prompt",
    "",
    workflow.promptTemplate.trim(),
    "",
    "## Applicable Repo Skills",
    "",
    skillSection,
    "",
    "## Remediation Contract",
    "",
    "- Work only in the prepared PR worktree and branch.",
    "- If the PR is merge-conflicted, update the PR branch against the target branch, resolve conflicts explicitly, and rerun relevant verification.",
    "- If the PR is draft, verify whether it is complete, run required checks, and mark it ready for review only when it satisfies the issue and PR contract.",
    "- Inspect each active Codex Review thread and decide whether it needs a code, docs, test, or generated-artifact fix.",
    "- Implement only in-scope fixes for this PR and preserve unrelated local or user-authored changes.",
    "- Run the relevant verification for the files touched.",
    "- Commit and push the PR branch with `--force-with-lease` only if a rebase or history rewrite is needed.",
    "- Reply to or resolve Codex Review threads only after the branch update makes the comment fixed or obsolete.",
    "- Do not perform production writes, provider writeback, secret disclosure, destructive git, or manual merges.",
    "- Finish with changed files, verification evidence, safety notes, and thread actions taken.",
    "",
  ].join("\n");
}

function ensureReviewRemediationDispatch({ pr, issues, dispatchConfig, workflow, skills, dryRun }) {
  const issue = issueForPullRequest(pr, issues);
  const targetBranch = pr.target_branch || dispatchConfig.defaultTargetBranch;
  const workspacePath = workspaceForReviewRemediation({ pr, issue, dispatchConfig });
  const repoPath = path.join(workspacePath, "repo");
  const logsPath = path.join(workspacePath, "logs");
  const promptPath = path.join(workspacePath, `review-remediation-${pr.number}.md`);
  const statePath = path.join(workspacePath, "state.json");
  const result = {
    number: pr.number,
    title: pr.title,
    issue_number: issue?.number || null,
    status: dryRun ? "would-remediate" : "prepared",
    branch: pr.head_ref,
    target_branch: targetBranch,
    workspace: workspacePath,
    repo: repoPath,
    prompt_path: promptPath,
    state_path: statePath,
    runner_status: dispatchConfig.agentRunnerCommand ? "runner_configured" : "needs_agent_runner",
    reason: "",
  };

  if (!fs.existsSync(repoPath)) {
    if (!isSafePrBranch(pr.head_ref)) {
      return { ...result, status: "blocked", reason: `unsafe PR branch for automatic remediation: ${pr.head_ref}` };
    }
    if (!dryRun) {
      run("git", ["fetch", "--prune", "origin"], { cwd: repoRoot });
    }
    if (!refExists(`origin/${pr.head_ref}`)) {
      return { ...result, status: "blocked", reason: `remote PR branch origin/${pr.head_ref} does not exist` };
    }
    const existingWorktree = worktreeForBranch(pr.head_ref);
    if (existingWorktree) {
      return { ...result, status: "blocked", reason: `PR branch is already checked out at ${existingWorktree.path}` };
    }
    if (dryRun) {
      return { ...result, reason: `dry-run would create missing PR remediation workspace at ${repoPath}` };
    }
    fs.mkdirSync(workspacePath, { recursive: true });
    if (refExists(pr.head_ref)) {
      run("git", ["branch", "-f", pr.head_ref, `origin/${pr.head_ref}`], { cwd: repoRoot });
      run("git", ["worktree", "add", repoPath, pr.head_ref], { cwd: repoRoot });
    } else {
      run("git", ["worktree", "add", "-b", pr.head_ref, repoPath, `origin/${pr.head_ref}`], { cwd: repoRoot });
    }
  }
  const status = cleanStatus(repoPath);
  if (!status.clean) {
    return { ...result, status: "blocked", reason: status.blocker, dirty_files: status.dirty_files };
  }
  const branch = currentBranch(repoPath);
  if (!branch.ok) {
    return { ...result, status: "blocked", reason: branch.reason };
  }
  if (branch.branch !== pr.head_ref) {
    return { ...result, status: "blocked", reason: `prepared PR workspace is on ${branch.branch || "(detached)"}, expected ${pr.head_ref}` };
  }

  if (dryRun) {
    return { ...result, reason: "dry-run verified remediation prerequisites and performs no prompt, state, runner, or PR mutations" };
  }

  fs.mkdirSync(logsPath, { recursive: true });
  const prompt = renderReviewRemediationPrompt({ pr, issue, dispatchConfig, workflow, skills });
  fs.writeFileSync(promptPath, `${prompt}\n`);
  const runner = dispatchConfig.agentRunnerCommand
    ? runAgentRunner({
        commandLine: dispatchConfig.agentRunnerCommand,
        repoPath,
        promptPath,
        prompt,
        issue,
        branchName: pr.head_ref,
        targetBranch: result.target_branch,
        logsPath,
        timeoutMs: dispatchConfig.agentRunnerTimeoutMs,
        codexHomeRoot: dispatchConfig.agentRunnerCodexHomeRoot,
        workspacePath,
      })
    : { status: result.status, runner_status: result.runner_status, reason: "no repo-owned agent runner command is configured yet" };
  const priorState = fs.existsSync(statePath) ? JSON.parse(fs.readFileSync(statePath, "utf8")) : {};
  const state = {
    ...priorState,
    issue_number: issue?.number || null,
    issue_url: issue?.url || "",
    title: issue?.title || pr.title,
    target_branch: result.target_branch,
    branch: pr.head_ref,
    workspace_path: workspacePath,
    repo_path: repoPath,
    status: runner.status,
    runner_status: runner.runner_status,
    review_remediation_pr: pr.number,
    review_remediation_prompt_path: promptPath,
    updated_at: new Date().toISOString(),
    runner,
    last_event:
      runner.runner_status === "completed"
        ? "review remediation agent completed"
        : runner.runner_status === "failed"
          ? "review remediation agent failed"
          : "review remediation prompt prepared; no repo-owned agent runner command is configured yet",
  };
  fs.writeFileSync(statePath, `${JSON.stringify(state, null, 2)}\n`);
  return { ...result, status: runner.status, runner_status: runner.runner_status, reason: runner.reason };
}

function ensureDispatchWorkspace({ issue, dispatchConfig, workflow, skills, dryRun }) {
  const workspacePath = issueWorkspacePath(issue, dispatchConfig);
  const repoPath = path.join(workspacePath, "repo");
  const logsPath = path.join(workspacePath, "logs");
  const promptPath = path.join(workspacePath, "prompt.md");
  const statePath = path.join(workspacePath, "state.json");
  const branchName = issueBranchName(issue, dispatchConfig);
  const targetBranch = issueTargetBranch(issue, dispatchConfig);
  const result = {
    number: issue.number,
    title: issue.title,
    status: dryRun ? "would-prepare" : "prepared",
    branch: branchName,
    target_branch: targetBranch,
    workspace: workspacePath,
    repo: repoPath,
    prompt_path: promptPath,
    state_path: statePath,
    runner_status: dispatchConfig.agentRunnerCommand ? "runner_configured" : "needs_agent_runner",
    reason: "",
  };

  if (dryRun) {
    result.reason = "dry-run performs no branch, worktree, prompt, or state mutations";
    return result;
  }

  fs.mkdirSync(logsPath, { recursive: true });
  if (fs.existsSync(repoPath)) {
    const status = cleanStatus(repoPath);
    if (!status.clean) {
      return { ...result, status: "blocked", reason: status.blocker, dirty_files: status.dirty_files };
    }
    const branch = currentBranch(repoPath);
    if (!branch.ok) {
      return { ...result, status: "blocked", reason: branch.reason };
    }
    if (branch.branch !== branchName) {
      return { ...result, status: "blocked", reason: `prepared issue workspace is on ${branch.branch || "(detached)"}, expected ${branchName}` };
    }
  } else {
    run("git", ["fetch", "--prune", "origin"], { cwd: repoRoot });
    const existingWorktree = worktreeForBranch(branchName);
    if (existingWorktree) {
      return {
        ...result,
        status: "blocked",
        reason: `branch is already checked out at ${existingWorktree.path}`,
      };
    }
    if (!refExists(`origin/${targetBranch}`)) {
      return { ...result, status: "blocked", reason: `target branch origin/${targetBranch} does not exist` };
    }
    if (refExists(branchName)) {
      run("git", ["worktree", "add", repoPath, branchName], { cwd: repoRoot });
    } else {
      run("git", ["worktree", "add", "-b", branchName, repoPath, `origin/${targetBranch}`], { cwd: repoRoot });
    }
  }

  let priorState = {};
  if (fs.existsSync(statePath)) {
    try {
      priorState = JSON.parse(fs.readFileSync(statePath, "utf8"));
    } catch (error) {
      return { ...result, status: "blocked", reason: `failed to parse existing state.json: ${error.message}` };
    }
  }
  const prompt = renderIssuePrompt({ issue, dispatchConfig, workflow, skills });
  fs.writeFileSync(promptPath, `${prompt}\n`);
  const runner = dispatchConfig.agentRunnerCommand
    ? runAgentRunner({
        commandLine: dispatchConfig.agentRunnerCommand,
        repoPath,
        promptPath,
        prompt,
        issue,
        branchName,
        targetBranch,
        logsPath,
        timeoutMs: dispatchConfig.agentRunnerTimeoutMs,
        codexHomeRoot: dispatchConfig.agentRunnerCodexHomeRoot,
        workspacePath,
      })
    : { status: result.status, runner_status: result.runner_status, reason: "no repo-owned agent runner command is configured yet" };
  const state = {
    ...priorState,
    issue_number: issue.number,
    issue_url: issue.url,
    title: issue.title,
    target_branch: targetBranch,
    branch: branchName,
    workspace_path: workspacePath,
    repo_path: repoPath,
    status: runner.status,
    runner_status: runner.runner_status,
    prompt_path: promptPath,
    attempts: Number(priorState.attempts || 0) + 1,
    created_at: priorState.created_at || new Date().toISOString(),
    updated_at: new Date().toISOString(),
    runner,
    last_event:
      runner.runner_status === "completed"
        ? "workspace prepared and agent runner completed"
        : runner.runner_status === "failed"
          ? "workspace prepared and agent runner failed"
          : "workspace prepared; no repo-owned agent runner command is configured yet",
  };
  fs.writeFileSync(statePath, `${JSON.stringify(state, null, 2)}\n`);
  return { ...result, status: runner.status, runner_status: runner.runner_status, reason: runner.reason };
}

function readMonitorConfig(workflowConfig) {
  const configured = workflowConfig.maintenance?.ui_improvements_monitor || {};
  return {
    ...DEFAULT_MONITOR,
    ...snakeToCamel(configured),
    healthUrls: configured.health_urls || DEFAULT_MONITOR.healthUrls,
    devServers: configured.dev_servers || DEFAULT_MONITOR.devServers,
    reconcileWorktreeRoot: configured.reconcile_worktree_root || DEFAULT_MONITOR.reconcileWorktreeRoot,
    reconcilePrBranches:
      configured.reconcile_pr_branches === undefined
        ? DEFAULT_MONITOR.reconcilePrBranches
        : Boolean(configured.reconcile_pr_branches),
    safeBranchPrefixes: configured.safe_branch_prefixes || DEFAULT_MONITOR.safeBranchPrefixes,
    codexReviewAuthors: configured.codex_review_authors || DEFAULT_MONITOR.codexReviewAuthors,
    autoResolveOutdatedCodexReviewThreads:
      configured.auto_resolve_outdated_codex_review_threads === undefined
        ? DEFAULT_MONITOR.autoResolveOutdatedCodexReviewThreads
        : Boolean(configured.auto_resolve_outdated_codex_review_threads),
    latestCodeAllowedDirty: configured.latest_code_allowed_dirty || DEFAULT_MONITOR.latestCodeAllowedDirty,
    browserScreenshotRequired:
      configured.browser_screenshot_required === undefined
        ? DEFAULT_MONITOR.browserScreenshotRequired
        : Boolean(configured.browser_screenshot_required),
    lockMaxAgeMs: Number(configured.lock_max_age_ms || DEFAULT_MONITOR.lockMaxAgeMs),
  };
}

function snakeToCamel(object) {
  const result = {};
  for (const [key, value] of Object.entries(object || {})) {
    result[key.replace(/_([a-z])/g, (_match, letter) => letter.toUpperCase())] = value;
  }
  return result;
}

function acquireLock(lockPath, { now = new Date(), maxAgeMs = DEFAULT_MONITOR.lockMaxAgeMs } = {}) {
  try {
    writeLock(lockPath, now);
    return { acquired: true, stale_removed: false };
  } catch (error) {
    if (error.code !== "EEXIST") throw error;
  }

  const existing = readLockMetadata(lockPath);
  if (!isStaleLock(existing, now, maxAgeMs)) {
    throw new Error(
      `Active Symphony monitor lock at ${lockPath} owned by pid ${existing.pid || "unknown"} since ${
        existing.started_at || "unknown"
      }`,
    );
  }

  fs.rmSync(lockPath, { recursive: true, force: true });
  writeLock(lockPath, now);
  return { acquired: true, stale_removed: true, previous: existing };
}

function writeLock(lockPath, now) {
  fs.mkdirSync(lockPath);
  fs.writeFileSync(path.join(lockPath, "pid"), `${process.pid}\n`);
  fs.writeFileSync(path.join(lockPath, "started_at"), `${now.toISOString()}\n`);
}

function readLockMetadata(lockPath) {
  try {
    return {
      pid: fs.readFileSync(path.join(lockPath, "pid"), "utf8").trim(),
      started_at: fs.readFileSync(path.join(lockPath, "started_at"), "utf8").trim(),
      readable: true,
    };
  } catch (error) {
    return { pid: "", started_at: "", readable: false, error: error.message };
  }
}

function isStaleLock(metadata, now, maxAgeMs) {
  if (!metadata.readable || !metadata.pid || !metadata.started_at) return true;
  const started = Date.parse(metadata.started_at);
  if (!Number.isFinite(started)) return true;
  if (now.getTime() - started > maxAgeMs) return true;
  if (!/^\d+$/.test(metadata.pid)) return true;
  try {
    process.kill(Number(metadata.pid), 0);
    return false;
  } catch (error) {
    return true;
  }
}

function releaseLock(lockPath) {
  const pidPath = path.join(lockPath, "pid");
  if (fs.existsSync(pidPath) && fs.readFileSync(pidPath, "utf8").trim() === String(process.pid)) {
    fs.rmSync(lockPath, { recursive: true, force: true });
  }
}

function inspectLatestCode(config) {
  const result = {
    worktree: config.latestCodeWorktree,
    dirty: false,
    dirty_files: [],
    head: "",
    target: "",
    blocker: "",
  };
  if (!fs.existsSync(config.latestCodeWorktree)) {
    result.blocker = "latest-code worktree is missing";
    return result;
  }
  const status = run("git", ["status", "--porcelain"], { cwd: config.latestCodeWorktree, allowFailure: true });
  if (status.failed) {
    result.blocker = status.stderr || "failed to inspect latest-code status";
    return result;
  }
  result.dirty_files = status
    ? status
        .split(/\r?\n/)
        .filter(Boolean)
        .map((line) => line.slice(3))
    : [];
  result.dirty = result.dirty_files.some(
    (file) => !config.latestCodeAllowedDirty.some((prefix) => file.startsWith(prefix)),
  );
  if (result.dirty) {
    result.blocker = "latest-code worktree has real local edits";
  }
  result.head = run("git", ["rev-parse", "--short", "HEAD"], { cwd: config.latestCodeWorktree, allowFailure: true });
  result.target = run("git", ["rev-parse", "--short", `origin/${config.targetBranch}`], {
    cwd: repoRoot,
    allowFailure: true,
  });
  return result;
}

async function checkHealth(urls) {
  const checks = [];
  for (const url of urls) {
    try {
      const response = await fetch(url, { redirect: "manual" });
      checks.push({ url, status: response.status, ok: response.status >= 200 && response.status < 400 });
    } catch (error) {
      checks.push({ url, status: 0, ok: false, error: error.message });
    }
  }
  return checks;
}

function browserEvaluationsFor(config, latestCode, healthChecks) {
  if (latestCode.blocker || healthChecks.some((check) => !check.ok)) return [];
  const browserUrlOrigin = new URL(config.browserDefaultUrl).origin;
  return [
    {
      url: config.browserDefaultUrl,
      purpose: "Refresh and self-evaluate the latest ui-improvements dashboard after monitor validation.",
      persona: "it_admin",
      expected_visible_behavior:
        "The IT Admin dashboard loads inside the shared shell without visible error overlays, major text overlap, or broken navigation chrome.",
      screenshot_required: Boolean(config.browserScreenshotRequired),
      interaction_steps: ["Open or preserve the current local app URL", "Reload after server refresh", "Capture DOM notes and screenshot"],
      persona_setup: `npm run dev:persona -- it_admin --base-url ${browserUrlOrigin}`,
      acceptance_checks: [
        "HTTP route loads successfully",
        "Shared shell/sidebar/header are visible",
        "No obvious overlapping text or controls in the first viewport",
        "No access-denied page for the IT Admin persona",
      ],
    },
  ];
}

function safePathSegment(value) {
  return String(value).replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "");
}

function isSafePrBranch(branchName, prefixes = DEFAULT_MONITOR.safeBranchPrefixes) {
  return prefixes.some((prefix) => branchName.startsWith(prefix));
}

function parseWorktreeList(output) {
  const entries = [];
  let current = {};
  for (const line of output.split(/\r?\n/)) {
    if (!line.trim()) {
      if (current.path) entries.push(current);
      current = {};
      continue;
    }
    const [key, ...rest] = line.split(" ");
    const value = rest.join(" ");
    if (key === "worktree") current.path = value;
    if (key === "HEAD") current.head = value;
    if (key === "branch") current.branch = value.replace(/^refs\/heads\//, "");
    if (key === "detached") current.detached = true;
  }
  if (current.path) entries.push(current);
  return entries;
}

function worktreeForBranch(branchName) {
  const output = run("git", ["worktree", "list", "--porcelain"], { cwd: repoRoot, allowFailure: true });
  if (output.failed) return null;
  return parseWorktreeList(output).find((entry) => entry.branch === branchName) || null;
}

function refExists(refName) {
  const result = run("git", ["rev-parse", "--verify", "--quiet", refName], { cwd: repoRoot, allowFailure: true });
  return !result.failed;
}

function cleanStatus(cwd) {
  const status = run("git", ["status", "--porcelain"], { cwd, allowFailure: true });
  if (status.failed) {
    return { clean: false, blocker: status.stderr || "failed to inspect worktree status" };
  }
  return {
    clean: !status,
    blocker: status ? "worktree has local edits" : "",
    dirty_files: status
      ? status
          .split(/\r?\n/)
          .filter(Boolean)
          .map((line) => line.slice(3))
      : [],
  };
}

function currentBranch(cwd) {
  const branch = run("git", ["branch", "--show-current"], { cwd, allowFailure: true });
  if (branch.failed) {
    return { ok: false, branch: "", reason: branch.stderr || "failed to inspect current branch" };
  }
  return { ok: true, branch, reason: "" };
}

function branchEqualsRemote(cwd, branchName) {
  const local = run("git", ["rev-parse", "HEAD"], { cwd, allowFailure: true });
  const remote = run("git", ["rev-parse", `origin/${branchName}`], { cwd, allowFailure: true });
  if (local.failed || remote.failed) return false;
  return local === remote;
}

function ensurePrWorktree(config, branchName, prNumber) {
  const existing = worktreeForBranch(branchName);
  if (existing) return { cwd: existing.path, reused: true };

  fs.mkdirSync(config.reconcileWorktreeRoot, { recursive: true });
  const worktreePath = path.join(config.reconcileWorktreeRoot, `pr-${prNumber}-${safePathSegment(branchName)}`);
  if (fs.existsSync(worktreePath)) {
    const status = cleanStatus(worktreePath);
    if (!status.clean) {
      return {
        cwd: worktreePath,
        reused: true,
        blocker: status.blocker,
        dirty_files: status.dirty_files,
      };
    }
    const branch = run("git", ["branch", "--show-current"], { cwd: worktreePath, allowFailure: true });
    if (!branch.failed && branch === branchName) return { cwd: worktreePath, reused: true };
  }

  if (refExists(branchName)) {
    run("git", ["worktree", "add", worktreePath, branchName], { cwd: repoRoot });
    return { cwd: worktreePath, reused: false };
  }

  run("git", ["worktree", "add", "-b", branchName, worktreePath, `origin/${branchName}`], { cwd: repoRoot });
  return { cwd: worktreePath, reused: false };
}

function reconcilePullRequestBranches({ prs, reviewThreads, config, dryRun }) {
  const results = [];
  if (!config.safeRebase || !config.reconcilePrBranches) {
    return results;
  }

  const unresolvedByPr = new Map(reviewThreads.map((entry) => [entry.number, entry.unresolved_threads.length]));
  for (const pr of queuePullRequests(prs)) {
    const result = {
      number: pr.number,
      head: pr.headRefName,
      action: "skipped",
      status: "skipped",
      reason: "",
      unresolved_review_threads: unresolvedByPr.get(pr.number) || 0,
      worktree: "",
      before: "",
      after: "",
    };
    results.push(result);

    if (pr.isDraft) {
      result.reason = "draft PR";
      continue;
    }
    if (!isSafePrBranch(pr.headRefName, config.safeBranchPrefixes)) {
      result.reason = "branch prefix is not in safe_branch_prefixes";
      continue;
    }
    if (!refExists(`origin/${pr.headRefName}`)) {
      result.reason = "remote PR branch is missing";
      continue;
    }
    if (dryRun) {
      result.action = "would-rebase";
      result.status = "dry-run";
      result.reason = "dry-run performs no branch mutations";
      continue;
    }

    try {
      const worktree = ensurePrWorktree(config, pr.headRefName, pr.number);
      result.worktree = worktree.cwd;
      if (worktree.blocker) {
        result.status = "blocked";
        result.reason = worktree.blocker;
        result.dirty_files = worktree.dirty_files || [];
        continue;
      }

      run("git", ["fetch", "--prune", "origin"], { cwd: worktree.cwd });
      const status = cleanStatus(worktree.cwd);
      if (!status.clean) {
        result.status = "blocked";
        result.reason = status.blocker;
        result.dirty_files = status.dirty_files;
        continue;
      }

      if (!branchEqualsRemote(worktree.cwd, pr.headRefName)) {
        result.status = "blocked";
        result.reason = "local branch diverges from origin; refusing to overwrite unknown local work";
        continue;
      }

      result.before = run("git", ["rev-parse", "--short", "HEAD"], { cwd: worktree.cwd });
      const rebase = run("git", ["rebase", `origin/${config.targetBranch}`], { cwd: worktree.cwd, allowFailure: true });
      if (rebase.failed) {
        run("git", ["rebase", "--abort"], { cwd: worktree.cwd, allowFailure: true });
        result.status = "blocked";
        result.reason = rebase.stderr || rebase.stdout || "rebase failed";
        continue;
      }

      result.after = run("git", ["rev-parse", "--short", "HEAD"], { cwd: worktree.cwd });
      if (result.before === result.after) {
        result.action = "checked";
        result.status = "up-to-date";
        result.reason = "already based on latest target branch";
        continue;
      }

      run("git", ["push", "--force-with-lease", "origin", `HEAD:${pr.headRefName}`], { cwd: worktree.cwd });
      result.action = "rebased-and-pushed";
      result.status = "updated";
      result.reason = `rebased onto origin/${config.targetBranch} and pushed with --force-with-lease`;
    } catch (error) {
      result.status = "blocked";
      result.reason = error.stderr || error.message || String(error);
    }
  }
  return results;
}

function workspaceRoot(config) {
  if (config.workspace?.root_env && process.env[config.workspace.root_env]) {
    return process.env[config.workspace.root_env];
  }
  if (config.workspace?.root_default) {
    return path.resolve(repoRoot, config.workspace.root_default);
  }
  return config.workspace?.root || path.join(os.tmpdir(), "accountsdot-symphony");
}

function writeState(workflowConfig, status) {
  const root = workspaceRoot(workflowConfig);
  fs.mkdirSync(root, { recursive: true });
  const statePath = path.join(root, "ui-improvements-monitor-status.json");
  const eventPath = path.join(root, "runs.jsonl");
  fs.writeFileSync(statePath, `${JSON.stringify(status, null, 2)}\n`);
  fs.appendFileSync(eventPath, `${JSON.stringify({ timestamp: new Date().toISOString(), ...status })}\n`);
  return { statePath, eventPath };
}

function writeDispatchState(dispatchConfig, status) {
  fs.mkdirSync(dispatchConfig.workspaceRoot, { recursive: true });
  const statePath = path.join(dispatchConfig.workspaceRoot, "dispatcher-status.json");
  const eventPath = path.join(dispatchConfig.workspaceRoot, "runs.jsonl");
  fs.writeFileSync(statePath, `${JSON.stringify(status, null, 2)}\n`);
  fs.appendFileSync(eventPath, `${JSON.stringify({ timestamp: new Date().toISOString(), ...status })}\n`);
  return { statePath, eventPath };
}

async function report({ json = false } = {}) {
  const workflow = readWorkflow();
  const monitor = readMonitorConfig(workflow.config);
  const prs = listOpenPullRequests(monitor.targetBranch);
  const issues = listOpenIssues();
  const skills = discoverSkills();
  const queued = queuePullRequests(prs).map((pr, index) => ({
    priority: index + 1,
    number: pr.number,
    title: pr.title,
    head: pr.headRefName,
    draft: pr.isDraft,
    merge_state: pr.mergeStateStatus,
    reason: queueReason(pr),
    skills: routeSkills(`${pr.title} ${pr.headRefName}`, skills).map((skill) => ({
      skill_name: skill.skill_name,
      skill_path: skill.skill_path,
      reason_selected: skill.reason_selected,
      instructions_included: skill.instructions_included,
      missing_or_blocked: skill.missing_or_blocked,
    })),
  }));
  const status = {
    command: "report",
    generated_at: new Date().toISOString(),
    target_branch: monitor.targetBranch,
    pr_queue: queued,
    uncovered_open_issues: openIssuesWithoutOpenPr(issues, prs).map((issue) => ({
      number: issue.number,
      title: issue.title,
      labels: issue.labels.map((label) => label.name),
    })),
    browser_evaluations: [],
    browser_results: [],
    mutations_performed: [],
  };
  printStatus(status, json);
  return status;
}

async function sync({ dryRun = false, json = false, maxRuns = null } = {}) {
  const workflow = readWorkflow();
  const dispatchConfig = readDispatchConfig(workflow.config);
  const prConfig = readPullRequestConfig(workflow.config, dispatchConfig);
  const skills = discoverSkills();
  const issues = listOpenIssues();
  const targetBranches = [
    dispatchConfig.defaultTargetBranch,
    prConfig.targetBranch,
    ...issues.map((issue) => issueTargetBranch(issue, dispatchConfig)),
  ];
  const prs = listOpenPullRequestsForBases(targetBranches);
  const mergedPrs = listMergedPullRequestsForBases(targetBranches);
  const mergedWorkspaceStates = markMergedIssueWorkspaceStates({ issues, mergedPrs, dispatchConfig, dryRun });
  const prReviewThreads = prConfig.inspectBeforeDispatch ? fetchReviewThreads(prConfig.targetBranch) : [];
  const prSignals = prConfig.inspectBeforeDispatch ? fetchPullRequestReviewSignals(prConfig.targetBranch) : new Map();
  const targetPrs = prs.filter((pr) => pr.baseRefName === prConfig.targetBranch);
  const prMergeQueue = queuePullRequests(targetPrs).map((pr) =>
    evaluatePullRequestForMerge({
      pr,
      reviewThreads: prReviewThreads,
      signals: prSignals.get(pr.number) || { reviews: [], comments: [] },
      config: prConfig,
    }),
  );
  const prMergeResults = [];
  if (prConfig.inspectBeforeDispatch) {
    for (const evaluation of prMergeQueue) {
      if (evaluation.status !== "ready_to_merge") continue;
      prMergeResults.push(mergePullRequest({ evaluation, config: prConfig, dryRun }));
    }
  }
  const maxSelected = maxRuns || dispatchConfig.maxConcurrentRuns;
  const reviewRemediationCandidates = prConfig.remediateBlockedPrs
    ? prMergeQueue.filter((entry) => shouldRemediateBlockedPullRequest(entry))
    : [];
  const reviewRemediations = reviewRemediationCandidates
    .slice(0, Math.min(maxSelected, prConfig.maxReviewRemediationsPerTick))
    .map((entry) => ensureReviewRemediationDispatch({ pr: entry, issues, dispatchConfig, workflow, skills, dryRun }));
  const remediationSlotsUsed = reviewRemediations.filter((entry) => remediationConsumesDispatchSlot(entry)).length;
  const remainingDispatchSlots = Math.max(0, maxSelected - remediationSlotsUsed);
  const readyToMergeRemaining = prMergeQueue.some(
    (entry) => entry.status === "ready_to_merge" && !prMergeResults.some((result) => result.number === entry.number && result.status === "merged"),
  );
  const shouldPauseDispatchForPrQueue =
    prConfig.inspectBeforeDispatch &&
    readyToMergeRemaining;
  const queue = rankedDispatchQueue(issues, prs, mergedPrs, dispatchConfig);
  const selected = shouldPauseDispatchForPrQueue ? [] : queue.filter((entry) => entry.eligible).slice(0, remainingDispatchSlots);
  const dispatches = [];

  for (const entry of selected) {
    const issue = issues.find((candidate) => candidate.number === entry.number);
    dispatches.push(ensureDispatchWorkspace({ issue, dispatchConfig, workflow, skills, dryRun }));
  }

  const status = {
    command: "sync",
    dry_run: dryRun,
    generated_at: new Date().toISOString(),
    dispatcher: {
      workspace_root: dispatchConfig.workspaceRoot,
      default_target_branch: dispatchConfig.defaultTargetBranch,
      max_concurrent_runs: dispatchConfig.maxConcurrentRuns,
      max_attempts: dispatchConfig.maxAttempts,
      max_selected_this_tick: maxSelected,
      active_labels: dispatchConfig.activeLabels,
      blocked_labels: dispatchConfig.blockedLabels,
      agent_runner_configured: Boolean(dispatchConfig.agentRunnerCommand),
      agent_runner_command: dispatchConfig.agentRunnerCommand,
    },
    pull_request_queue: {
      target_branch: prConfig.targetBranch,
      inspect_before_dispatch: prConfig.inspectBeforeDispatch,
      auto_merge_clean_prs: prConfig.autoMergeCleanPrs,
      merge_method: prConfig.mergeMethod,
      review_wait_policy: prConfig.reviewWaitPolicy,
      items: prMergeQueue,
      merge_results: prMergeResults,
      review_remediations: reviewRemediations,
      merged_workspace_states: mergedWorkspaceStates,
      dispatch_paused_for_pr_queue: shouldPauseDispatchForPrQueue,
      remaining_issue_dispatch_slots: shouldPauseDispatchForPrQueue ? 0 : remainingDispatchSlots,
    },
    issue_queue: queue.map(({ priority_score, ...entry }) => entry),
    selected_issues: selected.map(({ priority_score, ...entry }) => entry),
    dispatches,
    status: prMergeResults.some((result) => result.status === "merged")
      ? "merged_prs"
      : reviewRemediations.some((result) => result.status === "succeeded")
        ? "review_remediation_complete"
      : reviewRemediations.some((result) => result.status === "failed")
        ? "review_remediation_failed"
      : reviewRemediations.some((result) => result.status === "blocked")
        ? "review_remediation_blocked"
      : prMergeQueue.some((entry) => entry.status === "blocked")
        ? "pr_queue_blocked"
      : prMergeQueue.some((entry) => entry.status === "waiting_for_codex_review")
        ? "waiting_for_codex_review"
      : dispatches.some((dispatch) => dispatch.status === "blocked")
      ? "blocked"
      : dispatches.some((dispatch) => dispatch.status === "failed")
        ? "agent_runner_failed"
      : dispatches.some((dispatch) => dispatch.status === "succeeded")
        ? "agent_runner_complete"
      : dispatches.some((dispatch) => dispatch.status === "prepared")
        ? dispatchConfig.agentRunnerCommand
          ? "prepared"
          : "prepared_needs_agent_runner"
        : selected.length === 0
          ? "idle"
          : "dry-run",
    mutations_performed: dryRun
      ? []
      : [
          ...prMergeResults
            .filter((result) => result.status === "merged")
            .map((result) => `merged PR #${result.number} with ${result.merge_method}`),
          ...reviewRemediations
            .filter((result) => ["prepared", "succeeded", "failed"].includes(result.status))
            .map((result) => `review remediation ${result.status} for PR #${result.number} in ${result.repo}`),
          ...mergedWorkspaceStates
            .filter((result) => result.status === "merged")
            .map((result) => `marked issue #${result.issue_number} workspace merged by PR #${result.pr_number}`),
          ...dispatches
            .filter((dispatch) => ["prepared", "succeeded", "failed"].includes(dispatch.status))
            .map((dispatch) => `${dispatch.status} ${dispatch.branch} in ${dispatch.repo}`),
        ],
  };

  if (!dryRun) {
    status.state_files = writeDispatchState(dispatchConfig, status);
  }
  printStatus(status, json);
  return status;
}

async function uiMonitor({ dryRun = false, json = false } = {}) {
  const workflow = readWorkflow();
  const monitor = readMonitorConfig(workflow.config);
  const mutations = [];
  if (!dryRun) {
    acquireLock(monitor.lockPath);
  }
  try {
    if (!dryRun) {
      run("git", ["fetch", "--prune", "origin"], { cwd: repoRoot });
      mutations.push("git fetch --prune origin");
    }
    const prs = listOpenPullRequests(monitor.targetBranch);
    const issues = listOpenIssues();
    let reviewThreads = fetchReviewThreads(monitor.targetBranch);
    const reviewRemediations = remediateReviewThreads({
      reviewThreads,
      config: monitor,
      dryRun,
    });
    for (const remediation of reviewRemediations) {
      if (remediation.status === "resolved") {
        mutations.push(`resolved outdated Codex Review thread on PR #${remediation.number}`);
      }
    }
    if (reviewRemediations.some((remediation) => remediation.status === "resolved")) {
      reviewThreads = fetchReviewThreads(monitor.targetBranch);
    }
    const prReconciliations = reconcilePullRequestBranches({
      prs,
      reviewThreads,
      config: monitor,
      dryRun,
    });
    for (const reconciliation of prReconciliations) {
      if (reconciliation.status === "updated") {
        mutations.push(`rebased and pushed PR #${reconciliation.number} (${reconciliation.head})`);
      }
    }
    const latestCode = inspectLatestCode(monitor);
    const healthChecks = latestCode.blocker ? [] : await checkHealth(monitor.healthUrls);
    const status = {
      command: "ui-monitor",
      dry_run: dryRun,
      generated_at: new Date().toISOString(),
      target_branch: monitor.targetBranch,
      pr_queue: queuePullRequests(prs).map((pr, index) => ({
        priority: index + 1,
        number: pr.number,
        title: pr.title,
        head: pr.headRefName,
        draft: pr.isDraft,
        merge_state: pr.mergeStateStatus,
        reason: queueReason(pr),
      })),
      unresolved_review_threads: reviewThreads
        .filter((entry) => entry.unresolved_threads.length > 0)
        .map((entry) => ({ number: entry.number, count: entry.unresolved_threads.length })),
      outdated_unresolved_review_threads: reviewThreads
        .filter((entry) => entry.outdated_unresolved_threads.length > 0)
        .map((entry) => ({ number: entry.number, count: entry.outdated_unresolved_threads.length })),
      review_remediations: reviewRemediations,
      pr_reconciliations: prReconciliations,
      uncovered_open_issues: openIssuesWithoutOpenPr(issues, prs).map((issue) => ({
        number: issue.number,
        title: issue.title,
      })),
      latest_code: latestCode,
      health_checks: healthChecks,
      browser_evaluations: browserEvaluationsFor(monitor, latestCode, healthChecks),
      browser_results: [],
      status: latestCode.blocker
        ? "blocked"
        : reviewRemediations.some((result) => result.status === "blocked")
          ? "review_remediation_blocked"
        : prReconciliations.some((result) => result.status === "blocked")
          ? "pr_reconciliation_blocked"
        : healthChecks.some((check) => !check.ok)
          ? "dev_server_unhealthy"
          : "needs_browser_evaluation",
      mutations_performed: mutations,
    };
    if (!dryRun) {
      status.state_files = writeState(workflow.config, status);
    }
    printStatus(status, json);
    return status;
  } finally {
    if (!dryRun) {
      releaseLock(monitor.lockPath);
    }
  }
}

function recordBrowserResults({ browserResultsPath, json = false } = {}) {
  if (!browserResultsPath) {
    throw new Error("record-browser-results requires --browser-results PATH");
  }
  const workflow = readWorkflow();
  const root = workspaceRoot(workflow.config);
  const statePath = path.join(root, "ui-improvements-monitor-status.json");
  const status = fs.existsSync(statePath) ? JSON.parse(fs.readFileSync(statePath, "utf8")) : {};
  const browserResults = JSON.parse(fs.readFileSync(browserResultsPath, "utf8"));
  status.browser_results = validateBrowserResults(Array.isArray(browserResults) ? browserResults : [browserResults]);
  status.status = status.browser_results.some((result) => result.status === "failed")
    ? "browser_evaluation_failed"
    : status.browser_results.some((result) => result.status === "blocked")
      ? "browser_evaluation_blocked"
      : "browser_evaluation_complete";
  status.updated_at = new Date().toISOString();
  const files = writeState(workflow.config, status);
  status.state_files = files;
  printStatus(status, json);
  return status;
}

function validateBrowserResults(results) {
  const allowedStatuses = new Set(["passed", "failed", "blocked"]);
  return results.map((result, index) => {
    for (const key of ["url", "status", "evidence", "findings", "checked_at"]) {
      if (!(key in result)) {
        throw new Error(`Browser result ${index} is missing ${key}`);
      }
    }
    if (!allowedStatuses.has(result.status)) {
      throw new Error(`Browser result ${index} has unsupported status ${result.status}`);
    }
    return result;
  });
}

function printStatus(status, jsonOnly) {
  if (jsonOnly) {
    console.log(JSON.stringify(status, null, 2));
    return;
  }
  console.log(JSON.stringify(status, null, 2));
}

async function selfTest() {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), "symphony-runner-test-"));
  try {
    const workflow = readWorkflow();
    assert.equal(workflow.config.name, "wizard-symphony-workflow");
    assert.ok(workflow.promptTemplate.includes("WIZARD Symphony Agent Workflow"));

    const fakeSkillsRoot = path.join(tempRoot, "skills");
    fs.mkdirSync(path.join(fakeSkillsRoot, "wizard-ui-hardening"), { recursive: true });
    fs.mkdirSync(path.join(fakeSkillsRoot, "wizard-code-documentation"), { recursive: true });
    fs.writeFileSync(path.join(fakeSkillsRoot, "wizard-ui-hardening", "SKILL.md"), "# UI\nUse for .pen work.\n");
    fs.writeFileSync(
      path.join(fakeSkillsRoot, "wizard-code-documentation", "SKILL.md"),
      "# Docs\nUse for route and API docs.\n",
    );
    const fakeSkills = discoverSkills(fakeSkillsRoot);
    assert.deepEqual(
      routeSkills("Fix shared shell .pen route docs", fakeSkills).map((skill) => skill.skill_name),
      ["wizard-ui-hardening", "wizard-code-documentation"],
    );
    assert.equal(routeSkills("Fix shared shell .pen route docs", fakeSkills)[0].instructions_included, true);

    fs.mkdirSync(path.join(fakeSkillsRoot, "broken-skill"), { recursive: true });
    assert.equal(discoverSkills(fakeSkillsRoot).some((skill) => skill.missing_or_blocked), true);

    const queued = queuePullRequests([
      { number: 2, isDraft: true, mergeStateStatus: "CLEAN" },
      { number: 1, isDraft: false, mergeStateStatus: "CLEAN" },
      { number: 3, isDraft: false, mergeStateStatus: "DIRTY" },
    ]);
    assert.deepEqual(
      queued.map((pr) => pr.number),
      [1, 3, 2],
    );
    assert.equal(isSafePrBranch("codex/issue-242-browser-bridge"), true);
    assert.equal(isSafePrBranch("issue-227-room-moves-edit-ownership"), true);
    assert.equal(isSafePrBranch("feature/unknown-user-branch"), false);
    const codexThread = {
      id: "thread-1",
      isResolved: false,
      isOutdated: true,
      comments: {
        nodes: [
          {
            author: { login: "chatgpt-codex-connector" },
            body: "Derive persona setup base URL from monitor config",
            path: "scripts/symphony_runner.mjs",
            originalLine: 486,
            url: "https://example.invalid/thread",
          },
        ],
      },
    };
    assert.equal(isCodexReviewThread(codexThread, DEFAULT_MONITOR), true);
    assert.equal(
      remediateReviewThreads({
        reviewThreads: [{ number: 244, review_threads: [codexThread] }],
        config: DEFAULT_MONITOR,
        dryRun: true,
      })[0].action,
      "would-resolve-outdated-thread",
    );
    assert.equal(
      remediateReviewThreads({
        reviewThreads: [{ number: 244, review_threads: [{ ...codexThread, isOutdated: false }] }],
        config: DEFAULT_MONITOR,
        dryRun: true,
      })[0].status,
      "blocked",
    );
    assert.deepEqual(
      parseWorktreeList("worktree /tmp/repo\nHEAD abc123\nbranch refs/heads/codex/example\n\n")[0],
      { path: "/tmp/repo", head: "abc123", branch: "codex/example" },
    );
    const dispatchConfig = readDispatchConfig({
      tracker: { active_labels: ["agent-ready"], blocked_labels: ["blocked", "human-only"] },
      branching: {
        integration_branch: "dev",
        branch_template: "codex/issue-{number}-{slug}",
      },
      dispatch: { max_concurrent_runs: 2, workspace_root: tempRoot },
    });
    assert.equal(dispatchConfig.agentRunnerCodexHomeRoot, path.join(tempRoot, ".codex-agent-homes"));
    const sourceCodexHomeName = "source-codex-home";
    const sourceCodexHome = path.join(tempRoot, sourceCodexHomeName);
    fs.mkdirSync(sourceCodexHome, { recursive: true });
    fs.writeFileSync(path.join(sourceCodexHome, "auth.json"), "{}\n");
    fs.writeFileSync(path.join(sourceCodexHome, "config.toml"), "# test\n");
    fs.mkdirSync(path.join(sourceCodexHome, "cache"), { recursive: true });
    fs.mkdirSync(path.join(sourceCodexHome, "vendor_imports"), { recursive: true });
    const oldCodexHome = process.env.CODEX_HOME;
    const oldCwd = process.cwd();
    process.chdir(tempRoot);
    process.env.CODEX_HOME = sourceCodexHomeName;
    const preparedCodexHome = prepareAgentCodexHome({
      codexHomeRoot: dispatchConfig.agentRunnerCodexHomeRoot,
      workspacePath: path.join(tempRoot, "issue-900264-reference-input-snapshot-integrity"),
    });
    process.chdir(oldCwd);
    if (oldCodexHome === undefined) {
      delete process.env.CODEX_HOME;
    } else {
      process.env.CODEX_HOME = oldCodexHome;
    }
    assert.equal(
      preparedCodexHome,
      path.join(dispatchConfig.agentRunnerCodexHomeRoot, "issue-900264-reference-input-snapshot-integrity"),
    );
    assert.equal(fs.readlinkSync(path.join(preparedCodexHome, "auth.json")), path.join(fs.realpathSync(sourceCodexHome), "auth.json"));
    assert.equal(fs.existsSync(path.join(preparedCodexHome, "state_5.sqlite")), false);
    assert.equal(fs.existsSync(path.join(preparedCodexHome, "cache")), false);
    assert.equal(fs.existsSync(path.join(preparedCodexHome, "vendor_imports")), false);
    const runnerLogsPath = path.join(tempRoot, "runner-logs");
    fs.mkdirSync(runnerLogsPath, { recursive: true });
    const invalidCodexHomeRoot = path.join(tempRoot, "not-a-directory");
    fs.writeFileSync(invalidCodexHomeRoot, "");
    const prepFailure = runAgentRunner({
      commandLine: `${process.execPath} --version`,
      repoPath: tempRoot,
      promptPath: path.join(tempRoot, "prompt.md"),
      prompt: "",
      issue: { number: 1, url: "https://example.invalid/1" },
      branchName: "codex/issue-1-test",
      targetBranch: "phase-0-platform-foundation",
      logsPath: runnerLogsPath,
      timeoutMs: 1000,
      codexHomeRoot: invalidCodexHomeRoot,
      workspacePath: path.join(tempRoot, "issue-1-test"),
    });
    assert.equal(prepFailure.status, "failed");
    assert.equal(prepFailure.runner_status, "failed");
    assert.match(prepFailure.reason, /ENOTDIR|not a directory/i);
    const prConfig = readPullRequestConfig(
      {
        tracker: { blocked_labels: ["blocked"] },
        pull_requests: {
          auto_merge_clean_prs: true,
          codex_review_bot: "chatgpt-codex-connector[bot]",
          codex_review_success_reactions: ["THUMBS_UP"],
        },
      },
      dispatchConfig,
    );
    const readyPr = {
      number: 286,
      title: "Fixes #266",
      headRefName: "codex/issue-266-worker-crash-lease-recovery",
      isDraft: false,
      mergeStateStatus: "CLEAN",
      statusCheckRollup: [],
      labels: [],
    };
    const readyEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [],
        comments: [
          {
            reactionGroups: [
              {
                content: "THUMBS_UP",
                users: { nodes: [{ login: "chatgpt-codex-connector" }] },
              },
            ],
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(readyEvaluation.status, "ready_to_merge");
    assert.equal(readyEvaluation.bot_thumbs_up, true);
    const topLevelReactionEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [],
        comments: [],
        reactionGroups: [
          {
            content: "THUMBS_UP",
            users: { nodes: [{ login: "chatgpt-codex-connector[bot]" }] },
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(topLevelReactionEvaluation.status, "ready_to_merge");
    assert.equal(topLevelReactionEvaluation.bot_thumbs_up, true);
    assert.equal(shouldRemediateBlockedPullRequest({ status: "blocked", head_ref: "codex/example", blockers: ["merge state DIRTY"], unresolved_codex_review_threads: 0, unresolved_codex_review_thread_summaries: [] }), true);
    assert.equal(shouldRemediateBlockedPullRequest({ status: "blocked", head_ref: "codex/example", blockers: ["draft PR"], unresolved_codex_review_threads: 0, unresolved_codex_review_thread_summaries: [] }), true);
    assert.equal(shouldRemediateBlockedPullRequest({ status: "blocked", head_ref: "codex/example", blockers: ["check state FAILURE"], unresolved_codex_review_threads: 0, unresolved_codex_review_thread_summaries: [] }), false);
    assert.equal(
      evaluatePullRequestForMerge({
        pr: { ...readyPr, labels: [{ name: "blocked" }] },
        reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
        signals: { reviews: [], comments: [] },
        config: prConfig,
      }).status,
      "blocked",
    );
    const phaseIssue = {
      number: 900264,
      title: "P0-0A-001: Reference Input Snapshot Integrity",
      body: "Target branch: phase-0-platform-foundation.\n\n## Acceptance Criteria\n\n- [ ] Snapshot references are immutable.",
      labels: [{ name: "agent-ready" }, { name: "phase-0" }],
      url: "https://example.invalid/issues/900264",
    };
    const blockedIssue = {
      ...phaseIssue,
      number: 265,
      labels: [{ name: "agent-ready" }, { name: "blocked" }],
    };
    assert.equal(issueTargetBranch(phaseIssue, dispatchConfig), "phase-0-platform-foundation");
    assert.equal(
      issueTargetBranch(
        { ...phaseIssue, body: "Target branch: phase-0-platform-foundation. PRs for this issue must target phase-0-platform-foundation, not dev." },
        dispatchConfig,
      ),
      "phase-0-platform-foundation",
    );
    assert.equal(
      issueTargetBranch({ ...phaseIssue, body: "Target branch: https://example.invalid/docs/testing/test-matrix.md" }, dispatchConfig),
      dispatchConfig.defaultTargetBranch,
    );
    assert.equal(issueTargetBranch({ ...phaseIssue, body: "Target branch: phase-0-platform-foundation," }, dispatchConfig), "phase-0-platform-foundation");
    assert.equal(issueBranchName(phaseIssue, dispatchConfig), "codex/issue-900264-reference-input-snapshot-integrity");
    assert.equal(classifyIssueForDispatch(phaseIssue, [], [], dispatchConfig).eligible, true);
    assert.equal(classifyIssueForDispatch(blockedIssue, [], [], dispatchConfig).eligible, false);
    const retryDispatchConfig = { ...dispatchConfig, workspaceRoot: path.join(tempRoot, "retry"), maxAttempts: 4 };
    const retryWorkspace = issueWorkspacePath(phaseIssue, retryDispatchConfig);
    fs.mkdirSync(retryWorkspace, { recursive: true });
    const retryStatePath = path.join(retryWorkspace, "state.json");
    fs.writeFileSync(retryStatePath, JSON.stringify({ status: "failed", attempts: 1 }));
    assert.equal(classifyIssueForDispatch(phaseIssue, [], [], retryDispatchConfig).eligible, true);
    fs.writeFileSync(retryStatePath, JSON.stringify({ status: "failed", attempts: 4 }));
    assert.equal(classifyIssueForDispatch(phaseIssue, [], [], retryDispatchConfig).eligible, false);
    assert.equal(
      classifyIssueForDispatch(
        phaseIssue,
        [{ number: 310, title: "Fixes #900264", headRefName: "codex/issue-900264-fix" }],
        [],
        dispatchConfig,
      )
        .eligible,
      false,
    );
    const mergedPhasePr = {
      number: 311,
      title: "Fixes #900264",
      headRefName: "codex/issue-900264-reference-input-snapshot-integrity",
      baseRefName: "phase-0-platform-foundation",
      url: "https://example.invalid/pull/311",
      mergedAt: "2026-05-21T00:00:00Z",
    };
    assert.equal(
      classifyIssueForDispatch(
        { ...phaseIssue, updatedAt: "2026-05-21T01:00:00Z" },
        [],
        [mergedPhasePr],
        { ...dispatchConfig, workspaceRoot: path.join(tempRoot, "historical") },
      ).status,
      "eligible",
    );
    assert.equal(
      classifyIssueForDispatch(
        { ...phaseIssue, updatedAt: "2026-05-20T23:00:00Z" },
        [],
        [mergedPhasePr],
        { ...dispatchConfig, workspaceRoot: path.join(tempRoot, "historical") },
      ).status,
      "merged",
    );
    const mergedStateWorkspace = issueWorkspacePath(phaseIssue, dispatchConfig);
    fs.mkdirSync(mergedStateWorkspace, { recursive: true });
    const mergedStatePath = path.join(mergedStateWorkspace, "state.json");
    fs.writeFileSync(
      mergedStatePath,
      `${JSON.stringify({ status: "succeeded", issue_number: 900264, branch: mergedPhasePr.headRefName }, null, 2)}\n`,
    );
    const mergedClassification = classifyIssueForDispatch(phaseIssue, [], [mergedPhasePr], dispatchConfig);
    assert.equal(mergedClassification.status, "merged");
    assert.equal(mergedClassification.merged_pr.number, 311);
    assert.equal(mergedClassification.eligible, false);
    const wrongBaseIssue = { ...phaseIssue, number: 900265, title: "P0-0A-002: Environment Role Separation" };
    assert.equal(
      classifyIssueForDispatch(
        wrongBaseIssue,
        [],
        [{ ...mergedPhasePr, title: "Fixes #900265", headRefName: "codex/issue-900265-environment-role-separation", baseRefName: "ui-improvements" }],
        dispatchConfig,
      ).status,
      "eligible",
    );
    assert.equal(
      rankedDispatchQueue([blockedIssue, phaseIssue], [], [], { ...dispatchConfig, workspaceRoot: path.join(tempRoot, "rank") })[0].number,
      900264,
    );
    assert.ok(renderIssuePrompt({ issue: phaseIssue, dispatchConfig, workflow, skills: fakeSkills }).includes("Target branch: phase-0-platform-foundation"));
    assert.deepEqual(splitCommandLine("codex --ask-for-approval never exec --cd {repo} -"), [
      "codex",
      "--ask-for-approval",
      "never",
      "exec",
      "--cd",
      "{repo}",
      "-",
    ]);
    assert.equal(renderCommandToken("{repo}/prompt.md", { repo: "/tmp/example" }), "/tmp/example/prompt.md");
    assert.equal(
      issueForPullRequest({ title: "Ship v2 remediation", head_ref: "codex/v2-review-remediation" }, [
        { number: 2, title: "Unrelated issue" },
      ]),
      null,
    );
    assert.equal(
      issueForPullRequest({ title: "Fixes #267", head_ref: "codex/phase0-review-remediation-autonomy" }, [
        { number: 267, title: "Dispatch review remediation" },
      ])?.number,
      267,
    );

    const reviewWorkflow = { promptTemplate: "Review workflow prompt" };
    const mergedStateResults = markMergedIssueWorkspaceStates({
      issues: [phaseIssue],
      mergedPrs: [mergedPhasePr],
      dispatchConfig,
      dryRun: false,
    });
    assert.equal(mergedStateResults[0].status, "merged");
    const mergedState = JSON.parse(fs.readFileSync(mergedStatePath, "utf8"));
    assert.equal(mergedState.status, "merged");
    assert.equal(mergedState.merged_pr, 311);

    const oldSlugWorkspace = path.join(tempRoot, "issue-267-old-title");
    const oldSlugRepo = path.join(oldSlugWorkspace, "repo");
    fs.mkdirSync(oldSlugRepo, { recursive: true });
    run("git", ["init"], { cwd: oldSlugRepo });
    run("git", ["checkout", "-b", "codex/issue-267-review-remediation"], { cwd: oldSlugRepo });
    assert.equal(findIssueWorkspace({ number: 267, title: "New title" }, dispatchConfig), oldSlugWorkspace);
    const slugDriftResult = ensureReviewRemediationDispatch({
      pr: {
        number: 288,
        title: "Fixes #267",
        head_ref: "codex/issue-267-review-remediation",
        target_branch: "custom-target",
        unresolved_codex_review_thread_summaries: [],
      },
      issues: [{ number: 267, title: "New title", body: "", url: "https://example.invalid/issues/267" }],
      dispatchConfig,
      workflow: reviewWorkflow,
      skills: fakeSkills,
      dryRun: true,
    });
    assert.equal(slugDriftResult.status, "would-remediate");
    assert.equal(slugDriftResult.workspace, oldSlugWorkspace);
    assert.equal(slugDriftResult.target_branch, "custom-target");

    const closedIssueWorkspace = path.join(tempRoot, "pr-289");
    const closedIssueRepo = path.join(closedIssueWorkspace, "repo");
    fs.mkdirSync(closedIssueRepo, { recursive: true });
    run("git", ["init"], { cwd: closedIssueRepo });
    run("git", ["checkout", "-b", "codex/closed-issue-review"], { cwd: closedIssueRepo });
    const closedIssueResult = ensureReviewRemediationDispatch({
      pr: {
        number: 289,
        title: "Fix closed source issue review feedback",
        head_ref: "codex/closed-issue-review",
        target_branch: "phase-0-platform-foundation",
        unresolved_codex_review_thread_summaries: [],
      },
      issues: [],
      dispatchConfig,
      workflow: reviewWorkflow,
      skills: fakeSkills,
      dryRun: true,
    });
    assert.equal(closedIssueResult.status, "would-remediate");
    assert.equal(closedIssueResult.issue_number, null);

    const staleStateWorkspace = path.join(tempRoot, "issue-267-a-stale-title");
    const staleStateRepo = path.join(staleStateWorkspace, "repo");
    fs.mkdirSync(staleStateRepo, { recursive: true });
    run("git", ["init"], { cwd: staleStateRepo });
    run("git", ["checkout", "-b", "codex/stale-review-branch"], { cwd: staleStateRepo });
    fs.writeFileSync(
      path.join(staleStateWorkspace, "state.json"),
      `${JSON.stringify({ issue_number: 267, branch: "codex/stale-review-branch" }, null, 2)}\n`,
    );
    const exactPrWorkspace = path.join(tempRoot, "issue-267-z-current-title");
    const exactPrRepo = path.join(exactPrWorkspace, "repo");
    fs.mkdirSync(exactPrRepo, { recursive: true });
    run("git", ["init"], { cwd: exactPrRepo });
    run("git", ["checkout", "-b", "codex/current-review-branch"], { cwd: exactPrRepo });
    fs.writeFileSync(
      path.join(exactPrWorkspace, "state.json"),
      `${JSON.stringify({ issue_number: 267, review_remediation_pr: 292, branch: "codex/current-review-branch" }, null, 2)}\n`,
    );
    assert.equal(
      workspaceForReviewRemediation({
        pr: { number: 292, head_ref: "codex/current-review-branch" },
        issue: { number: 267, title: "Current title" },
        dispatchConfig,
      }),
      exactPrWorkspace,
    );

    const wrongBranchWorkspace = path.join(tempRoot, "pr-290");
    const wrongBranchRepo = path.join(wrongBranchWorkspace, "repo");
    fs.mkdirSync(wrongBranchRepo, { recursive: true });
    run("git", ["init"], { cwd: wrongBranchRepo });
    run("git", ["checkout", "-b", "codex/wrong-branch"], { cwd: wrongBranchRepo });
    assert.match(
      ensureReviewRemediationDispatch({
        pr: {
          number: 290,
          title: "Fixes #267",
          head_ref: "codex/expected-branch",
          target_branch: "phase-0-platform-foundation",
          unresolved_codex_review_thread_summaries: [],
        },
        issues: [],
        dispatchConfig,
        workflow: reviewWorkflow,
        skills: fakeSkills,
        dryRun: true,
      }).reason,
      /expected codex\/expected-branch/,
    );
    assert.equal(
      ensureReviewRemediationDispatch({
        pr: {
          number: 291,
          title: "No prepared workspace",
          head_ref: "codex/missing-workspace",
          target_branch: "phase-0-platform-foundation",
          unresolved_codex_review_thread_summaries: [],
        },
        issues: [],
        dispatchConfig,
        workflow: reviewWorkflow,
        skills: fakeSkills,
        dryRun: true,
      }).status,
      "blocked",
    );
    assert.equal(
      ensureReviewRemediationDispatch({
        pr: {
          number: 292,
          title: "Unsafe branch",
          head_ref: "feature/manual-branch",
          target_branch: "phase-0-platform-foundation",
          unresolved_codex_review_thread_summaries: [],
        },
        issues: [],
        dispatchConfig,
        workflow: reviewWorkflow,
        skills: fakeSkills,
        dryRun: true,
      }).status,
      "blocked",
    );

    const evals = browserEvaluationsFor(DEFAULT_MONITOR, { blocker: "" }, [{ ok: true }, { ok: true }]);
    assert.equal(evals.length, 1);
    assert.equal(evals[0].screenshot_required, false);
    assert.equal(evals[0].persona_setup, "npm run dev:persona -- it_admin --base-url http://localhost:5173");
    assert.equal(
      browserEvaluationsFor({ ...DEFAULT_MONITOR, browserDefaultUrl: "http://127.0.0.1:6173/dashboard/it-admin" }, { blocker: "" }, [
        { ok: true },
      ])[0].persona_setup,
      "npm run dev:persona -- it_admin --base-url http://127.0.0.1:6173",
    );
    assert.equal(browserEvaluationsFor(DEFAULT_MONITOR, { blocker: "dirty" }, [{ ok: true }]).length, 0);
    assert.equal(
      validateBrowserResults([
        { url: "http://localhost:5173/dashboard/it-admin", status: "blocked", evidence: [], findings: [], checked_at: "now" },
      ]).length,
      1,
    );

    const staleLock = path.join(tempRoot, "stale.lock");
    fs.mkdirSync(staleLock);
    fs.writeFileSync(path.join(staleLock, "pid"), "999999\n");
    fs.writeFileSync(path.join(staleLock, "started_at"), "2020-01-01T00:00:00.000Z\n");
    assert.equal(acquireLock(staleLock, { now: new Date("2026-01-01T00:00:00.000Z"), maxAgeMs: 1000 }).stale_removed, true);
    releaseLock(staleLock);

    const activeLock = path.join(tempRoot, "active.lock");
    fs.mkdirSync(activeLock);
    fs.writeFileSync(path.join(activeLock, "pid"), `${process.pid}\n`);
    fs.writeFileSync(path.join(activeLock, "started_at"), `${new Date().toISOString()}\n`);
    assert.throws(() => acquireLock(activeLock, { maxAgeMs: DEFAULT_MONITOR.lockMaxAgeMs }), /Active Symphony monitor lock/);
    fs.rmSync(activeLock, { recursive: true, force: true });

    console.log("symphony runner self-tests passed");
  } finally {
    fs.rmSync(tempRoot, { recursive: true, force: true });
  }
}

async function main() {
  const { command, options } = parseArgs(process.argv.slice(2));
  if (!command || options.help) {
    console.log(usage());
    return;
  }
  if (command === "report") {
    await report(options);
  } else if (command === "sync") {
    await sync(options);
  } else if (command === "ui-monitor") {
    await uiMonitor(options);
  } else if (command === "record-browser-results") {
    recordBrowserResults(options);
  } else if (command === "test") {
    await selfTest();
  } else {
    throw new Error(`Unknown command: ${command}\n${usage()}`);
  }
}

main().catch((error) => {
  console.error(error.stack || error.message);
  process.exitCode = 1;
});
