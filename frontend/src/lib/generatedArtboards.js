import { useEffect, useState } from "react";
import { loadGeneratedArtboard } from "../generated/artboards.generated.js";
import { devMark, devMeasureAsync } from "./devPerformance";

const resolvedArtboards = new Map();
const prefetchedArtboards = new Set();

/**
 * loadArtboard returns cached generated artboards or imports them once for static PEN pages.
 * DEV route-performance runs use the sanitized timing marker to separate generated artboard
 * import cost from Browser navigation/load and readiness polling time.
 */
export async function loadArtboard(key) {
  if (!key) {
    throw new Error("Generated artboard key is required.");
  }
  if (resolvedArtboards.has(key)) {
    return resolvedArtboards.get(key);
  }
  const artboard = await devMeasureAsync("generated-artboard-import", { artboardKey: key }, () =>
    loadGeneratedArtboard(key)
  );
  resolvedArtboards.set(key, artboard);
  return artboard;
}

/**
 * prefetchArtboards warms generated .pen-derived JSON chunks after App has already
 * received an authorized session route list. It accepts artboard keys that were
 * derived from allowed routes, schedules at most one loader per key, and keeps
 * failures non-fatal because foreground navigation still reports artboard load
 * errors through useGeneratedArtboard.
 */
export function prefetchArtboards(keys) {
  const uniqueKeys = [...new Set(keys.filter(Boolean))];
  uniqueKeys.forEach((key) => {
    if (resolvedArtboards.has(key) || prefetchedArtboards.has(key)) {
      return;
    }
    prefetchedArtboards.add(key);
    loadArtboard(key).catch(() => {
      prefetchedArtboards.delete(key);
    });
  });
}

export function useGeneratedArtboard(key) {
  const [state, setState] = useState(() => {
    if (key && resolvedArtboards.has(key)) {
      return { artboard: resolvedArtboards.get(key), status: "ready", error: null };
    }
    return { artboard: null, status: key ? "loading" : "error", error: key ? null : new Error("Generated artboard key is required.") };
  });

  useEffect(() => {
    let cancelled = false;
    if (!key) {
      setState({ artboard: null, status: "error", error: new Error("Generated artboard key is required.") });
      return () => {
        cancelled = true;
      };
    }
    if (resolvedArtboards.has(key)) {
      setState({ artboard: resolvedArtboards.get(key), status: "ready", error: null });
      return () => {
        cancelled = true;
      };
    }

    setState({ artboard: null, status: "loading", error: null });
    loadArtboard(key)
      .then((artboard) => {
        if (!cancelled) {
          setState({ artboard, status: "ready", error: null });
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setState({ artboard: null, status: "error", error });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [key]);

  useEffect(() => {
    if (key && state.status === "ready") {
      devMark("generated-artboard-render", { artboardKey: key });
    }
  }, [key, state.status]);

  return state;
}
