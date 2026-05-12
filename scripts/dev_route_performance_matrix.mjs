import { readFileSync } from "node:fs";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const DEFAULT_OUTPUT_DIR = "artifacts/performance";
const DEFAULT_BASE_URL = "http://localhost:5173";
const DEFAULT_TIMEOUT_MS = 15000;
const DEFAULT_REFRESH_SAMPLES = 2;
const DEFAULT_TRANSITION_BATCH_SIZE = 50;
const DEFAULT_REFRESH_BATCH_SIZE = 12;
const IT_ADMIN_PERSONA_LABEL = "IT Admin";
const REPO_ROOT = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const ROUTE_REGISTRY_PATH = path.join(REPO_ROOT, "frontend/src/lib/routeRegistry.js");

const ROUTE_VARIANTS = new Map([
  ["/search", [{ suffix: "?q=alex", label: "search alex", expectedText: "alex" }]],
  [
    "/room-moves/bulk-draft",
    [
      { suffix: "?draft_id=rm-draft-101", label: "bulk draft rm-draft-101", expectedText: "Batch Move" },
      { suffix: "?draft_id=rm-draft-103", label: "bulk draft rm-draft-103", expectedText: "Site Rollover" },
    ],
  ],
]);

const EXPECTED_TEXT_BY_PATH = new Map([
  ["/dashboard", "Dashboard"],
  ["/dashboard/it-admin", "IT Admin"],
  ["/dashboard/hr-lifecycle", "HR Lifecycle"],
  ["/dashboard/site-admin", "Site-level view"],
  ["/search", "Search"],
  ["/onboarding", "Onboarding"],
  ["/offboarding", "Offboarding"],
  ["/departing-seniors", "Departing Seniors"],
  ["/room-moves", "Room Moves"],
  ["/room-moves/bulk-draft", "Room Moves"],
  ["/phone-directory/by-person", "Phone Directory"],
  ["/phone-directory/by-department", "Phone Directory"],
  ["/phone-directory/by-room", "Phone Directory"],
  ["/data-quality", "Data Quality"],
  ["/frequent-fliers", "Frequent Fliers"],
  ["/student-data-cleanup", "Student Data Cleanup"],
  ["/reports", "Reports"],
  ["/reports/sync-transparency", "Sync Transparency"],
  ["/reports/ticketing-human-work", "Ticketing and Human Work"],
  ["/admin", "Admin"],
  ["/my-profile", "My Profile"],
]);

function nowISO() {
  return new Date().toISOString();
}

function sleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function normalizeUrl(baseUrl, route) {
  return new URL(route, baseUrl).toString();
}

function timeoutError(label, timeoutMs) {
  const error = new Error(`${label} timed out after ${timeoutMs}ms`);
  error.code = "browser_timeout";
  return error;
}

async function withTimeout(label, timeoutMs, operation) {
  let timer;
  try {
    return await Promise.race([
      operation(),
      new Promise((_, reject) => {
        timer = setTimeout(() => reject(timeoutError(label, timeoutMs)), timeoutMs);
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
}

function routeLabel(route) {
  if (route.label) {
    return route.label;
  }
  if (route.path === "/dashboard") {
    return "dashboard redirect";
  }
  return route.path.replace(/^\//, "") || "root";
}

function routeExpectedText(route) {
  return route.expectedText || EXPECTED_TEXT_BY_PATH.get(route.path) || routeLabel(route);
}

function isBrowserPipeFailure(row) {
  const message = `${row?.error ?? ""} ${row?.logs?.map((log) => log.message).join(" ") ?? ""}`;
  return /IAB pipe|Browser turn|Transport closed/i.test(message);
}

function failureClassForRow(row) {
  if (row.status === "ok") {
    return null;
  }
  if (isBrowserPipeFailure(row)) {
    return "browser_pipe_failure";
  }
  if (row.status === "timeout") {
    return "app_timeout";
  }
  if (row.status === "not_ready" && (row.finalUrl || row.title)) {
    return "harness_expectation_failure";
  }
  return "browser_error";
}

function withFailureClass(row) {
  return {
    ...row,
    failureClass: failureClassForRow(row),
  };
}

export function buildRouteTargets() {
  return buildRouteTargetsFromRoutes(loadAppRoutesFromRegistrySync());
}

function extractAppRoutesSource(source) {
  const marker = "export const APP_ROUTES =";
  const start = source.indexOf(marker);
  if (start === -1) {
    throw new Error(`Unable to find APP_ROUTES in ${ROUTE_REGISTRY_PATH}`);
  }
  const arrayStart = source.indexOf("[", start);
  if (arrayStart === -1) {
    throw new Error(`Unable to find APP_ROUTES array in ${ROUTE_REGISTRY_PATH}`);
  }

  let depth = 0;
  for (let index = arrayStart; index < source.length; index += 1) {
    const char = source[index];
    if (char === "[") {
      depth += 1;
    } else if (char === "]") {
      depth -= 1;
      if (depth === 0) {
        return source.slice(arrayStart, index + 1);
      }
    }
  }

  throw new Error(`Unable to parse APP_ROUTES array in ${ROUTE_REGISTRY_PATH}`);
}

function evaluateAppRoutesSource(arraySource) {
  return Function(`"use strict"; return (${arraySource});`)();
}

function loadAppRoutesFromRegistrySync() {
  const source = readFileSync(ROUTE_REGISTRY_PATH, "utf8");
  return evaluateAppRoutesSource(extractAppRoutesSource(source));
}

export function buildRouteTargetsFromRoutes(appRoutes) {
  const targets = [];
  for (const route of appRoutes) {
    if (route.public || route.path === "/login") {
      continue;
    }
    const base = {
      path: route.path,
      url: route.path,
      kind: route.kind,
      artboardKey: route.artboardKey ?? null,
      label: routeLabel(route),
      expectedText: routeExpectedText(route),
    };
    targets.push(base);

    const variants = ROUTE_VARIANTS.get(route.path) ?? [];
    for (const variant of variants) {
      targets.push({
        ...base,
        url: `${route.path}${variant.suffix}`,
        label: variant.label,
        expectedText: variant.expectedText || base.expectedText,
        variantOf: route.path,
      });
    }
  }
  return targets;
}

export async function loadRouteTargets() {
  const source = await readFile(ROUTE_REGISTRY_PATH, "utf8");
  return buildRouteTargetsFromRoutes(evaluateAppRoutesSource(extractAppRoutesSource(source)));
}

export function buildDirectedEdges(routes) {
  const edges = [];
  for (const from of routes) {
    for (const to of routes) {
      if (from.url === to.url) {
        continue;
      }
      edges.push({ from, to });
    }
  }
  return edges;
}

export function buildEdgeCoveragePath(routes) {
  if (routes.length < 2) {
    return routes.slice();
  }

  const adjacency = new Map(routes.map((route) => [route.url, routes.filter((next) => next.url !== route.url).map((next) => next.url)]));
  const routeByUrl = new Map(routes.map((route) => [route.url, route]));
  const stack = [routes[0].url];
  const circuit = [];

  while (stack.length > 0) {
    const current = stack[stack.length - 1];
    const next = adjacency.get(current)?.pop();
    if (next) {
      stack.push(next);
    } else {
      circuit.push(stack.pop());
    }
  }

  return circuit.reverse().map((url) => routeByUrl.get(url));
}

export function validateCoverage(routes, pathRoutes) {
  const expectedEdges = new Set(buildDirectedEdges(routes).map((edge) => `${edge.from.url}->${edge.to.url}`));
  const actualEdges = new Set();
  const duplicates = [];
  const selfTransitions = [];

  for (let index = 1; index < pathRoutes.length; index += 1) {
    const from = pathRoutes[index - 1];
    const to = pathRoutes[index];
    const key = `${from.url}->${to.url}`;
    if (from.url === to.url) {
      selfTransitions.push(key);
    }
    if (actualEdges.has(key)) {
      duplicates.push(key);
    }
    actualEdges.add(key);
  }

  const missing = [...expectedEdges].filter((key) => !actualEdges.has(key));
  const extra = [...actualEdges].filter((key) => !expectedEdges.has(key));

  return {
    routeCount: routes.length,
    expectedEdgeCount: expectedEdges.size,
    actualEdgeCount: actualEdges.size,
    pathNodeCount: pathRoutes.length,
    missing,
    extra,
    duplicates,
    selfTransitions,
    valid:
      missing.length === 0 &&
      extra.length === 0 &&
      duplicates.length === 0 &&
      selfTransitions.length === 0 &&
      actualEdges.size === expectedEdges.size,
  };
}

async function collectLogs(tab) {
  try {
    return await tab.dev.logs({ levels: ["error", "warn"], limit: 50 });
  } catch (error) {
    return [{ level: "error", message: `Unable to collect browser logs: ${error.message}` }];
  }
}

async function readReadyState(tab, route, timeoutMs) {
  const startedAt = Date.now();
  const expectedText = routeExpectedText(route);
  let snapshot = "";
  let title = "";
  let finalUrl = "";

  while (Date.now() - startedAt < timeoutMs) {
    title = (await tab.title()) || "";
    finalUrl = (await tab.url()) || "";
    snapshot = await tab.playwright.domSnapshot();
    if (snapshot.includes(expectedText) && !snapshot.includes("Preparing the DEV session.")) {
      return {
        ready: true,
        expectedText,
        title,
        finalUrl,
      };
    }
    await sleep(100);
  }

  return {
    ready: false,
    expectedText,
    title,
    finalUrl,
  };
}

async function measureNavigation(tab, baseUrl, from, to, timeoutMs) {
  const startedAt = Date.now();
  const startedAtISO = nowISO();
  try {
    await withTimeout(`navigate ${from.url} -> ${to.url}`, timeoutMs, async () => {
      await tab.goto(normalizeUrl(baseUrl, to.url));
      await tab.playwright.waitForLoadState({ state: "load", timeoutMs });
    });
    const ready = await withTimeout(`ready ${to.url}`, timeoutMs, () => readReadyState(tab, to, timeoutMs));
    return withFailureClass({
      type: "transition",
      from: from.url,
      to: to.url,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs: Date.now() - startedAt,
      finalUrl: ready.finalUrl,
      title: ready.title,
      expectedText: ready.expectedText,
      ready: ready.ready,
      status: ready.ready ? "ok" : "not_ready",
      logs: await collectLogs(tab),
    });
  } catch (error) {
    return withFailureClass({
      type: "transition",
      from: from.url,
      to: to.url,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs: Date.now() - startedAt,
      finalUrl: await tab.url().catch(() => ""),
      title: await tab.title().catch(() => ""),
      expectedText: routeExpectedText(to),
      ready: false,
      status: error.code === "browser_timeout" ? "timeout" : "error",
      error: error.message,
      logs: await collectLogs(tab),
    });
  }
}

async function measureRefresh(tab, baseUrl, route, sample, timeoutMs) {
  const startedAt = Date.now();
  const startedAtISO = nowISO();
  try {
    await withTimeout(`open ${route.url} before refresh`, timeoutMs, async () => {
      await tab.goto(normalizeUrl(baseUrl, route.url));
      await tab.playwright.waitForLoadState({ state: "load", timeoutMs });
    });
    await withTimeout(`refresh ${route.url}`, timeoutMs, async () => {
      await tab.reload();
      await tab.playwright.waitForLoadState({ state: "load", timeoutMs });
    });
    const ready = await withTimeout(`ready after refresh ${route.url}`, timeoutMs, () => readReadyState(tab, route, timeoutMs));
    return withFailureClass({
      type: "refresh",
      route: route.url,
      sample,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs: Date.now() - startedAt,
      finalUrl: ready.finalUrl,
      title: ready.title,
      expectedText: ready.expectedText,
      ready: ready.ready,
      status: ready.ready ? "ok" : "not_ready",
      logs: await collectLogs(tab),
    });
  } catch (error) {
    return withFailureClass({
      type: "refresh",
      route: route.url,
      sample,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs: Date.now() - startedAt,
      finalUrl: await tab.url().catch(() => ""),
      title: await tab.title().catch(() => ""),
      expectedText: routeExpectedText(route),
      ready: false,
      status: error.code === "browser_timeout" ? "timeout" : "error",
      error: error.message,
      logs: await collectLogs(tab),
    });
  }
}

async function ensureItAdminPersona(tab, timeoutMs) {
  const toggle = tab.playwright.getByRole("button", { name: "Expand DEV persona switcher", exact: true });
  if ((await toggle.count()) !== 1) {
    return { ensured: false, reason: "DEV persona switcher toggle not found" };
  }

  const toggleText = await toggle.innerText({ timeoutMs }).catch(() => "");
  if (toggleText.includes(IT_ADMIN_PERSONA_LABEL)) {
    return { ensured: true, alreadySelected: true };
  }

  await toggle.click({ timeoutMs });
  const select = tab.playwright.locator("#dev-persona-select");
  if ((await select.count()) !== 1) {
    return { ensured: false, reason: "DEV persona select not found" };
  }
  await select.selectOption({ label: IT_ADMIN_PERSONA_LABEL }, { timeoutMs });
  return { ensured: true, alreadySelected: false };
}

function summarizeTimings(rows, keyName) {
  const groups = new Map();
  for (const row of rows) {
    const key = row[keyName];
    const group = groups.get(key) ?? [];
    group.push(row);
    groups.set(key, group);
  }

  return [...groups.entries()].map(([key, group]) => {
    const okRows = group.filter((row) => row.status === "ok");
    const elapsed = okRows.map((row) => row.elapsedMs).sort((a, b) => a - b);
    return {
      key,
      count: group.length,
      ok: okRows.length,
      failed: group.length - okRows.length,
      minMs: elapsed[0] ?? null,
      medianMs: elapsed.length ? elapsed[Math.floor(elapsed.length / 2)] : null,
      maxMs: elapsed[elapsed.length - 1] ?? null,
    };
  });
}

function failureCounts(rows) {
  return rows.reduce((counts, row) => {
    const key = row.failureClass || "ok";
    counts[key] = (counts[key] ?? 0) + 1;
    return counts;
  }, {});
}

function lastValidTransitionBefore(transitions, failedIndex) {
  return [...transitions]
    .reverse()
    .find((row) => row.type === "transition" && row.status === "ok" && row.index < failedIndex);
}

function buildResult({
  generatedAt,
  baseUrl,
  timeoutMs,
  refreshSamples,
  persona,
  routes,
  coverage,
  startTransitionIndex,
  maxTransitions,
  refreshStartIndex,
  refreshLimit,
  includeRefreshes,
  transitions,
  refreshes,
  stoppedAtTransitionIndex,
  stopReason,
  devServerHealthy,
}) {
  const nextTransitionIndex =
    stoppedAtTransitionIndex ?? (transitions.length ? startTransitionIndex + transitions.length : startTransitionIndex);
  return {
    generatedAt,
    baseUrl,
    timeoutMs,
    refreshSamples,
    persona,
    routes,
    coverage,
    transitionRange: {
      startIndex: startTransitionIndex,
      endIndex: transitions.length ? startTransitionIndex + transitions.length - 1 : null,
      maxTransitions,
    },
    refreshRange: {
      startIndex: refreshStartIndex,
      limit: refreshLimit,
      includeRefreshes,
    },
    nextTransitionIndex,
    resumeFromTransitionIndex: stopReason === "browser_pipe_failure" ? nextTransitionIndex : null,
    stoppedAtTransitionIndex,
    stopReason,
    devServerHealthy,
    browserSessionRestartNeeded: stopReason === "browser_pipe_failure",
    transitions,
    refreshes,
    failureCounts: {
      transitions: failureCounts(transitions),
      refreshes: failureCounts(refreshes),
    },
    transitionSummaryByTarget: summarizeTimings(transitions, "to"),
    refreshSummaryByRoute: summarizeTimings(refreshes, "route"),
  };
}

async function writeMatrixArtifacts(result, outputDir, stamp) {
  const absoluteOutputDir = path.resolve(REPO_ROOT, outputDir);
  await mkdir(absoluteOutputDir, { recursive: true });
  const jsonPath = path.join(absoluteOutputDir, `dev-route-performance-${stamp}.json`);
  const markdownPath = path.join(absoluteOutputDir, `dev-route-performance-${stamp}.md`);
  await writeFile(jsonPath, `${JSON.stringify(result, null, 2)}\n`, "utf8");
  await writeFile(markdownPath, renderMarkdownSummary(result), "utf8");
  return {
    json: jsonPath,
    markdown: markdownPath,
  };
}

function cleanRowsAfterPipeFailure(rows) {
  const pipeIndex = rows.findIndex(isBrowserPipeFailure);
  if (pipeIndex === -1) {
    return rows;
  }
  return rows.slice(0, pipeIndex + 1);
}

export async function mergePerformanceArtifacts({
  outputDir = DEFAULT_OUTPUT_DIR,
  inputDir = outputDir,
} = {}) {
  const absoluteInputDir = path.resolve(REPO_ROOT, inputDir);
  const files = (await readdir(absoluteInputDir))
    .filter((name) => /^dev-route-performance-\d{4}-.*\.json$/.test(name))
    .filter((name) => !name.includes("merged"))
    .sort();

  if (files.length === 0) {
    throw new Error(`No dev-route-performance JSON artifacts found in ${absoluteInputDir}`);
  }

  const artifacts = [];
  for (const file of files) {
    const payload = JSON.parse(await readFile(path.join(absoluteInputDir, file), "utf8"));
    artifacts.push({ file, payload });
  }

  const routes = artifacts[0].payload.routes;
  const coverage = artifacts[0].payload.coverage;
  const transitionByIndex = new Map();
  const refreshes = [];
  const sourceFiles = [];

  for (const artifact of artifacts) {
    sourceFiles.push(artifact.file);
    const transitionStartIndex = artifact.payload.transitionRange?.startIndex ?? 1;
    for (const [offset, rawRow] of cleanRowsAfterPipeFailure(artifact.payload.transitions ?? []).entries()) {
      const row = {
        ...rawRow,
        index: rawRow.index ?? transitionStartIndex + offset,
        failureClass: rawRow.failureClass ?? failureClassForRow(rawRow),
      };
      transitionByIndex.set(row.index, row);
    }
    refreshes.push(
      ...cleanRowsAfterPipeFailure(artifact.payload.refreshes ?? []).map((row) => ({
        ...row,
        failureClass: row.failureClass ?? failureClassForRow(row),
      }))
    );
  }

  const transitions = [...transitionByIndex.values()].sort((a, b) => a.index - b.index);
  const firstPipeFailure = transitions.find(isBrowserPipeFailure) || refreshes.find(isBrowserPipeFailure) || null;
  const result = buildResult({
    generatedAt: nowISO(),
    baseUrl: artifacts[0].payload.baseUrl ?? DEFAULT_BASE_URL,
    timeoutMs: artifacts[0].payload.timeoutMs ?? DEFAULT_TIMEOUT_MS,
    refreshSamples: artifacts[0].payload.refreshSamples ?? DEFAULT_REFRESH_SAMPLES,
    persona: artifacts[0].payload.persona ?? null,
    routes,
    coverage,
    startTransitionIndex: transitions[0]?.index ?? 1,
    maxTransitions: coverage.expectedEdgeCount,
    refreshStartIndex: 0,
    refreshLimit: routes.length,
    includeRefreshes: true,
    transitions,
    refreshes,
    stoppedAtTransitionIndex: firstPipeFailure?.index ?? null,
    stopReason: firstPipeFailure ? "browser_pipe_failure" : "",
    devServerHealthy: null,
  });
  result.sourceFiles = sourceFiles;
  result.cleanedAfterPipeFailures = true;

  const stamp = `merged-${result.generatedAt.replace(/[:.]/g, "-")}`;
  const filesWritten = await writeMatrixArtifacts(result, outputDir, stamp);
  return {
    ...result,
    files: filesWritten,
  };
}

function renderMarkdownSummary(result) {
  const transitionFailures = result.transitions.filter((row) => row.status !== "ok");
  const refreshFailures = result.refreshes.filter((row) => row.status !== "ok");
  const firstBrowserPipeFailure =
    result.transitions.find(isBrowserPipeFailure) || result.refreshes.find(isBrowserPipeFailure) || null;
  const failedTransitionIndex = firstBrowserPipeFailure?.index ?? null;
  const lastValidEdge = failedTransitionIndex
    ? lastValidTransitionBefore(result.transitions, failedTransitionIndex)
    : result.transitions.filter((row) => row.status === "ok").at(-1);
  const slowestTransitions = result.transitions
    .filter((row) => row.status === "ok")
    .sort((a, b) => b.elapsedMs - a.elapsedMs)
    .slice(0, 10);
  const slowestRefreshes = result.refreshes
    .filter((row) => row.status === "ok")
    .sort((a, b) => b.elapsedMs - a.elapsedMs)
    .slice(0, 10);

  const lines = [
    "# DEV Route Performance Matrix",
    "",
    `Generated: ${result.generatedAt}`,
    `Base URL: ${result.baseUrl}`,
    `Routes: ${result.routes.length}`,
    `Directed transitions: ${result.coverage.actualEdgeCount}/${result.coverage.expectedEdgeCount}`,
    `Refresh samples: ${result.refreshes.length}`,
    `Coverage valid: ${result.coverage.valid ? "yes" : "no"}`,
    result.transitionRange
      ? `Transition range: ${result.transitionRange.startIndex}-${result.transitionRange.endIndex ?? "end"}`
      : null,
    result.nextTransitionIndex ? `Next transition index: ${result.nextTransitionIndex}` : null,
    result.stopReason ? `Stop reason: ${result.stopReason}` : null,
    firstBrowserPipeFailure
      ? `First Browser pipe failure: ${firstBrowserPipeFailure.from ?? firstBrowserPipeFailure.route} -> ${firstBrowserPipeFailure.to ?? "refresh"}`
      : null,
    result.resumeFromTransitionIndex ? `Resume from transition index: ${result.resumeFromTransitionIndex}` : null,
    "",
    "## Runbook",
    "",
    `Exact failed edge: ${
      firstBrowserPipeFailure
        ? `\`${firstBrowserPipeFailure.from ?? firstBrowserPipeFailure.route}\` -> \`${firstBrowserPipeFailure.to ?? "refresh"}\``
        : "none"
    }`,
    `Last valid edge: ${
      lastValidEdge ? `\`${lastValidEdge.from}\` -> \`${lastValidEdge.to}\`` : "none"
    }`,
    `Next resume index: ${result.resumeFromTransitionIndex ?? result.nextTransitionIndex ?? "n/a"}`,
    `Dev server healthy: ${result.devServerHealthy === null ? "unknown" : result.devServerHealthy ? "yes" : "no"}`,
    `Browser session restart needed: ${result.browserSessionRestartNeeded ? "yes" : "no"}`,
    "",
    "## Failure Classes",
    "",
    `Transition classes: \`${JSON.stringify(result.failureCounts?.transitions ?? {})}\``,
    `Refresh classes: \`${JSON.stringify(result.failureCounts?.refreshes ?? {})}\``,
    "",
    "## Slowest Transitions",
    "",
    "| From | To | ms | Final URL |",
    "| --- | --- | ---: | --- |",
    ...slowestTransitions.map((row) => `| \`${row.from}\` | \`${row.to}\` | ${row.elapsedMs} | \`${row.finalUrl}\` |`),
    "",
    "## Slowest Refreshes",
    "",
    "| Route | Sample | ms | Final URL |",
    "| --- | ---: | ---: | --- |",
    ...slowestRefreshes.map((row) => `| \`${row.route}\` | ${row.sample} | ${row.elapsedMs} | \`${row.finalUrl}\` |`),
    "",
    "## Failures",
    "",
    `Transition failures: ${transitionFailures.length}`,
    `Refresh failures: ${refreshFailures.length}`,
  ].filter((line) => line !== null);

  if (transitionFailures.length > 0) {
    lines.push("", "### Transition Failures", "", "| Index | From | To | Status | Class | Error |", "| ---: | --- | --- | --- | --- | --- |");
    lines.push(...transitionFailures.map((row) => `| ${row.index ?? ""} | \`${row.from}\` | \`${row.to}\` | ${row.status} | ${row.failureClass ?? ""} | ${row.error ?? ""} |`));
  }

  if (refreshFailures.length > 0) {
    lines.push("", "### Refresh Failures", "", "| Route | Sample | Status | Class | Error |", "| --- | ---: | --- | --- | --- |");
    lines.push(...refreshFailures.map((row) => `| \`${row.route}\` | ${row.sample} | ${row.status} | ${row.failureClass ?? ""} | ${row.error ?? ""} |`));
  }

  return `${lines.join("\n")}\n`;
}

export async function runDevRoutePerformanceMatrix({
  tab,
  baseUrl = DEFAULT_BASE_URL,
  outputDir = DEFAULT_OUTPUT_DIR,
  timeoutMs = DEFAULT_TIMEOUT_MS,
  refreshSamples = DEFAULT_REFRESH_SAMPLES,
  maxTransitions = DEFAULT_TRANSITION_BATCH_SIZE,
  startTransitionIndex = 1,
  includeRefreshes = false,
  refreshStartIndex = 0,
  refreshLimit = DEFAULT_REFRESH_BATCH_SIZE,
  stopOnBrowserPipeError = true,
  devServerHealthy = null,
} = {}) {
  if (!tab) {
    throw new Error("runDevRoutePerformanceMatrix requires the Browser skill tab object.");
  }

  const routes = buildRouteTargets();
  const edgePath = buildEdgeCoveragePath(routes);
  const coverage = validateCoverage(routes, edgePath);
  const persona = await ensureItAdminPersona(tab, timeoutMs);
  const generatedAt = nowISO();
  const stamp = generatedAt.replace(/[:.]/g, "-");
  const transitions = [];
  const refreshes = [];
  let stopReason = "";
  let stoppedAtTransitionIndex = null;
  let nextTransitionIndex = startTransitionIndex;

  const persist = async () => {
    const result = buildResult({
      generatedAt,
      baseUrl,
      timeoutMs,
      refreshSamples,
      persona,
      routes,
      coverage,
      startTransitionIndex,
      maxTransitions,
      refreshStartIndex,
      refreshLimit,
      includeRefreshes,
      transitions,
      refreshes,
      stoppedAtTransitionIndex,
      stopReason,
      devServerHealthy,
    });
    return writeMatrixArtifacts(result, outputDir, stamp);
  };

  for (let index = startTransitionIndex; index < edgePath.length && transitions.length < maxTransitions; index += 1) {
    const row = await measureNavigation(tab, baseUrl, edgePath[index - 1], edgePath[index], timeoutMs);
    row.index = index;
    transitions.push(row);
    nextTransitionIndex = index + 1;
    await persist();
    if (stopOnBrowserPipeError && isBrowserPipeFailure(row)) {
      stopReason = "browser_pipe_failure";
      stoppedAtTransitionIndex = index;
      await persist();
      break;
    }
  }

  if (includeRefreshes && !stopReason) {
    const refreshRoutes = routes.slice(refreshStartIndex, refreshStartIndex + refreshLimit);
    for (let routeOffset = 0; routeOffset < refreshRoutes.length && !stopReason; routeOffset += 1) {
      const route = refreshRoutes[routeOffset];
      for (let sample = 1; sample <= refreshSamples; sample += 1) {
        const row = await measureRefresh(tab, baseUrl, route, sample, timeoutMs);
        refreshes.push(row);
        await persist();
        if (stopOnBrowserPipeError && isBrowserPipeFailure(row)) {
          stopReason = "browser_pipe_failure";
          await persist();
          break;
        }
      }
    }
  }

  const result = buildResult({
    generatedAt,
    baseUrl,
    timeoutMs,
    refreshSamples,
    persona,
    routes,
    coverage,
    startTransitionIndex,
    maxTransitions,
    refreshStartIndex,
    refreshLimit,
    includeRefreshes,
    stoppedAtTransitionIndex,
    stopReason,
    devServerHealthy,
    transitions,
    refreshes,
  });
  const files = await writeMatrixArtifacts(result, outputDir, stamp);

  return {
    ...result,
    nextTransitionIndex,
    files,
  };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [, , command, maybeDir] = process.argv;
  if (command === "merge" || command === "--merge") {
    const result = await mergePerformanceArtifacts({
      inputDir: maybeDir || DEFAULT_OUTPUT_DIR,
      outputDir: DEFAULT_OUTPUT_DIR,
    });
    console.log(JSON.stringify({ files: result.files, transitions: result.transitions.length, refreshes: result.refreshes.length }, null, 2));
  } else {
    const routes = buildRouteTargets();
    const edgePath = buildEdgeCoveragePath(routes);
    const coverage = validateCoverage(routes, edgePath);
    const payload = {
      routes,
      coverage,
      defaults: {
        transitionBatchSize: DEFAULT_TRANSITION_BATCH_SIZE,
        refreshBatchSize: DEFAULT_REFRESH_BATCH_SIZE,
        includeRefreshes: false,
      },
      firstTransitions: edgePath.slice(1, 8).map((route, index) => ({
        from: edgePath[index].url,
        to: route.url,
      })),
    };
    console.log(JSON.stringify(payload, null, 2));
  }
}
