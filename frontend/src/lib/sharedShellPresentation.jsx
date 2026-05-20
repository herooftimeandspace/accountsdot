import React, { useEffect, useId, useRef, useState } from "react";
import * as lucideIcons from "lucide-static";
import { RuntimeDrawer } from "../components/RuntimeDrawer";
import { RuntimeSelectDropdown } from "../components/RuntimeDropdown";
import { sharedShellSpec } from "../generated/artboards.generated.js";
import { buildVisibleNavGroups, navDestinationForKey, visibleNavChildrenForKey } from "./routeRegistry";
import { helpContentForRoute } from "./routeHelpContent";

const SIDEBAR_TEMPLATE = {
  firstLabelY: 140,
  rowStep: 49,
  nestedRowFirstOffset: 30,
  nestedRowStep: 28,
  afterNestedGap: 14,
  iconX: 21,
  disclosureX: 224,
  labelX: 58,
  nestedDotX: 62,
  nestedLabelX: 82,
  highlightX: 12,
  highlightWidth: 236,
  highlightHeight: 42,
  nestedHighlightX: 38,
  nestedHighlightWidth: 210,
  nestedHighlightHeight: 28,
};

const NAV_ICON_MARKUP = {
  dashboard: lucideIcons.LayoutDashboard,
  onboarding: lucideIcons.UserRoundPlus,
  offboarding: lucideIcons.UserRoundMinus,
  departingSeniors: lucideIcons.GraduationCap,
  roomMoves: lucideIcons.MoveHorizontal,
  phoneDirectory: lucideIcons.Phone,
  dataQuality: lucideIcons.AlertTriangle,
  frequentFliers: lucideIcons.Plane,
  studentDataCleanup: lucideIcons.GraduationCap,
  reports: lucideIcons.FileBarChart2,
  admin: lucideIcons.Shield,
};

const NAV_LABELS = {
  dashboard: "Dashboard",
  onboarding: "Onboarding",
  offboarding: "Offboarding",
  departingSeniors: "Departing Seniors",
  roomMoves: "Room Moves",
  phoneDirectory: "Phone Directory",
  dataQuality: "Data Quality",
  frequentFliers: "Frequent Fliers",
  studentDataCleanup: "Student Data Cleanup",
  reports: "Reports",
  admin: "Admin",
};

const DEFAULT_STATIC_REFRESH_METADATA = "Last refreshed\nMay 3, 2026 9:00 AM PT";
const STATIC_PAGE_REFRESH_METADATA = {
  "dashboard-it-admin": DEFAULT_STATIC_REFRESH_METADATA,
  "dashboard-hr-lifecycle": DEFAULT_STATIC_REFRESH_METADATA,
  onboarding: DEFAULT_STATIC_REFRESH_METADATA,
  offboarding: DEFAULT_STATIC_REFRESH_METADATA,
  "room-moves": DEFAULT_STATIC_REFRESH_METADATA,
  "frequent-fliers": DEFAULT_STATIC_REFRESH_METADATA,
  "student-data-cleanup": DEFAULT_STATIC_REFRESH_METADATA,
  reports: DEFAULT_STATIC_REFRESH_METADATA,
  "reports-sync-transparency": DEFAULT_STATIC_REFRESH_METADATA,
};

const SCOPE_STATIC_NODE_IDS = [
  sharedShellSpec.sharedShellIds.scopeField,
  sharedShellSpec.sharedShellIds.scopeTitle,
  sharedShellSpec.sharedShellIds.scopeSubtitle,
  "p14",
  "p15",
  "p16",
  "p17",
  "p18",
  "p19",
  "p20",
  "p21",
];

function estimateTextHeight(node, textOverrides) {
  const content = String(textOverrides?.[node.id] ?? node.content ?? "");
  const fontSize = node.fontSize ?? 14;
  const width = node.width ?? Math.ceil(content.length * fontSize * 0.6);
  const approxCharsPerLine = Math.max(1, Math.floor(width / Math.max(fontSize * 0.58, 1)));
  const hardLines = content.split("\n");
  const lineCount = hardLines.reduce(
    (total, line) => total + Math.max(1, Math.ceil(Math.max(line.length, 1) / approxCharsPerLine)),
    0
  );
  return node.height ?? Math.ceil(lineCount * fontSize * 1.35);
}

function nodeBounds(node, textOverrides) {
  if (!node) {
    return null;
  }

  const width = node.width ?? 0;
  const height = node.type === "text" ? estimateTextHeight(node, textOverrides) : (node.height ?? 0);
  return {
    left: node.x ?? 0,
    top: node.y ?? 0,
    right: (node.x ?? 0) + width,
    bottom: (node.y ?? 0) + height,
    width,
    height,
  };
}

function mergeBounds(current, next) {
  if (!current) {
    return next;
  }
  if (!next) {
    return current;
  }
  return {
    left: Math.min(current.left, next.left),
    top: Math.min(current.top, next.top),
    right: Math.max(current.right, next.right),
    bottom: Math.max(current.bottom, next.bottom),
  };
}

function textContent(node, textOverrides = {}) {
  if (!node) {
    return "";
  }
  return String(textOverrides?.[node.id] ?? node.content ?? "");
}

function containsBounds(outer, inner) {
  if (!outer || !inner) {
    return false;
  }
  return (
    inner.left >= outer.left &&
    inner.right <= outer.right &&
    inner.top >= outer.top &&
    inner.bottom <= outer.bottom
  );
}

function findTopRightRefreshButtonBounds(nodeIndex, textOverrides = {}) {
  const nodes = Array.from(nodeIndex.values());
  const refreshTextNode = nodes
    .filter((node) => node.type === "text" && textContent(node, textOverrides).trim() === "Refresh")
    .sort((left, right) => {
      if ((left.y ?? 0) !== (right.y ?? 0)) {
        return (left.y ?? 0) - (right.y ?? 0);
      }
      return (right.x ?? 0) - (left.x ?? 0);
    })[0];

  if (!refreshTextNode) {
    return null;
  }

  const refreshTextBounds = nodeBounds(refreshTextNode, textOverrides);
  const refreshFrame = nodes
    .filter(
      (node) =>
        node.type === "frame" &&
        typeof node.fill === "string" &&
        node.fill.toUpperCase() === "#CEB770"
    )
    .filter((node) => {
      const bounds = nodeBounds(node, textOverrides);
      return containsBounds(
        {
          left: bounds.left - 4,
          top: bounds.top - 4,
          right: bounds.right + 4,
          bottom: bounds.bottom + 4,
        },
        refreshTextBounds
      );
    })
    .sort((left, right) => {
      const leftBounds = nodeBounds(left, textOverrides);
      const rightBounds = nodeBounds(right, textOverrides);
      const leftArea = (leftBounds.right - leftBounds.left) * (leftBounds.bottom - leftBounds.top);
      const rightArea = (rightBounds.right - rightBounds.left) * (rightBounds.bottom - rightBounds.top);
      return leftArea - rightArea;
    })[0];

  if (!refreshFrame) {
    return {
      left: refreshTextBounds.left - 28,
      top: refreshTextBounds.top - 11,
      right: refreshTextBounds.right + 28,
      bottom: refreshTextBounds.bottom + 11,
      width: refreshTextBounds.right - refreshTextBounds.left + 56,
      height: refreshTextBounds.bottom - refreshTextBounds.top + 22,
    };
  }

  return nodeBounds(refreshFrame, textOverrides);
}

function parseRefreshMetadata(value) {
  const normalized = String(value ?? "")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  if (normalized.length === 0) {
    return null;
  }

  const firstLine = normalized[0];
  const remainder = firstLine.replace(/^Last refreshed:?/i, "").trim();
  const detail = [remainder, ...normalized.slice(1)].filter(Boolean).join(" ");

  return {
    label: "Last refreshed",
    value: detail || "Recently updated",
  };
}

/**
 * normalizePageSyncControl prepares the page sync/refresh primitive for createSharedShellRenderOverlay.
 * React page components pass either passive freshness text or an intentional action contract; this helper
 * converts both shapes into one render model so callers do not hand-place Last refreshed, Refresh, or Sync now
 * controls. It does not mutate provider or DEV mock state itself; action side effects remain in the page callback.
 */
function normalizePageSyncControl(pageSyncControl, fallbackRefreshMetadata) {
  const source = pageSyncControl ?? (fallbackRefreshMetadata ? { lastRefreshed: fallbackRefreshMetadata } : null);
  if (!source) {
    return null;
  }

  const parsed = parseRefreshMetadata(source.lastRefreshed ?? source.refreshMetadata ?? fallbackRefreshMetadata);
  if (!parsed && !source.label) {
    return null;
  }

  const label = String(source.label ?? "").trim();
  const loadingLabel = String(source.loadingLabel ?? "").trim();
  const resolvedLabel = source.loading && loadingLabel ? loadingLabel : label;
  const actionName = String(source.ariaLabel ?? source.accessibleName ?? resolvedLabel).trim();
  const nextSyncText = String(source.nextSyncText ?? source.nextSync ?? "").trim();
  const hasAction = typeof source.onAction === "function";

  return {
    label: resolvedLabel,
    actionName: actionName || resolvedLabel || "Refresh page data",
    lastRefreshedLabel: source.lastRefreshedLabel ?? parsed?.label ?? "Last refreshed",
    lastRefreshedValue: source.lastRefreshedValue ?? parsed?.value ?? "",
    nextSyncText,
    disabled: Boolean(source.disabled || source.loading || (!hasAction && label)),
    loading: Boolean(source.loading),
    onAction: hasAction ? source.onAction : undefined,
    primary: source.primary !== false,
  };
}

/**
 * SharedShellPageSyncControl renders the shared runtime primitive for page-level freshness controls.
 * Generated .pen artboards still provide the canonical header geometry, while this overlay supplies the
 * real button semantics, disabled/loading state, and optional next-sync text used by pages such as Data
 * Quality and Student Data Cleanup. A click only calls the page-owned callback; the primitive never writes
 * to providers, databases, or DEV mock stores directly.
 */
function SharedShellPageSyncControl({ buttonBounds, pageSyncControl }) {
  const control = normalizePageSyncControl(pageSyncControl);
  if (!buttonBounds || !control) {
    return null;
  }

  const metadataWidth = control.nextSyncText ? 184 : 156;
  const buttonLabel = control.label;
  const usesStaticRefreshVisual = buttonLabel === "Refresh" && control.primary && !control.loading;
  const buttonAriaLabel = [
    control.actionName,
    control.lastRefreshedValue ? `${control.lastRefreshedLabel} ${control.lastRefreshedValue}` : "",
    control.nextSyncText,
  ]
    .filter(Boolean)
    .join(". ");

  return (
    <>
      {control.lastRefreshedValue ? (
        <div
          aria-hidden="true"
          className="shared-shell-page-sync__meta"
          style={{
            position: "absolute",
            left: buttonBounds.left - metadataWidth - 5,
            top: buttonBounds.top,
            width: metadataWidth,
            height: buttonBounds.height,
            zIndex: 24,
          }}
        >
          <span className="shared-shell-page-sync__label">{control.lastRefreshedLabel}</span>
          <span className="shared-shell-page-sync__value">{control.lastRefreshedValue}</span>
          {control.nextSyncText ? (
            <span className="shared-shell-page-sync__next">{control.nextSyncText}</span>
          ) : null}
        </div>
      ) : null}
      {buttonLabel ? (
        <button
          type="button"
          className={[
            "shared-shell-page-sync__button",
            control.primary ? "shared-shell-page-sync__button--primary" : "",
            usesStaticRefreshVisual ? "shared-shell-page-sync__button--static-visual" : "",
          ]
            .filter(Boolean)
            .join(" ")}
          aria-label={buttonAriaLabel}
          aria-busy={control.loading ? "true" : undefined}
          disabled={control.disabled}
          onClick={control.onAction}
          style={{
            position: "absolute",
            left: buttonBounds.left,
            top: buttonBounds.top,
            width: Math.max(buttonBounds.width, buttonLabel.length > 8 ? 104 : buttonBounds.width),
            height: buttonBounds.height,
            zIndex: 25,
          }}
        >
          {usesStaticRefreshVisual ? null : <span>{buttonLabel}</span>}
        </button>
      ) : null}
    </>
  );
}

export function deriveInitials(persona) {
  const explicit = String(persona?.initials ?? "").trim();
  if (explicit) {
    return explicit.slice(0, 2).toUpperCase();
  }

  const parts = String(persona?.display_name ?? "")
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  if (parts.length === 0) {
    return "??";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return `${parts[0][0] ?? ""}${parts[parts.length - 1][0] ?? ""}`.toUpperCase();
}

export function buildSharedShellTextOverrides(session) {
  if (!session?.current_persona) {
    return {};
  }

  const { sharedShellIds } = sharedShellSpec;
  const persona = session.current_persona;

  return {
    [sharedShellIds.scopeTitle]: session.shell?.scope_title ?? "",
    [sharedShellIds.scopeSubtitle]: session.shell?.scope_subtitle ?? "",
    [sharedShellIds.searchPlaceholder]: session.shell?.search_placeholder ?? "",
    [sharedShellIds.notificationCount]: session.shell?.notification_count ?? "",
    [sharedShellIds.initials]: deriveInitials(persona),
    [sharedShellIds.userName]: persona.display_name ?? "",
    [sharedShellIds.userRole]: persona.label ?? "",
    [sharedShellIds.platformStatusValue]: session.shell?.platform_status ?? "",
  };
}

export function buildSharedShellImageOverrides(session) {
  const profilePhotoUrl = session?.current_persona?.profile_photo_url;
  if (!profilePhotoUrl) {
    return {};
  }

  return {
    [sharedShellSpec.sharedShellIds.avatar]: profilePhotoUrl,
  };
}

export function staticRefreshMetadataForArtboard(artboardKey) {
  return STATIC_PAGE_REFRESH_METADATA[artboardKey] ?? null;
}

export function buildSharedShellHiddenNodeIds(session, options = {}) {
  const hiddenNodeIds = [];
  const visibleNavGroups = new Set(buildVisibleNavGroups(session));
  const {
    hideNavHighlight = false,
    hideSearchPlaceholder = false,
    hideAllNavGroups = false,
  } = options;

  Object.entries(sharedShellSpec.navGroups).forEach(([navKey, nodeIds]) => {
    if (hideAllNavGroups || !visibleNavGroups.has(navKey)) {
      hiddenNodeIds.push(...nodeIds);
    }
  });

  if (hideNavHighlight) {
    hiddenNodeIds.push(sharedShellSpec.sharedShellIds.navHighlight);
  }

  if (session?.current_persona?.profile_photo_url) {
    hiddenNodeIds.push(sharedShellSpec.sharedShellIds.initials);
  }

  if (hideSearchPlaceholder) {
    hiddenNodeIds.push(sharedShellSpec.sharedShellIds.searchPlaceholder);
  }

  hiddenNodeIds.push(...SCOPE_STATIC_NODE_IDS);

  return hiddenNodeIds;
}

function navLabelContent(navKey, nodeIndex, textOverrides) {
  if (NAV_LABELS[navKey]) {
    return NAV_LABELS[navKey];
  }
  const labelNode = nodeIndex.get(sharedShellSpec.navLabelIds[navKey]);
  if (!labelNode) {
    return navKey;
  }
  if (Object.prototype.hasOwnProperty.call(textOverrides, labelNode.id)) {
    return textOverrides[labelNode.id];
  }
  return labelNode.content ?? navKey;
}

function navIconMarkup(navKey) {
  const svg = NAV_ICON_MARKUP[navKey];
  if (!svg) {
    return "";
  }
  return svg
    .replace(/width="[^"]+"/, 'width="18"')
    .replace(/height="[^"]+"/, 'height="18"');
}

function sidebarRowMetrics(index, row = {}) {
  // `npm run pen:lint` checks that sidebarRowMetrics(index) remains the shared source for compact sidebar row geometry.
  const depth = row.depth ?? 0;
  const labelY = row.labelY ?? SIDEBAR_TEMPLATE.firstLabelY + index * SIDEBAR_TEMPLATE.rowStep;
  const highlightHeight = depth > 0 ? SIDEBAR_TEMPLATE.nestedHighlightHeight : SIDEBAR_TEMPLATE.highlightHeight;
  const rowCenter = labelY + (depth > 0 ? 7 : 9);
  return {
    labelY,
    rowCenter,
    iconTop: rowCenter - 9,
    disclosureTop: rowCenter - 4,
    dotTop: rowCenter - 3,
    highlightTop: rowCenter - highlightHeight / 2,
    highlightX: depth > 0 ? SIDEBAR_TEMPLATE.nestedHighlightX : SIDEBAR_TEMPLATE.highlightX,
    highlightWidth: depth > 0 ? SIDEBAR_TEMPLATE.nestedHighlightWidth : SIDEBAR_TEMPLATE.highlightWidth,
    highlightHeight,
    labelX: depth > 0 ? SIDEBAR_TEMPLATE.nestedLabelX : SIDEBAR_TEMPLATE.labelX,
    dotX: SIDEBAR_TEMPLATE.nestedDotX,
  };
}

function SharedShellSidebarRow({
  index,
  rowKey,
  depth = 0,
  navKey,
  destination,
  label,
  isActive,
  hasChildren = false,
  labelY,
  onNavigate,
}) {
  const metrics = sidebarRowMetrics(index, { depth, labelY });
  return (
    <React.Fragment key={rowKey}>
      {isActive ? (
        <div
          aria-hidden="true"
          className="shared-shell-nav__highlight"
          style={{
            position: "absolute",
            left: metrics.highlightX,
            top: metrics.highlightTop,
            width: metrics.highlightWidth,
            height: metrics.highlightHeight,
            zIndex: 66,
          }}
        />
      ) : null}
      {depth > 0 ? (
        <span
          aria-hidden="true"
          className={`shared-shell-nav__dot${isActive ? " shared-shell-nav__dot--active" : ""}`}
          style={{
            position: "absolute",
            left: metrics.dotX,
            top: metrics.dotTop,
            zIndex: 2,
          }}
        />
      ) : (
        <div
          aria-hidden="true"
          className="shared-shell-nav__icon"
          style={{
            position: "absolute",
            left: SIDEBAR_TEMPLATE.iconX,
            top: metrics.iconTop,
            zIndex: 2,
          }}
          dangerouslySetInnerHTML={{ __html: navIconMarkup(navKey) }}
        />
      )}
      <div
        aria-hidden="true"
        className={`shared-shell-nav__label${depth > 0 ? " shared-shell-nav__label--nested" : ""}`}
        style={{
          position: "absolute",
          left: metrics.labelX,
          top: metrics.labelY,
          zIndex: 2,
        }}
      >
        {label}
      </div>
      {hasChildren ? (
        <span
          aria-hidden="true"
          className="shared-shell-nav__disclosure"
          style={{
            position: "absolute",
            left: SIDEBAR_TEMPLATE.disclosureX,
            top: metrics.disclosureTop,
            zIndex: 2,
          }}
        />
      ) : null}
      <button
        type="button"
        className="pen-hotspot pen-hotspot--nav"
        aria-label={`Open ${label}`}
        title={`Open ${label}`}
        onClick={() => onNavigate(destination)}
        style={{
          position: "absolute",
          left: metrics.highlightX,
          top: metrics.highlightTop,
          width: metrics.highlightWidth,
          height: metrics.highlightHeight,
          border: 0,
          background: "transparent",
          padding: 0,
          cursor: "pointer",
          zIndex: 3,
        }}
      />
      {/* WCAG 2.4.4/4.1.2: visual sidebar labels are aria-hidden; this native button carries the name and role. */}
    </React.Fragment>
  );
}

/**
 * buildVisibleSidebarRows flattens role-authorized sidebar parents and
 * documented nested route buttons into one visual list. Route-backed page modes
 * can remain top-level-only when their page owns in-page mode controls, so the y
 * positions are calculated after route filtering instead of assuming every child
 * route renders in the sidebar.
 */
function buildVisibleSidebarRows(session) {
  const rows = [];
  let labelY = SIDEBAR_TEMPLATE.firstLabelY;
  buildVisibleNavGroups(session).forEach((navKey) => {
    const children = visibleNavChildrenForKey(navKey, session);
    const destination = navDestinationForKey(navKey, session);
    rows.push({
      key: navKey,
      navKey,
      depth: 0,
      labelY,
      destination,
      hasChildren: children.length > 0,
      hasChildWithSameDestination: children.some((child) => child.path === destination),
    });
    if (children.length === 0) {
      labelY += SIDEBAR_TEMPLATE.rowStep;
      return;
    }
    children.forEach((child, childIndex) => {
      rows.push({
        key: `${navKey}:${child.path}`,
        navKey,
        depth: 1,
        labelY: labelY + SIDEBAR_TEMPLATE.nestedRowFirstOffset + childIndex * SIDEBAR_TEMPLATE.nestedRowStep,
        destination: child.path,
        label: child.label,
      });
    });
    labelY +=
      SIDEBAR_TEMPLATE.nestedRowFirstOffset +
      children.length * SIDEBAR_TEMPLATE.nestedRowStep +
      SIDEBAR_TEMPLATE.afterNestedGap;
  });
  return rows.filter((row) => row.destination);
}

function sidebarRowActive(row, activeNavKey, activeRoutePath) {
  if (activeRoutePath) {
    // When a parent defaults to a documented child route, the child owns the selected visual state.
    if (row.depth === 0 && row.hasChildWithSameDestination) {
      return false;
    }
    if (row.destination === activeRoutePath) {
      return true;
    }
    return row.depth === 0 && !row.hasChildren && row.navKey === activeNavKey;
  }
  return row.depth === 0 && row.navKey === activeNavKey;
}

export function defaultScopeDropdownForSession(session) {
  const visibleSites = Array.isArray(session?.visible_sites) ? session.visible_sites : [];
  const isDistrictWide = String(session?.shell?.scope_title ?? "").toLowerCase().includes("district");
  const options = [
    ...(isDistrictWide ? [{ id: "district-wide", label: "District-wide" }] : []),
    ...visibleSites.map((site) => ({
      id: site.id,
      label: site.name,
    })),
  ];
  const fallbackSite = session?.current_site_id || session?.default_site_id || options[0]?.id || "current";
  const urlScope = typeof window === "undefined" ? "" : new URLSearchParams(window.location.search).get("site_id");
  const value = options.some((option) => option.id === urlScope)
    ? urlScope
    : isDistrictWide
      ? "district-wide"
      : fallbackSite;

  return {
    label: "Header scope",
    value,
    options: options.length
      ? options
      : [{ id: session?.shell?.scope_title ?? "current", label: session?.shell?.scope_title ?? "Current scope" }],
    onChange: (nextScope) => {
      if (typeof window === "undefined") {
        return;
      }
      const nextUrl = new URL(window.location.href);
      if (nextScope && nextScope !== "district-wide") {
        nextUrl.searchParams.set("site_id", nextScope);
      } else {
        nextUrl.searchParams.delete("site_id");
      }
      window.history.pushState({}, "", `${nextUrl.pathname}${nextUrl.search}`);
      const navigationEvent =
        typeof PopStateEvent === "function" ? new PopStateEvent("popstate") : new Event("popstate");
      window.dispatchEvent(navigationEvent);
    },
  };
}

function SharedShellSearchOverlay({ bounds, iconBounds, initialQuery, placeholder, onSearch }) {
  const [value, setValue] = useState(initialQuery ?? "");

  useEffect(() => {
    setValue(initialQuery ?? "");
  }, [initialQuery]);

  if (!bounds || typeof onSearch !== "function") {
    return null;
  }

  const iconInset = iconBounds ? Math.max(32, iconBounds.right - bounds.left + 14) : 42;

  return (
    // WCAG 1.3.1/3.3.2/4.1.2: shared search is exposed as a search landmark with a named input.
    <form
      className="shared-shell-search"
      style={{
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: Math.max(0, bounds.right - bounds.left),
        height: Math.max(0, bounds.bottom - bounds.top),
        zIndex: 2,
      }}
      role="search"
      onSubmit={(event) => {
        event.preventDefault();
        onSearch(value.trim());
      }}
    >
      <input
        type="search"
        className="shared-shell-search__input"
        aria-label="Search directory"
        placeholder={placeholder || "Search directory"}
        value={value}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key !== "Enter") {
            return;
          }
          event.preventDefault();
          onSearch(value.trim());
        }}
        style={{ paddingLeft: `${iconInset}px` }}
      />
    </form>
  );
}

export function SharedShellScopeDropdown({
  bounds,
  label = "Header scope",
  value = "",
  options = [],
  onChange = null,
}) {
  if (!bounds) {
    return null;
  }

  const normalizedOptions = options.length ? options : [{ id: value || "current", label: value || "Current scope" }];
  const selectedValue =
    normalizedOptions.some((option) => option.id === value) ? value : normalizedOptions[0]?.id ?? "";

  return (
    // WCAG 1.3.1/3.3.2/4.1.2: the shared scope selector exposes a named button/listbox control.
    <div
      className="shared-shell-scope-dropdown"
      style={{
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: Math.max(0, bounds.right - bounds.left),
        height: Math.max(0, bounds.bottom - bounds.top),
        zIndex: 3,
      }}
    >
      <RuntimeSelectDropdown
        label={label}
        value={selectedValue}
        options={normalizedOptions.map((option) => ({ value: option.id, label: option.label }))}
        onChange={(nextValue) => onChange?.(nextValue)}
        className="shared-shell-scope-dropdown__runtime"
        buttonClassName="shared-shell-scope-dropdown__select"
      />
    </div>
  );
}

/**
 * SharedShellHelpOverlay places the transparent shell help hotspot over the generated help icon and opens the same bounded right drawer used by runtime row-detail surfaces. It is called from createSharedShellRenderOverlay for every implemented page with a shared shell, so the page-specific operator help copy stays attached to the canonical shell primitive instead of each page inventing drawer placement.
 */
function SharedShellHelpOverlay({ bounds, helpContent }) {
  const [isOpen, setIsOpen] = useState(false);

  if (!bounds || !helpContent) {
    return null;
  }

  const sections = (helpContent.sections || []).map((section) => ({
    heading: section.heading,
    paragraphs: section.paragraphs || (section.body ? [section.body] : []),
  }));

  return (
    <>
      <button
        type="button"
        className="shared-shell-help-button"
        aria-expanded={isOpen}
        aria-label={`${isOpen ? "Close" : "Open"} help for ${helpContent.title}`}
        title={`${isOpen ? "Close" : "Open"} help for ${helpContent.title}`}
        onClick={() => setIsOpen((current) => !current)}
        style={{
          position: "absolute",
          left: bounds.left - 4,
          top: bounds.top - 4,
          width: Math.max(44, bounds.right - bounds.left + 8),
          height: Math.max(44, bounds.bottom - bounds.top + 8),
          zIndex: 4,
        }}
      />
      {isOpen ? (
        <RuntimeDrawer
          title={helpContent.title}
          onClose={() => setIsOpen(false)}
          className="shared-shell-help-runtime-drawer"
        >
          <div className="shared-shell-help-drawer">
            {sections.map((section) => (
              <section key={section.heading}>
                <h3>{section.heading}</h3>
                {section.paragraphs.map((paragraph) => (
                  <p key={paragraph}>{paragraph}</p>
                ))}
              </section>
            ))}
          </div>
        </RuntimeDrawer>
      ) : null}
    </>
  );
}

async function runDefaultDevLogout() {
  await fetch("/api/v1/dev/logout", {
    method: "POST",
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });
  window.location.assign("/login");
}

function SharedShellAccountMenu({ bounds, onNavigate, onLogout }) {
  const [isOpen, setIsOpen] = useState(false);
  const menuId = useId();
  const rootRef = useRef(null);

  useEffect(() => {
    if (!isOpen) {
      return undefined;
    }
    function handlePointerDown(event) {
      if (!rootRef.current?.contains(event.target)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [isOpen]);

  if (!bounds) {
    return null;
  }

  function closeAndNavigate(path) {
    setIsOpen(false);
    onNavigate?.(path);
  }

  async function closeAndLogout() {
    setIsOpen(false);
    if (typeof onLogout === "function") {
      await onLogout();
      return;
    }
    await runDefaultDevLogout();
  }

  return (
    <div
      ref={rootRef}
      className="shared-shell-account-menu"
      style={{
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: Math.max(0, bounds.right - bounds.left),
        height: Math.max(0, bounds.bottom - bounds.top),
        zIndex: 73,
      }}
    >
      <button
        type="button"
        className="shared-shell-account-menu__button"
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-controls={menuId}
        aria-label="Open account menu"
        onClick={() => setIsOpen((current) => !current)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            setIsOpen(false);
          }
        }}
      />
      {isOpen ? (
        <div id={menuId} className="shared-shell-account-menu__panel" role="menu" aria-label="Account menu">
          <button type="button" role="menuitem" onClick={() => closeAndNavigate("/my-profile")}>
            My Profile
          </button>
          <button type="button" role="menuitem" onClick={() => void closeAndLogout()}>
            Sign Out
          </button>
        </div>
      ) : null}
    </div>
  );
}

function defaultHelpContent(activeNavKey, activeRoutePath) {
  return helpContentForRoute(activeRoutePath, activeNavKey);
}

export function createSharedShellRenderOverlay({
  session,
  onNavigate,
  activeNavKey = null,
  activeRoutePath = null,
  onSearch = null,
  searchQuery = "",
  refreshMetadata = null,
  pageSyncControl = null,
  helpContent = null,
  scopeDropdown = null,
  onLogout = null,
}) {
  if (
    !session?.authenticated ||
    !session?.authorized ||
    (typeof onNavigate !== "function" && typeof onSearch !== "function")
  ) {
    return null;
  }

  const visibleSidebarRows = buildVisibleSidebarRows(session);

  return ({ nodeIndex, textOverrides = {} }) => {
    const searchBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.searchField), textOverrides);
    const searchIconBounds = nodeBounds(
      nodeIndex.get(sharedShellSpec.sharedShellIds.searchIcon),
      textOverrides
    );
    const scopeBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.scopeField), textOverrides);
    const refreshButtonBounds = findTopRightRefreshButtonBounds(nodeIndex, textOverrides);
    const resolvedPageSyncControl = normalizePageSyncControl(pageSyncControl, refreshMetadata);
    const helpIconBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.helpIcon), textOverrides);
    const accountBoxBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.accountBox), textOverrides);
    const resolvedHelpContent = helpContent ?? defaultHelpContent(activeNavKey, activeRoutePath);
    const resolvedScopeDropdown = scopeDropdown ?? defaultScopeDropdownForSession(session);

    return [
      <SharedShellPageSyncControl
        key="shared-shell-page-sync"
        buttonBounds={refreshButtonBounds}
        pageSyncControl={resolvedPageSyncControl}
      />,
      <SharedShellScopeDropdown
        key="shared-shell-scope"
        bounds={scopeBounds}
        label={resolvedScopeDropdown.label}
        value={resolvedScopeDropdown.value}
        options={resolvedScopeDropdown.options}
        onChange={resolvedScopeDropdown.onChange}
      />,
      <SharedShellSearchOverlay
        key="shared-shell-search"
        bounds={searchBounds}
        iconBounds={searchIconBounds}
        initialQuery={searchQuery}
        placeholder={session?.shell?.search_placeholder ?? ""}
        onSearch={onSearch}
      />,
      <SharedShellHelpOverlay
        key="shared-shell-help"
        bounds={helpIconBounds}
        helpContent={resolvedHelpContent}
      />,
      <SharedShellAccountMenu
        key="shared-shell-account-menu"
        bounds={accountBoxBounds}
        onNavigate={onNavigate}
        onLogout={onLogout}
      />,
      ...visibleSidebarRows.map((row, index) => {
        return (
          <SharedShellSidebarRow
            key={row.key}
            rowKey={row.key}
            index={index}
            depth={row.depth}
            labelY={row.labelY}
            navKey={row.navKey}
            destination={row.destination}
            label={row.label ?? navLabelContent(row.navKey, nodeIndex, textOverrides)}
            isActive={sidebarRowActive(row, activeNavKey, activeRoutePath)}
            hasChildren={row.hasChildren}
            onNavigate={onNavigate}
          />
        );
      }),
    ];
  };
}
