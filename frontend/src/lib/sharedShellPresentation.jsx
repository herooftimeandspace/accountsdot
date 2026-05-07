import React, { useEffect, useState } from "react";
import * as lucideIcons from "lucide-static";
import { sharedShellSpec } from "../generated/artboards.generated.js";
import { buildVisibleNavGroups, navDestinationForKey } from "./routeRegistry";

const SIDEBAR_TEMPLATE = {
  firstLabelY: 128,
  rowStep: 49,
  iconX: 21,
  labelX: 58,
  highlightX: 12,
  highlightWidth: 236,
  highlightHeight: 42,
};

const NAV_ICON_MARKUP = {
  dashboard: lucideIcons.LayoutDashboard,
  onboarding: lucideIcons.UserRoundPlus,
  offboarding: lucideIcons.UserRoundMinus,
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
  onboarding: "Staff Onboarding",
  offboarding: "Offboarding",
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
  "dashboard-site-admin": DEFAULT_STATIC_REFRESH_METADATA,
  onboarding: DEFAULT_STATIC_REFRESH_METADATA,
  offboarding: DEFAULT_STATIC_REFRESH_METADATA,
  "room-moves": DEFAULT_STATIC_REFRESH_METADATA,
  "frequent-fliers": DEFAULT_STATIC_REFRESH_METADATA,
  "student-data-cleanup": DEFAULT_STATIC_REFRESH_METADATA,
  reports: DEFAULT_STATIC_REFRESH_METADATA,
  "reports-sync-transparency": DEFAULT_STATIC_REFRESH_METADATA,
  "reports-ticketing-human-work": DEFAULT_STATIC_REFRESH_METADATA,
  admin: DEFAULT_STATIC_REFRESH_METADATA,
  "my-profile": DEFAULT_STATIC_REFRESH_METADATA,
};

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

  return {
    left: node.x ?? 0,
    top: node.y ?? 0,
    right: (node.x ?? 0) + (node.width ?? 0),
    bottom:
      (node.y ?? 0) +
      (node.type === "text" ? estimateTextHeight(node, textOverrides) : (node.height ?? 0)),
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

function SharedShellRefreshMetadataOverlay({ buttonBounds, refreshMetadata }) {
  const parsed = parseRefreshMetadata(refreshMetadata);
  if (!buttonBounds || !parsed) {
    return null;
  }

  const width = 156;

  return (
    <div
      aria-hidden="true"
      className="shared-shell-refresh-meta"
      style={{
        position: "absolute",
        left: buttonBounds.left - width - 5,
        top: buttonBounds.top,
        width,
        height: buttonBounds.height,
        zIndex: 2,
      }}
    >
      <span className="shared-shell-refresh-meta__label">{parsed.label}</span>
      <span className="shared-shell-refresh-meta__value">{parsed.value}</span>
    </div>
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

function sidebarRowMetrics(index) {
  const labelY = SIDEBAR_TEMPLATE.firstLabelY + index * SIDEBAR_TEMPLATE.rowStep;
  return {
    labelY,
    iconTop: labelY - 2,
    highlightTop: labelY - 10,
  };
}

function SharedShellSidebarRow({
  index,
  navKey,
  destination,
  label,
  isActive,
  onNavigate,
}) {
  const metrics = sidebarRowMetrics(index);
  return (
    <React.Fragment key={navKey}>
      {isActive ? (
        <div
          aria-hidden="true"
          className="shared-shell-nav__highlight"
          style={{
            position: "absolute",
            left: SIDEBAR_TEMPLATE.highlightX,
            top: metrics.highlightTop,
            width: SIDEBAR_TEMPLATE.highlightWidth,
            height: SIDEBAR_TEMPLATE.highlightHeight,
            zIndex: 0,
          }}
        />
      ) : null}
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
      <div
        aria-hidden="true"
        className="shared-shell-nav__label"
        style={{
          position: "absolute",
          left: SIDEBAR_TEMPLATE.labelX,
          top: metrics.labelY,
          zIndex: 2,
        }}
      >
        {label}
      </div>
      <button
        type="button"
        className="pen-hotspot pen-hotspot--nav"
        aria-label={`Open ${label}`}
        title={`Open ${label}`}
        onClick={() => onNavigate(destination)}
        style={{
          position: "absolute",
          left: SIDEBAR_TEMPLATE.highlightX,
          top: metrics.highlightTop,
          width: SIDEBAR_TEMPLATE.highlightWidth,
          height: SIDEBAR_TEMPLATE.highlightHeight,
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

export function createSharedShellRenderOverlay({
  session,
  onNavigate,
  activeNavKey = null,
  onSearch = null,
  searchQuery = "",
  refreshMetadata = null,
}) {
  if (
    !session?.authenticated ||
    !session?.authorized ||
    (typeof onNavigate !== "function" && typeof onSearch !== "function")
  ) {
    return null;
  }

  const visibleNavGroups = buildVisibleNavGroups(session);

  return ({ nodeIndex, textOverrides = {} }) => {
    const searchBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.searchField), textOverrides);
    const searchIconBounds = nodeBounds(
      nodeIndex.get(sharedShellSpec.sharedShellIds.searchIcon),
      textOverrides
    );
    const refreshButtonBounds = findTopRightRefreshButtonBounds(nodeIndex, textOverrides);

    return [
      <SharedShellRefreshMetadataOverlay
        key="shared-shell-refresh-meta"
        buttonBounds={refreshButtonBounds}
        refreshMetadata={refreshMetadata}
      />,
      <SharedShellSearchOverlay
        key="shared-shell-search"
        bounds={searchBounds}
        iconBounds={searchIconBounds}
        initialQuery={searchQuery}
        placeholder={session?.shell?.search_placeholder ?? ""}
        onSearch={onSearch}
      />,
      ...visibleNavGroups.map((navKey, index) => {
        const destination = navDestinationForKey(navKey, session);
        if (!destination) {
          return null;
        }
        return (
          <SharedShellSidebarRow
            key={navKey}
            index={index}
            navKey={navKey}
            destination={destination}
            label={navLabelContent(navKey, nodeIndex, textOverrides)}
            isActive={activeNavKey === navKey}
            onNavigate={onNavigate}
          />
        );
      }),
    ];
  };
}
