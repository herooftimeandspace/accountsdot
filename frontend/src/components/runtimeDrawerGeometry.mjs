export const SHARED_HEADER_HEIGHT = 76;
export const DEFAULT_RUNTIME_DRAWER_BOUNDS = { left: 1278, top: SHARED_HEADER_HEIGHT, width: 390, height: 818 };
export const DEFAULT_RUNTIME_DRAWER_WIDTH = 440;
export const DEFAULT_RUNTIME_DRAWER_RIGHT_INSET = 0;
export const MIN_RUNTIME_DRAWER_WIDTH = 320;
export const MIN_RUNTIME_DRAWER_HEIGHT = 340;
export const RUNTIME_DRAWER_HEIGHT_RATIO = 0.6;
export const RUNTIME_DRAWER_BOTTOM_GAP = 24;
export const RUNTIME_DRAWER_MOBILE_BREAKPOINT = 760;

function clamp(value, min, max) {
  if (max < min) {
    return max;
  }
  return Math.min(Math.max(value, min), max);
}

export function runtimeDrawerHeightForViewport(viewportHeight, headerHeight = SHARED_HEADER_HEIGHT) {
  const availableHeight = Math.max(220, viewportHeight - headerHeight - RUNTIME_DRAWER_BOTTOM_GAP);
  const targetHeight = Math.round(viewportHeight * RUNTIME_DRAWER_HEIGHT_RATIO);
  return clamp(targetHeight, Math.min(MIN_RUNTIME_DRAWER_HEIGHT, availableHeight), availableHeight);
}

export function resolveRuntimeDrawerPlacement({
  artboardRect = null,
  artboardOffsetWidth = 0,
  headerRect = null,
  mountedInArtboard = false,
  bounds = null,
  viewportWidth,
  viewportHeight,
  headerHeight = SHARED_HEADER_HEIGHT,
}) {
  const height = runtimeDrawerHeightForViewport(viewportHeight, headerHeight);
  const mobile = viewportWidth <= RUNTIME_DRAWER_MOBILE_BREAKPOINT;
  if (mobile) {
    return {
      position: "fixed",
      left: 0,
      top: headerHeight,
      width: viewportWidth,
      height,
      zIndex: 80,
    };
  }

  if (!artboardRect) {
    const width = Math.min(DEFAULT_RUNTIME_DRAWER_WIDTH, Math.max(MIN_RUNTIME_DRAWER_WIDTH, viewportWidth - DEFAULT_RUNTIME_DRAWER_RIGHT_INSET));
    return {
      position: "fixed",
      left: Math.max(0, viewportWidth - width - DEFAULT_RUNTIME_DRAWER_RIGHT_INSET),
      top: headerHeight,
      width,
      height,
      zIndex: 80,
    };
  }

  const artboardWidth = artboardOffsetWidth || artboardRect.width || 1;
  const scale = artboardRect.width / artboardWidth || 1;
  const styleScale = mountedInArtboard ? scale : 1;
  const visualTop = headerRect?.bottom ?? headerHeight;
  const requestedWidth = bounds ? bounds.width * scale : DEFAULT_RUNTIME_DRAWER_WIDTH;
  const maxContainedWidth = Math.max(MIN_RUNTIME_DRAWER_WIDTH, Math.min(artboardRect.width, viewportWidth - DEFAULT_RUNTIME_DRAWER_RIGHT_INSET));
  const width = Math.min(Math.max(MIN_RUNTIME_DRAWER_WIDTH, requestedWidth), maxContainedWidth);
  const pageRight = headerRect?.right ?? artboardRect.right;
  const drawerRight = Math.min(pageRight - (bounds ? 0 : DEFAULT_RUNTIME_DRAWER_RIGHT_INSET * scale), viewportWidth - DEFAULT_RUNTIME_DRAWER_RIGHT_INSET);
  const requestedLeft = bounds ? artboardRect.left + bounds.left * scale : drawerRight - width;
  const maxLeft = drawerRight - width;
  const left = clamp(requestedLeft, Math.max(0, artboardRect.left), Math.max(0, maxLeft));

  return {
    position: "fixed",
    left: left / styleScale,
    top: visualTop / styleScale,
    width: width / styleScale,
    height: height / styleScale,
    zIndex: 80,
  };
}

export function resolveRuntimeDrawerStyle(drawerElement, bounds) {
  const mountedArtboard = drawerElement.closest(".pen-stage__artboard");
  const artboard = mountedArtboard ?? document.querySelector(".pen-stage__artboard");
  const header = document.querySelector(".pen-node--shared-shell-header");
  return resolveRuntimeDrawerPlacement({
    artboardRect: artboard ? artboard.getBoundingClientRect() : null,
    artboardOffsetWidth: artboard?.offsetWidth ?? 0,
    headerRect: header ? header.getBoundingClientRect() : null,
    mountedInArtboard: Boolean(mountedArtboard),
    bounds,
    viewportWidth: window.innerWidth,
    viewportHeight: window.innerHeight,
  });
}
