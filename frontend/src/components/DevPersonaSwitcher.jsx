import { useEffect, useId, useLayoutEffect, useMemo, useState } from "react";

import { sharedShellSpec } from "../generated/artboards.generated.js";

/**
 * DevPersonaSwitcher renders the DEV-only mock-session control inside the shared sidebar bounds. App owns the persona-switch side effect and routing fallback, while this component owns the collapsed/expanded control state, current-persona labeling, and polite status announcements used during demos.
 */
export function DevPersonaSwitcher({
  session,
  personaId,
  pendingPersonaId = null,
  pendingLabel = "",
  sessionState,
  onChange,
}) {
  if (!import.meta.env.DEV) {
    return null;
  }

  const personas = session?.personas ?? [];
  const panelId = useId();
  const statusLabel = pendingPersonaId ? "switching" : sessionState;
  const statusDetail = pendingLabel ? `Switching to ${pendingLabel}...` : null;
  const activePersonaLabel =
    personas.find((persona) => persona.id === personaId)?.label || personaId || "Unknown";
  const [expanded, setExpanded] = useState(false);

  const platformStatusValueSelector = useMemo(() => {
    const nodeId = sharedShellSpec?.sharedShellIds?.platformStatusValue;
    return nodeId ? `[data-node-id="${nodeId}"]` : null;
  }, []);

  const [anchorStyle, setAnchorStyle] = useState(null);

  useLayoutEffect(() => {
    if (!platformStatusValueSelector) {
      return undefined;
    }

    function updateAnchor() {
      const sidebarNodes = Array.from(document.querySelectorAll('[data-shared-shell-sticky="sidebar"]'));
      const sidebarBounds = sidebarNodes.reduce(
        (bounds, node) => {
          const rect = node.getBoundingClientRect();
          if (!rect || rect.width <= 0 || rect.height <= 0) {
            return bounds;
          }
          return {
            left: Math.min(bounds.left, rect.left),
            right: Math.max(bounds.right, rect.right),
          };
        },
        { left: Number.POSITIVE_INFINITY, right: Number.NEGATIVE_INFINITY },
      );
      const platformStatusNode = document.querySelector(platformStatusValueSelector);
      if (!platformStatusNode || !Number.isFinite(sidebarBounds.left) || !Number.isFinite(sidebarBounds.right)) {
        setAnchorStyle(null);
        return;
      }
      const rect = platformStatusNode.getBoundingClientRect();
      if (!rect || Number.isNaN(rect.bottom)) {
        setAnchorStyle(null);
        return;
      }
      const sidebarWidth = Math.max(0, sidebarBounds.right - sidebarBounds.left);
      const horizontalPadding = sidebarWidth >= 180 ? 16 : 8;
      const desiredTop = Math.round(rect.bottom + 8);
      const viewportBottom = window.innerHeight || document.documentElement.clientHeight || desiredTop;
      const clampedTop = Math.min(desiredTop, Math.max(0, viewportBottom - 48));
      const nextStyle = {
        left: `${Math.round(sidebarBounds.left + horizontalPadding)}px`,
        top: `${Math.max(0, clampedTop)}px`,
        width: `${Math.max(0, Math.round(sidebarWidth - horizontalPadding * 2))}px`,
      };
      setAnchorStyle((current) => {
        if (current?.left === nextStyle.left && current?.top === nextStyle.top && current?.width === nextStyle.width) {
          return current;
        }
        return nextStyle;
      });
    }

    updateAnchor();
    window.addEventListener("resize", updateAnchor);
    window.addEventListener("shared-shell-sidebar-scroll", updateAnchor);

    const observer = new MutationObserver(() => {
      updateAnchor();
    });
    observer.observe(document.body, { childList: true, subtree: true });

    return () => {
      window.removeEventListener("resize", updateAnchor);
      window.removeEventListener("shared-shell-sidebar-scroll", updateAnchor);
      observer.disconnect();
    };
  }, [platformStatusValueSelector]);

  useEffect(() => {
    if (pendingPersonaId) {
      setExpanded(false);
    }
  }, [pendingPersonaId]);

  function toggleExpanded() {
    setExpanded((current) => !current);
  }

  function handleToggleKeyDown(event) {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    toggleExpanded();
  }

  function handlePersonaChange(nextPersonaId) {
    setExpanded(false);
    onChange(nextPersonaId);
  }

  return (
    <aside
      className="dev-toolbar"
      style={anchorStyle ?? undefined}
      aria-label="Development persona controls"
    >
      <button
        type="button"
        className="dev-toolbar__toggle"
        aria-expanded={expanded}
        aria-controls={panelId}
        aria-label={`${expanded ? "Collapse" : "Expand"} DEV persona switcher`}
        onClick={toggleExpanded}
        onKeyDown={handleToggleKeyDown}
      >
        <span>DEV</span>
        <strong>{activePersonaLabel}</strong>
      </button>
      {expanded ? (
        <div id={panelId} className="dev-toolbar__panel">
          <h2 className="dev-toolbar__title">Dev-only Persona Switcher</h2>
          <label className="dev-toolbar__label" htmlFor="dev-persona-select">
            Current persona
          </label>
          <select
            id="dev-persona-select"
            className="dev-toolbar__select"
            value={personaId}
            onChange={(event) => handlePersonaChange(event.target.value)}
          >
            {personas.length > 0 ? (
              personas.map((persona) => (
                <option key={persona.id} value={persona.id}>
                  {persona.label}
                </option>
              ))
            ) : (
              <option value={personaId}>{personaId}</option>
            )}
          </select>
          {/* WCAG 4.1.3: persona-session load state changes announce without moving focus. */}
          <div className="dev-toolbar__meta" aria-live="polite">
            <span>Status</span>
            <strong>{statusLabel}</strong>
          </div>
          {statusDetail ? <p className="dev-toolbar__status-detail">{statusDetail}</p> : null}
        </div>
      ) : null}
    </aside>
  );
}
