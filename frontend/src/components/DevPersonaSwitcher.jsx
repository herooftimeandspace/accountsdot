import { useEffect, useId, useState } from "react";

/**
 * DevPersonaSwitcher renders the UI surface for frontend/src/components/DevPersonaSwitcher.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
    <aside className="dev-toolbar" aria-label="Development persona controls">
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
