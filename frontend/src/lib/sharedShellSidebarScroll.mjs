const SIDEBAR_VIEWPORT_MARGIN = 12;

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
