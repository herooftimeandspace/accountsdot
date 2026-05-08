export function RuntimeDetailList({ items }) {
  const visibleItems = items.filter((item) => item && item.value !== undefined && item.value !== null && item.value !== "");

  if (!visibleItems.length) {
    return null;
  }

  return (
    <dl className="runtime-drawer__details">
      {visibleItems.map((item) => (
        <div key={item.label}>
          <dt>{item.label}</dt>
          <dd>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function RuntimeDrawer({ title, onClose, children, bounds = null, className = "", ariaLive = "polite" }) {
  const titleText = String(title);
  const titleId = `runtime-drawer-title-${titleText.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`;
  const boundedStyle = bounds
    ? {
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: bounds.width,
        height: bounds.height,
        zIndex: 32,
      }
    : undefined;

  return (
    <aside
      className={`runtime-drawer ${bounds ? "runtime-drawer--bounded" : ""} ${className}`.trim()}
      aria-labelledby={titleId}
      aria-live={ariaLive}
      style={boundedStyle}
    >
      <div className="runtime-drawer__panel">
        <div className="runtime-drawer__header">
          <h2 id={titleId}>{titleText}</h2>
          <button
            type="button"
            className="runtime-drawer__close"
            aria-label={`Close ${titleText.toLowerCase()} drawer`}
            onClick={onClose}
          >
            &times;
          </button>
        </div>
        {children}
      </div>
    </aside>
  );
}

export function RowHotspotOverlay({ rows, selectedId, onSelect, ariaLabel }) {
  return (
    <div aria-label={ariaLabel}>
      {rows.map((row) => (
        <button
          key={row.id}
          type="button"
          className="runtime-row-hotspot"
          aria-label={row.ariaLabel}
          aria-pressed={selectedId === row.id}
          onClick={() => onSelect(row)}
          style={{
            left: row.left,
            top: row.top,
            width: row.width,
            height: row.height,
          }}
        />
      ))}
    </div>
  );
}
