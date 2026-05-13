import { useEffect, useState } from "react";
import { loadGeneratedArtboard } from "../generated/artboards.generated.js";

const resolvedArtboards = new Map();

export async function loadArtboard(key) {
  if (!key) {
    throw new Error("Generated artboard key is required.");
  }
  if (resolvedArtboards.has(key)) {
    return resolvedArtboards.get(key);
  }
  const artboard = await loadGeneratedArtboard(key);
  resolvedArtboards.set(key, artboard);
  return artboard;
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

  return state;
}
