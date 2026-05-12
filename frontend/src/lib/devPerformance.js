const PERF_PREFIX = "wizard";

function canRecordPerformance() {
  return Boolean(import.meta.env.DEV && typeof performance !== "undefined");
}

export function devMark(name, detail = {}) {
  if (!canRecordPerformance()) {
    return;
  }
  const markName = `${PERF_PREFIX}:${name}:${Date.now()}`;
  performance.mark(markName, { detail });
  if (typeof console !== "undefined" && console.debug) {
    console.debug(`[perf] ${name}`, detail);
  }
}

export async function devMeasureAsync(name, detail, operation) {
  if (!canRecordPerformance()) {
    return operation();
  }

  const startName = `${PERF_PREFIX}:${name}:start:${Date.now()}`;
  const endName = `${PERF_PREFIX}:${name}:end:${Date.now()}`;
  performance.mark(startName, { detail });
  try {
    return await operation();
  } finally {
    performance.mark(endName, { detail });
    performance.measure(`${PERF_PREFIX}:${name}`, {
      start: startName,
      end: endName,
      detail,
    });
    const measure = performance.getEntriesByName(`${PERF_PREFIX}:${name}`).at(-1);
    if (typeof console !== "undefined" && console.debug) {
      console.debug(`[perf] ${name}`, {
        ...detail,
        elapsedMs: measure ? Math.round(measure.duration) : null,
      });
    }
  }
}
