import { useEffect, useRef } from "react";

export const DEFAULT_RUNTIME_DRAWER_BOUNDS = { left: 1278, top: 92, width: 390, height: 802 };

function shouldCloseDrawerForPointerTarget(target) {
  if (!(target instanceof Element)) {
    return false;
  }
  if (target.closest(".runtime-drawer")) {
    return false;
  }
  if (target.closest("button, a, input, select, textarea, label, summary, [role='button'], [role='link'], [tabindex]")) {
    return false;
  }
  return true;
}

/**
 * RuntimeDetailList renders label/value metadata inside shared drawers. Page drawers pass already-authorized display values here, and the helper drops empty values so rows do not create blank assistive-technology stops or visual gaps.
 */
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

/**
 * RuntimeDrawer is the shared right-hand drawer used by implemented pages for row details, manual workflow forms, and page help. It owns the visual shell, outside-click close behavior, and initial/restore focus handling so every caller gets the same accessible overlay behavior whether the drawer is bounded to an artboard region or pinned to the shell's right edge.
 */
export function RuntimeDrawer({ title, onClose, children, bounds = null, className = "", ariaLive = "polite" }) {
  const closeButtonRef = useRef(null);
  const restoreFocusRef = useRef(null);
  const titleText = String(title);
  const titleId = `runtime-drawer-title-${titleText.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`;
  const devToolbarClass = import.meta.env.DEV ? "runtime-drawer--dev-toolbar-offset" : "";
  const boundedStyle = bounds
    ? {
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: bounds.width,
        height: bounds.height,
        zIndex: 80,
    }
    : undefined;

  useEffect(() => {
    restoreFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    closeButtonRef.current?.focus({ preventScroll: true });

    return () => {
      if (restoreFocusRef.current?.isConnected) {
        restoreFocusRef.current.focus({ preventScroll: true });
      }
    };
  }, []);

  useEffect(() => {
    function handleDocumentPointerDown(event) {
      if (shouldCloseDrawerForPointerTarget(event.target)) {
        onClose();
      }
    }

    document.addEventListener("pointerdown", handleDocumentPointerDown, true);
    return () => document.removeEventListener("pointerdown", handleDocumentPointerDown, true);
  }, [onClose]);

  function handleCloseButtonPointerDown(event) {
    event.stopPropagation();
    onClose();
  }

  function handleCloseButtonClick(event) {
    event.stopPropagation();
    onClose();
  }

  return (
    <aside
      className={`runtime-drawer ${bounds ? "runtime-drawer--bounded" : ""} ${devToolbarClass} ${className}`.trim()}
      aria-labelledby={titleId}
      aria-live={ariaLive}
      style={boundedStyle}
    >
      <div className="runtime-drawer__panel">
        <div className="runtime-drawer__header">
          <h2 id={titleId}>{titleText}</h2>
          <button
            type="button"
            ref={closeButtonRef}
            className="runtime-drawer__close"
            aria-label={`Close ${titleText.toLowerCase()} drawer`}
            onPointerDown={handleCloseButtonPointerDown}
            onClick={handleCloseButtonClick}
          >
            &times;
          </button>
        </div>
        {children}
      </div>
    </aside>
  );
}

/**
 * RowHotspotOverlay places transparent buttons over generated artboard table rows. StaticPenPage and runtime-migrating pages use it to keep the `.pen` table geometry intact while React owns row selection state and drawer opening behavior.
 */
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
