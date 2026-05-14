const PERF_PREFIX = "wizard";
const BUFFER_LIMIT = 200;

function canRecordPerformance() {
  return Boolean(import.meta.env.DEV && typeof performance !== "undefined");
}

function sanitizeDetail(detail) {
  return Object.fromEntries(
    Object.entries(detail ?? {}).filter(([, value]) =>
      value === null || ["string", "number", "boolean"].includes(typeof value)
    )
  );
}

function recordEntry(entry) {
  const recordedEntry = {
    recordedAt: new Date().toISOString(),
    ...entry,
  };
  if (typeof window === "undefined") {
    return;
  }
  const entries = Array.isArray(window.__wizardDevPerformanceEntries)
    ? window.__wizardDevPerformanceEntries
    : [];
  entries.push(recordedEntry);
  window.__wizardDevPerformanceEntries = entries.slice(-BUFFER_LIMIT);
  if (typeof console !== "undefined" && console.debug) {
    console.debug("[wizard-perf]", JSON.stringify(recordedEntry));
  }
}

/**
 * devMark emits DEV-only route and render markers consumed by the Browser route matrix.
 * Callers pass only non-secret route labels, artboard keys, and UI state so artifacts
 * can explain phase timing without copying session payloads or provider data.
 */
export function devMark(name, detail = {}) {
  if (!canRecordPerformance()) {
    return;
  }
  const sanitizedDetail = sanitizeDetail(detail);
  const markName = `${PERF_PREFIX}:${name}:${Date.now()}`;
  performance.mark(markName, { detail: sanitizedDetail });
  recordEntry({ type: "mark", name, detail: sanitizedDetail });
  if (typeof console !== "undefined" && console.debug) {
    console.debug(`[perf] ${name}`, sanitizedDetail);
  }
}

/**
 * devMeasureAsync wraps DEV-only async phases such as session fetches or artboard imports.
 * The return value and errors are left unchanged; only sanitized timing metadata is added
 * to the browser performance buffer for the local route-performance harness.
 */
export async function devMeasureAsync(name, detail, operation) {
  if (!canRecordPerformance()) {
    return operation();
  }

  const sanitizedDetail = sanitizeDetail(detail);
  const startName = `${PERF_PREFIX}:${name}:start:${Date.now()}`;
  const endName = `${PERF_PREFIX}:${name}:end:${Date.now()}`;
  performance.mark(startName, { detail: sanitizedDetail });
  try {
    return await operation();
  } finally {
    performance.mark(endName, { detail: sanitizedDetail });
    performance.measure(`${PERF_PREFIX}:${name}`, {
      start: startName,
      end: endName,
      detail: sanitizedDetail,
    });
    const measure = performance.getEntriesByName(`${PERF_PREFIX}:${name}`).at(-1);
    const elapsedMs = measure ? Math.round(measure.duration) : null;
    recordEntry({
      type: "measure",
      name,
      detail: sanitizedDetail,
      elapsedMs,
    });
    if (typeof console !== "undefined" && console.debug) {
      console.debug(`[perf] ${name}`, {
        ...sanitizedDetail,
        elapsedMs,
      });
    }
  }
}
