import { useEffect } from "react";

/**
 * shouldCloseDrawerForPointerTarget documents runtime data flow for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
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
 * RuntimeDetailList renders the UI surface for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * RuntimeDrawer renders the UI surface for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function RuntimeDrawer({ title, onClose, children, bounds = null, className = "", ariaLive = "polite" }) {
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
    /**
     * handleDocumentPointerDown handles the user or network event for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
     */
    function handleDocumentPointerDown(event) {
      if (shouldCloseDrawerForPointerTarget(event.target)) {
        onClose();
      }
    }

    document.addEventListener("pointerdown", handleDocumentPointerDown, true);
    return () => document.removeEventListener("pointerdown", handleDocumentPointerDown, true);
  }, [onClose]);

  /**
   * handleCloseButtonPointerDown handles the user or network event for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
   */
  function handleCloseButtonPointerDown(event) {
    event.stopPropagation();
    onClose();
  }

  /**
   * handleCloseButtonClick handles the user or network event for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
   */
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
 * RowHotspotOverlay renders the UI surface for frontend/src/components/RuntimeDrawer.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
