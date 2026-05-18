import { useEffect, useLayoutEffect, useRef, useState } from "react";

const SHARED_HEADER_HEIGHT = 76;
export const DEFAULT_RUNTIME_DRAWER_BOUNDS = { left: 1278, top: SHARED_HEADER_HEIGHT, width: 390, height: 818 };
const DEFAULT_RUNTIME_DRAWER_WIDTH = 440;
const DEFAULT_RUNTIME_DRAWER_RIGHT_INSET = 4;
const MIN_RUNTIME_DRAWER_WIDTH = 320;

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
 * resolveArtboardRelativeDrawerStyle maps drawer geometry from generated artboard coordinates to viewport-fixed CSS so row/help drawers share the same right edge even when mounted outside the PenArtboard overlay.
 */
function resolveArtboardRelativeDrawerStyle(drawerElement, bounds) {
  const artboard = drawerElement.closest(".pen-stage__artboard") ?? document.querySelector(".pen-stage__artboard");
  if (!artboard) {
    return bounds
      ? {
          position: "fixed",
          left: bounds.left,
          top: SHARED_HEADER_HEIGHT,
          width: bounds.width,
          zIndex: 80,
        }
      : undefined;
  }

  const artboardRect = artboard.getBoundingClientRect();
  const artboardWidth = artboard.offsetWidth || artboardRect.width || 1;
  const scale = artboardRect.width / artboardWidth || 1;
  const requestedWidth = bounds ? bounds.width * scale : DEFAULT_RUNTIME_DRAWER_WIDTH;
  const width = Math.min(
    Math.max(MIN_RUNTIME_DRAWER_WIDTH, requestedWidth),
    Math.max(MIN_RUNTIME_DRAWER_WIDTH, artboardRect.width)
  );
  const drawerRight = artboardRect.right - (bounds ? 0 : DEFAULT_RUNTIME_DRAWER_RIGHT_INSET * scale);
  const requestedLeft = bounds ? artboardRect.left + bounds.left * scale : drawerRight - width;
  const maxLeft = drawerRight - width;
  const left = Math.max(artboardRect.left, Math.min(requestedLeft, maxLeft));

  return {
    position: "fixed",
    left,
    top: SHARED_HEADER_HEIGHT,
    width,
    zIndex: 80,
  };
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
 * RuntimeDrawer is the shared right-hand drawer used by implemented pages for row details, manual workflow forms, and page help. It owns the shell-level top offset, outside-click close behavior, and initial/restore focus handling so every caller gets the same accessible overlay behavior anchored below the shared header. The drawer panel intentionally avoids normal internal scrolling; long content expands the page scroll range instead of trapping keyboard, wheel, or trackpad users inside a nested scroll box.
 */
export function RuntimeDrawer({ title, onClose, children, bounds = null, className = "", ariaLive = "polite" }) {
  const closeButtonRef = useRef(null);
  const drawerRef = useRef(null);
  const restoreFocusRef = useRef(null);
  const [resolvedStyle, setResolvedStyle] = useState(undefined);
  const titleText = String(title);
  const titleId = `runtime-drawer-title-${titleText.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`;
  const devToolbarClass = import.meta.env.DEV ? "runtime-drawer--dev-toolbar-offset" : "";

  useLayoutEffect(() => {
    const drawerElement = drawerRef.current;
    if (!drawerElement) {
      return undefined;
    }

    let frame = 0;
    const updateResolvedStyle = () => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => {
        setResolvedStyle(resolveArtboardRelativeDrawerStyle(drawerElement, bounds));
      });
    };

    const artboard = drawerElement.closest(".pen-stage__artboard");
    const resizeObserver = new ResizeObserver(updateResolvedStyle);
    if (artboard) {
      resizeObserver.observe(artboard);
    }
    updateResolvedStyle();
    window.addEventListener("resize", updateResolvedStyle);
    window.addEventListener("scroll", updateResolvedStyle, true);

    return () => {
      window.cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      window.removeEventListener("resize", updateResolvedStyle);
      window.removeEventListener("scroll", updateResolvedStyle, true);
    };
  }, [bounds]);

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
      ref={drawerRef}
      className={`runtime-drawer ${bounds ? "runtime-drawer--bounded" : ""} ${devToolbarClass} ${className}`.trim()}
      aria-labelledby={titleId}
      aria-live={ariaLive}
      style={resolvedStyle}
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
