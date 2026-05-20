export const SHARED_HEADER_HEIGHT = 76;
export const DEFAULT_RUNTIME_DRAWER_WIDTH = 440;
export const DEFAULT_RUNTIME_DRAWER_RIGHT_INSET = 4;
export const MIN_RUNTIME_DRAWER_WIDTH = 320;

function clamp(value, min, max) {
  return Math.max(min, Math.min(value, max));
}

/**
 * resolveArtboardDrawerStyle returns artboard-local drawer geometry for row-detail and help drawers mounted inside a generated `.pen` artboard overlay. RuntimeDrawer uses the returned absolute positioning so the drawer participates in PenArtboard's content-height measurement; that measurement extends the shared page scroll range when a short page opens a long drawer.
 */
export function resolveArtboardDrawerStyle({ bounds = null, artboardWidth = 1, scale = 1 }) {
  const safeScale = Number.isFinite(scale) && scale > 0 ? scale : 1;
  const safeArtboardWidth = Math.max(MIN_RUNTIME_DRAWER_WIDTH, Number.isFinite(artboardWidth) ? artboardWidth : 1);
  const top = SHARED_HEADER_HEIGHT / safeScale;

  if (bounds) {
    const requestedWidth = Number.isFinite(bounds.width) ? bounds.width : DEFAULT_RUNTIME_DRAWER_WIDTH / safeScale;
    const width = clamp(requestedWidth, MIN_RUNTIME_DRAWER_WIDTH / safeScale, safeArtboardWidth);
    const requestedLeft = Number.isFinite(bounds.left) ? bounds.left : safeArtboardWidth - width;
    const left = clamp(requestedLeft, 0, safeArtboardWidth - width);
    return {
      position: "absolute",
      left,
      top,
      width,
      zIndex: 80,
    };
  }

  const width = clamp(DEFAULT_RUNTIME_DRAWER_WIDTH / safeScale, MIN_RUNTIME_DRAWER_WIDTH / safeScale, safeArtboardWidth);
  return {
    position: "absolute",
    left: safeArtboardWidth - DEFAULT_RUNTIME_DRAWER_RIGHT_INSET - width,
    top,
    width,
    zIndex: 80,
  };
}

/**
 * resolveFallbackFixedDrawerStyle keeps RuntimeDrawer usable in rare callers that mount outside the `.pen` artboard tree. Those drawers cannot contribute to artboard height measurement, so implemented-page row/help drawers should stay inside PenArtboard overlays whenever they need the shared page scroll range.
 */
export function resolveFallbackFixedDrawerStyle(bounds = null) {
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
