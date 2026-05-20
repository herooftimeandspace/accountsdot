import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { nextRuntimeDrawerSelectionForId } from "./runtimeDrawerController.mjs";
import { resolveRuntimeDrawerStyle } from "./runtimeDrawerGeometry.mjs";

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
 * RuntimeDrawer is the shared right-hand drawer used by implemented pages for row details, manual workflow forms, and page help. It owns shell-relative placement, outside-click and Escape close behavior, fixed-height internal scrolling, and focus management so every caller gets the same accessible overlay behavior anchored below the shared header.
 */
export function RuntimeDrawer({
  title,
  onClose,
  children,
  bounds = null,
  className = "",
  ariaLive = "polite",
  variant = "inspector",
  isDirty = false,
}) {
  const closeButtonRef = useRef(null);
  const drawerRef = useRef(null);
  const restoreFocusRef = useRef(null);
  const [resolvedStyle, setResolvedStyle] = useState(undefined);
  const titleText = String(title);
  const titleId = `runtime-drawer-title-${titleText.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`;
  const devToolbarClass = import.meta.env.DEV ? "runtime-drawer--dev-toolbar-offset" : "";
  const isModal = variant === "modal";

  function requestClose() {
    if (isModal && isDirty && !window.confirm("Discard unsaved drawer changes?")) {
      return;
    }
    onClose();
  }

  useLayoutEffect(() => {
    const drawerElement = drawerRef.current;
    if (!drawerElement) {
      return undefined;
    }

    let frame = 0;
    const applyResolvedStyle = () => {
      setResolvedStyle(resolveRuntimeDrawerStyle(drawerElement, bounds));
    };
    const updateResolvedStyle = () => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => {
        applyResolvedStyle();
      });
    };

    const artboard = drawerElement.closest(".pen-stage__artboard");
    const resizeObserver = new ResizeObserver(updateResolvedStyle);
    if (artboard) {
      resizeObserver.observe(artboard);
    }
    applyResolvedStyle();
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
      if (!shouldCloseDrawerForPointerTarget(event.target)) {
        return;
      }
      if (isModal) {
        event.preventDefault();
        event.stopPropagation();
      }
      requestClose();
    }

    document.addEventListener("pointerdown", handleDocumentPointerDown, true);
    return () => document.removeEventListener("pointerdown", handleDocumentPointerDown, true);
  }, [isDirty, isModal, onClose]);

  useEffect(() => {
    function handleDocumentKeyDown(event) {
      if (event.key === "Escape") {
        event.preventDefault();
        requestClose();
        return;
      }
      if (!isModal || event.key !== "Tab") {
        return;
      }
      const focusable = Array.from(
        drawerRef.current?.querySelectorAll(
          "a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), summary, [tabindex]:not([tabindex='-1'])"
        ) ?? []
      ).filter((element) => element instanceof HTMLElement && !element.hidden);
      if (!focusable.length) {
        event.preventDefault();
        closeButtonRef.current?.focus();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }

    document.addEventListener("keydown", handleDocumentKeyDown, true);
    return () => document.removeEventListener("keydown", handleDocumentKeyDown, true);
  }, [isDirty, isModal, onClose]);

  function handleCloseButtonPointerDown(event) {
    event.stopPropagation();
    requestClose();
  }

  function handleCloseButtonClick(event) {
    event.stopPropagation();
    requestClose();
  }

  return (
    <aside
      ref={drawerRef}
      className={`runtime-drawer ${bounds ? "runtime-drawer--bounded" : ""} runtime-drawer--${variant} ${devToolbarClass} ${className}`.trim()}
      aria-labelledby={titleId}
      aria-live={ariaLive}
      aria-modal={isModal ? "true" : undefined}
      role={isModal ? "dialog" : undefined}
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
        <div className="runtime-drawer__body">{children}</div>
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
          onClick={() => onSelect(nextRuntimeDrawerSelectionForId(selectedId, row))}
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
