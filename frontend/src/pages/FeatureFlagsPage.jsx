import { useCallback, useEffect, useState } from "react";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { PenArtboard } from "../lib/PenArtboard";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const ARTBOARD_KEY = "admin-feature-flags";
const FEATURE_FLAGS_ENDPOINT = "/api/v1/dev/feature-flags";
const FEATURE_FLAGS_HEADING_ID = "feature-flags-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const ARTBOARD_WIDTH = 1672;
const PANE_RIGHT_GUTTER = 48;
const PANE_WIDTH = ARTBOARD_WIDTH - PANE_LEFT - PANE_RIGHT_GUTTER;

async function readJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(payload?.message || `Request failed with ${response.status}`);
    error.status = response.status;
    error.payload = payload;
    throw error;
  }
  return payload;
}

function StatusChip({ enabled, children }) {
  return (
    <span className={`feature-flags-runtime__chip ${enabled ? "feature-flags-runtime__chip--on" : "feature-flags-runtime__chip--off"}`}>
      {children}
    </span>
  );
}

function TargetToggle({ flag, target, targetType, busyTargetKey, onToggle }) {
  if (target.read_only) {
    return <StatusChip enabled={target.enabled}>Always on</StatusChip>;
  }
  const targetKey = `${flag.key}:${targetType}:${target.id}`;
  const busy = busyTargetKey === targetKey;

  return (
    <label className="feature-flags-runtime__toggle">
      <input
        type="checkbox"
        checked={target.enabled}
        disabled={busy}
        onChange={(event) => onToggle(flag, targetType, target.id, event.target.checked)}
      />
      <span>{target.enabled ? "Enabled" : "Disabled"}</span>
    </label>
  );
}

function FeatureFlagCard({ flag, busyTargetKey, onToggle }) {
  const activePersonaTargets = flag.persona_targets.filter((target) => target.enabled);
  const activeSiteTargets = flag.site_targets.filter((target) => target.enabled);

  return (
    <article className="feature-flags-runtime__flag">
      <header className="feature-flags-runtime__flag-header">
        <div>
          <h2>{flag.label}</h2>
          <p>{flag.description}</p>
        </div>
        <StatusChip enabled={flag.effective_for_it_admin}>IT Admin always on</StatusChip>
      </header>
      <div className="feature-flags-runtime__meta">
        <span>{flag.feature_route}</span>
        <span>{flag.routes.join(", ")}</span>
      </div>
      <section className="feature-flags-runtime__indicators" aria-label={`${flag.label} active indicators`}>
        {activePersonaTargets.map((target) => (
          <StatusChip key={`persona-${target.id}`} enabled={target.enabled}>{target.label}</StatusChip>
        ))}
        {activeSiteTargets.map((target) => (
          <StatusChip key={`site-${target.id}`} enabled={target.enabled}>{target.label}</StatusChip>
        ))}
      </section>
      <div className="feature-flags-runtime__matrix">
        <section aria-label={`${flag.label} persona targets`}>
          <h3>Personas</h3>
          <div className="feature-flags-runtime__target-list">
            {flag.persona_targets.map((target) => (
              <div key={target.id} className="feature-flags-runtime__target-row">
                <span>{target.label}</span>
                <TargetToggle flag={flag} target={target} targetType="persona" busyTargetKey={busyTargetKey} onToggle={onToggle} />
              </div>
            ))}
          </div>
        </section>
        <section aria-label={`${flag.label} site targets`}>
          <h3>Sites</h3>
          <div className="feature-flags-runtime__target-list">
            {flag.site_targets.map((target) => (
              <div key={target.id} className="feature-flags-runtime__target-row">
                <span>{target.label}</span>
                <TargetToggle flag={flag} target={target} targetType="site" busyTargetKey={busyTargetKey} onToggle={onToggle} />
              </div>
            ))}
          </div>
        </section>
      </div>
    </article>
  );
}

/**
 * FeatureFlagsOverlay renders the native controls layered over the Feature Flags `.pen` shell. `PenArtboard` owns long-content measurement for every route, so this page only supplies its page-local pane geometry and feature-flag interaction state.
 */
function FeatureFlagsOverlay({ payload, state, message, busyTargetKey, onToggle }) {
  return (
    <section
      className="feature-flags-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        zIndex: 2,
      }}
      aria-labelledby={FEATURE_FLAGS_HEADING_ID}
    >
      <header className="feature-flags-runtime__header">
        <div>
          <h1 id={FEATURE_FLAGS_HEADING_ID}>Feature Flags</h1>
          <p>IT Admin route-level controls for persona and site slices.</p>
        </div>
        <StatusChip enabled>IT Admin override active</StatusChip>
      </header>
      {state === "loading" ? <p className="feature-flags-runtime__status" role="status">Loading feature flags...</p> : null}
      {message ? <p className="feature-flags-runtime__status" role={state === "error" ? "alert" : "status"}>{message}</p> : null}
      {payload?.flags?.length ? (
        <div className="feature-flags-runtime__grid">
          {payload.flags.map((flag) => (
            <FeatureFlagCard key={flag.key} flag={flag} busyTargetKey={busyTargetKey} onToggle={onToggle} />
          ))}
        </div>
      ) : null}
    </section>
  );
}

export function FeatureFlagsPage({ session, onNavigate, onSearch, searchQuery, onUnauthorized, onForbidden }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [payload, setPayload] = useState(null);
  const [state, setState] = useState("loading");
  const [message, setMessage] = useState("");
  const [busyTargetKey, setBusyTargetKey] = useState("");
  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? "admin",
    activeRoutePath: "/admin/feature-flags",
    refreshMetadata: null,
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Feature Flags",
        textOverrides,
      })
    : { title: "Feature Flags", items: [] };

  const loadFeatureFlags = useCallback(async (signal) => {
    setState("loading");
    setMessage("");
    try {
      const nextPayload = await readJSON(
        await fetch(FEATURE_FLAGS_ENDPOINT, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          signal,
        })
      );
      setPayload(nextPayload);
      setState("ready");
    } catch (error) {
      if (signal?.aborted) {
        return;
      }
      if (error.status === 401) {
        onUnauthorized?.();
        return;
      }
      if (error.status === 403) {
        onForbidden?.();
        return;
      }
      setState("error");
      setMessage(error.message);
    }
  }, [onForbidden, onUnauthorized]);

  useEffect(() => {
    const controller = new AbortController();
    void loadFeatureFlags(controller.signal);
    return () => controller.abort();
  }, [loadFeatureFlags]);

  const handleToggle = useCallback(async (flag, targetType, targetId, enabled) => {
    setBusyTargetKey(`${flag.key}:${targetType}:${targetId}`);
    setMessage("");
    try {
      const updatedFlag = await readJSON(
        await fetch(`${FEATURE_FLAGS_ENDPOINT}/${encodeURIComponent(flag.key)}`, {
          method: "PUT",
          credentials: "same-origin",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            targets: [{ target_type: targetType, target_id: targetId, enabled }],
          }),
        })
      );
      setPayload((current) => ({
        ...current,
        flags: (current?.flags ?? []).map((candidate) => (candidate.key === updatedFlag.key ? updatedFlag : candidate)),
      }));
      setState("ready");
      setMessage(`${updatedFlag.label} updated.`);
    } catch (error) {
      if (error.status === 401) {
        onUnauthorized?.();
        return;
      }
      if (error.status === 403) {
        onForbidden?.();
        return;
      }
      setMessage(error.payload?.message || error.message);
    } finally {
      setBusyTargetKey("");
    }
  }, [onForbidden, onUnauthorized]);

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <FeatureFlagsOverlay
        payload={payload}
        state={state}
        message={message}
        busyTargetKey={busyTargetKey}
        onToggle={handleToggle}
      />
    </>
  ), [busyTargetKey, handleToggle, message, payload, sharedShellRenderOverlay, state]);

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Feature Flags</h1>
          <p>Preparing the generated Feature Flags artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Feature Flags unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={FEATURE_FLAGS_HEADING_ID}>
      <section className="sr-only" aria-labelledby={`${FEATURE_FLAGS_HEADING_ID}-summary`}>
        <h1 id={`${FEATURE_FLAGS_HEADING_ID}-summary`}>{semanticSummary.title}</h1>
        <ul>
          {semanticSummary.items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      </section>
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
      </div>
    </main>
  );
}
