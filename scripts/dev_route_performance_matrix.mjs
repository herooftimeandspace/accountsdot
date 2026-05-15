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
const DEFAULT_MAX_BROWSER_BATCHES_PER_CALL = 1;
const DEFAULT_TRANSITION_BUDGET_WARNING_MS = 3000;
const DEFAULT_TRANSITION_BUDGET_FAILURE_MS = 7000;
const DEFAULT_REFRESH_BUDGET_WARNING_MS = 3000;
const DEFAULT_REFRESH_BUDGET_FAILURE_MS = 7000;
const IT_ADMIN_PERSONA_LABEL = "IT Admin";
const REPO_ROOT = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const ROUTE_REGISTRY_PATH = path.join(REPO_ROOT, "frontend/src/lib/routeRegistry.js");
const PHASE_TIMING_KEYS = [
  "navigationLoadMs",
  "readinessPollingMs",
  "setupNavigationLoadMs",
  "frontendSessionFetchMs",
  "frontendGeneratedArtboardImportMs",
];

// Route variants stay content-sensitive by default so query-specific pages, such as
// search results, still prove that the expected body content rendered. Use
// allowTitleAndUrlReadiness only for static generated-page variants whose mock
// body text is intentionally not durable, such as annotated room-move drafts.
const ROUTE_VARIANTS = new Map([
  ["/search", [{ suffix: "?q=alex", label: "search alex", expectedText: "Alex", expectedTitle: "Global Search" }]],
  [
    "/room-moves/bulk-draft",
    [
      {
        suffix: "?draft_id=rm-draft-101",
        label: "bulk draft rm-draft-101",
        expectedText: "Batch Move",
        expectedTitle: "Room Moves",
        allowTitleAndUrlReadiness: true,
      },
      {
        suffix: "?draft_id=rm-draft-103",
        label: "bulk draft rm-draft-103",
        expectedText: "Site Rollover",
        expectedTitle: "Room Moves",
        allowTitleAndUrlReadiness: true,
      },
    ],
  ],
]);

const EXPECTED_TEXT_BY_PATH = new Map([
  ["/dashboard", "Dashboard"],
  ["/dashboard/it-admin", "IT Admin"],
  ["/dashboard/hr-lifecycle", "HR Lifecycle"],
  ["/dashboard/site-admin", "Site Admin Dashboard"],
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
  ["/admin", "Admin"],
  ["/admin/feature-flags", "Feature Flags"],
  ["/my-profile", "My Profile"],
]);

const EXPECTED_TITLE_BY_PATH = new Map([
  ["/dashboard/it-admin", "IT Admin Dashboard"],
  ["/dashboard/hr-lifecycle", "Human Resources Dashboard"],
  ["/dashboard/site-admin", "Site Admin Dashboard"],
  ["/admin/feature-flags", "Feature Flags"],
]);

const LOADING_TEXT_MARKERS = [
  "Preparing the DEV session.",
  "Preparing the requested page.",
  "Preparing the generated page artboard.",
  "Routing to the correct page.",
];
const GENERATED_ARTBOARD_LOADING_PATTERN = /Preparing the generated .+ artboard\./;

function nowISO() {
  return new Date().toISOString();
}

function elapsedMsSince(startedAt) {
  return Date.now() - startedAt;
}

function parsePositiveInteger(value, fallback, label) {
  if (value === undefined || value === null || value === "") {
    return fallback;
  }
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed <= 0) {
    throw new Error(`${label} must be a positive integer number of milliseconds.`);
  }
  return parsed;
}

function readBudgetEnv(name) {
  return typeof process !== "undefined" ? process.env[name] : undefined;
}

export function defaultPerformanceBudgets() {
  return {
    transitions: {
      warningMs: parsePositiveInteger(
        readBudgetEnv("ROUTE_PERF_TRANSITION_WARNING_MS"),
        DEFAULT_TRANSITION_BUDGET_WARNING_MS,
        "ROUTE_PERF_TRANSITION_WARNING_MS"
      ),
      failureMs: parsePositiveInteger(
        readBudgetEnv("ROUTE_PERF_TRANSITION_FAILURE_MS"),
        DEFAULT_TRANSITION_BUDGET_FAILURE_MS,
        "ROUTE_PERF_TRANSITION_FAILURE_MS"
      ),
    },
    refreshes: {
      warningMs: parsePositiveInteger(
        readBudgetEnv("ROUTE_PERF_REFRESH_WARNING_MS"),
        DEFAULT_REFRESH_BUDGET_WARNING_MS,
        "ROUTE_PERF_REFRESH_WARNING_MS"
      ),
      failureMs: parsePositiveInteger(
        readBudgetEnv("ROUTE_PERF_REFRESH_FAILURE_MS"),
        DEFAULT_REFRESH_BUDGET_FAILURE_MS,
        "ROUTE_PERF_REFRESH_FAILURE_MS"
      ),
    },
  };
}

function normalizePerformanceBudgets(overrides = {}) {
  overrides ??= {};
  const defaults = defaultPerformanceBudgets();
  const transitions = {
    warningMs: parsePositiveInteger(
      overrides.transitions?.warningMs ?? overrides.transitionWarningMs,
      defaults.transitions.warningMs,
      "transition warning budget"
    ),
    failureMs: parsePositiveInteger(
      overrides.transitions?.failureMs ?? overrides.transitionFailureMs,
      defaults.transitions.failureMs,
      "transition failure budget"
    ),
  };
  const refreshes = {
    warningMs: parsePositiveInteger(
      overrides.refreshes?.warningMs ?? overrides.refreshWarningMs,
      defaults.refreshes.warningMs,
      "refresh warning budget"
    ),
    failureMs: parsePositiveInteger(
      overrides.refreshes?.failureMs ?? overrides.refreshFailureMs,
      defaults.refreshes.failureMs,
      "refresh failure budget"
    ),
  };

  if (transitions.warningMs >= transitions.failureMs) {
    throw new Error("Transition warning budget must be lower than transition failure budget.");
  }
  if (refreshes.warningMs >= refreshes.failureMs) {
    throw new Error("Refresh warning budget must be lower than refresh failure budget.");
  }

  return { transitions, refreshes };
}

function sleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

async function fetchWithTimeout(url, timeoutMs) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, { signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}

// The Browser batch helper calls this before touching the active tab so a
// missing APP_ENV=development API or broken Vite proxy is recorded as a DEV
// startup failure rather than misclassified as route readiness evidence.
export async function checkDevRoutePerformancePreflight({ baseUrl = DEFAULT_BASE_URL, timeoutMs = DEFAULT_TIMEOUT_MS } = {}) {
  const sessionUrl = normalizeUrl(baseUrl, "/api/v1/dev/session");
  const startedAt = Date.now();
  try {
    const response = await fetchWithTimeout(sessionUrl, timeoutMs);
    const body = await response.text();
    let payload = null;
    try {
      payload = body ? JSON.parse(body) : null;
    } catch {
      payload = null;
    }
    const healthy =
      response.ok &&
      payload?.environment === "development" &&
      Array.isArray(payload?.personas) &&
      payload.personas.length > 0;
    return {
      healthy,
      checkedAt: nowISO(),
      elapsedMs: elapsedMsSince(startedAt),
      url: sessionUrl,
      status: response.status,
      statusText: response.statusText,
      environment: payload?.environment ?? null,
      personaCount: Array.isArray(payload?.personas) ? payload.personas.length : null,
      reason: healthy ? "" : "DEV session endpoint did not return development persona metadata.",
    };
  } catch (error) {
    return {
      healthy: false,
      checkedAt: nowISO(),
      elapsedMs: elapsedMsSince(startedAt),
      url: sessionUrl,
      status: null,
      statusText: "",
      environment: null,
      personaCount: null,
      reason: error.name === "AbortError" ? `DEV session preflight timed out after ${timeoutMs}ms.` : error.message,
    };
  }
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

function routeExpectedTitle(route) {
  return route.expectedTitle || EXPECTED_TITLE_BY_PATH.get(route.path) || routeExpectedText(route);
}

function routePathMatches(finalUrl, route) {
  if (!finalUrl) {
    return false;
  }
  try {
    const parsed = new URL(finalUrl);
    return `${parsed.pathname}${parsed.search}` === route.url;
  } catch {
    return finalUrl.endsWith(route.url);
  }
}

function snapshotHasLoadingMarker(snapshot) {
  return (
    LOADING_TEXT_MARKERS.some((marker) => snapshot.includes(marker)) ||
    GENERATED_ARTBOARD_LOADING_PATTERN.test(snapshot)
  );
}

function canUseTitleAndUrlReadiness(route) {
  return !route.variantOf || route.allowTitleAndUrlReadiness === true;
}

function routePlanSignature(routes) {
  return routes.map((route) => ({
    url: route.url,
    path: route.path,
    kind: route.kind,
    expectedText: routeExpectedText(route),
    expectedTitle: routeExpectedTitle(route),
    allowTitleAndUrlReadiness: route.allowTitleAndUrlReadiness === true,
  }));
}

function routePlanSignaturesMatch(leftRoutes, rightRoutes) {
  return JSON.stringify(routePlanSignature(leftRoutes)) === JSON.stringify(routePlanSignature(rightRoutes));
}

function markdownCell(value) {
  return String(value ?? "")
    .replace(/\\/g, "\\\\")
    .replace(/\|/g, "\\|")
    .replace(/\r?\n/g, "<br>");
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
    phaseTimings: normalizePhaseTimings(row),
    failureClass: failureClassForRow(row),
  };
}

function budgetForRow(row, budgets) {
  return row.type === "refresh" ? budgets.refreshes : budgets.transitions;
}

function budgetStatusForRow(row, budget) {
  if (row.status !== "ok" || !Number.isFinite(row.elapsedMs)) {
    return "not_evaluated";
  }
  if (row.elapsedMs > budget.failureMs) {
    return "failure";
  }
  if (row.elapsedMs > budget.warningMs) {
    return "warning";
  }
  return "ok";
}

function withPerformanceBudget(row, budgets) {
  const budget = budgetForRow(row, budgets);
  const budgetStatus = budgetStatusForRow(row, budget);
  const threshold = budgetStatus === "failure" ? budget.failureMs : budgetStatus === "warning" ? budget.warningMs : null;
  return {
    ...row,
    budgetStatus,
    budgetWarningMs: budget.warningMs,
    budgetFailureMs: budget.failureMs,
    budgetExceededByMs: threshold === null ? 0 : Math.max(0, row.elapsedMs - threshold),
  };
}

function applyPerformanceBudgets(rows, budgets) {
  return rows.map((row) => withPerformanceBudget(row, budgets));
}

function budgetCounts(rows) {
  return rows.reduce(
    (counts, row) => {
      const key = row.budgetStatus || "not_evaluated";
      counts[key] = (counts[key] ?? 0) + 1;
      return counts;
    },
    { ok: 0, warning: 0, failure: 0, not_evaluated: 0 }
  );
}

function budgetRowSummary(row) {
  return {
    type: row.type,
    from: row.from,
    to: row.to,
    route: row.route,
    sample: row.sample,
    index: row.index,
    elapsedMs: row.elapsedMs,
    budgetStatus: row.budgetStatus,
    budgetWarningMs: row.budgetWarningMs,
    budgetFailureMs: row.budgetFailureMs,
    budgetExceededByMs: row.budgetExceededByMs,
    readinessSignal: row.readinessSignal,
    status: row.status,
  };
}

function buildBudgetSummary(transitions, refreshes) {
  const transitionWarnings = transitions.filter((row) => row.budgetStatus === "warning");
  const transitionFailures = transitions.filter((row) => row.budgetStatus === "failure");
  const refreshWarnings = refreshes.filter((row) => row.budgetStatus === "warning");
  const refreshFailures = refreshes.filter((row) => row.budgetStatus === "failure");
  return {
    transitions: {
      counts: budgetCounts(transitions),
      warnings: transitionWarnings.map(budgetRowSummary),
      failures: transitionFailures.map(budgetRowSummary),
    },
    refreshes: {
      counts: budgetCounts(refreshes),
      warnings: refreshWarnings.map(budgetRowSummary),
      failures: refreshFailures.map(budgetRowSummary),
    },
    totals: {
      warnings: transitionWarnings.length + refreshWarnings.length,
      failures: transitionFailures.length + refreshFailures.length,
    },
  };
}

function phaseTimingRowSummary(row, phaseKey) {
  return {
    type: row.type,
    from: row.from,
    to: row.to,
    route: row.route,
    sample: row.sample,
    index: row.index,
    phaseMs: row.phaseTimings?.[phaseKey] ?? null,
    totalMs: row.phaseTimings?.totalMs ?? row.elapsedMs ?? null,
    elapsedMs: row.elapsedMs,
    readinessSignal: row.readinessSignal,
    status: row.status,
  };
}

function medianMs(sortedValues) {
  if (!sortedValues.length) {
    return null;
  }
  const midpoint = Math.floor(sortedValues.length / 2);
  if (sortedValues.length % 2 === 1) {
    return sortedValues[midpoint];
  }
  return (sortedValues[midpoint - 1] + sortedValues[midpoint]) / 2;
}

function summarizePhaseTimingRows(rows, phaseKey, limit = 5) {
  const measuredRows = rows
    .filter((row) => row.status === "ok" && Number.isFinite(row.phaseTimings?.[phaseKey]))
    .sort((a, b) => b.phaseTimings[phaseKey] - a.phaseTimings[phaseKey]);
  const values = measuredRows.map((row) => row.phaseTimings[phaseKey]).sort((a, b) => a - b);

  return {
    sampleCount: values.length,
    minMs: values[0] ?? null,
    medianMs: medianMs(values),
    maxMs: values[values.length - 1] ?? null,
    slowestRows: measuredRows.slice(0, limit).map((row) => phaseTimingRowSummary(row, phaseKey)),
  };
}

function buildPhaseTimingSummary(transitions, refreshes) {
  const rows = [...transitions, ...refreshes];
  return Object.fromEntries(PHASE_TIMING_KEYS.map((phaseKey) => [phaseKey, summarizePhaseTimingRows(rows, phaseKey)]));
}

function normalizePhaseTimings(row) {
  const existing = row.phaseTimings ?? {};
  return {
    totalMs: numberOrNull(existing.totalMs ?? row.elapsedMs),
    navigationLoadMs: numberOrNull(existing.navigationLoadMs),
    readinessPollingMs: numberOrNull(existing.readinessPollingMs),
    setupNavigationLoadMs: numberOrNull(existing.setupNavigationLoadMs),
    frontendSessionFetchMs: numberOrNull(existing.frontendSessionFetchMs),
    frontendGeneratedArtboardImportMs: numberOrNull(existing.frontendGeneratedArtboardImportMs),
    frontendGeneratedArtboardRenderMarks: numberOrNull(existing.frontendGeneratedArtboardRenderMarks),
    routeRenderCommitMarks: numberOrNull(existing.routeRenderCommitMarks),
  };
}

function numberOrNull(value) {
  return Number.isFinite(value) ? value : null;
}

function summarizeFrontendTimingEntries(entries) {
  const summary = {
    frontendSessionFetchMs: null,
    frontendGeneratedArtboardImportMs: null,
    frontendGeneratedArtboardRenderMarks: 0,
    routeRenderCommitMarks: 0,
    measureCount: 0,
    markCount: 0,
  };

  for (const entry of entries) {
    if (entry?.type === "measure") {
      summary.measureCount += 1;
      if (entry.name === "session-fetch") {
        summary.frontendSessionFetchMs = (summary.frontendSessionFetchMs ?? 0) + (entry.elapsedMs ?? 0);
      }
      if (entry.name === "generated-artboard-import") {
        summary.frontendGeneratedArtboardImportMs =
          (summary.frontendGeneratedArtboardImportMs ?? 0) + (entry.elapsedMs ?? 0);
      }
    }
    if (entry?.type === "mark") {
      summary.markCount += 1;
      if (entry.name === "generated-artboard-render") {
        summary.frontendGeneratedArtboardRenderMarks += 1;
      }
      if (entry.name === "route-render-commit") {
        summary.routeRenderCommitMarks += 1;
      }
    }
  }

  return summary;
}

function buildPhaseTimings({
  totalMs,
  navigationLoadMs = null,
  readinessPollingMs = null,
  setupNavigationLoadMs = null,
  frontendTimings = {},
}) {
  return normalizePhaseTimings({
    elapsedMs: totalMs,
    phaseTimings: {
      totalMs,
      navigationLoadMs,
      readinessPollingMs,
      setupNavigationLoadMs,
      frontendSessionFetchMs: frontendTimings.frontendSessionFetchMs,
      frontendGeneratedArtboardImportMs: frontendTimings.frontendGeneratedArtboardImportMs,
      frontendGeneratedArtboardRenderMarks: frontendTimings.frontendGeneratedArtboardRenderMarks,
      routeRenderCommitMarks: frontendTimings.routeRenderCommitMarks,
    },
  });
}

function parseFrontendTimingLog(message) {
  const marker = "[wizard-perf]";
  const markerIndex = message.indexOf(marker);
  if (markerIndex === -1) {
    return null;
  }
  const payload = message.slice(markerIndex + marker.length).trim();
  try {
    const entry = JSON.parse(payload);
    return {
      type: entry.type,
      name: entry.name,
      elapsedMs: entry.elapsedMs ?? null,
    };
  } catch {
    return null;
  }
}

async function readFrontendTimings(tab, sinceISO) {
  try {
    const sinceMs = Date.parse(sinceISO);
    const logs = await tab.dev.logs({ levels: ["debug"], filter: "[wizard-perf]", limit: 200 });
    return summarizeFrontendTimingEntries(
      logs
        .filter((log) => {
          const loggedAt = Date.parse(log.timestamp);
          return Number.isNaN(sinceMs) || Number.isNaN(loggedAt) || loggedAt >= sinceMs;
        })
        .map((log) => parseFrontendTimingLog(log.message))
        .filter(Boolean)
    );
  } catch {
    return summarizeFrontendTimingEntries([]);
  }
}

function isMeasuredRow(row) {
  return row && !isBrowserPipeFailure(row);
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
      expectedTitle: routeExpectedTitle(route),
    };
    targets.push(base);

    const variants = ROUTE_VARIANTS.get(route.path) ?? [];
    for (const variant of variants) {
      targets.push({
        ...base,
        url: `${route.path}${variant.suffix}`,
        label: variant.label,
        expectedText: variant.expectedText || base.expectedText,
        expectedTitle: variant.expectedTitle || base.expectedTitle,
        allowTitleAndUrlReadiness: variant.allowTitleAndUrlReadiness === true,
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

function buildRefreshJobs(routes, refreshSamples) {
  return routes.flatMap((route, routeIndex) =>
    Array.from({ length: refreshSamples }, (_, sampleIndex) => ({
      route,
      routeIndex,
      sample: sampleIndex + 1,
      key: `${route.url}::${sampleIndex + 1}`,
    }))
  );
}

function refreshKeyForRow(row) {
  return `${row.route}::${row.sample}`;
}

function artifactMatchesRoutePlan(payload, routes, coverage) {
  return (
    Array.isArray(payload?.routes) &&
    payload.routes.length === routes.length &&
    payload.coverage?.expectedEdgeCount === coverage.expectedEdgeCount &&
    routePlanSignaturesMatch(payload.routes, routes)
  );
}

async function loadCurrentPerformanceArtifacts(inputDir, routes, coverage) {
  const absoluteInputDir = path.resolve(REPO_ROOT, inputDir);
  let files;
  try {
    files = (await readdir(absoluteInputDir))
      .filter((name) => /^dev-route-performance-\d{4}-.*\.json$/.test(name))
      .sort();
  } catch (error) {
    if (error.code === "ENOENT") {
      return [];
    }
    throw error;
  }

  const artifacts = [];
  for (const file of files) {
    const payload = JSON.parse(await readFile(path.join(absoluteInputDir, file), "utf8"));
    if (artifactMatchesRoutePlan(payload, routes, coverage)) {
      artifacts.push({ file, payload });
    }
  }
  return artifacts;
}

function completedTransitionIndexesFromArtifacts(artifacts) {
  const completed = new Set();
  for (const artifact of artifacts) {
    const transitionStartIndex = artifact.payload.transitionRange?.startIndex ?? 1;
    for (const [offset, rawRow] of (artifact.payload.transitions ?? []).entries()) {
      const row = {
        ...rawRow,
        index: rawRow.index ?? transitionStartIndex + offset,
      };
      if (isMeasuredRow(row)) {
        completed.add(row.index);
      }
    }
  }
  return completed;
}

function completedRefreshKeysFromArtifacts(artifacts, refreshJobs) {
  const completed = new Set();
  const plannedRefreshKeys = new Set(refreshJobs.map((job) => job.key));
  for (const artifact of artifacts) {
    for (const row of artifact.payload.refreshes ?? []) {
      const key = refreshKeyForRow(row);
      if (isMeasuredRow(row) && plannedRefreshKeys.has(key)) {
        completed.add(key);
      }
    }
  }
  return completed;
}

function nextTransitionBatch(completedIndexes, edgeCount, transitionBatchSize) {
  let startIndex = null;
  for (let index = 1; index <= edgeCount; index += 1) {
    if (!completedIndexes.has(index)) {
      startIndex = index;
      break;
    }
  }
  if (startIndex === null) {
    return null;
  }

  let maxTransitions = 0;
  for (let index = startIndex; index <= edgeCount && maxTransitions < transitionBatchSize; index += 1) {
    if (completedIndexes.has(index)) {
      break;
    }
    maxTransitions += 1;
  }
  return { startTransitionIndex: startIndex, maxTransitions };
}

function nextRefreshBatch(completedRefreshKeys, refreshJobs, refreshBatchSize) {
  const jobs = [];
  for (const job of refreshJobs) {
    if (!completedRefreshKeys.has(job.key)) {
      jobs.push(job);
      if (jobs.length >= refreshBatchSize) {
        break;
      }
    }
  }
  return jobs.length ? jobs : null;
}

// Used by the CLI batch-plan command and the Browser helper to inspect local
// row-level artifacts before selecting the next bounded route-performance batch.
export async function planDevRoutePerformanceBatches({
  outputDir = DEFAULT_OUTPUT_DIR,
  refreshSamples = DEFAULT_REFRESH_SAMPLES,
  transitionBatchSize = DEFAULT_TRANSITION_BATCH_SIZE,
  refreshBatchSize = DEFAULT_REFRESH_BATCH_SIZE,
  includeTransitions = true,
  includeRefreshes = true,
} = {}) {
  const routes = buildRouteTargets();
  const edgePath = buildEdgeCoveragePath(routes);
  const coverage = validateCoverage(routes, edgePath);
  const artifacts = await loadCurrentPerformanceArtifacts(outputDir, routes, coverage);
  const completedTransitions = completedTransitionIndexesFromArtifacts(artifacts);
  const refreshJobs = buildRefreshJobs(routes, refreshSamples);
  const completedRefreshes = completedRefreshKeysFromArtifacts(artifacts, refreshJobs);
  const nextTransition = includeTransitions
    ? nextTransitionBatch(completedTransitions, coverage.expectedEdgeCount, transitionBatchSize)
    : null;
  const nextRefresh = includeRefreshes ? nextRefreshBatch(completedRefreshes, refreshJobs, refreshBatchSize) : null;

  return {
    outputDir,
    artifactCount: artifacts.length,
    routeCount: routes.length,
    transitionBatchSize,
    refreshBatchSize,
    refreshSamples,
    transitions: {
      completed: completedTransitions.size,
      total: coverage.expectedEdgeCount,
      remaining: coverage.expectedEdgeCount - completedTransitions.size,
      next: nextTransition,
    },
    refreshes: {
      completed: completedRefreshes.size,
      total: refreshJobs.length,
      remaining: refreshJobs.length - completedRefreshes.size,
      next: nextRefresh
        ? {
            count: nextRefresh.length,
            first: { route: nextRefresh[0].route.url, sample: nextRefresh[0].sample },
            last: { route: nextRefresh.at(-1).route.url, sample: nextRefresh.at(-1).sample },
          }
        : null,
    },
    complete: (!includeTransitions || !nextTransition) && (!includeRefreshes || !nextRefresh),
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
  const expectedTitle = routeExpectedTitle(route);
  let snapshot = "";
  let title = "";
  let finalUrl = "";
  let readinessSignal = "";
  let pollCount = 0;

  while (Date.now() - startedAt < timeoutMs) {
    pollCount += 1;
    title = (await tab.title()) || "";
    finalUrl = (await tab.url()) || "";
    snapshot = await tab.playwright.domSnapshot();
    const hasLoadingMarker = snapshotHasLoadingMarker(snapshot);
    if (snapshot.includes(expectedText) && !hasLoadingMarker) {
      readinessSignal = "expected_text";
      return {
        ready: true,
        readinessSignal,
        expectedText,
        expectedTitle,
        title,
        finalUrl,
        pollingMs: elapsedMsSince(startedAt),
        pollCount,
      };
    }
    if (
      canUseTitleAndUrlReadiness(route) &&
      routePathMatches(finalUrl, route) &&
      title.includes(expectedTitle) &&
      !hasLoadingMarker
    ) {
      readinessSignal = "title_and_url";
      return {
        ready: true,
        readinessSignal,
        expectedText,
        expectedTitle,
        title,
        finalUrl,
        pollingMs: elapsedMsSince(startedAt),
        pollCount,
      };
    }
    await sleep(100);
  }

  return {
    ready: false,
    readinessSignal,
    expectedText,
    expectedTitle,
    title,
    finalUrl,
    pollingMs: elapsedMsSince(startedAt),
    pollCount,
  };
}

async function measureNavigation(tab, baseUrl, from, to, timeoutMs) {
  const startedAt = Date.now();
  const startedAtISO = nowISO();
  let navigationLoadMs = null;
  let readinessPollingMs = null;
  let frontendTimings = summarizeFrontendTimingEntries([]);
  try {
    const navigationStartedAt = Date.now();
    await withTimeout(`navigate ${from.url} -> ${to.url}`, timeoutMs, async () => {
      await tab.goto(normalizeUrl(baseUrl, to.url));
      await tab.playwright.waitForLoadState({ state: "load", timeoutMs });
    });
    navigationLoadMs = elapsedMsSince(navigationStartedAt);
    const ready = await withTimeout(`ready ${to.url}`, timeoutMs, () => readReadyState(tab, to, timeoutMs));
    readinessPollingMs = ready.pollingMs;
    frontendTimings = await readFrontendTimings(tab, startedAtISO);
    const elapsedMs = elapsedMsSince(startedAt);
    return withFailureClass({
      type: "transition",
      from: from.url,
      to: to.url,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs,
      phaseTimings: buildPhaseTimings({
        totalMs: elapsedMs,
        navigationLoadMs,
        readinessPollingMs,
        frontendTimings,
      }),
      finalUrl: ready.finalUrl,
      title: ready.title,
      expectedText: ready.expectedText,
      expectedTitle: ready.expectedTitle,
      readinessSignal: ready.readinessSignal,
      readinessPollCount: ready.pollCount,
      ready: ready.ready,
      status: ready.ready ? "ok" : "not_ready",
      logs: await collectLogs(tab),
    });
  } catch (error) {
    frontendTimings = await readFrontendTimings(tab, startedAtISO);
    const elapsedMs = elapsedMsSince(startedAt);
    return withFailureClass({
      type: "transition",
      from: from.url,
      to: to.url,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs,
      phaseTimings: buildPhaseTimings({
        totalMs: elapsedMs,
        navigationLoadMs,
        readinessPollingMs,
        frontendTimings,
      }),
      finalUrl: await tab.url().catch(() => ""),
      title: await tab.title().catch(() => ""),
      expectedText: routeExpectedText(to),
      expectedTitle: routeExpectedTitle(to),
      readinessSignal: "",
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
  let setupNavigationLoadMs = null;
  let navigationLoadMs = null;
  let readinessPollingMs = null;
  let frontendTimings = summarizeFrontendTimingEntries([]);
  try {
    const setupStartedAt = Date.now();
    await withTimeout(`open ${route.url} before refresh`, timeoutMs, async () => {
      await tab.goto(normalizeUrl(baseUrl, route.url));
      await tab.playwright.waitForLoadState({ state: "load", timeoutMs });
    });
    setupNavigationLoadMs = elapsedMsSince(setupStartedAt);
    const navigationStartedAt = Date.now();
    await withTimeout(`refresh ${route.url}`, timeoutMs, async () => {
      await tab.reload();
      await tab.playwright.waitForLoadState({ state: "load", timeoutMs });
    });
    navigationLoadMs = elapsedMsSince(navigationStartedAt);
    const ready = await withTimeout(`ready after refresh ${route.url}`, timeoutMs, () => readReadyState(tab, route, timeoutMs));
    readinessPollingMs = ready.pollingMs;
    frontendTimings = await readFrontendTimings(tab, startedAtISO);
    const elapsedMs = elapsedMsSince(startedAt);
    return withFailureClass({
      type: "refresh",
      route: route.url,
      sample,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs,
      phaseTimings: buildPhaseTimings({
        totalMs: elapsedMs,
        navigationLoadMs,
        readinessPollingMs,
        setupNavigationLoadMs,
        frontendTimings,
      }),
      finalUrl: ready.finalUrl,
      title: ready.title,
      expectedText: ready.expectedText,
      expectedTitle: ready.expectedTitle,
      readinessSignal: ready.readinessSignal,
      readinessPollCount: ready.pollCount,
      ready: ready.ready,
      status: ready.ready ? "ok" : "not_ready",
      logs: await collectLogs(tab),
    });
  } catch (error) {
    frontendTimings = await readFrontendTimings(tab, startedAtISO);
    const elapsedMs = elapsedMsSince(startedAt);
    return withFailureClass({
      type: "refresh",
      route: route.url,
      sample,
      startedAt: startedAtISO,
      endedAt: nowISO(),
      elapsedMs,
      phaseTimings: buildPhaseTimings({
        totalMs: elapsedMs,
        navigationLoadMs,
        readinessPollingMs,
        setupNavigationLoadMs,
        frontendTimings,
      }),
      finalUrl: await tab.url().catch(() => ""),
      title: await tab.title().catch(() => ""),
      expectedText: routeExpectedText(route),
      expectedTitle: routeExpectedTitle(route),
      readinessSignal: "",
      ready: false,
      status: error.code === "browser_timeout" ? "timeout" : "error",
      error: error.message,
      logs: await collectLogs(tab),
    });
  }
}

async function runRefreshJobBatch({
  tab,
  baseUrl,
  outputDir,
  timeoutMs,
  refreshSamples,
  refreshJobs,
  routes,
  coverage,
  persona,
  stopOnBrowserPipeError,
  devServerHealthy,
  devServerPreflight = null,
}) {
  const generatedAt = nowISO();
  const stamp = generatedAt.replace(/[:.]/g, "-");
  const refreshes = [];
  let stopReason = "";

  const persist = async () => {
    const result = buildResult({
      generatedAt,
      baseUrl,
      timeoutMs,
      refreshSamples,
      persona,
      routes,
      coverage,
      startTransitionIndex: 1,
      maxTransitions: 0,
      refreshStartIndex: refreshJobs[0]?.routeIndex ?? 0,
      refreshLimit: refreshJobs.length,
      includeRefreshes: true,
      transitions: [],
      refreshes,
      stoppedAtTransitionIndex: null,
      stopReason,
      devServerHealthy,
      devServerPreflight,
    });
    result.refreshJobRange = refreshJobs.length
      ? {
          first: { route: refreshJobs[0].route.url, sample: refreshJobs[0].sample },
          last: { route: refreshJobs.at(-1).route.url, sample: refreshJobs.at(-1).sample },
          count: refreshJobs.length,
        }
      : null;
    return writeMatrixArtifacts(result, outputDir, stamp);
  };

  for (const job of refreshJobs) {
    const row = await measureRefresh(tab, baseUrl, job.route, job.sample, timeoutMs);
    row.routeIndex = job.routeIndex;
    refreshes.push(row);
    await persist();
    if (stopOnBrowserPipeError && isBrowserPipeFailure(row)) {
      stopReason = "browser_pipe_failure";
      await persist();
      break;
    }
  }

  const files = await persist();
  return {
    phase: "refresh",
    stopReason,
    refreshes,
    files,
  };
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
      medianMs: medianMs(elapsed),
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
  budgets,
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
  devServerPreflight = null,
}) {
  const performanceBudgets = normalizePerformanceBudgets(budgets);
  const normalizedTransitions = transitions.map((row) => ({
    ...row,
    phaseTimings: normalizePhaseTimings(row),
  }));
  const normalizedRefreshes = refreshes.map((row) => ({
    ...row,
    phaseTimings: normalizePhaseTimings(row),
  }));
  const budgetedTransitions = applyPerformanceBudgets(normalizedTransitions, performanceBudgets);
  const budgetedRefreshes = applyPerformanceBudgets(normalizedRefreshes, performanceBudgets);
  const nextTransitionIndex =
    stoppedAtTransitionIndex ?? (budgetedTransitions.length ? startTransitionIndex + budgetedTransitions.length : startTransitionIndex);
  return {
    generatedAt,
    baseUrl,
    timeoutMs,
    refreshSamples,
    performanceBudgets,
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
    devServerPreflight,
    browserSessionRestartNeeded: stopReason === "browser_pipe_failure",
    transitions: budgetedTransitions,
    refreshes: budgetedRefreshes,
    failureCounts: {
      transitions: failureCounts(budgetedTransitions),
      refreshes: failureCounts(budgetedRefreshes),
    },
    budgetSummary: buildBudgetSummary(budgetedTransitions, budgetedRefreshes),
    phaseTimingSummary: buildPhaseTimingSummary(budgetedTransitions, budgetedRefreshes),
    transitionSummaryByTarget: summarizeTimings(budgetedTransitions, "to"),
    refreshSummaryByRoute: summarizeTimings(budgetedRefreshes, "route"),
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

function validateTransitionRowsCoverage(routes, transitions) {
  const expectedEdges = new Set(buildDirectedEdges(routes).map((edge) => `${edge.from.url}->${edge.to.url}`));
  const actualEdges = new Set();
  const duplicates = [];
  const selfTransitions = [];
  const invalidRows = [];

  for (const row of transitions) {
    if (!row.from || !row.to) {
      invalidRows.push(row.index ?? null);
      continue;
    }
    const key = `${row.from}->${row.to}`;
    if (row.from === row.to) {
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
    missing,
    extra,
    duplicates,
    selfTransitions,
    invalidRows,
    valid:
      missing.length === 0 &&
      extra.length === 0 &&
      duplicates.length === 0 &&
      selfTransitions.length === 0 &&
      invalidRows.length === 0 &&
      actualEdges.size === expectedEdges.size,
  };
}

function missingTransitionIndexes(transitions, expectedEdgeCount) {
  const indexes = new Set(
    transitions
      .map((row) => row.index)
      .filter((index) => Number.isInteger(index) && index >= 1 && index <= expectedEdgeCount)
  );
  const missing = [];
  for (let index = 1; index <= expectedEdgeCount; index += 1) {
    if (!indexes.has(index)) {
      missing.push(index);
    }
  }
  return missing;
}

function uniqueSortedNumbers(values) {
  return [...new Set(values.filter((value) => Number.isInteger(value)))].sort((a, b) => a - b);
}

function buildRefreshCoverage(routes, refreshSamples, refreshes) {
  const expectedKeys = new Set(buildRefreshJobs(routes, refreshSamples).map((job) => job.key));
  const actualKeys = new Set();
  const duplicates = [];
  const invalidRows = [];

  for (const row of refreshes) {
    const key = refreshKeyForRow(row);
    if (!row.route || !Number.isInteger(row.sample)) {
      invalidRows.push(key);
      continue;
    }
    if (actualKeys.has(key)) {
      duplicates.push(key);
    }
    actualKeys.add(key);
  }

  const missing = [...expectedKeys].filter((key) => !actualKeys.has(key));
  const extra = [...actualKeys].filter((key) => !expectedKeys.has(key));

  return {
    expectedCount: expectedKeys.size,
    actualCount: actualKeys.size,
    missing,
    extra,
    duplicates: [...new Set(duplicates)].sort(),
    invalidRows,
    valid: missing.length === 0 && extra.length === 0 && duplicates.length === 0 && invalidRows.length === 0,
  };
}

function buildQualityGate(result, { duplicateTransitionIndexes = [] } = {}) {
  const transitionFailures = result.transitions.filter((row) => row.status !== "ok");
  const refreshFailures = result.refreshes.filter((row) => row.status !== "ok");
  const failedRows = [...transitionFailures, ...refreshFailures];
  const appTimeoutRows = failedRows.filter((row) => row.failureClass === "app_timeout");
  const browserTransportRows = failedRows.filter((row) => row.failureClass === "browser_pipe_failure");
  const expectedTransitionCount = result.transitionCoverage?.expectedEdgeCount ?? result.coverage.expectedEdgeCount;
  const missingIndexes = missingTransitionIndexes(result.transitions, expectedTransitionCount);
  const duplicateIndexes = uniqueSortedNumbers(duplicateTransitionIndexes);
  const routeCountMismatch =
    result.currentRoutePlan?.routeCount !== undefined && result.currentRoutePlan.routeCount !== result.routes.length;
  const directedEdgeCountMismatch =
    result.currentRoutePlan?.expectedEdgeCount !== undefined &&
    result.currentRoutePlan.expectedEdgeCount !== expectedTransitionCount;
  const refreshCoverage = buildRefreshCoverage(result.routes, result.refreshSamples, result.refreshes);
  const staleCurrentRoutePlan = result.currentRoutePlan?.differsFromMergedArtifacts === true;
  const invalidDirectedEdgeCoverage = result.transitionCoverage?.valid === false || result.coverage?.valid === false;
  const devServerHealthFailed = result.devServerHealthy === false;

  const blockers = {
    transitionFailures: transitionFailures.length,
    refreshFailures: refreshFailures.length,
    browserTransportFailures: browserTransportRows.length,
    appTimeoutRows: appTimeoutRows.length,
    staleCurrentRoutePlan,
    missingTransitionIndexes: missingIndexes.length,
    duplicateTransitionIndexes: duplicateIndexes.length,
    missingRefreshSamples: refreshCoverage.missing.length,
    duplicateRefreshSamples: refreshCoverage.duplicates.length,
    extraRefreshSamples: refreshCoverage.extra.length,
    invalidRefreshRows: refreshCoverage.invalidRows.length,
    invalidDirectedEdgeCoverage,
    routeCountMismatch,
    directedEdgeCountMismatch,
    devServerHealthFailed,
  };

  const passed = Object.values(blockers).every((value) => value === 0 || value === false);
  const messages = [
    blockers.transitionFailures ? `transition failures=${blockers.transitionFailures}` : null,
    blockers.refreshFailures ? `refresh failures=${blockers.refreshFailures}` : null,
    blockers.browserTransportFailures ? `Browser transport failures=${blockers.browserTransportFailures}` : null,
    blockers.appTimeoutRows ? `app timeout rows=${blockers.appTimeoutRows}` : null,
    blockers.staleCurrentRoutePlan ? "current route plan differs from merged artifacts" : null,
    blockers.missingTransitionIndexes ? `missing transition indexes=${blockers.missingTransitionIndexes}` : null,
    blockers.duplicateTransitionIndexes ? `duplicate transition indexes=${blockers.duplicateTransitionIndexes}` : null,
    blockers.missingRefreshSamples ? `missing refresh samples=${blockers.missingRefreshSamples}` : null,
    blockers.duplicateRefreshSamples ? `duplicate refresh samples=${blockers.duplicateRefreshSamples}` : null,
    blockers.extraRefreshSamples ? `extra refresh samples=${blockers.extraRefreshSamples}` : null,
    blockers.invalidRefreshRows ? `invalid refresh rows=${blockers.invalidRefreshRows}` : null,
    blockers.invalidDirectedEdgeCoverage ? "directed-edge coverage is invalid" : null,
    blockers.routeCountMismatch ? "route-count mismatch with current route plan" : null,
    blockers.directedEdgeCountMismatch ? "directed-transition count mismatch with current route plan" : null,
    blockers.devServerHealthFailed ? "DEV route preflight reported unhealthy server startup" : null,
  ].filter(Boolean);

  return {
    mode: "strict",
    passed,
    blockers,
    expectedTransitionCount,
    actualTransitionCount: result.transitions.length,
    expectedRefreshCount: refreshCoverage.expectedCount,
    actualRefreshCount: refreshCoverage.actualCount,
    refreshCoverage,
    missingTransitionIndexes: missingIndexes,
    duplicateTransitionIndexes: duplicateIndexes,
    messages,
  };
}

function formatQualityGateFailure(result) {
  const messages = result.qualityGate?.messages ?? ["unknown route performance quality-gate failure"];
  const artifactLine = result.files
    ? `Artifacts: markdown=${result.files.markdown}; json=${result.files.json}`
    : "Artifacts: unavailable";
  return [
    "Route performance strict merge quality gate failed.",
    `Blocking counts: ${messages.join("; ")}.`,
    artifactLine,
  ].join("\n");
}

function parseValueArg(args, index, name) {
  const arg = args[index];
  if (arg.includes("=")) {
    return { value: arg.slice(arg.indexOf("=") + 1), consumed: 1 };
  }
  const value = args[index + 1];
  if (!value || value.startsWith("--")) {
    throw new Error(`${name} requires a millisecond value.`);
  }
  return { value, consumed: 2 };
}

function isValueFlag(arg, name) {
  return arg === name || arg.startsWith(`${name}=`);
}

function parseMergeCliArgs(args) {
  let strict = false;
  let inputDir = DEFAULT_OUTPUT_DIR;
  let budgetStrict = false;
  const budgets = {};
  let hasBudgetOverrides = false;

  for (let index = 0; index < args.length; ) {
    const arg = args[index];
    if (arg === "--strict") {
      strict = true;
      index += 1;
      continue;
    }
    if (arg === "--no-strict") {
      strict = false;
      index += 1;
      continue;
    }
    if (arg === "--budget-strict") {
      budgetStrict = true;
      index += 1;
      continue;
    }
    if (arg === "--no-budget-strict") {
      budgetStrict = false;
      index += 1;
      continue;
    }
    if (isValueFlag(arg, "--transition-warning-ms")) {
      const parsed = parseValueArg(args, index, "--transition-warning-ms");
      budgets.transitionWarningMs = parsed.value;
      hasBudgetOverrides = true;
      index += parsed.consumed;
      continue;
    }
    if (isValueFlag(arg, "--transition-failure-ms")) {
      const parsed = parseValueArg(args, index, "--transition-failure-ms");
      budgets.transitionFailureMs = parsed.value;
      hasBudgetOverrides = true;
      index += parsed.consumed;
      continue;
    }
    if (isValueFlag(arg, "--refresh-warning-ms")) {
      const parsed = parseValueArg(args, index, "--refresh-warning-ms");
      budgets.refreshWarningMs = parsed.value;
      hasBudgetOverrides = true;
      index += parsed.consumed;
      continue;
    }
    if (isValueFlag(arg, "--refresh-failure-ms")) {
      const parsed = parseValueArg(args, index, "--refresh-failure-ms");
      budgets.refreshFailureMs = parsed.value;
      index += parsed.consumed;
      hasBudgetOverrides = true;
      continue;
    }
    if (arg.startsWith("--")) {
      throw new Error(`Unknown merge option: ${arg}`);
    }
    inputDir = arg;
    index += 1;
  }

  return {
    inputDir,
    strict,
    budgetStrict,
    budgets: hasBudgetOverrides ? normalizePerformanceBudgets(budgets) : undefined,
  };
}

function budgetGatePassed(result) {
  return (result.budgetSummary?.totals?.failures ?? 0) === 0;
}

function formatBudgetGateFailure(result) {
  const failures = result.budgetSummary?.totals?.failures ?? 0;
  const transitionFailures = result.budgetSummary?.transitions?.counts?.failure ?? 0;
  const refreshFailures = result.budgetSummary?.refreshes?.counts?.failure ?? 0;
  return [
    "Route performance budget gate failed.",
    `Budget failures: total=${failures}; transitions=${transitionFailures}; refreshes=${refreshFailures}.`,
    result.files ? `Artifacts: markdown=${result.files.markdown}; json=${result.files.json}` : "Artifacts: unavailable",
  ].join("\n");
}

function rowLabel(row) {
  if (row.type === "transition") {
    return `${row.from} -> ${row.to}`;
  }
  return `${row.route} refresh ${row.sample}`;
}

function slowestRowsByPhase(rows, phaseKey, limit = 5) {
  return rows
    .filter((row) => row.status === "ok" && Number.isFinite(row.phaseTimings?.[phaseKey]))
    .sort((a, b) => b.phaseTimings[phaseKey] - a.phaseTimings[phaseKey])
    .slice(0, limit);
}

function renderPhaseRows(rows, phaseKey) {
  if (rows.length === 0) {
    return [`| ${markdownCell(phaseKey)} | none | n/a | n/a |`];
  }
  return rows.map((row) => {
    const timing = row.phaseTimings ?? {};
    return `| ${markdownCell(phaseKey)} | \`${markdownCell(rowLabel(row))}\` | ${timing[phaseKey]} | ${timing.totalMs ?? row.elapsedMs ?? ""} |`;
  });
}

export async function mergePerformanceArtifacts({
  outputDir = DEFAULT_OUTPUT_DIR,
  inputDir = outputDir,
  budgets,
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
  const seenTransitionIndexes = new Set();
  const duplicateTransitionIndexes = [];
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
      if (seenTransitionIndexes.has(row.index)) {
        duplicateTransitionIndexes.push(row.index);
      }
      seenTransitionIndexes.add(row.index);
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
  const transitionCoverage = validateTransitionRowsCoverage(routes, transitions);
  const firstPipeFailure = transitions.find(isBrowserPipeFailure) || refreshes.find(isBrowserPipeFailure) || null;
  const devServerHealthValues = artifacts
    .map((artifact) => artifact.payload.devServerHealthy)
    .filter((value) => value !== null && value !== undefined);
  const mergedDevServerHealthy = devServerHealthValues.length === 0
    ? null
    : devServerHealthValues.every(Boolean);
  const mergedDevServerPreflight =
    artifacts.find((artifact) => artifact.payload.devServerPreflight)?.payload.devServerPreflight ?? null;
  const result = buildResult({
    generatedAt: nowISO(),
    baseUrl: artifacts[0].payload.baseUrl ?? DEFAULT_BASE_URL,
    timeoutMs: artifacts[0].payload.timeoutMs ?? DEFAULT_TIMEOUT_MS,
    refreshSamples: artifacts[0].payload.refreshSamples ?? DEFAULT_REFRESH_SAMPLES,
    budgets: budgets ?? artifacts[0].payload.performanceBudgets,
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
    devServerHealthy: mergedDevServerHealthy,
    devServerPreflight: mergedDevServerPreflight,
  });
  result.sourceFiles = sourceFiles;
  result.cleanedAfterPipeFailures = true;
  result.transitionCoverage = transitionCoverage;
  const currentRoutes = buildRouteTargets();
  const currentCoverage = validateCoverage(currentRoutes, buildEdgeCoveragePath(currentRoutes));
  result.currentRoutePlan = {
    routeCount: currentRoutes.length,
    expectedEdgeCount: currentCoverage.expectedEdgeCount,
    differsFromMergedArtifacts:
      currentRoutes.length !== routes.length ||
      currentCoverage.expectedEdgeCount !== coverage.expectedEdgeCount ||
      !routePlanSignaturesMatch(currentRoutes, routes),
  };
  result.qualityGate = buildQualityGate(result, { duplicateTransitionIndexes });

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
  const transitionBudgetWarnings = result.transitions.filter((row) => row.budgetStatus === "warning");
  const transitionBudgetFailures = result.transitions.filter((row) => row.budgetStatus === "failure");
  const refreshBudgetWarnings = result.refreshes.filter((row) => row.budgetStatus === "warning");
  const refreshBudgetFailures = result.refreshes.filter((row) => row.budgetStatus === "failure");
  const budgetWarnings = [...transitionBudgetWarnings, ...refreshBudgetWarnings];
  const budgetFailures = [...transitionBudgetFailures, ...refreshBudgetFailures];
  const appTimeoutFailures = [...transitionFailures, ...refreshFailures].filter(
    (row) => row.failureClass === "app_timeout"
  );
  const browserTransportFailures = [...transitionFailures, ...refreshFailures].filter(
    (row) => row.failureClass === "browser_pipe_failure"
  );
  const okRows = [...result.transitions, ...result.refreshes].filter((row) => row.status === "ok");
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
    result.transitionCoverage
      ? `Merged directed transitions: ${result.transitionCoverage.actualEdgeCount}/${result.transitionCoverage.expectedEdgeCount}`
      : null,
    `Refresh samples: ${result.refreshes.length}`,
    `Coverage valid: ${result.coverage.valid ? "yes" : "no"}`,
    result.transitionCoverage ? `Merged transition coverage valid: ${result.transitionCoverage.valid ? "yes" : "no"}` : null,
    `Transition budget: warning > ${result.performanceBudgets.transitions.warningMs} ms; failure > ${result.performanceBudgets.transitions.failureMs} ms`,
    `Refresh budget: warning > ${result.performanceBudgets.refreshes.warningMs} ms; failure > ${result.performanceBudgets.refreshes.failureMs} ms`,
    result.currentRoutePlan
      ? `Current route plan: ${result.currentRoutePlan.routeCount} routes / ${result.currentRoutePlan.expectedEdgeCount} directed transitions`
      : null,
    result.currentRoutePlan?.differsFromMergedArtifacts
      ? "Current route plan differs from merged artifact route coverage; run a fresh matrix instead of resuming this merged artifact."
      : null,
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
    result.devServerPreflight
      ? `DEV preflight: ${result.devServerPreflight.status ?? "no status"} ${result.devServerPreflight.reason || "ok"}`
      : null,
    `Browser session restart needed: ${result.browserSessionRestartNeeded ? "yes" : "no"}`,
    "",
    "## Failure Classes",
    "",
    `Transition classes: \`${JSON.stringify(result.failureCounts?.transitions ?? {})}\``,
    `Refresh classes: \`${JSON.stringify(result.failureCounts?.refreshes ?? {})}\``,
    `App timeout rows: ${appTimeoutFailures.length}`,
    `Browser transport rows: ${browserTransportFailures.length}`,
    "",
    "## Strict Quality Gate",
    "",
    result.qualityGate
      ? `Status: ${result.qualityGate.passed ? "pass" : "fail"}`
      : "Status: not evaluated",
    result.qualityGate
      ? `Blocking counts: \`${JSON.stringify(result.qualityGate.blockers)}\``
      : null,
    result.qualityGate?.missingTransitionIndexes?.length
      ? `Missing transition indexes: ${result.qualityGate.missingTransitionIndexes.join(", ")}`
      : null,
    result.qualityGate?.duplicateTransitionIndexes?.length
      ? `Duplicate transition indexes: ${result.qualityGate.duplicateTransitionIndexes.join(", ")}`
      : null,
    result.qualityGate?.refreshCoverage?.missing?.length
      ? `Missing refresh samples: ${result.qualityGate.refreshCoverage.missing.join(", ")}`
      : null,
    result.qualityGate?.refreshCoverage?.duplicates?.length
      ? `Duplicate refresh samples: ${result.qualityGate.refreshCoverage.duplicates.join(", ")}`
      : null,
    "",
    "## Performance Budgets",
    "",
    `Budget warning rows: ${budgetWarnings.length}`,
    `Budget failure rows: ${budgetFailures.length}`,
    `Transition budget counts: \`${JSON.stringify(result.budgetSummary?.transitions?.counts ?? {})}\``,
    `Refresh budget counts: \`${JSON.stringify(result.budgetSummary?.refreshes?.counts ?? {})}\``,
    "",
    "## Slowest Transitions",
    "",
    "| From | To | ms | Ready Signal | Final URL |",
    "| --- | --- | ---: | --- | --- |",
    ...slowestTransitions.map((row) => `| \`${markdownCell(row.from)}\` | \`${markdownCell(row.to)}\` | ${row.elapsedMs} | ${markdownCell(row.readinessSignal)} | \`${markdownCell(row.finalUrl)}\` |`),
    "",
    "## Slowest Phase Timings",
    "",
    "| Phase | Row | Phase ms | Total ms |",
    "| --- | --- | ---: | ---: |",
    ...renderPhaseRows(slowestRowsByPhase(okRows, "navigationLoadMs"), "navigationLoadMs"),
    ...renderPhaseRows(slowestRowsByPhase(okRows, "readinessPollingMs"), "readinessPollingMs"),
    ...renderPhaseRows(slowestRowsByPhase(okRows, "frontendSessionFetchMs"), "frontendSessionFetchMs"),
    ...renderPhaseRows(slowestRowsByPhase(okRows, "frontendGeneratedArtboardImportMs"), "frontendGeneratedArtboardImportMs"),
    "",
    "## Slowest Refreshes",
    "",
    "| Route | Sample | ms | Ready Signal | Final URL |",
    "| --- | ---: | ---: | --- | --- |",
    ...slowestRefreshes.map((row) => `| \`${markdownCell(row.route)}\` | ${row.sample} | ${row.elapsedMs} | ${markdownCell(row.readinessSignal)} | \`${markdownCell(row.finalUrl)}\` |`),
    "",
    "## Failures",
    "",
    `Transition failures: ${transitionFailures.length}`,
    `Refresh failures: ${refreshFailures.length}`,
  ].filter((line) => line !== null);

  if (transitionFailures.length > 0) {
    lines.push("", "### Transition Failures", "", "| Index | From | To | Status | Class | Expected | Final Title | Error |", "| ---: | --- | --- | --- | --- | --- | --- | --- |");
    lines.push(...transitionFailures.map((row) => `| ${row.index ?? ""} | \`${markdownCell(row.from)}\` | \`${markdownCell(row.to)}\` | ${markdownCell(row.status)} | ${markdownCell(row.failureClass)} | ${markdownCell(row.expectedText)} | ${markdownCell(row.title)} | ${markdownCell(row.error)} |`));
  }

  if (refreshFailures.length > 0) {
    lines.push("", "### Refresh Failures", "", "| Route | Sample | Status | Class | Expected | Final Title | Error |", "| --- | ---: | --- | --- | --- | --- | --- |");
    lines.push(...refreshFailures.map((row) => `| \`${markdownCell(row.route)}\` | ${row.sample} | ${markdownCell(row.status)} | ${markdownCell(row.failureClass)} | ${markdownCell(row.expectedText)} | ${markdownCell(row.title)} | ${markdownCell(row.error)} |`));
  }

  if (budgetWarnings.length > 0) {
    lines.push("", "### Budget Warnings", "", "| Type | Route / Edge | Index / Sample | ms | Warning ms | Failure ms | Ready Signal |", "| --- | --- | ---: | ---: | ---: | ---: | --- |");
    lines.push(
      ...budgetWarnings.map((row) => {
        const route = row.type === "transition" ? `${row.from} -> ${row.to}` : row.route;
        const index = row.type === "transition" ? row.index ?? "" : row.sample ?? "";
        return `| ${markdownCell(row.type)} | \`${markdownCell(route)}\` | ${index} | ${row.elapsedMs} | ${row.budgetWarningMs} | ${row.budgetFailureMs} | ${markdownCell(row.readinessSignal)} |`;
      })
    );
  }

  if (budgetFailures.length > 0) {
    lines.push("", "### Budget Failures", "", "| Type | Route / Edge | Index / Sample | ms | Warning ms | Failure ms | Ready Signal |", "| --- | --- | ---: | ---: | ---: | ---: | --- |");
    lines.push(
      ...budgetFailures.map((row) => {
        const route = row.type === "transition" ? `${row.from} -> ${row.to}` : row.route;
        const index = row.type === "transition" ? row.index ?? "" : row.sample ?? "";
        return `| ${markdownCell(row.type)} | \`${markdownCell(route)}\` | ${index} | ${row.elapsedMs} | ${row.budgetWarningMs} | ${row.budgetFailureMs} | ${markdownCell(row.readinessSignal)} |`;
      })
    );
  }

  return `${lines.join("\n")}\n`;
}

export async function runDevRoutePerformanceMatrix({
  tab,
  baseUrl = DEFAULT_BASE_URL,
  outputDir = DEFAULT_OUTPUT_DIR,
  timeoutMs = DEFAULT_TIMEOUT_MS,
  refreshSamples = DEFAULT_REFRESH_SAMPLES,
  budgets,
  maxTransitions = DEFAULT_TRANSITION_BATCH_SIZE,
  startTransitionIndex = 1,
  includeRefreshes = false,
  refreshStartIndex = 0,
  refreshLimit = DEFAULT_REFRESH_BATCH_SIZE,
  stopOnBrowserPipeError = true,
  devServerHealthy = null,
  devServerPreflight = null,
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

  const persist = async () => {
    const result = buildResult({
      generatedAt,
      baseUrl,
      timeoutMs,
      refreshSamples,
      budgets,
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
      devServerPreflight,
    });
    return writeMatrixArtifacts(result, outputDir, stamp);
  };

  for (let index = startTransitionIndex; index < edgePath.length && transitions.length < maxTransitions; index += 1) {
    const row = await measureNavigation(tab, baseUrl, edgePath[index - 1], edgePath[index], timeoutMs);
    row.index = index;
    transitions.push(row);
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
    budgets,
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
    devServerPreflight,
    transitions,
    refreshes,
  });
  const files = await writeMatrixArtifacts(result, outputDir, stamp);

  return {
    ...result,
    files,
  };
}

// Browser-skill entrypoint for issue-sized evidence collection. Callers provide
// the active Browser tab; this helper resumes from matching local artifacts,
// runs bounded transition batches before refresh batches, and relies on the
// lower-level row writer so each measured row survives Browser interruptions.
export async function runDevRoutePerformanceBatches({
  tab,
  baseUrl = DEFAULT_BASE_URL,
  outputDir = DEFAULT_OUTPUT_DIR,
  timeoutMs = DEFAULT_TIMEOUT_MS,
  refreshSamples = DEFAULT_REFRESH_SAMPLES,
  transitionBatchSize = DEFAULT_TRANSITION_BATCH_SIZE,
  refreshBatchSize = DEFAULT_REFRESH_BATCH_SIZE,
  includeTransitions = true,
  includeRefreshes = true,
  maxBatches = DEFAULT_MAX_BROWSER_BATCHES_PER_CALL,
  stopOnBrowserPipeError = true,
  devServerHealthy = null,
  devServerPreflight = true,
} = {}) {
  if (!tab) {
    throw new Error("runDevRoutePerformanceBatches requires the Browser skill tab object.");
  }

  const routes = buildRouteTargets();
  const edgePath = buildEdgeCoveragePath(routes);
  const coverage = validateCoverage(routes, edgePath);
  const refreshJobs = buildRefreshJobs(routes, refreshSamples);
  const preflight =
    devServerPreflight === false
      ? null
      : await checkDevRoutePerformancePreflight({
          baseUrl,
          timeoutMs,
        });

  if (preflight && !preflight.healthy) {
    const generatedAt = nowISO();
    const stamp = generatedAt.replace(/[:.]/g, "-");
    const result = buildResult({
      generatedAt,
      baseUrl,
      timeoutMs,
      refreshSamples,
      persona: { ensured: false, reason: "DEV route preflight failed before Browser persona selection" },
      routes,
      coverage,
      startTransitionIndex: 1,
      maxTransitions: 0,
      refreshStartIndex: 0,
      refreshLimit: 0,
      includeRefreshes,
      transitions: [],
      refreshes: [],
      stoppedAtTransitionIndex: null,
      stopReason: "dev_server_unhealthy",
      devServerHealthy: false,
      devServerPreflight: preflight,
    });
    const files = await writeMatrixArtifacts(result, outputDir, stamp);
    const plan = await planDevRoutePerformanceBatches({
      outputDir,
      refreshSamples,
      transitionBatchSize,
      refreshBatchSize,
      includeTransitions,
      includeRefreshes,
    });
    return {
      ...plan,
      maxBatches,
      batches: [
        {
          phase: "preflight",
          measured: 0,
          stopReason: "dev_server_unhealthy",
          devServerPreflight: preflight,
          files,
        },
      ],
      stopped: true,
      devServerHealthy: false,
      devServerPreflight: preflight,
    };
  }

  const persona = await ensureItAdminPersona(tab, timeoutMs);
  const batches = [];
  let stopped = false;

  for (let batchNumber = 0; batchNumber < maxBatches && !stopped; batchNumber += 1) {
    const artifacts = await loadCurrentPerformanceArtifacts(outputDir, routes, coverage);
    const completedTransitions = completedTransitionIndexesFromArtifacts(artifacts);
    const completedRefreshes = completedRefreshKeysFromArtifacts(artifacts, refreshJobs);
    const transitionBatch = includeTransitions
      ? nextTransitionBatch(completedTransitions, coverage.expectedEdgeCount, transitionBatchSize)
      : null;

    if (transitionBatch) {
      const result = await runDevRoutePerformanceMatrix({
        tab,
        baseUrl,
        outputDir,
        timeoutMs,
        refreshSamples,
        maxTransitions: transitionBatch.maxTransitions,
        startTransitionIndex: transitionBatch.startTransitionIndex,
        includeRefreshes: false,
        stopOnBrowserPipeError,
        devServerHealthy: preflight ? true : devServerHealthy,
        devServerPreflight: preflight,
      });
      batches.push({
        phase: "transition",
        startTransitionIndex: transitionBatch.startTransitionIndex,
        maxTransitions: transitionBatch.maxTransitions,
        measured: result.transitions.length,
        stopReason: result.stopReason,
        files: result.files,
      });
      stopped = Boolean(result.stopReason);
      continue;
    }

    const refreshBatch = includeRefreshes ? nextRefreshBatch(completedRefreshes, refreshJobs, refreshBatchSize) : null;
    if (refreshBatch) {
      const result = await runRefreshJobBatch({
        tab,
        baseUrl,
        outputDir,
        timeoutMs,
        refreshSamples,
        refreshJobs: refreshBatch,
        routes,
        coverage,
        persona,
        stopOnBrowserPipeError,
        devServerHealthy: preflight ? true : devServerHealthy,
        devServerPreflight: preflight,
      });
      batches.push({
        phase: "refresh",
        first: refreshBatch[0] ? { route: refreshBatch[0].route.url, sample: refreshBatch[0].sample } : null,
        last: refreshBatch.at(-1) ? { route: refreshBatch.at(-1).route.url, sample: refreshBatch.at(-1).sample } : null,
        planned: refreshBatch.length,
        measured: result.refreshes.length,
        stopReason: result.stopReason,
        files: result.files,
      });
      stopped = Boolean(result.stopReason);
      continue;
    }

    break;
  }

  const plan = await planDevRoutePerformanceBatches({
    outputDir,
    refreshSamples,
    transitionBatchSize,
    refreshBatchSize,
    includeTransitions,
    includeRefreshes,
  });

  return {
    ...plan,
    maxBatches,
    batches,
    stopped,
    devServerHealthy: preflight ? true : devServerHealthy,
    devServerPreflight: preflight,
  };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [, , command, ...args] = process.argv;
  if (command === "merge" || command === "--merge") {
    const options = parseMergeCliArgs(args);
    const result = await mergePerformanceArtifacts({
      inputDir: options.inputDir,
      outputDir: DEFAULT_OUTPUT_DIR,
      budgets: options.budgets,
    });
    console.log(
      JSON.stringify(
        {
          files: result.files,
          transitions: result.transitions.length,
          refreshes: result.refreshes.length,
          qualityGate: result.qualityGate,
          budgetSummary: result.budgetSummary,
        },
        null,
        2
      )
    );
    if (options.strict && !result.qualityGate.passed) {
      console.error(formatQualityGateFailure(result));
      process.exitCode = 1;
    }
    if (options.budgetStrict && !budgetGatePassed(result)) {
      console.error(formatBudgetGateFailure(result));
      process.exitCode = 1;
    }
  } else if (command === "batch-plan" || command === "--batch-plan") {
    const result = await planDevRoutePerformanceBatches({
      outputDir: args[0] || DEFAULT_OUTPUT_DIR,
    });
    console.log(JSON.stringify(result, null, 2));
  } else {
    const routes = buildRouteTargets();
    const edgePath = buildEdgeCoveragePath(routes);
    const coverage = validateCoverage(routes, edgePath);
    const budgets = defaultPerformanceBudgets();
    const payload = {
      routes,
      coverage,
      defaults: {
        transitionBatchSize: DEFAULT_TRANSITION_BATCH_SIZE,
        refreshBatchSize: DEFAULT_REFRESH_BATCH_SIZE,
        includeRefreshes: false,
        performanceBudgets: budgets,
      },
      firstTransitions: edgePath.slice(1, 8).map((route, index) => ({
        from: edgePath[index].url,
        to: route.url,
      })),
    };
    console.log(JSON.stringify(payload, null, 2));
  }
}
