/**
 * resolveDevToolbarAnchorStyle converts the measured shared-sidebar and
 * Platform Status bounds into the fixed DEV persona switcher placement. The
 * React switcher calls this after the generated shell renders so the collapsed
 * pill follows the actual sidebar width instead of relying on a hard-coded
 * toolbar size.
 */
export function resolveDevToolbarAnchorStyle({
  sidebarLeft,
  sidebarRight,
  platformStatusBottom,
}) {
  if (
    !Number.isFinite(sidebarLeft) ||
    !Number.isFinite(sidebarRight) ||
    !Number.isFinite(platformStatusBottom) ||
    sidebarRight <= sidebarLeft
  ) {
    return null;
  }

  const sidebarWidth = Math.max(0, sidebarRight - sidebarLeft);
  const horizontalPadding = sidebarWidth >= 180 ? 16 : 8;
  const anchoredWidth = sidebarWidth - horizontalPadding * 2;
  if (anchoredWidth <= 0) {
    return null;
  }

  return {
    left: `${Math.round(sidebarLeft + horizontalPadding)}px`,
    top: `${Math.max(0, Math.round(platformStatusBottom + 8))}px`,
    width: `${Math.round(anchoredWidth)}px`,
  };
}
