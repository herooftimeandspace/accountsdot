#!/usr/bin/env node
import assert from "node:assert/strict";
import { execFileSync, spawn } from "node:child_process";
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
  autoResolveOutdatedCodexReviewThreads: true,
  codexReviewBot: "chatgpt-codex-connector[bot]",
  codexReviewSuccessReactions: ["THUMBS_UP", "+1"],
  codexReviewInProgressReactions: ["EYES"],
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

let cachedGitHubRepository = null;

function githubRepositorySlug() {
  if (cachedGitHubRepository) return cachedGitHubRepository;
  if (process.env.GH_REPO && /^[^/\s]+\/[^/\s]+$/.test(process.env.GH_REPO)) {
    cachedGitHubRepository = process.env.GH_REPO;
    return cachedGitHubRepository;
  }
  const repository = ghJson(["repo", "view", "--json", "nameWithOwner"]);
  if (!repository?.nameWithOwner || !/^[^/\s]+\/[^/\s]+$/.test(repository.nameWithOwner)) {
    throw new Error("Could not resolve GitHub repository from GH_REPO or gh repo view");
  }
  cachedGitHubRepository = repository.nameWithOwner;
  return cachedGitHubRepository;
}

function githubRepositoryParts() {
  const [owner, repo] = githubRepositorySlug().split("/");
  return { owner, repo };
}

function githubRepoApiPath(suffix) {
  return `repos/${githubRepositorySlug()}/${suffix}`;
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

function listOpenIssuesForLabel(label = "") {
  const issues = [];
  for (let page = 1; ; page += 1) {
    const args = [
      "api",
      "--method",
      "GET",
      githubRepoApiPath("issues"),
      "-f",
      "state=open",
      "-f",
      "per_page=100",
      "-f",
      `page=${page}`,
    ];
    if (label) {
      args.push("-f", `labels=${label}`);
    }
    const rawPageItems = ghJson(args);
    issues.push(...rawPageItems.filter((issue) => !issue.pull_request));
    if (rawPageItems.length < 100) break;
  }
  return issues;
}

function normalizeIssueFromApi(issue, comments = null) {
  return {
    number: issue.number,
    title: issue.title || "",
    body: issue.body || "",
    comments: comments || hydrateIssueComments(issue.number),
    labels: Array.isArray(issue.labels) ? issue.labels.map((label) => ({ name: label.name || String(label) })) : [],
    url: issue.html_url || issue.url || "",
    updatedAt: issue.updated_at || "",
    assignees: issue.assignees || [],
  };
}

function hydrateIssueComments(issueNumber) {
  const comments = [];
  for (let page = 1; ; page += 1) {
    const pageItems = ghJson([
      "api",
      "--method",
      "GET",
      githubRepoApiPath(`issues/${issueNumber}/comments`),
      "-f",
      "per_page=100",
      "-f",
      `page=${page}`,
    ]);
    comments.push(
      ...pageItems.map((comment) => ({
        id: comment.id,
        author: comment.user ? { login: comment.user.login } : null,
        body: comment.body || "",
        createdAt: comment.created_at || "",
        updatedAt: comment.updated_at || "",
        url: comment.html_url || "",
      })),
    );
    if (pageItems.length < 100) break;
  }
  return comments;
}

function mergeIssuesByNumber(issueLists) {
  const merged = new Map();
  for (const issues of issueLists) {
    for (const issue of issues || []) {
      merged.set(issue.number, issue);
    }
  }
  return [...merged.values()];
}

function listOpenIssues(activeLabels = []) {
  const issueLists = [listOpenIssuesForLabel()];
  for (const label of activeLabels) {
    issueLists.push(listOpenIssuesForLabel(label));
  }
  return mergeIssuesByNumber(issueLists).map((issue) => normalizeIssueFromApi(issue, hydrateIssueComments(issue.number)));
}

function fetchReviewThreads(baseRef) {
  const { owner, repo } = githubRepositoryParts();
  const query =
    'query($owner:String!,$repo:String!,$base:String!){ repository(owner:$owner,name:$repo){ pullRequests(first:100, states:OPEN, baseRefName:$base) { nodes { number reviewThreads(first:100) { nodes { id isResolved isOutdated comments(first:10){ nodes { author { login } body createdAt path line originalLine url } } } } } } } }';
  const result = ghJson([
    "api",
    "graphql",
    "-f",
    `query=${query}`,
    "-F",
    `owner=${owner}`,
    "-F",
    `repo=${repo}`,
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
  const { owner, repo } = githubRepositoryParts();
  const query =
    'query($owner:String!,$repo:String!,$base:String!){ repository(owner:$owner,name:$repo){ pullRequests(first:100, states:OPEN, baseRefName:$base) { nodes { number reactionGroups { content users(first:20){ nodes { login } } } reviews(first:50){ nodes { author { login } state submittedAt } } comments(first:50){ nodes { author { login } body createdAt reactionGroups { content users(first:20){ nodes { login } } } } } } } } }';
  const result = ghJson([
    "api",
    "graphql",
    "-f",
    `query=${query}`,
    "-F",
    `owner=${owner}`,
    "-F",
    `repo=${repo}`,
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

function reconcileOutdatedPullRequestReviewThreads({ reviewThreads, config, dryRun }) {
  return remediateReviewThreads({
    reviewThreads,
    config: {
      codexReviewAuthors: config.codexReviewAuthors,
      autoResolveOutdatedCodexReviewThreads: config.autoResolveOutdatedCodexReviewThreads,
    },
    dryRun,
  });
}

function resolveRemediatedReviewThreads({ pr, config, dryRun, branchUpdated, resolveThread = resolveReviewThread }) {
  const threadSummaries = pr.unresolved_codex_review_thread_summaries || [];
  if (threadSummaries.length === 0) return [];
  return threadSummaries.map((thread) => {
    const result = {
      number: pr.number,
      action: dryRun ? "would-resolve-remediated-thread" : "resolve-remediated-thread",
      status: dryRun ? "dry-run" : "skipped",
      reason: "",
      ...thread,
    };
    if (dryRun) {
      result.reason = "dry-run performs no review-thread mutations";
      return result;
    }
    if (!config.autoResolveOutdatedCodexReviewThreads) {
      result.status = "blocked";
      result.reason = "auto_resolve_outdated_codex_review_threads is disabled";
      return result;
    }
    if (!branchUpdated) {
      result.status = "blocked";
      result.reason = "review remediation completed without a branch update, so the automation did not resolve active review feedback";
      return result;
    }
    try {
      resolveThread(thread.thread_id);
      result.status = "resolved";
      result.reason = "review remediation succeeded and updated the PR branch, so the handed-off Codex Review thread was resolved";
    } catch (error) {
      result.status = "blocked";
      result.reason = error.stderr || error.message || String(error);
    }
    return result;
  });
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
  return hasBotReaction(signals, botLogin, successReactions);
}

function hasBotInProgressReaction(signals, config) {
  const inProgressReactions = new Set((config.codexReviewInProgressReactions || []).map((reaction) => String(reaction).toUpperCase()));
  const botLogin = config.codexReviewBot || "";
  return hasBotReaction(signals, botLogin, inProgressReactions);
}

function reactionGroupsHaveBotReaction(reactionGroups, botLogin, allowedReactions) {
  return (reactionGroups || []).some((group) => {
    if (!allowedReactions.has(String(group.content || "").toUpperCase())) return false;
    return (group.users?.nodes || []).some((user) => authorMatches(user.login, [botLogin]));
  });
}

function hasBotReaction(signals, botLogin, allowedReactions) {
  if (reactionGroupsHaveBotReaction(signals.reactionGroups, botLogin, allowedReactions)) return true;
  return (signals.comments || []).some((comment) =>
    reactionGroupsHaveBotReaction(comment.reactionGroups, botLogin, allowedReactions),
  );
}

function timestampMs(value) {
  const parsed = Date.parse(value || "");
  return Number.isNaN(parsed) ? 0 : parsed;
}

function latestCodexReviewResponseTime(signals, config) {
  const authors = config.codexReviewAuthors || [];
  return Math.max(
    0,
    ...(signals.reviews || [])
      .filter((review) => authorMatches(review.author?.login, authors))
      .map((review) => timestampMs(review.submittedAt)),
  );
}

function hasPendingCodexReviewRequest(signals, config) {
  const inProgressReactions = new Set((config.codexReviewInProgressReactions || []).map((reaction) => String(reaction).toUpperCase()));
  const successReactions = new Set((config.codexReviewSuccessReactions || []).map((reaction) => String(reaction).toUpperCase()));
  const botLogin = config.codexReviewBot || "";
  const latestReviewTime = latestCodexReviewResponseTime(signals, config);
  return (signals.comments || []).some((comment) => {
    if (!/@codex\b/i.test(String(comment.body || ""))) return false;
    if (timestampMs(comment.createdAt) < latestReviewTime) return false;
    if (reactionGroupsHaveBotReaction(comment.reactionGroups, botLogin, successReactions)) return false;
    return true;
  });
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

function mergeStateBlocker(mergeStateStatus) {
  const state = String(mergeStateStatus || "").toUpperCase();
  if (state === "CLEAN") return "";
  if (state === "UNKNOWN") return "";
  return `merge state ${mergeStateStatus}`;
}

function evaluatePullRequestForMerge({ pr, reviewThreads, signals, config }) {
  const threadEntry = reviewThreadsForPr(reviewThreads, pr.number);
  const blockers = [];
  const warnings = [];
  const labelBlockers = prLabelNames(pr).filter((label) => (config.blockedLabels || []).includes(label));
  if (pr.isDraft) blockers.push("draft PR");
  const mergeBlocker = mergeStateBlocker(pr.mergeStateStatus);
  if (mergeBlocker) blockers.push(mergeBlocker);
  if (String(pr.mergeStateStatus || "").toUpperCase() === "UNKNOWN") {
    warnings.push("mergeability is temporarily unknown; GitHub merge command must make the final server-side decision");
  }
  if (labelBlockers.length > 0) blockers.push(`blocked labels: ${labelBlockers.join(", ")}`);
  const checkState = statusRollupState(pr.statusCheckRollup);
  if (checkState === "failing") blockers.push("status checks are failing or pending");
  if (threadEntry.unresolved_threads.some((thread) => isCodexReviewThread(thread, { codexReviewAuthors: config.codexReviewAuthors }))) {
    blockers.push("unresolved current Codex Review thread");
  }
  if (hasRequestedChanges(signals, config)) blockers.push("Codex Review requested changes");

  const codexReviewResponse = hasCodexReviewResponse({ signals, threadEntry, config });
  const botThumbsUp = hasBotSuccessReaction(signals, config);
  const botEyes = hasBotInProgressReaction(signals, config);
  const pendingCodexReviewRequest = hasPendingCodexReviewRequest(signals, config);
  if (pendingCodexReviewRequest) {
    if (botEyes) warnings.push("Codex Review is looking at a newer @codex review request after chatgpt-codex-connector bot eyes reaction");
    else warnings.push("waiting for Codex Review response or bot reaction on a newer @codex review request");
  } else if (!codexReviewResponse && !botThumbsUp) {
    if (botEyes) warnings.push("Codex Review is looking at the PR after chatgpt-codex-connector bot eyes reaction");
    else {
      warnings.push("waiting for Codex Review response or bot thumbs-up reaction");
    }
  }
  if (!codexReviewResponse && botThumbsUp && config.noReviewWithBotThumbsUpIsClean) {
    warnings.push("no Codex Review response, but chatgpt-codex-connector bot thumbs-up is configured as clean evidence");
  }

  const ready =
    !pendingCodexReviewRequest &&
    blockers.length === 0 &&
    (codexReviewResponse || (botThumbsUp && config.noReviewWithBotThumbsUpIsClean));
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
    bot_eyes: botEyes,
    pending_codex_review_request: pendingCodexReviewRequest,
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
    agentRunnerTimeoutMs: Number(dispatch.agent_runner_timeout_ms ?? 6 * 60 * 60 * 1000),
    agentRunnerIdleTimeoutMs: Number(dispatch.agent_runner_idle_timeout_ms ?? 120000),
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
    autoResolveOutdatedCodexReviewThreads:
      configured.auto_resolve_outdated_codex_review_threads === undefined
        ? DEFAULT_PULL_REQUESTS.autoResolveOutdatedCodexReviewThreads
        : Boolean(configured.auto_resolve_outdated_codex_review_threads),
    codexReviewBot: configured.codex_review_bot || DEFAULT_PULL_REQUESTS.codexReviewBot,
    codexReviewSuccessReactions:
      configured.codex_review_success_reactions || DEFAULT_PULL_REQUESTS.codexReviewSuccessReactions,
    codexReviewInProgressReactions:
      configured.codex_review_in_progress_reactions || DEFAULT_PULL_REQUESTS.codexReviewInProgressReactions,
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
  const workspacePath = findIssueWorkspace(issue, dispatchConfig);
  const statePath = path.join(workspacePath, "state.json");
  const promptPath = path.join(workspacePath, "prompt.md");
  const repoPath = path.join(workspacePath, "repo");
  if (fs.existsSync(statePath)) {
    try {
      const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
      const activeStatuses = new Set(["prepared", "running", "waiting_retry", "human_review"]);
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
      if (state.status === "failed") {
        const parsedAttempts = Number(state.attempts || 1);
        const attempts = Number.isFinite(parsedAttempts) ? parsedAttempts : dispatchConfig.maxAttempts;
        if (attempts >= dispatchConfig.maxAttempts) {
          return {
            workspace: workspacePath,
            prompt_path: promptPath,
            state_path: statePath,
            status: state.status,
            attempts,
            retryable: false,
          };
        }
        return null;
      }
      if (state.status === "succeeded") return null;
    } catch (error) {
      return { workspace: workspacePath, prompt_path: promptPath, state_path: statePath, status: "unreadable_state", blocker: error.message };
    }
  }
  const branchName = issueBranchName(issue, dispatchConfig);
  const existingWorktree = worktreeForBranch(branchName);
  if (existingWorktree) {
    if (path.resolve(existingWorktree.path) === path.resolve(repoPath)) return null;
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

function issueDispatchWorkspacePath(issue, dispatchConfig) {
  const currentPath = issueWorkspacePath(issue, dispatchConfig);
  const resolvedPath = findIssueWorkspace(issue, dispatchConfig);
  if (path.resolve(currentPath) === path.resolve(resolvedPath)) return currentPath;
  const branchName = issueBranchName(issue, dispatchConfig);
  if (workspaceRepoMatchesBranch(resolvedPath, branchName)) return resolvedPath;
  const statePath = path.join(resolvedPath, "state.json");
  if (fs.existsSync(statePath)) {
    try {
      const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
      if (String(state.branch || "") === branchName) return resolvedPath;
    } catch {
      return resolvedPath;
    }
  }
  return currentPath;
}

const WORKTREE_TEARDOWN_GENERATED_CACHE_DIRS = [".gomodcache", ".gocache", path.join("node_modules", ".cache")];

function worktreeRemoveFailureNeedsGeneratedCachePermissionRetry(reason) {
  return /permission denied/i.test(reason || "") && /(?:failed to remove|\.gomodcache|\.gocache|node_modules[/\\]\.cache)/i.test(reason || "");
}

function makeTreeUserWritable(targetPath) {
  const stat = fs.lstatSync(targetPath);
  if (stat.isSymbolicLink()) return;
  const writableMode = stat.mode | (stat.isDirectory() ? 0o700 : 0o600);
  fs.chmodSync(targetPath, writableMode);
  if (!stat.isDirectory()) return;
  for (const entry of fs.readdirSync(targetPath)) {
    makeTreeUserWritable(path.join(targetPath, entry));
  }
}

function normalizeGeneratedCachePermissionsForTeardown(repoPath) {
  const fixes = [];
  for (const relativePath of WORKTREE_TEARDOWN_GENERATED_CACHE_DIRS) {
    const targetPath = path.join(repoPath, relativePath);
    if (!fs.existsSync(targetPath)) continue;
    try {
      makeTreeUserWritable(targetPath);
      fixes.push({ path: relativePath, status: "made-user-writable" });
    } catch (error) {
      fixes.push({ path: relativePath, status: "blocked", reason: error.message });
    }
  }
  return fixes;
}

function removeCleanWorktreeForMergedIssue(repoPath) {
  const firstAttempt = run("git", ["worktree", "remove", repoPath], { cwd: repoRoot, allowFailure: true });
  if (!firstAttempt.failed) return { ok: true, retried_after_cache_permission_fix: false };
  const firstReason = firstAttempt.stderr || firstAttempt.stdout || `git worktree remove failed with status ${firstAttempt.status}`;
  if (!worktreeRemoveFailureNeedsGeneratedCachePermissionRetry(firstReason)) {
    return { ok: false, reason: firstReason, retried_after_cache_permission_fix: false };
  }
  const permissionFixes = normalizeGeneratedCachePermissionsForTeardown(repoPath);
  if (permissionFixes.some((fix) => fix.status === "blocked")) {
    return {
      ok: false,
      reason: firstReason,
      retried_after_cache_permission_fix: true,
      teardown_permission_fixes: permissionFixes,
    };
  }
  const secondAttempt = run("git", ["worktree", "remove", repoPath], { cwd: repoRoot, allowFailure: true });
  if (!secondAttempt.failed) {
    return {
      ok: true,
      retried_after_cache_permission_fix: true,
      teardown_permission_fixes: permissionFixes,
    };
  }
  return {
    ok: false,
    reason: secondAttempt.stderr || secondAttempt.stdout || `git worktree remove failed with status ${secondAttempt.status}`,
    first_reason: firstReason,
    retried_after_cache_permission_fix: true,
    teardown_permission_fixes: permissionFixes,
  };
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
          if (status.blocker !== "worktree has local edits") {
            results.push({ ...result, status: "blocked", teardown_status: "blocked", reason: status.blocker, dirty_files: status.dirty_files });
            continue;
          }
          if (dryRun) {
            results.push({
              ...result,
              dirty_files: status.dirty_files,
              teardown_status: "would-preserve-dirty-worktree-and-remove",
              workspace_preparation: {
                reason: "dry-run would preserve dirty merged workspace edits before teardown",
                dirty_files: status.dirty_files,
              },
            });
            continue;
          }
          const preserved = preserveDirtyWorkspaceBeforeRefresh({
            repoPath,
            logsPath: path.join(workspacePath, "logs"),
            prNumber: mergedPr.number,
          });
          if (!preserved.ok) {
            results.push({
              ...result,
              status: "blocked",
              teardown_status: "blocked",
              reason: preserved.reason,
              dirty_files: status.dirty_files,
              workspace_preparation: preserved,
            });
            continue;
          }
          result.dirty_files = status.dirty_files;
          result.workspace_preparation = preserved;
        }
        if (!dryRun) {
          const removed = removeCleanWorktreeForMergedIssue(repoPath);
          if (!removed.ok) {
            results.push({
              ...result,
              status: "blocked",
              teardown_status: "blocked",
              reason: removed.reason,
              first_reason: removed.first_reason,
              retried_after_cache_permission_fix: removed.retried_after_cache_permission_fix,
              teardown_permission_fixes: removed.teardown_permission_fixes,
            });
            continue;
          }
          if (removed.teardown_permission_fixes) {
            result.teardown_permission_fixes = removed.teardown_permission_fixes;
          }
          if (removed.retried_after_cache_permission_fix) {
            result.teardown_status = "removed-worktree-after-cache-permission-fix";
          }
        }
        if (dryRun) {
          result.teardown_status = "would-remove-clean-worktree";
        } else if (result.teardown_status === "not-needed") {
          result.teardown_status = "removed-worktree";
        }
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

function hasUnattemptedReadyToMergePr(prMergeQueue, prMergeResults) {
  return prMergeQueue.some(
    (entry) => entry.status === "ready_to_merge" && !prMergeResults.some((result) => result.number === entry.number),
  );
}

function renderIssuePrompt({ issue, dispatchConfig, workflow, skills, dirtyStatus = null, workspaceState = null, workspacePath = null }) {
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
  const renderedWorkspacePath = workspacePath || issueWorkspacePath(issue, dispatchConfig);
  const existingWorkspaceSection =
    dirtyStatus?.dirty_files?.length > 0 || workspaceState?.status
      ? [
          "## Existing Workspace State",
          "",
          workspaceState?.status ? `Previous runner state: ${workspaceState.status}` : "",
          workspaceState?.runner_status ? `Previous runner status: ${workspaceState.runner_status}` : "",
          dirtyStatus?.dirty_files?.length > 0
            ? [
                "The prepared issue worktree already contains local edits for this same issue branch. Treat them as previous automation work for this issue.",
                "Inspect them before changing code, preserve in-scope work, repair or finish anything incomplete, then commit, push, and open/update the PR.",
                "",
                "Dirty files at dispatch start:",
                dirtyStatus.dirty_files.map((file) => `- ${file}`).join("\n"),
              ].join("\n")
            : "",
          "",
        ]
          .filter(Boolean)
          .join("\n")
      : "";
  return [
    "# Symphony Issue Dispatch Prompt",
    "",
    `Issue: #${issue.number} ${issue.title}`,
    `Issue URL: ${issue.url}`,
    `Target branch: ${targetBranch}`,
    `Working branch: ${branchName}`,
    `Workspace root: ${renderedWorkspacePath}`,
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
    existingWorkspaceSection,
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

function terminateChildProcess(child) {
  if (!child || child.killed) return;
  try {
    if (process.platform === "win32") {
      child.kill("SIGTERM");
    } else {
      process.kill(-child.pid, "SIGTERM");
    }
  } catch {
    try {
      child.kill("SIGTERM");
    } catch {
      // The child may have exited between timeout handling and termination.
    }
  }
}

function forceKillChildProcess(child) {
  if (!child || !child.pid) return;
  try {
    if (process.platform === "win32") {
      child.kill("SIGKILL");
    } else {
      process.kill(-child.pid, "SIGKILL");
    }
  } catch {
    try {
      child.kill("SIGKILL");
    } catch {
      // The child may have exited after the graceful termination attempt.
    }
  }
}

function appendBoundedTail(current, text, maxChars = 64 * 1024) {
  const next = `${current || ""}${text || ""}`;
  return next.length > maxChars ? next.slice(-maxChars) : next;
}

function isCompletedCodexTurnLine(line) {
  const trimmed = String(line || "").trim();
  if (!trimmed) return false;
  try {
    return JSON.parse(trimmed).type === "turn.completed";
  } catch {
    return false;
  }
}

function consumeCodexStdoutEvents({ chunk, remainder }) {
  const combined = `${remainder || ""}${chunk || ""}`;
  const lines = combined.split(/\r?\n/);
  const nextRemainder = lines.pop() || "";
  return {
    completedCodexTurn: lines.some((line) => isCompletedCodexTurnLine(line)),
    remainder: nextRemainder,
  };
}

function writeAgentRunnerLogs(logsPath, stdout, stderr) {
  const stdoutPath = path.join(logsPath, "agent-stdout.log");
  const stderrPath = path.join(logsPath, "agent-stderr.log");
  fs.writeFileSync(stdoutPath, stdout || "");
  fs.writeFileSync(stderrPath, stderr || "");
  return { stdoutPath, stderrPath };
}

function classifyAgentRunnerFailure({ error, stdoutTail, stderrTail, hasOutput, timedOut, idleTimedOut, completedCodexTurn }) {
  if (idleTimedOut && completedCodexTurn) {
    return {
      status: "succeeded",
      runner_status: "completed",
      reason: "agent runner emitted turn.completed and was reaped after idle timeout",
    };
  }
  if (idleTimedOut) {
    const reason = hasOutput ? "agent runner idle timeout without completion" : "agent runner produced no output before idle timeout";
    return { status: "failed", runner_status: "failed", reason };
  }
  if (timedOut) {
    return { status: "failed", runner_status: "failed", reason: "agent runner exceeded timeout" };
  }
  return {
    status: "failed",
    runner_status: "failed",
    reason: String(stderrTail || stdoutTail || error?.message || "agent runner failed").trim(),
  };
}

async function runAgentRunner({
  commandLine,
  repoPath,
  promptPath,
  prompt,
  issue,
  branchName,
  targetBranch,
  logsPath,
  timeoutMs,
  idleTimeoutMs,
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
  fs.mkdirSync(logsPath, { recursive: true });
  const stdoutPath = path.join(logsPath, "agent-stdout.log");
  const stderrPath = path.join(logsPath, "agent-stderr.log");
  try {
    codexHome = prepareAgentCodexHome({ codexHomeRoot, workspacePath }) || "";
  } catch (error) {
    const completedAt = new Date().toISOString();
    writeAgentRunnerLogs(logsPath, "", String(error.message || error));
    return {
      status: "failed",
      runner_status: "failed",
      command: [cmd, ...args],
      codex_home_path: codexHome,
      started_at: startedAt,
      completed_at: completedAt,
      stdout_path: stdoutPath,
      stderr_path: stderrPath,
      exit_status: null,
      reason: String(error.message || "failed to prepare agent runner").trim(),
    };
  }

  return new Promise((resolve) => {
    let stdoutTail = "";
    let stderrTail = "";
    let stdoutRemainder = "";
    let completedCodexTurn = false;
    let hasOutput = false;
    let exitStatus = null;
    let exitSignal = null;
    let timedOut = false;
    let idleTimedOut = false;
    let settled = false;
    let idleTimer = null;
    let timeoutTimer = null;
    let forceTimer = null;

    const child = spawn(cmd, args, {
      cwd: repoPath,
      stdio: ["pipe", "pipe", "pipe"],
      detached: process.platform !== "win32",
      env: codexHome ? { ...process.env, CODEX_HOME: codexHome } : process.env,
    });

    const finish = ({ error = null } = {}) => {
      if (settled) return;
      settled = true;
      clearTimeout(idleTimer);
      clearTimeout(timeoutTimer);
      clearTimeout(forceTimer);
      const completedAt = new Date().toISOString();
      completedCodexTurn = completedCodexTurn || isCompletedCodexTurnLine(stdoutRemainder);
      if (!error && exitStatus === 0 && !exitSignal) {
        resolve({
          status: "succeeded",
          runner_status: "completed",
          command: [cmd, ...args],
          codex_home_path: codexHome,
          started_at: startedAt,
          completed_at: completedAt,
          stdout_path: stdoutPath,
          stderr_path: stderrPath,
          reason: "agent runner completed successfully",
        });
        return;
      }
      const classification = classifyAgentRunnerFailure({
        error,
        stdoutTail,
        stderrTail,
        hasOutput,
        timedOut,
        idleTimedOut,
        completedCodexTurn,
      });
      resolve({
        ...classification,
        command: [cmd, ...args],
        codex_home_path: codexHome,
        started_at: startedAt,
        completed_at: completedAt,
        stdout_path: stdoutPath,
        stderr_path: stderrPath,
        exit_status: exitStatus,
        exit_signal: exitSignal,
      });
    };

    const terminate = ({ idle = false, total = false }) => {
      if (settled) return;
      idleTimedOut = idleTimedOut || idle;
      timedOut = timedOut || total;
      terminateChildProcess(child);
      forceTimer = setTimeout(() => {
        forceKillChildProcess(child);
        finish();
      }, 5000);
    };

    const resetIdleTimer = () => {
      clearTimeout(idleTimer);
      if (!idleTimeoutMs || idleTimeoutMs <= 0) return;
      idleTimer = setTimeout(() => terminate({ idle: true }), idleTimeoutMs);
    };

    child.on("error", (error) => finish({ error }));
    child.on("close", (status, signal) => {
      exitStatus = status;
      exitSignal = signal;
      finish();
    });
    child.stdout.on("data", (chunk) => {
      const text = chunk.toString();
      hasOutput = hasOutput || text.length > 0;
      stdoutTail = appendBoundedTail(stdoutTail, text);
      const parsed = consumeCodexStdoutEvents({ chunk: text, remainder: stdoutRemainder });
      stdoutRemainder = parsed.remainder;
      completedCodexTurn = completedCodexTurn || parsed.completedCodexTurn;
      fs.appendFileSync(stdoutPath, text);
      resetIdleTimer();
    });
    child.stderr.on("data", (chunk) => {
      const text = chunk.toString();
      hasOutput = hasOutput || text.length > 0;
      stderrTail = appendBoundedTail(stderrTail, text);
      fs.appendFileSync(stderrPath, text);
      resetIdleTimer();
    });

    fs.writeFileSync(stdoutPath, "");
    fs.writeFileSync(stderrPath, "");
    child.stdin.end(prompt);
    resetIdleTimer();
    if (timeoutMs && timeoutMs > 0) {
      timeoutTimer = setTimeout(() => terminate({ total: true }), timeoutMs);
    }
  });
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

function preparedPrWorkspaceState(repoPath, branchName) {
  const branch = currentBranch(repoPath);
  if (!branch.ok) return { ok: false, branch: "", detached: false, reason: branch.reason };
  if (branch.branch === branchName) return { ok: true, branch: branch.branch, detached: false, reason: "" };
  if (branch.branch) {
    return {
      ok: false,
      branch: branch.branch,
      detached: false,
      reason: `prepared PR workspace is on ${branch.branch}, expected ${branchName}`,
    };
  }
  const head = revParse(repoPath, "HEAD");
  const remote = revParse(repoPath, `origin/${branchName}`);
  if (head.ok && remote.ok && head.oid === remote.oid) {
    return { ok: true, branch: "", detached: true, reason: `detached at origin/${branchName}` };
  }
  return {
    ok: false,
    branch: "",
    detached: true,
    reason: `prepared PR workspace is detached and not at origin/${branchName}`,
  };
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

function renderReviewRemediationPrompt({ pr, issue, dispatchConfig, workflow, skills, dirtyStatus, workspaceState }) {
  const targetBranch = pr.target_branch || "phase-0-platform-foundation";
  const branchName = pr.head_ref;
  const detachedWorkspace = Boolean(workspaceState?.detached);
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
              `- Thread ID: ${thread.thread_id || "(missing)"}`,
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
  const dirtyWorkspaceSection =
    dirtyStatus?.dirty_files?.length > 0
      ? [
          "## Existing Local Workspace Edits",
          "",
          "This prepared PR remediation workspace already has local edits. Treat them as previous automation work for this PR branch, not as a reason to stop.",
          "",
          dirtyStatus.dirty_files.map((file) => `- ${file}`).join("\n"),
          "",
          "Inspect these edits before changing files. Keep in-scope fixes, finish or correct incomplete work, run verification, commit, and push the PR branch. Do not discard local edits unless they are clearly generated noise or contradicted by the PR scope; explain any discarded files in the final report.",
          "",
        ].join("\n")
      : "";
  const workspacePrepSection =
    dirtyStatus?.workspace_preparation
      ? [
          "## Workspace Preparation Notes",
          "",
          dirtyStatus.workspace_preparation.reason ? `- ${dirtyStatus.workspace_preparation.reason}` : "",
          dirtyStatus.workspace_preparation.stash_ref
            ? `- Stale pre-refresh edits were preserved in ${dirtyStatus.workspace_preparation.stash_ref}. Inspect or apply that stash only if the refreshed PR head is missing required in-scope work.`
            : "",
          dirtyStatus.workspace_preparation.stash_oid
            ? `- Stable stash commit: ${dirtyStatus.workspace_preparation.stash_oid}`
            : "",
          dirtyStatus.workspace_preparation.preserved_head_ref
            ? `- Local-only pre-refresh commits were preserved at ${dirtyStatus.workspace_preparation.preserved_head_ref}. Inspect that ref only if the refreshed PR head is missing required in-scope work.`
            : "",
          dirtyStatus.workspace_preparation.status_manifest_path
            ? `- Pre-refresh dirty file manifest: ${dirtyStatus.workspace_preparation.status_manifest_path}`
            : "",
          dirtyStatus.workspace_preparation.target_merge_status
            ? `- Target-branch merge preflight: ${dirtyStatus.workspace_preparation.target_merge_status}`
            : "",
          dirtyStatus.workspace_preparation.target_merge_output
            ? `- Target-branch merge output: ${dirtyStatus.workspace_preparation.target_merge_output}`
            : "",
          "",
        ]
          .filter((line) => line !== "")
          .join("\n")
      : "";
  return [
    "# Symphony PR Review Remediation Prompt",
    "",
    `Pull request: #${pr.number} ${pr.title}`,
    `PR URL: ${pr.url}`,
    `Linked issue: ${issue ? `#${issue.number} ${issue.title}` : "(not resolved)"}`,
    `Target branch: ${targetBranch}`,
    `Working branch: ${branchName}`,
    detachedWorkspace ? `Checkout mode: detached at origin/${branchName}` : "Checkout mode: branch checkout",
    "",
    "## PR Blockers To Resolve",
    "",
    blockerList,
    "",
    "## Codex Review Threads To Address",
    "",
    threadList,
    "",
    dirtyWorkspaceSection,
    workspacePrepSection,
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
    detachedWorkspace
      ? `- This workspace is detached because the PR branch is already checked out elsewhere. Commit fixes here and push with \`git push --force-with-lease origin HEAD:${branchName}\` after verification.`
      : "- Push the prepared PR branch after verification; use `--force-with-lease` only if a rebase or history rewrite is needed.",
    "- If the PR is merge-conflicted, update the PR branch against the target branch, resolve conflicts explicitly, and rerun relevant verification.",
    "- If the PR is draft, verify whether it is complete, run required checks, and mark it ready for review only when it satisfies the issue and PR contract.",
    "- Inspect each active Codex Review thread and decide whether it needs a code, docs, test, or generated-artifact fix.",
    "- Implement only in-scope fixes for this PR and preserve unrelated local or user-authored changes.",
    "- Run the relevant verification for the files touched.",
    "- After committing and pushing fixes, revisit every listed Codex Review thread. Resolve a thread only when the pushed branch makes the feedback fixed or obsolete; otherwise leave it open and explain the remaining blocker.",
    "- Prefer resolving obsolete GitHub review conversations with the listed Thread ID. If resolution is unavailable, leave a concise thread reply with the fixing commit and verification evidence so the conversation can be closed manually.",
    "- Do not perform production writes, provider writeback, secret disclosure, destructive git, or manual merges.",
    "- Finish with changed files, verification evidence, safety notes, and thread actions taken.",
    "",
  ].join("\n");
}

async function ensureReviewRemediationDispatch({ pr, issues, dispatchConfig, workflow, skills, dryRun, prConfig = DEFAULT_PULL_REQUESTS }) {
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

  if (!dryRun) {
    run("git", ["fetch", "--prune", "origin"], { cwd: repoRoot });
  }

  if (!fs.existsSync(repoPath)) {
    if (!isSafePrBranch(pr.head_ref)) {
      return { ...result, status: "blocked", reason: `unsafe PR branch for automatic remediation: ${pr.head_ref}` };
    }
    if (!refExists(`origin/${pr.head_ref}`)) {
      return { ...result, status: "blocked", reason: `remote PR branch origin/${pr.head_ref} does not exist` };
    }
    const existingWorktree = worktreeForBranch(pr.head_ref);
    if (existingWorktree) {
      result.checkout_mode = "detached";
      result.reason = `PR branch is already checked out at ${existingWorktree.path}; remediation will use a detached workspace`;
    }
    if (dryRun) {
      return {
        ...result,
        checkout_mode: existingWorktree ? "detached" : "branch",
        reason: existingWorktree
          ? `dry-run would create detached PR remediation workspace at ${repoPath} because ${pr.head_ref} is already checked out at ${existingWorktree.path}`
          : `dry-run would create missing PR remediation workspace at ${repoPath}`,
      };
    }
    fs.mkdirSync(workspacePath, { recursive: true });
    if (existingWorktree) {
      run("git", ["worktree", "add", "--detach", repoPath, `origin/${pr.head_ref}`], { cwd: repoRoot });
    } else if (refExists(pr.head_ref)) {
      run("git", ["branch", "-f", pr.head_ref, `origin/${pr.head_ref}`], { cwd: repoRoot });
      run("git", ["worktree", "add", repoPath, pr.head_ref], { cwd: repoRoot });
    } else {
      run("git", ["worktree", "add", "-b", pr.head_ref, repoPath, `origin/${pr.head_ref}`], { cwd: repoRoot });
    }
  }
  const status = cleanStatus(repoPath);
  if (!status.clean && status.blocker !== "worktree has local edits") {
    return { ...result, status: "blocked", reason: status.blocker, dirty_files: status.dirty_files };
  }
  const workspaceState = preparedPrWorkspaceState(repoPath, pr.head_ref);
  if (!workspaceState.ok) {
    return { ...result, status: "blocked", reason: workspaceState.reason };
  }
  result.checkout_mode = workspaceState.detached ? "detached" : "branch";
  const preparation = prepareReviewRemediationWorkspace({ repoPath, logsPath, pr, targetBranch, dryRun });
  if (!preparation.ok) {
    return {
      ...result,
      status: preparation.status || "blocked",
      reason: preparation.reason,
      dirty_files: preparation.dirty_files || status.dirty_files,
      workspace_preparation: preparation.preparation || null,
    };
  }
  const preparedStatus = cleanStatus(repoPath);

  if (dryRun) {
    return {
      ...result,
      dirty_files: preparedStatus.dirty_files,
      workspace_preparation: preparation.preparation,
      reason:
        preparation.preparation?.reason ||
        (preparedStatus.clean
          ? "dry-run verified remediation prerequisites and performs no prompt, state, runner, or PR mutations"
          : "dry-run would launch remediation runner with existing local workspace edits preserved"),
    };
  }

  fs.mkdirSync(logsPath, { recursive: true });
  const prompt = renderReviewRemediationPrompt({
    pr,
    issue,
    dispatchConfig,
    workflow,
    skills,
    workspaceState,
    dirtyStatus: preparedStatus.clean
      ? { workspace_preparation: preparation.preparation }
      : { ...preparedStatus, workspace_preparation: preparation.preparation },
  });
  fs.writeFileSync(promptPath, `${prompt}\n`);
  const runner = dispatchConfig.agentRunnerCommand
    ? await runAgentRunner({
        commandLine: dispatchConfig.agentRunnerCommand,
        repoPath,
        promptPath,
        prompt,
        issue,
        branchName: pr.head_ref,
        targetBranch: result.target_branch,
        logsPath,
        timeoutMs: dispatchConfig.agentRunnerTimeoutMs,
        idleTimeoutMs: dispatchConfig.agentRunnerIdleTimeoutMs,
        codexHomeRoot: dispatchConfig.agentRunnerCodexHomeRoot,
        workspacePath,
      })
    : { status: result.status, runner_status: result.runner_status, reason: "no repo-owned agent runner command is configured yet" };
  const postRunnerHead = revParse(repoPath, "HEAD");
  const branchUpdated =
    runner.status === "succeeded" &&
    postRunnerHead.ok &&
    preparation.preparation?.local_head &&
    postRunnerHead.oid !== preparation.preparation.local_head;
  const reviewThreadResolutions =
    runner.status === "succeeded"
      ? resolveRemediatedReviewThreads({
          pr,
          config: prConfig,
          dryRun: false,
          branchUpdated,
        })
      : [];
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
    dirty_files_at_start: preparedStatus.dirty_files,
    workspace_preparation: preparation.preparation,
    review_thread_resolutions: reviewThreadResolutions,
    checkout_mode: result.checkout_mode,
    last_event:
      runner.runner_status === "completed"
        ? "review remediation agent completed"
        : runner.runner_status === "failed"
          ? "review remediation agent failed"
          : "review remediation prompt prepared; no repo-owned agent runner command is configured yet",
  };
  fs.writeFileSync(statePath, `${JSON.stringify(state, null, 2)}\n`);
  return {
    ...result,
    status: runner.status,
    runner_status: runner.runner_status,
    reason: runner.reason || (preparedStatus.clean ? preparation.preparation?.reason || "" : "launched remediation runner with existing local workspace edits preserved"),
    dirty_files: preparedStatus.dirty_files,
    workspace_preparation: preparation.preparation,
    review_thread_resolutions: reviewThreadResolutions,
    checkout_mode: result.checkout_mode,
  };
}

async function ensureDispatchWorkspace({ issue, dispatchConfig, workflow, skills, dryRun }) {
  const workspacePath = issueDispatchWorkspacePath(issue, dispatchConfig);
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
  let priorState = {};
  if (fs.existsSync(statePath)) {
    try {
      priorState = JSON.parse(fs.readFileSync(statePath, "utf8"));
    } catch (error) {
      return { ...result, status: "blocked", reason: `failed to parse existing state.json: ${error.message}` };
    }
  }
  let dirtyStatus = null;
  if (fs.existsSync(repoPath)) {
    const branch = currentBranch(repoPath);
    if (!branch.ok) {
      return { ...result, status: "blocked", reason: branch.reason };
    }
    if (branch.branch !== branchName) {
      return { ...result, status: "blocked", reason: `prepared issue workspace is on ${branch.branch || "(detached)"}, expected ${branchName}` };
    }
    const dirtyInspection = dispatchableDirtyStatus(repoPath);
    if (!dirtyInspection.ok) {
      return { ...result, status: "blocked", reason: dirtyInspection.reason, dirty_files: dirtyInspection.dirty_files };
    }
    dirtyStatus = dirtyInspection.dirtyStatus;
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

  const prompt = renderIssuePrompt({ issue, dispatchConfig, workflow, skills, dirtyStatus, workspaceState: priorState, workspacePath });
  fs.writeFileSync(promptPath, `${prompt}\n`);
  const runner = dispatchConfig.agentRunnerCommand
    ? await runAgentRunner({
        commandLine: dispatchConfig.agentRunnerCommand,
        repoPath,
        promptPath,
        prompt,
        issue,
        branchName,
        targetBranch,
        logsPath,
        timeoutMs: dispatchConfig.agentRunnerTimeoutMs,
        idleTimeoutMs: dispatchConfig.agentRunnerIdleTimeoutMs,
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
    dirty_files_at_start: dirtyStatus?.dirty_files || [],
    last_event:
      runner.runner_status === "completed"
        ? "workspace prepared and agent runner completed"
        : runner.runner_status === "failed"
          ? "workspace prepared and agent runner failed"
          : "workspace prepared; no repo-owned agent runner command is configured yet",
  };
  fs.writeFileSync(statePath, `${JSON.stringify(state, null, 2)}\n`);
  return {
    ...result,
    status: runner.status,
    runner_status: runner.runner_status,
    reason:
      runner.reason ||
      (dirtyStatus?.dirty_files?.length > 0 ? "launched issue runner with existing local workspace edits preserved" : ""),
    dirty_files: dirtyStatus?.dirty_files || [],
  };
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
          .map((line) => line.match(/^.. ?(.+)$/)?.[1] || line.slice(3))
      : [],
    has_unmerged: mergeFailureHasConflicts(cwd),
  };
}

function dispatchableDirtyStatus(cwd) {
  const status = cleanStatus(cwd);
  if (status.clean) return { ok: true, dirtyStatus: null, status };
  if (status.blocker === "worktree has local edits") return { ok: true, dirtyStatus: status, status };
  return { ok: false, reason: status.blocker, dirty_files: status.dirty_files || [], status };
}

function currentBranch(cwd) {
  const branch = run("git", ["branch", "--show-current"], { cwd, allowFailure: true });
  if (branch.failed) {
    return { ok: false, branch: "", reason: branch.stderr || "failed to inspect current branch" };
  }
  return { ok: true, branch, reason: "" };
}

function revParse(cwd, refName) {
  const output = run("git", ["rev-parse", "--verify", refName], { cwd, allowFailure: true });
  if (output.failed) return { ok: false, oid: "", reason: output.stderr || `failed to resolve ${refName}` };
  return { ok: true, oid: output, reason: "" };
}

function branchUpstream(cwd) {
  const output = run("git", ["rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"], { cwd, allowFailure: true });
  if (output.failed) return "";
  return output;
}

function branchEqualsRemote(cwd, branchName) {
  const local = run("git", ["rev-parse", "HEAD"], { cwd, allowFailure: true });
  const remote = run("git", ["rev-parse", `origin/${branchName}`], { cwd, allowFailure: true });
  if (local.failed || remote.failed) return false;
  return local === remote;
}

function gitTimestamp() {
  return new Date().toISOString().replace(/[:.]/g, "-");
}

function localOnlyCommitCount(cwd, remoteBranch) {
  const output = run("git", ["rev-list", "--count", `${remoteBranch}..HEAD`], { cwd, allowFailure: true });
  if (output.failed) return 0;
  return Number(output) || 0;
}

function preserveLocalHeadBeforeRefresh({ repoPath, prNumber }) {
  const refName = `refs/symphony/preserved/pr-${prNumber}-pre-refresh-${gitTimestamp()}`;
  const preserved = run("git", ["update-ref", refName, "HEAD"], { cwd: repoPath, allowFailure: true });
  if (preserved.failed) {
    return {
      ok: false,
      reason: preserved.stderr || preserved.stdout || "failed to preserve local-only commits before refresh",
      preserved_head_ref: refName,
    };
  }
  const oid = revParse(repoPath, refName);
  return {
    ok: true,
    reason: "local-only remediation commits were preserved before refreshing to the current PR head",
    preserved_head_ref: refName,
    preserved_head_oid: oid.ok ? oid.oid : "",
  };
}

function writeDirtyWorkspaceManifest({ repoPath, logsPath }) {
  const timestamp = gitTimestamp();
  fs.mkdirSync(logsPath, { recursive: true });
  const statusManifestPath = path.join(logsPath, `pre-refresh-dirty-status-${timestamp}.txt`);
  const status = run("git", ["status", "--porcelain=v1", "-uall"], { cwd: repoPath, allowFailure: true });
  fs.writeFileSync(statusManifestPath, status.failed ? status.stderr || status.stdout || "" : `${status}\n`);
  return statusManifestPath;
}

function preserveDirtyWorkspaceBeforeRefresh({ repoPath, logsPath, prNumber }) {
  const timestamp = gitTimestamp();
  const statusManifestPath = writeDirtyWorkspaceManifest({ repoPath, logsPath });
  const stashMessage = `symphony-pr-${prNumber}-pre-refresh-${timestamp}`;
  const stash = run("git", ["stash", "push", "--include-untracked", "-m", stashMessage], {
    cwd: repoPath,
    allowFailure: true,
  });
  if (stash.failed) {
    return {
      ok: false,
      reason: stash.stderr || stash.stdout || "failed to preserve dirty PR remediation workspace before refresh",
      status_manifest_path: statusManifestPath,
    };
  }
  const stashList = run("git", ["stash", "list", "--format=%H%x00%gd%x00%s"], { cwd: repoPath, allowFailure: true });
  const stashEntry = stashList.failed
    ? []
    : String(stashList)
        .split(/\r?\n/)
        .map((line) => line.split("\0"))
        .find(([_oid, _ref, subject]) => String(subject || "").includes(stashMessage)) || [];
  const [stashOid = "", stashRef = ""] = stashEntry;
  return {
    ok: true,
    reason: "stale local remediation edits were stashed before refreshing to the current PR head",
    stash_ref: stashRef,
    stash_oid: stashOid,
    stash_message: stashMessage,
    status_manifest_path: statusManifestPath,
  };
}

function mergeFailureHasConflicts(cwd) {
  const unmerged = run("git", ["diff", "--name-only", "--diff-filter=U"], { cwd, allowFailure: true });
  return !unmerged.failed && String(unmerged).trim().length > 0;
}

function prNeedsTargetMergePreflight(pr) {
  return (
    String(pr.merge_state || pr.mergeStateStatus || "").toUpperCase() === "DIRTY" ||
    (pr.blockers || []).some((blocker) => /^merge state DIRTY$/i.test(String(blocker)))
  );
}

function prepareReviewRemediationWorkspace({ repoPath, logsPath, pr, targetBranch, dryRun }) {
  const remoteBranch = `origin/${pr.head_ref}`;
  const remoteHead = revParse(repoPath, remoteBranch);
  if (!remoteHead.ok) {
    return { ok: false, status: "blocked", reason: `remote PR branch ${remoteBranch} does not exist` };
  }
  const localHead = revParse(repoPath, "HEAD");
  if (!localHead.ok) return { ok: false, status: "blocked", reason: localHead.reason };

  const status = cleanStatus(repoPath);
  if (!status.clean && status.blocker !== "worktree has local edits") {
    return { ok: false, status: "blocked", reason: status.blocker, dirty_files: status.dirty_files };
  }

  const upstream = branchUpstream(repoPath);
  const needsRefresh = localHead.oid !== remoteHead.oid;
  const wrongUpstream = upstream && upstream !== remoteBranch;
  const localOnlyCommits = needsRefresh ? localOnlyCommitCount(repoPath, remoteBranch) : 0;
  const needsTargetMerge = prNeedsTargetMergePreflight(pr);
  const needsCleanTargetMergeBase = needsTargetMerge && !status.clean;
  const canReuseExistingConflicts = needsTargetMerge && status.has_unmerged && !needsRefresh && !wrongUpstream;
  const preparation = {
    reason: "",
    local_head: localHead.oid,
    remote_head: remoteHead.oid,
    upstream,
    remote_branch: remoteBranch,
    local_only_commits: localOnlyCommits,
    dirty_files: status.dirty_files,
    has_unmerged: status.has_unmerged,
  };

  if (dryRun) {
    if (canReuseExistingConflicts) {
      return {
        ok: true,
        status: "would-dispatch-existing-conflicts",
        preparation: {
          ...preparation,
          reason: "dry-run would dispatch the existing unresolved merge state to the remediation worker",
          target_merge_status: "existing-conflicts-left-for-remediation-worker",
        },
      };
    }
    if (needsRefresh || wrongUpstream || needsCleanTargetMergeBase) {
      return {
        ok: true,
        status: "would-refresh",
        preparation: {
          ...preparation,
          reason: `dry-run would preserve local edits and refresh PR remediation workspace to ${remoteBranch}`,
        },
      };
    }
    if (needsTargetMerge) {
      return {
        ok: true,
        status: "would-merge-target",
        preparation: {
          ...preparation,
          reason: `dry-run would merge origin/${targetBranch} into the PR branch before remediation`,
        },
      };
    }
    return { ok: true, status: "current", preparation };
  }

  if (canReuseExistingConflicts) {
    const statusManifestPath = writeDirtyWorkspaceManifest({ repoPath, logsPath });
    return {
      ok: true,
      status: "conflicts-left-for-remediation-worker",
      dirty_files: status.dirty_files,
      preparation: {
        ...preparation,
        reason: "existing unresolved merge state left for the remediation worker",
        target_merge_status: "existing-conflicts-left-for-remediation-worker",
        status_manifest_path: statusManifestPath,
      },
    };
  }

  if (needsRefresh || wrongUpstream || needsCleanTargetMergeBase) {
    if (localOnlyCommits > 0) {
      const preservedHead = preserveLocalHeadBeforeRefresh({ repoPath, prNumber: pr.number });
      if (!preservedHead.ok) {
        return {
          ok: false,
          status: "blocked",
          reason: preservedHead.reason,
          dirty_files: status.dirty_files,
          preparation: { ...preparation, ...preservedHead },
        };
      }
      Object.assign(preparation, preservedHead);
    }
    let refreshStatus = status;
    if (refreshStatus.has_unmerged) {
      const statusManifestPath = writeDirtyWorkspaceManifest({ repoPath, logsPath });
      const aborted = run("git", ["merge", "--abort"], { cwd: repoPath, allowFailure: true });
      if (aborted.failed) {
        return {
          ok: false,
          status: "blocked",
          reason: aborted.stderr || aborted.stdout || "failed to abort stale unresolved merge before refresh",
          dirty_files: refreshStatus.dirty_files,
          preparation: {
            ...preparation,
            target_merge_status: "failed-to-abort-stale-conflicts-before-refresh",
            status_manifest_path: statusManifestPath,
          },
        };
      }
      Object.assign(preparation, {
        reason: "stale unresolved merge state was recorded and aborted before refreshing to the current PR head",
        status_manifest_path: statusManifestPath,
        target_merge_status: "stale-conflicts-aborted-before-refresh",
      });
      refreshStatus = cleanStatus(repoPath);
    }
    if (!refreshStatus.clean) {
      const preserved = preserveDirtyWorkspaceBeforeRefresh({ repoPath, logsPath, prNumber: pr.number });
      if (!preserved.ok) {
        return {
          ok: false,
          status: "blocked",
          reason: preserved.reason,
          dirty_files: refreshStatus.dirty_files,
          preparation: { ...preparation, ...preserved },
        };
      }
      Object.assign(preparation, preserved);
    }
    run("git", ["branch", "--set-upstream-to", remoteBranch], { cwd: repoPath, allowFailure: true });
    run("git", ["reset", "--hard", remoteBranch], { cwd: repoPath });
    preparation.reason = preparation.reason || `refreshed PR remediation workspace to ${remoteBranch}`;
  }

  const targetRef = `origin/${targetBranch}`;
  const targetHead = revParse(repoPath, targetRef);
  if (needsTargetMerge && targetHead.ok) {
    const merge = run("git", ["merge", "--no-edit", targetRef], { cwd: repoPath, allowFailure: true });
    if (merge.failed && !mergeFailureHasConflicts(repoPath)) {
      run("git", ["merge", "--abort"], { cwd: repoPath, allowFailure: true });
      return {
        ok: false,
        status: "blocked",
        reason: String(merge.stderr || merge.stdout || `git merge failed with status ${merge.status}`).slice(0, 2000),
        dirty_files: cleanStatus(repoPath).dirty_files,
        preparation: {
          ...preparation,
          target_merge_status: "failed-before-conflicts",
          target_merge_output: String(merge.stderr || merge.stdout || `git merge failed with status ${merge.status}`).slice(0, 2000),
        },
      };
    }
    preparation.target_merge_status = merge.failed ? "conflicts-left-for-remediation-worker" : "merged-target-before-remediation";
    preparation.target_merge_output = merge.failed
      ? String(merge.stderr || merge.stdout || `git merge failed with status ${merge.status}`).slice(0, 2000)
      : String(merge || "").slice(0, 2000);
  }

  return { ok: true, status: "prepared", preparation };
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
  const dispatchConfig = readDispatchConfig(workflow.config);
  const monitor = readMonitorConfig(workflow.config);
  const prs = listOpenPullRequests(monitor.targetBranch);
  const issues = listOpenIssues(dispatchConfig.activeLabels);
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

function syncStatusFromResults({
  prMergeResults,
  prReviewThreadRemediations,
  reviewRemediations,
  prMergeQueue,
  dispatches,
  selected,
  dispatchConfig,
}) {
  if (prMergeResults.some((result) => result.status === "merged")) return "merged_prs";
  if (reviewRemediations.some((result) => result.status === "failed")) return "review_remediation_failed";
  if (reviewRemediations.some((result) => result.status === "blocked")) return "review_remediation_blocked";
  if (reviewRemediations.some((result) => result.status === "succeeded")) return "review_remediation_complete";
  if (prReviewThreadRemediations.some((result) => result.status === "resolved")) return "review_threads_resolved";
  if (dispatches.some((dispatch) => dispatch.status === "blocked")) return "blocked";
  if (dispatches.some((dispatch) => dispatch.status === "failed")) return "agent_runner_failed";
  if (dispatches.some((dispatch) => dispatch.status === "succeeded")) return "agent_runner_complete";
  if (dispatches.some((dispatch) => dispatch.status === "prepared")) {
    return dispatchConfig.agentRunnerCommand ? "prepared" : "prepared_needs_agent_runner";
  }
  if (prMergeQueue.some((entry) => entry.status === "blocked")) return "pr_queue_blocked";
  if (prMergeQueue.some((entry) => entry.status === "waiting_for_codex_review")) return "waiting_for_codex_review";
  return selected.length === 0 ? "idle" : "dry-run";
}

async function sync({ dryRun = false, json = false, maxRuns = null } = {}) {
  const workflow = readWorkflow();
  const dispatchConfig = readDispatchConfig(workflow.config);
  const prConfig = readPullRequestConfig(workflow.config, dispatchConfig);
  const skills = discoverSkills();
  const issues = listOpenIssues(dispatchConfig.activeLabels);
  const targetBranches = [
    dispatchConfig.defaultTargetBranch,
    prConfig.targetBranch,
    ...issues.map((issue) => issueTargetBranch(issue, dispatchConfig)),
  ];
  const prs = listOpenPullRequestsForBases(targetBranches);
  const mergedPrs = listMergedPullRequestsForBases(targetBranches);
  const mergedWorkspaceStates = markMergedIssueWorkspaceStates({ issues, mergedPrs, dispatchConfig, dryRun });
  let prReviewThreads = prConfig.inspectBeforeDispatch ? fetchReviewThreads(prConfig.targetBranch) : [];
  const prReviewThreadRemediations = prConfig.inspectBeforeDispatch
    ? reconcileOutdatedPullRequestReviewThreads({ reviewThreads: prReviewThreads, config: prConfig, dryRun })
    : [];
  if (!dryRun && prReviewThreadRemediations.some((remediation) => remediation.status === "resolved")) {
    prReviewThreads = fetchReviewThreads(prConfig.targetBranch);
  }
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
  const selectedReviewRemediationCandidates = reviewRemediationCandidates.slice(
    0,
    Math.min(maxSelected, prConfig.maxReviewRemediationsPerTick),
  );
  const reviewRemediationPromises = selectedReviewRemediationCandidates.map((entry) =>
    ensureReviewRemediationDispatch({ pr: entry, issues, dispatchConfig, workflow, skills, dryRun, prConfig }),
  );
  const reviewRemediations = await Promise.all(reviewRemediationPromises);
  const remediationSlotsUsed = reviewRemediations.filter(
    (entry) => !["blocked", "skipped"].includes(entry.status),
  ).length;
  const remainingDispatchSlots = Math.max(0, maxSelected - remediationSlotsUsed);
  const readyToMergeRemaining = hasUnattemptedReadyToMergePr(prMergeQueue, prMergeResults);
  const shouldPauseDispatchForPrQueue =
    prConfig.inspectBeforeDispatch &&
    readyToMergeRemaining;
  const queue = rankedDispatchQueue(issues, prs, mergedPrs, dispatchConfig);
  const selected = shouldPauseDispatchForPrQueue ? [] : queue.filter((entry) => entry.eligible).slice(0, remainingDispatchSlots);
  const dispatchPromises = selected.map((entry) => {
    const issue = issues.find((candidate) => candidate.number === entry.number);
    return ensureDispatchWorkspace({ issue, dispatchConfig, workflow, skills, dryRun });
  });
  const dispatches = await Promise.all(dispatchPromises);

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
      review_thread_remediations: prReviewThreadRemediations,
      merge_results: prMergeResults,
      review_remediations: reviewRemediations,
      merged_workspace_states: mergedWorkspaceStates,
      dispatch_paused_for_pr_queue: shouldPauseDispatchForPrQueue,
      remaining_issue_dispatch_slots: shouldPauseDispatchForPrQueue ? 0 : remainingDispatchSlots,
    },
    issue_queue: queue.map(({ priority_score, ...entry }) => entry),
    selected_issues: selected.map(({ priority_score, ...entry }) => entry),
    dispatches,
    status: syncStatusFromResults({
      prMergeResults,
      prReviewThreadRemediations,
      reviewRemediations,
      prMergeQueue,
      dispatches,
      selected,
      dispatchConfig,
    }),
    mutations_performed: dryRun
      ? []
      : [
          ...prMergeResults
            .filter((result) => result.status === "merged")
            .map((result) => `merged PR #${result.number} with ${result.merge_method}`),
          ...prReviewThreadRemediations
            .filter((result) => result.status === "resolved")
            .map((result) => `resolved outdated Codex Review thread on PR #${result.number}`),
          ...reviewRemediations
            .filter((result) => ["prepared", "succeeded", "failed"].includes(result.status))
            .map((result) => `review remediation ${result.status} for PR #${result.number} in ${result.repo}`),
          ...reviewRemediations.flatMap((result) =>
            (result.review_thread_resolutions || [])
              .filter((resolution) => resolution.status === "resolved")
              .map((resolution) => `resolved remediated Codex Review thread on PR #${resolution.number}`),
          ),
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
      reconcileOutdatedPullRequestReviewThreads({
        reviewThreads: [{ number: 244, review_threads: [codexThread] }],
        config: DEFAULT_PULL_REQUESTS,
        dryRun: true,
      }).map((result) => ({ number: result.number, action: result.action, status: result.status })),
      [{ number: 244, action: "would-resolve-outdated-thread", status: "dry-run" }],
    );
    const remediatedThreadPr = {
      number: 245,
      unresolved_codex_review_thread_summaries: [
        {
          thread_id: "thread-2",
          author: "chatgpt-codex-connector",
          path: "scripts/symphony_runner.mjs",
          line: 1,
          url: "https://example.invalid/thread-2",
          is_outdated: false,
          body_excerpt: "Fix this after remediation.",
        },
      ],
    };
    const resolvedThreadIds = [];
    const remediatedResolutions = resolveRemediatedReviewThreads({
      pr: remediatedThreadPr,
      config: DEFAULT_PULL_REQUESTS,
      dryRun: false,
      branchUpdated: true,
      resolveThread: (threadId) => resolvedThreadIds.push(threadId),
    });
    assert.deepEqual(resolvedThreadIds, ["thread-2"]);
    assert.equal(remediatedResolutions[0].status, "resolved");
    const noBranchUpdateResolutions = resolveRemediatedReviewThreads({
      pr: remediatedThreadPr,
      config: DEFAULT_PULL_REQUESTS,
      dryRun: false,
      branchUpdated: false,
      resolveThread: () => {
        throw new Error("should not resolve unchanged branch");
      },
    });
    assert.equal(noBranchUpdateResolutions[0].status, "blocked");
    assert.match(noBranchUpdateResolutions[0].reason, /without a branch update/);
    assert.equal(
      syncStatusFromResults({
        prMergeResults: [],
        prReviewThreadRemediations: [{ status: "resolved" }],
        reviewRemediations: [{ status: "failed" }],
        prMergeQueue: [],
        dispatches: [],
        selected: [],
        dispatchConfig: { agentRunnerCommand: "codex" },
      }),
      "review_remediation_failed",
    );
    assert.equal(
      syncStatusFromResults({
        prMergeResults: [],
        prReviewThreadRemediations: [],
        reviewRemediations: [],
        prMergeQueue: [{ number: 311, status: "waiting_for_codex_review" }],
        dispatches: [{ status: "succeeded" }],
        selected: [{ number: 900264 }],
        dispatchConfig: { agentRunnerCommand: "codex" },
      }),
      "agent_runner_complete",
    );
    assert.equal(
      syncStatusFromResults({
        prMergeResults: [],
        prReviewThreadRemediations: [],
        reviewRemediations: [],
        prMergeQueue: [{ number: 311, status: "waiting_for_codex_review" }],
        dispatches: [],
        selected: [],
        dispatchConfig: { agentRunnerCommand: "codex" },
      }),
      "waiting_for_codex_review",
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
    const prepFailure = await runAgentRunner({
      commandLine: `${process.execPath} --version`,
      repoPath: tempRoot,
      promptPath: path.join(tempRoot, "prompt.md"),
      prompt: "",
      issue: { number: 1, url: "https://example.invalid/1" },
      branchName: "codex/issue-1-test",
      targetBranch: "phase-0-platform-foundation",
      logsPath: runnerLogsPath,
      timeoutMs: 1000,
      idleTimeoutMs: 1000,
      codexHomeRoot: invalidCodexHomeRoot,
      workspacePath: path.join(tempRoot, "issue-1-test"),
    });
    assert.equal(prepFailure.status, "failed");
    assert.equal(prepFailure.runner_status, "failed");
    assert.match(prepFailure.reason, /ENOTDIR|not a directory/i);
    const completedHangScript = path.join(tempRoot, "completed-hang.mjs");
    fs.writeFileSync(
      completedHangScript,
      `process.stdout.write(JSON.stringify({ type: "turn.completed" }) + "\\n"); setInterval(() => {}, 1000);\n`,
    );
    const completedHangLogs = path.join(tempRoot, "completed-hang-logs");
    const completedHang = await runAgentRunner({
      commandLine: `${process.execPath} ${completedHangScript}`,
      repoPath: tempRoot,
      promptPath: path.join(tempRoot, "prompt.md"),
      prompt: "",
      issue: { number: 2, url: "https://example.invalid/2" },
      branchName: "codex/issue-2-test",
      targetBranch: "phase-0-platform-foundation",
      logsPath: completedHangLogs,
      timeoutMs: 5000,
      idleTimeoutMs: 1000,
      codexHomeRoot: "",
      workspacePath: path.join(tempRoot, "issue-2-test"),
    });
    assert.equal(completedHang.status, "succeeded");
    assert.equal(completedHang.runner_status, "completed");
    assert.match(completedHang.reason, /reaped after idle timeout/);
    assert.match(fs.readFileSync(path.join(completedHangLogs, "agent-stdout.log"), "utf8"), /turn.completed/);
    const silentHangScript = path.join(tempRoot, "silent-hang.mjs");
    fs.writeFileSync(silentHangScript, `setInterval(() => {}, 1000);\n`);
    const silentHang = await runAgentRunner({
      commandLine: `${process.execPath} ${silentHangScript}`,
      repoPath: tempRoot,
      promptPath: path.join(tempRoot, "prompt.md"),
      prompt: "",
      issue: { number: 3, url: "https://example.invalid/3" },
      branchName: "codex/issue-3-test",
      targetBranch: "phase-0-platform-foundation",
      logsPath: path.join(tempRoot, "silent-hang-logs"),
      timeoutMs: 5000,
      idleTimeoutMs: 100,
      codexHomeRoot: "",
      workspacePath: path.join(tempRoot, "issue-3-test"),
    });
    assert.equal(silentHang.status, "failed");
    assert.equal(silentHang.runner_status, "failed");
    assert.match(silentHang.reason, /no output before idle timeout/);
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
    const unknownMergeabilityEvaluation = evaluatePullRequestForMerge({
      pr: { ...readyPr, mergeStateStatus: "UNKNOWN" },
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
    assert.equal(unknownMergeabilityEvaluation.status, "ready_to_merge");
    assert.equal(unknownMergeabilityEvaluation.bot_thumbs_up, true);
    assert.match(unknownMergeabilityEvaluation.notes.join("; "), /mergeability is temporarily unknown/);
    const inProgressReactionEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [],
        comments: [],
        reactionGroups: [
          {
            content: "EYES",
            users: { nodes: [{ login: "chatgpt-codex-connector[bot]" }] },
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(inProgressReactionEvaluation.status, "waiting_for_codex_review");
    assert.equal(inProgressReactionEvaluation.bot_eyes, true);
    assert.match(inProgressReactionEvaluation.notes.join("; "), /looking at the PR/);
    const pendingRequestAfterReviewEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [
          {
            author: { login: "chatgpt-codex-connector" },
            state: "COMMENTED",
            submittedAt: "2026-05-22T01:00:00Z",
          },
        ],
        comments: [
          {
            body: "@codex",
            createdAt: "2026-05-22T02:00:00Z",
            reactionGroups: [
              {
                content: "EYES",
                users: { nodes: [{ login: "chatgpt-codex-connector[bot]" }] },
              },
            ],
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(pendingRequestAfterReviewEvaluation.status, "waiting_for_codex_review");
    assert.equal(pendingRequestAfterReviewEvaluation.pending_codex_review_request, true);
    assert.match(pendingRequestAfterReviewEvaluation.notes.join("; "), /newer @codex review request/);
    const unacknowledgedRequestAfterReviewEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [
          {
            author: { login: "chatgpt-codex-connector" },
            state: "COMMENTED",
            submittedAt: "2026-05-22T01:00:00Z",
          },
        ],
        comments: [
          {
            body: "@codex",
            createdAt: "2026-05-22T02:00:00Z",
            reactionGroups: [],
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(unacknowledgedRequestAfterReviewEvaluation.status, "waiting_for_codex_review");
    assert.equal(unacknowledgedRequestAfterReviewEvaluation.pending_codex_review_request, true);
    assert.match(unacknowledgedRequestAfterReviewEvaluation.notes.join("; "), /bot reaction on a newer @codex review request/);
    const cleanReactionAfterReviewEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [
          {
            author: { login: "chatgpt-codex-connector" },
            state: "COMMENTED",
            submittedAt: "2026-05-22T01:00:00Z",
          },
        ],
        comments: [
          {
            body: "@codex",
            createdAt: "2026-05-22T02:00:00Z",
            reactionGroups: [
              {
                content: "THUMBS_UP",
                users: { nodes: [{ login: "chatgpt-codex-connector[bot]" }] },
              },
            ],
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(cleanReactionAfterReviewEvaluation.status, "ready_to_merge");
    assert.equal(cleanReactionAfterReviewEvaluation.pending_codex_review_request, false);
    const completedRequestEvaluation = evaluatePullRequestForMerge({
      pr: readyPr,
      reviewThreads: [{ number: 286, review_threads: [], unresolved_threads: [] }],
      signals: {
        reviews: [
          {
            author: { login: "chatgpt-codex-connector" },
            state: "COMMENTED",
            submittedAt: "2026-05-22T03:00:00Z",
          },
        ],
        comments: [
          {
            body: "@codex",
            createdAt: "2026-05-22T02:00:00Z",
            reactionGroups: [
              {
                content: "EYES",
                users: { nodes: [{ login: "chatgpt-codex-connector[bot]" }] },
              },
            ],
          },
        ],
      },
      config: prConfig,
    });
    assert.equal(completedRequestEvaluation.status, "ready_to_merge");
    assert.equal(completedRequestEvaluation.pending_codex_review_request, false);
    assert.equal(shouldRemediateBlockedPullRequest({ status: "blocked", head_ref: "codex/example", blockers: ["merge state DIRTY"], unresolved_codex_review_threads: 0, unresolved_codex_review_thread_summaries: [] }), true);
    assert.equal(shouldRemediateBlockedPullRequest({ status: "blocked", head_ref: "codex/example", blockers: ["draft PR"], unresolved_codex_review_threads: 0, unresolved_codex_review_thread_summaries: [] }), true);
    assert.equal(shouldRemediateBlockedPullRequest({ status: "blocked", head_ref: "codex/example", blockers: ["check state FAILURE"], unresolved_codex_review_threads: 0, unresolved_codex_review_thread_summaries: [] }), false);
    assert.equal(
      hasUnattemptedReadyToMergePr([{ number: 302, status: "ready_to_merge" }], [{ number: 302, status: "blocked" }]),
      false,
    );
    assert.equal(
      hasUnattemptedReadyToMergePr(
        [
          { number: 302, status: "ready_to_merge" },
          { number: 305, status: "ready_to_merge" },
        ],
        [{ number: 302, status: "blocked" }],
      ),
      true,
    );
    const dirtyRemediationPrompt = renderReviewRemediationPrompt({
      pr: {
        number: 303,
        title: "Issue #280: Expose health pause and dependency state",
        url: "https://example.invalid/pull/303",
        head_ref: "codex/issue-280-health-endpoints-reflect-pause-and-dependency-st",
        target_branch: "phase-0-platform-foundation",
        blockers: ["unresolved current Codex Review thread"],
        unresolved_codex_review_thread_summaries: [],
      },
      issue: null,
      dispatchConfig,
      workflow: { promptTemplate: "Follow repo workflow." },
      skills: [],
      dirtyStatus: { dirty_files: ["internal/web/health.go", "cmd/provisioner/main.go"] },
    });
    assert.match(dirtyRemediationPrompt, /Existing Local Workspace Edits/);
    assert.match(dirtyRemediationPrompt, /internal\/web\/health\.go/);
    assert.match(dirtyRemediationPrompt, /previous automation work for this PR branch/);
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
    fs.writeFileSync(retryStatePath, JSON.stringify({ status: "failed", attempts: "legacy-corrupt" }));
    const invalidAttemptClassification = classifyIssueForDispatch(phaseIssue, [], [], retryDispatchConfig);
    assert.equal(invalidAttemptClassification.eligible, false);
    assert.match(invalidAttemptClassification.reason, /active workspace already exists/);
    fs.writeFileSync(retryStatePath, JSON.stringify({ status: "succeeded", attempts: 1 }));
    const succeededWorkspaceClassification = classifyIssueForDispatch(phaseIssue, [], [], retryDispatchConfig);
    assert.equal(succeededWorkspaceClassification.eligible, true);
    assert.equal(succeededWorkspaceClassification.status, "eligible");
    fs.rmSync(retryStatePath, { force: true });
    assert.equal(classifyIssueForDispatch(phaseIssue, [], [], retryDispatchConfig).eligible, true);
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
    assert.deepEqual(
      mergeIssuesByNumber([
        [{ number: 1, title: "unready newer issue" }],
        [{ number: 900264, title: "older agent-ready issue" }],
        [{ number: 1, title: "unready newer issue with full payload", body: "details" }],
      ]).map((issue) => [issue.number, issue.title, issue.body || ""]),
      [
        [1, "unready newer issue with full payload", "details"],
        [900264, "older agent-ready issue", ""],
      ],
    );
    assert.ok(renderIssuePrompt({ issue: phaseIssue, dispatchConfig, workflow, skills: fakeSkills }).includes("Target branch: phase-0-platform-foundation"));
    const dirtyPrompt = renderIssuePrompt({
      issue: phaseIssue,
      dispatchConfig,
      workflow,
      skills: fakeSkills,
      dirtyStatus: { dirty_files: ["docs/product/permissions-matrix.md"] },
      workspaceState: { status: "succeeded", runner_status: "completed" },
    });
    assert.match(dirtyPrompt, /Existing Workspace State/);
    assert.match(dirtyPrompt, /Previous runner state: succeeded/);
    assert.match(dirtyPrompt, /docs\/product\/permissions-matrix\.md/);
    const dirtyIssue = { ...phaseIssue, number: 900267, title: "P0-0A-004: Dirty stale issue workspace recovery" };
    const dirtyIssueWorkspace = issueWorkspacePath(dirtyIssue, dispatchConfig);
    const dirtyIssueRepo = path.join(dirtyIssueWorkspace, "repo");
    fs.mkdirSync(dirtyIssueRepo, { recursive: true });
    run("git", ["init"], { cwd: dirtyIssueRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: dirtyIssueRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: dirtyIssueRepo });
    run("git", ["checkout", "-b", issueBranchName(dirtyIssue, dispatchConfig)], { cwd: dirtyIssueRepo });
    fs.writeFileSync(path.join(dirtyIssueRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: dirtyIssueRepo });
    run("git", ["commit", "-m", "base"], { cwd: dirtyIssueRepo });
    fs.writeFileSync(path.join(dirtyIssueRepo, "README.md"), "dirty\n");
    const dirtyIssueDispatch = await ensureDispatchWorkspace({
      issue: dirtyIssue,
      dispatchConfig,
      workflow,
      skills: fakeSkills,
      dryRun: false,
    });
    assert.equal(dirtyIssueDispatch.status, "prepared");
    assert.deepEqual(dirtyIssueDispatch.dirty_files, ["README.md"]);
    assert.match(fs.readFileSync(path.join(dirtyIssueWorkspace, "prompt.md"), "utf8"), /Existing Workspace State/);
    assert.deepEqual(JSON.parse(fs.readFileSync(path.join(dirtyIssueWorkspace, "state.json"), "utf8")).dirty_files_at_start, ["README.md"]);
    const renamedIssue = { ...phaseIssue, number: 900268, title: "P0-0A-005: New renamed issue title" };
    const renamedWorkspace = path.join(tempRoot, "issue-900268-old-issue-title");
    const renamedRepo = path.join(renamedWorkspace, "repo");
    fs.mkdirSync(renamedRepo, { recursive: true });
    run("git", ["init"], { cwd: renamedRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: renamedRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: renamedRepo });
    run("git", ["checkout", "-b", issueBranchName(renamedIssue, dispatchConfig)], { cwd: renamedRepo });
    run("git", ["commit", "--allow-empty", "-m", "old slug workspace"], { cwd: renamedRepo });
    assert.equal(classifyIssueForDispatch(renamedIssue, [], [], dispatchConfig).eligible, true);
    const renamedDispatch = await ensureDispatchWorkspace({
      issue: renamedIssue,
      dispatchConfig,
      workflow,
      skills: fakeSkills,
      dryRun: false,
    });
    assert.equal(renamedDispatch.status, "prepared");
    assert.equal(renamedDispatch.workspace, renamedWorkspace);
    assert.equal(renamedDispatch.repo, renamedRepo);
    const unhealthyIssue = { ...phaseIssue, number: 900269, title: "P0-0A-006: Unhealthy workspace blocks dispatch" };
    const unhealthyWorkspace = issueWorkspacePath(unhealthyIssue, dispatchConfig);
    fs.mkdirSync(path.join(unhealthyWorkspace, "repo"), { recursive: true });
    const unhealthyDispatch = await ensureDispatchWorkspace({
      issue: unhealthyIssue,
      dispatchConfig,
      workflow,
      skills: fakeSkills,
      dryRun: false,
    });
    assert.equal(unhealthyDispatch.status, "blocked");
    assert.notEqual(unhealthyDispatch.reason, "worktree has local edits");
    const failedStatusIssue = { ...phaseIssue, number: 900270, title: "P0-0A-007: Failed status inspection blocks dispatch" };
    const failedStatusWorkspace = issueWorkspacePath(failedStatusIssue, dispatchConfig);
    const failedStatusRepo = path.join(failedStatusWorkspace, "repo");
    fs.mkdirSync(failedStatusRepo, { recursive: true });
    run("git", ["init"], { cwd: failedStatusRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: failedStatusRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: failedStatusRepo });
    run("git", ["checkout", "-b", issueBranchName(failedStatusIssue, dispatchConfig)], { cwd: failedStatusRepo });
    fs.writeFileSync(path.join(failedStatusRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: failedStatusRepo });
    run("git", ["commit", "-m", "base"], { cwd: failedStatusRepo });
    fs.writeFileSync(path.join(failedStatusRepo, ".git", "index"), "not a git index\n");
    const failedStatusDispatch = await ensureDispatchWorkspace({
      issue: failedStatusIssue,
      dispatchConfig,
      workflow,
      skills: fakeSkills,
      dryRun: false,
    });
    assert.equal(failedStatusDispatch.status, "blocked");
    assert.notEqual(failedStatusDispatch.reason, "worktree has local edits");
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

    const dirtyMergedIssue = { ...phaseIssue, number: 900266, title: "P0-0A-003: Dirty merged workspace teardown" };
    const dirtyMergedPr = {
      ...mergedPhasePr,
      number: 312,
      title: "Fixes #900266",
      headRefName: "codex/issue-900266-dirty-merged-workspace-teardown",
    };
    const dirtyMergedWorkspace = issueWorkspacePath(dirtyMergedIssue, dispatchConfig);
    const dirtyMergedRepo = path.join(dirtyMergedWorkspace, "repo");
    fs.mkdirSync(dirtyMergedRepo, { recursive: true });
    run("git", ["init"], { cwd: dirtyMergedRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: dirtyMergedRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: dirtyMergedRepo });
    fs.writeFileSync(path.join(dirtyMergedRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: dirtyMergedRepo });
    run("git", ["commit", "-m", "base"], { cwd: dirtyMergedRepo });
    fs.writeFileSync(path.join(dirtyMergedRepo, "README.md"), "dirty\n");
    fs.writeFileSync(
      path.join(dirtyMergedWorkspace, "state.json"),
      `${JSON.stringify({ status: "succeeded", issue_number: 900266, branch: dirtyMergedPr.headRefName }, null, 2)}\n`,
    );
    const dirtyMergedDryRun = markMergedIssueWorkspaceStates({
      issues: [dirtyMergedIssue],
      mergedPrs: [dirtyMergedPr],
      dispatchConfig,
      dryRun: true,
    });
    assert.equal(dirtyMergedDryRun[0].status, "would-mark-merged");
    assert.equal(dirtyMergedDryRun[0].teardown_status, "would-preserve-dirty-worktree-and-remove");
    assert.equal(
      worktreeRemoveFailureNeedsGeneratedCachePermissionRetry(
        "warning: failed to remove .gomodcache/golang.org/x/text@v0.21.0/file.go: Permission denied",
      ),
      true,
    );
    const generatedCacheRepo = path.join(tempRoot, "generated-cache-permissions");
    const generatedCacheLeaf = path.join(generatedCacheRepo, ".gomodcache", "golang.org", "x", "text@v0.21.0");
    const generatedCacheFile = path.join(generatedCacheLeaf, "tables.go");
    fs.mkdirSync(generatedCacheLeaf, { recursive: true });
    fs.writeFileSync(generatedCacheFile, "package text\n");
    fs.chmodSync(generatedCacheFile, 0o400);
    fs.chmodSync(generatedCacheLeaf, 0o500);
    const cachePermissionFixes = normalizeGeneratedCachePermissionsForTeardown(generatedCacheRepo);
    assert.deepEqual(cachePermissionFixes, [{ path: ".gomodcache", status: "made-user-writable" }]);
    assert.ok((fs.statSync(generatedCacheFile).mode & 0o600) === 0o600);
    assert.ok((fs.statSync(generatedCacheLeaf).mode & 0o700) === 0o700);

    const oldSlugWorkspace = path.join(tempRoot, "issue-267-old-title");
    const oldSlugRepo = path.join(oldSlugWorkspace, "repo");
    fs.mkdirSync(oldSlugRepo, { recursive: true });
    run("git", ["init"], { cwd: oldSlugRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: oldSlugRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: oldSlugRepo });
    run("git", ["checkout", "-b", "codex/issue-267-review-remediation"], { cwd: oldSlugRepo });
    run("git", ["commit", "--allow-empty", "-m", "old slug workspace"], { cwd: oldSlugRepo });
    run("git", ["update-ref", "refs/remotes/origin/codex/issue-267-review-remediation", "HEAD"], { cwd: oldSlugRepo });
    assert.equal(findIssueWorkspace({ number: 267, title: "New title" }, dispatchConfig), oldSlugWorkspace);
    assert.equal(
      issueDispatchWorkspacePath({ number: 267, title: "New title" }, dispatchConfig),
      issueWorkspacePath({ number: 267, title: "New title" }, dispatchConfig),
    );
    const legacyWorkspacePrompt = renderIssuePrompt({
      issue: { number: 267, title: "New title", body: "", url: "https://example.invalid/issues/267" },
      dispatchConfig,
      workflow: reviewWorkflow,
      skills: fakeSkills,
      workspacePath: oldSlugWorkspace,
    });
    assert.match(legacyWorkspacePrompt, new RegExp(`Workspace root: ${oldSlugWorkspace.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`));
    assert.doesNotMatch(legacyWorkspacePrompt, /Workspace root: .*issue-267-new-title/);
    const slugDriftResult = await ensureReviewRemediationDispatch({
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
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: closedIssueRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: closedIssueRepo });
    run("git", ["checkout", "-b", "codex/closed-issue-review"], { cwd: closedIssueRepo });
    run("git", ["commit", "--allow-empty", "-m", "closed issue workspace"], { cwd: closedIssueRepo });
    run("git", ["update-ref", "refs/remotes/origin/codex/closed-issue-review", "HEAD"], { cwd: closedIssueRepo });
    const closedIssueResult = await ensureReviewRemediationDispatch({
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

    const stalePrRepo = path.join(tempRoot, "stale-pr-repo");
    const stalePrLogs = path.join(tempRoot, "stale-pr-logs");
    fs.mkdirSync(stalePrRepo, { recursive: true });
    run("git", ["init"], { cwd: stalePrRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: stalePrRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: stalePrRepo });
    fs.writeFileSync(path.join(stalePrRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: stalePrRepo });
    run("git", ["commit", "-m", "base"], { cwd: stalePrRepo });
    run("git", ["checkout", "-b", "codex/stale-pr"], { cwd: stalePrRepo });
    fs.writeFileSync(path.join(stalePrRepo, "remote.txt"), "remote\n");
    run("git", ["add", "remote.txt"], { cwd: stalePrRepo });
    run("git", ["commit", "-m", "remote head"], { cwd: stalePrRepo });
    const remotePrHead = revParse(stalePrRepo, "HEAD").oid;
    run("git", ["update-ref", "refs/remotes/origin/codex/stale-pr", "HEAD"], { cwd: stalePrRepo });
    run("git", ["reset", "--hard", "HEAD~1"], { cwd: stalePrRepo });
    fs.writeFileSync(path.join(stalePrRepo, "README.md"), "local dirty edit\n");
    fs.writeFileSync(path.join(stalePrRepo, "untracked.txt"), "untracked dirty edit\n");
    const stalePrep = prepareReviewRemediationWorkspace({
      repoPath: stalePrRepo,
      logsPath: stalePrLogs,
      pr: { number: 293, head_ref: "codex/stale-pr", merge_state: "CLEAN", blockers: [] },
      targetBranch: "phase-0-platform-foundation",
      dryRun: false,
    });
    assert.equal(stalePrep.ok, true);
    assert.equal(revParse(stalePrRepo, "HEAD").oid, remotePrHead);
    assert.equal(cleanStatus(stalePrRepo).clean, true);
    assert.match(stalePrep.preparation.stash_ref, /^stash@\{\d+\}$/);
    assert.match(stalePrep.preparation.stash_oid, /^[0-9a-f]{40}$/);
    assert.equal(fs.existsSync(stalePrep.preparation.status_manifest_path), true);

    const localCommitRepo = path.join(tempRoot, "local-commit-pr-repo");
    fs.mkdirSync(localCommitRepo, { recursive: true });
    run("git", ["init"], { cwd: localCommitRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: localCommitRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: localCommitRepo });
    fs.writeFileSync(path.join(localCommitRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: localCommitRepo });
    run("git", ["commit", "-m", "base"], { cwd: localCommitRepo });
    run("git", ["checkout", "-b", "codex/local-commit-pr"], { cwd: localCommitRepo });
    run("git", ["update-ref", "refs/remotes/origin/codex/local-commit-pr", "HEAD"], { cwd: localCommitRepo });
    fs.writeFileSync(path.join(localCommitRepo, "local.txt"), "local-only\n");
    run("git", ["add", "local.txt"], { cwd: localCommitRepo });
    run("git", ["commit", "-m", "local-only commit"], { cwd: localCommitRepo });
    const localOnlyHead = revParse(localCommitRepo, "HEAD").oid;
    const localCommitPrep = prepareReviewRemediationWorkspace({
      repoPath: localCommitRepo,
      logsPath: path.join(tempRoot, "local-commit-pr-logs"),
      pr: { number: 294, head_ref: "codex/local-commit-pr", merge_state: "CLEAN", blockers: [] },
      targetBranch: "phase-0-platform-foundation",
      dryRun: false,
    });
    assert.equal(localCommitPrep.ok, true);
    assert.equal(localCommitPrep.preparation.local_only_commits, 1);
    assert.equal(localCommitPrep.preparation.preserved_head_oid, localOnlyHead);
    assert.equal(revParse(localCommitRepo, localCommitPrep.preparation.preserved_head_ref).oid, localOnlyHead);
    assert.equal(revParse(localCommitRepo, "HEAD").oid, revParse(localCommitRepo, "origin/codex/local-commit-pr").oid);

    const existingConflictRepo = path.join(tempRoot, "existing-conflict-pr-repo");
    fs.mkdirSync(existingConflictRepo, { recursive: true });
    run("git", ["init"], { cwd: existingConflictRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: existingConflictRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: existingConflictRepo });
    fs.writeFileSync(path.join(existingConflictRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: existingConflictRepo });
    run("git", ["commit", "-m", "base"], { cwd: existingConflictRepo });
    run("git", ["checkout", "-b", "codex/existing-conflict-pr"], { cwd: existingConflictRepo });
    fs.writeFileSync(path.join(existingConflictRepo, "README.md"), "pr edit\n");
    run("git", ["add", "README.md"], { cwd: existingConflictRepo });
    run("git", ["commit", "-m", "pr edit"], { cwd: existingConflictRepo });
    run("git", ["update-ref", "refs/remotes/origin/codex/existing-conflict-pr", "HEAD"], { cwd: existingConflictRepo });
    run("git", ["checkout", "-b", "phase-0-platform-foundation", "HEAD~1"], { cwd: existingConflictRepo });
    fs.writeFileSync(path.join(existingConflictRepo, "README.md"), "target edit\n");
    run("git", ["add", "README.md"], { cwd: existingConflictRepo });
    run("git", ["commit", "-m", "target edit"], { cwd: existingConflictRepo });
    run("git", ["update-ref", "refs/remotes/origin/phase-0-platform-foundation", "HEAD"], { cwd: existingConflictRepo });
    run("git", ["checkout", "codex/existing-conflict-pr"], { cwd: existingConflictRepo });
    run("git", ["merge", "--no-edit", "origin/phase-0-platform-foundation"], {
      cwd: existingConflictRepo,
      allowFailure: true,
    });
    assert.equal(mergeFailureHasConflicts(existingConflictRepo), true);
    const existingConflictPrep = prepareReviewRemediationWorkspace({
      repoPath: existingConflictRepo,
      logsPath: path.join(tempRoot, "existing-conflict-pr-logs"),
      pr: { number: 295, head_ref: "codex/existing-conflict-pr", merge_state: "DIRTY", blockers: ["merge state DIRTY"] },
      targetBranch: "phase-0-platform-foundation",
      dryRun: false,
    });
    assert.equal(existingConflictPrep.ok, true);
    assert.equal(existingConflictPrep.status, "conflicts-left-for-remediation-worker");
    assert.equal(existingConflictPrep.preparation.target_merge_status, "existing-conflicts-left-for-remediation-worker");
    assert.equal(fs.existsSync(existingConflictPrep.preparation.status_manifest_path), true);

    const staleConflictRepo = path.join(tempRoot, "stale-conflict-pr-repo");
    fs.mkdirSync(staleConflictRepo, { recursive: true });
    run("git", ["init"], { cwd: staleConflictRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: staleConflictRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: staleConflictRepo });
    fs.writeFileSync(path.join(staleConflictRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: staleConflictRepo });
    run("git", ["commit", "-m", "base"], { cwd: staleConflictRepo });
    run("git", ["checkout", "-b", "codex/stale-conflict-pr"], { cwd: staleConflictRepo });
    fs.writeFileSync(path.join(staleConflictRepo, "README.md"), "stale pr edit\n");
    run("git", ["add", "README.md"], { cwd: staleConflictRepo });
    run("git", ["commit", "-m", "stale pr edit"], { cwd: staleConflictRepo });
    const staleLocalHead = revParse(staleConflictRepo, "HEAD").oid;
    fs.writeFileSync(path.join(staleConflictRepo, "README.md"), "current pr edit\n");
    run("git", ["add", "README.md"], { cwd: staleConflictRepo });
    run("git", ["commit", "-m", "current pr edit"], { cwd: staleConflictRepo });
    const staleRemoteHead = revParse(staleConflictRepo, "HEAD").oid;
    run("git", ["update-ref", "refs/remotes/origin/codex/stale-conflict-pr", "HEAD"], { cwd: staleConflictRepo });
    run("git", ["checkout", "-b", "phase-0-platform-foundation", "HEAD~2"], { cwd: staleConflictRepo });
    fs.writeFileSync(path.join(staleConflictRepo, "README.md"), "target edit\n");
    run("git", ["add", "README.md"], { cwd: staleConflictRepo });
    run("git", ["commit", "-m", "target edit"], { cwd: staleConflictRepo });
    run("git", ["update-ref", "refs/remotes/origin/phase-0-platform-foundation", "HEAD"], { cwd: staleConflictRepo });
    run("git", ["checkout", "codex/stale-conflict-pr"], { cwd: staleConflictRepo });
    run("git", ["reset", "--hard", staleLocalHead], { cwd: staleConflictRepo });
    run("git", ["merge", "--no-edit", "origin/phase-0-platform-foundation"], {
      cwd: staleConflictRepo,
      allowFailure: true,
    });
    assert.equal(mergeFailureHasConflicts(staleConflictRepo), true);
    const staleConflictPrep = prepareReviewRemediationWorkspace({
      repoPath: staleConflictRepo,
      logsPath: path.join(tempRoot, "stale-conflict-pr-logs"),
      pr: { number: 296, head_ref: "codex/stale-conflict-pr", merge_state: "DIRTY", blockers: ["merge state DIRTY"] },
      targetBranch: "phase-0-platform-foundation",
      dryRun: false,
    });
    assert.equal(staleConflictPrep.ok, true);
    assert.equal(staleConflictPrep.status, "prepared");
    assert.equal(staleConflictPrep.preparation.target_merge_status, "conflicts-left-for-remediation-worker");
    assert.equal(staleConflictPrep.preparation.local_head, staleLocalHead);
    assert.equal(revParse(staleConflictRepo, "HEAD").oid, staleRemoteHead);
    assert.equal(fs.existsSync(staleConflictPrep.preparation.status_manifest_path), true);

    const detachedPrRepo = path.join(tempRoot, "detached-pr-repo");
    fs.mkdirSync(detachedPrRepo, { recursive: true });
    run("git", ["init"], { cwd: detachedPrRepo });
    run("git", ["config", "user.email", "symphony@example.invalid"], { cwd: detachedPrRepo });
    run("git", ["config", "user.name", "Symphony Test"], { cwd: detachedPrRepo });
    fs.writeFileSync(path.join(detachedPrRepo, "README.md"), "base\n");
    run("git", ["add", "README.md"], { cwd: detachedPrRepo });
    run("git", ["commit", "-m", "base"], { cwd: detachedPrRepo });
    run("git", ["checkout", "-b", "codex/detached-pr"], { cwd: detachedPrRepo });
    run("git", ["update-ref", "refs/remotes/origin/codex/detached-pr", "HEAD"], { cwd: detachedPrRepo });
    run("git", ["checkout", "--detach", "origin/codex/detached-pr"], { cwd: detachedPrRepo });
    const detachedState = preparedPrWorkspaceState(detachedPrRepo, "codex/detached-pr");
    assert.equal(detachedState.ok, true);
    assert.equal(detachedState.detached, true);

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
      (
        await ensureReviewRemediationDispatch({
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
        })
      ).reason,
      /expected codex\/expected-branch/,
    );
    assert.equal(
      (
        await ensureReviewRemediationDispatch({
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
        })
      ).status,
      "blocked",
    );
    assert.equal(
      (
        await ensureReviewRemediationDispatch({
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
        })
      ).status,
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
