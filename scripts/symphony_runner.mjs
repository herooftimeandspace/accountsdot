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
  latestCodeAllowedDirty: ["frontend/dist/", "tmp/", ".vite/"],
  browserDefaultUrl: "http://localhost:5173/dashboard/it-admin",
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
    "  ui-monitor             Run the lock-protected ui-improvements monitor.",
    "  record-browser-results Record Browser plugin results into runner state.",
    "  test                   Run runner self-tests.",
    "",
    "Options:",
    "  --dry-run              Do not mutate refs, branches, issues, PRs, dev servers, Browser, or runner state.",
    "  --json                 Print JSON only.",
    "  --browser-results PATH JSON file for record-browser-results.",
  ].join("\n");
}

function parseArgs(argv) {
  const [command, ...rest] = argv;
  const options = { dryRun: false, json: false, browserResultsPath: "" };
  const remaining = [...rest];
  while (remaining.length > 0) {
    const arg = remaining.shift();
    if (arg === "--dry-run") {
      options.dryRun = true;
    } else if (arg === "--json") {
      options.json = true;
    } else if (arg === "--browser-results") {
      options.browserResultsPath = remaining.shift() || "";
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
  const output = run("gh", args);
  return output ? JSON.parse(output) : null;
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
    "number,title,headRefName,isDraft,mergeStateStatus,reviewDecision,headRefOid,statusCheckRollup,updatedAt",
    "--limit",
    "100",
  ]);
}

function listOpenIssues() {
  return ghJson([
    "issue",
    "list",
    "--state",
    "open",
    "--json",
    "number,title,labels,url,updatedAt",
    "--limit",
    "100",
  ]);
}

function fetchReviewThreads(baseRef) {
  const query =
    'query($owner:String!,$repo:String!,$base:String!){ repository(owner:$owner,name:$repo){ pullRequests(first:100, states:OPEN, baseRefName:$base) { nodes { number reviewThreads(first:100) { nodes { isResolved isOutdated comments(first:10){ nodes { author { login } body createdAt path line originalLine } } } } } } } }';
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
    unresolved_threads: pr.reviewThreads.nodes.filter((thread) => !thread.isResolved && !thread.isOutdated),
  }));
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

function openIssuesWithoutOpenPr(issues, prs) {
  const prText = prs.map((pr) => `${pr.number} ${pr.title} ${pr.headRefName}`).join("\n").toLowerCase();
  return issues.filter((issue) => !prText.includes(`#${issue.number}`) && !prText.includes(`issue-${issue.number}`));
}

function readMonitorConfig(workflowConfig) {
  const configured = workflowConfig.maintenance?.ui_improvements_monitor || {};
  return {
    ...DEFAULT_MONITOR,
    ...snakeToCamel(configured),
    healthUrls: configured.health_urls || DEFAULT_MONITOR.healthUrls,
    devServers: configured.dev_servers || DEFAULT_MONITOR.devServers,
    latestCodeAllowedDirty: configured.latest_code_allowed_dirty || DEFAULT_MONITOR.latestCodeAllowedDirty,
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
  return [
    {
      url: config.browserDefaultUrl,
      purpose: "Refresh and self-evaluate the latest ui-improvements dashboard after monitor validation.",
      persona: "it_admin",
      expected_visible_behavior:
        "The IT Admin dashboard loads inside the shared shell without visible error overlays, major text overlap, or broken navigation chrome.",
      screenshot_required: true,
      interaction_steps: ["Open or preserve the current local app URL", "Reload after server refresh", "Capture DOM notes and screenshot"],
      persona_setup: "npm run dev:persona -- it_admin --base-url http://localhost:5173",
      acceptance_checks: [
        "HTTP route loads successfully",
        "Shared shell/sidebar/header are visible",
        "No obvious overlapping text or controls in the first viewport",
        "No access-denied page for the IT Admin persona",
      ],
    },
  ];
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
    const reviewThreads = fetchReviewThreads(monitor.targetBranch);
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

    const evals = browserEvaluationsFor(DEFAULT_MONITOR, { blocker: "" }, [{ ok: true }, { ok: true }]);
    assert.equal(evals.length, 1);
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
