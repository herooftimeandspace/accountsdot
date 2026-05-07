export function DevPersonaSwitcher({ session, personaId, sessionState, onChange }) {
  if (!import.meta.env.DEV) {
    return null;
  }

  const personas = session?.personas ?? [];
  const hintId = "dev-persona-switcher-hint";

  return (
    <aside className="dev-toolbar" aria-label="Development persona controls">
      <div className="dev-toolbar__eyebrow">Pre-Phase 0 DEV</div>
      <h2 className="dev-toolbar__title">Persona Switcher</h2>
      <label className="dev-toolbar__label" htmlFor="dev-persona-select">
        Current persona
      </label>
      <select
        id="dev-persona-select"
        className="dev-toolbar__select"
        value={personaId}
        aria-describedby={hintId}
        onChange={(event) => onChange(event.target.value)}
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
        <strong>{sessionState}</strong>
      </div>
      <p id={hintId} className="dev-toolbar__hint">
        DEV-only persona switching is backed by mock data from the Go app. Production auth is out of
        scope for pre-phase 0.
      </p>
    </aside>
  );
}
