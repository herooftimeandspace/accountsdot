const SIDEBAR_VIEWPORT_MARGIN = 12;
const WHEEL_DELTA_LINE = 16;

/**
 * clampSidebarOffset keeps the shared sidebar scroll offset within the range
 * that can reveal overflowing sidebar controls while preserving the unscrolled
 * top position on taller viewports. SharedShellSidebarScrollManager calls this
 * before writing the CSS variable consumed by fixed sidebar nodes and nav
 * hotspots.
 */
export function clampSidebarOffset(offset, { viewportHeight, contentBottom, margin = SIDEBAR_VIEWPORT_MARGIN }) {
  const numericOffset = Number.isFinite(offset) ? offset : 0;
  const safeViewportHeight = Math.max(0, Number(viewportHeight) || 0);
  const safeContentBottom = Math.max(0, Number(contentBottom) || 0);
  const maxNegativeOffset = Math.min(0, safeViewportHeight - safeContentBottom - margin);

  return Math.min(0, Math.max(maxNegativeOffset, numericOffset));
}

/**
 * sidebarOffsetForWheel translates wheel movement inside the fixed shared
 * sidebar into the independent sidebar-layer offset. The page body keeps its
 * normal scroll behavior because callers only invoke this helper for pointer
 * coordinates inside the sidebar bounds.
 */
export function sidebarOffsetForWheel(currentOffset, deltaY, geometry) {
  return clampSidebarOffset(currentOffset - deltaY, geometry);
}

/**
 * sidebarWheelDeltaPixels normalizes WheelEvent delta units before shared
 * sidebar scroll math runs. SharedShellSidebarScrollManager uses this so line
 * and page-mode wheels behave like pixel-mode trackpads instead of jumping
 * straight to the sidebar's clamp bounds.
 */
export function sidebarWheelDeltaPixels(deltaY, deltaMode, geometry = {}) {
  const numericDeltaY = Number.isFinite(deltaY) ? deltaY : 0;
  if (deltaMode === 1) {
    return numericDeltaY * WHEEL_DELTA_LINE;
  }
  if (deltaMode === 2) {
    const viewportHeight = Math.max(0, Number(geometry?.viewportHeight) || 0);
    return numericDeltaY * viewportHeight;
  }
  return numericDeltaY;
}

/**
 * sidebarOffsetForFocusedRect adjusts the shared sidebar offset enough to
 * bring a keyboard-focused sidebar control into the visible viewport. This
 * protects the DEV persona switcher and nested IT Admin routes for tab users on
 * short browser heights.
 */
export function sidebarOffsetForFocusedRect(currentOffset, rect, geometry) {
  if (!rect) {
    return clampSidebarOffset(currentOffset, geometry);
  }

  const margin = geometry?.margin ?? SIDEBAR_VIEWPORT_MARGIN;
  const viewportHeight = Math.max(0, Number(geometry?.viewportHeight) || 0);
  let nextOffset = currentOffset;

  if (rect.bottom > viewportHeight - margin) {
    nextOffset -= rect.bottom - (viewportHeight - margin);
  }
  if (rect.top < margin) {
    nextOffset += margin - rect.top;
  }

  return clampSidebarOffset(nextOffset, geometry);
}
