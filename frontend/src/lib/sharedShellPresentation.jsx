import React, { useEffect, useState } from "react";
import * as lucideIcons from "lucide-static";
import { RuntimeDrawer } from "../components/RuntimeDrawer";
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

const DEFAULT_HELP_BY_NAV_KEY = {
  dashboard: {
    title: "Dashboard help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This dashboard gives a quick view of current account, access, data quality, and workflow health.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Review the cards and tables for anything that needs attention. Select rows or use page actions when a page offers more detail.",
        ],
      },
    ],
  },
  onboarding: {
    title: "Onboarding help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page shows upcoming staff onboarding work. Each row is a person who needs accounts, access, rooms, or follow-up before they are fully ready.",
          "The status badge tells you whether the work is ready, running, waiting, missing information, or blocked.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Select a row to open details in the right drawer. The drawer explains what is happening and lists any action needed from HR, IT, or another system.",
          "Use Add Non-Escape Record when a contractor or other manual record needs onboarding before the person appears from Escape.",
        ],
      },
      {
        heading: "Warnings",
        paragraphs: [
          "A warning icon beside the start date means the start date is very close to the date the record was added. Some systems may not be ready by that date.",
          "Incomplete or blocked records need attention before normal onboarding can continue.",
        ],
      },
    ],
  },
  offboarding: {
    title: "Offboarding help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page tracks upcoming account retirement work, including accounts, licenses, devices, and security follow-up.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Review rows with blocked, manual action, or security risk statuses first. Select any row to open the right drawer, then follow the listed owner and resolution steps.",
          "Escape-backed end dates are read-only and must be corrected in Escape. Non-Escape and orphan account rows may show an end-date picker for HR and IT.",
        ],
      },
    ],
  },
  departingSeniors: {
    title: "Departing Seniors help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page lists seniors who are expected to leave at the end of the current school year.",
          "It shows each student, graduation year, student ID, district email, planned end date, and any outstanding devices from IncidentIQ.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Use the table search to find a student by name, email, graduation year, student ID, assigned asset serial, or assigned asset ID.",
          "IT and Device Wranglers can adjust a local end-date override, then deprovision the account when it is ready. A student stays on the list until the account is deprovisioned and all assigned devices are cleared.",
        ],
      },
    ],
  },
  roomMoves: {
    title: "Room Moves help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page helps review room moves and phone changes before scheduled cutover work runs.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Review warnings before scheduling a cutover. Rows marked for review need a person to resolve the warning before automation can safely continue.",
        ],
      },
    ],
  },
  phoneDirectory: {
    title: "Phone Directory help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page shows phone directory information by person, room, or department.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Use the mode buttons and filters to find the directory view you need. Select a result to see more detail when the page provides it.",
        ],
      },
    ],
  },
  dataQuality: {
    title: "Data Quality help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page lists data issues that can block or delay account and access work.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Start with high-severity issues and follow the next action listed for each row. Refresh when you need the latest DEV mock queue.",
        ],
      },
    ],
  },
  frequentFliers: {
    title: "Frequent Fliers help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page highlights people or devices that repeatedly need support attention.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Use the repeated patterns to decide where follow-up, cleanup, or prevention work may be needed.",
        ],
      },
    ],
  },
  studentDataCleanup: {
    title: "Student Data Cleanup help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page shows student data cleanup items that need correction in source systems.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Review the listed issue, update the source system named in the row, then return to confirm the cleanup is complete.",
        ],
      },
    ],
  },
  reports: {
    title: "Reports help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page collects operational reports for account, access, onboarding, offboarding, and sync work.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Choose the report that matches the question you need to answer. Use row details when available for follow-up context.",
        ],
      },
    ],
  },
  admin: {
    title: "Admin help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page shows administrator controls and health information for the DEV dashboard.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Use admin controls carefully and review warnings before changing shared settings or workflow behavior.",
        ],
      },
    ],
  },
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
    // WCAG 1.3.1/3.3.2/4.1.2: the shared scope selector is a native named form control.
    <label
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
      <span className="sr-only">{label}</span>
      <select
        className="shared-shell-scope-dropdown__select"
        aria-label={label}
        value={selectedValue}
        onChange={(event) => {
          onChange?.(event.target.value);
        }}
      >
        {normalizedOptions.map((option) => (
          <option key={option.id} value={option.id}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

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
        <RuntimeDrawer title={helpContent.title} onClose={() => setIsOpen(false)}>
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

function defaultHelpContent(activeNavKey) {
  if (activeNavKey && DEFAULT_HELP_BY_NAV_KEY[activeNavKey]) {
    return DEFAULT_HELP_BY_NAV_KEY[activeNavKey];
  }
  return {
    title: "Page help",
    sections: [
      {
        heading: "What this page shows",
        paragraphs: [
          "This page is part of The WIZARD staff dashboard and shows operational information for the current workflow.",
        ],
      },
      {
        heading: "How to use it",
        paragraphs: [
          "Review the visible rows, badges, and actions. If a row opens a drawer, use the drawer to see more detail and next steps.",
        ],
      },
    ],
  };
}

export function createSharedShellRenderOverlay({
  session,
  onNavigate,
  activeNavKey = null,
  onSearch = null,
  searchQuery = "",
  refreshMetadata = null,
  helpContent = null,
  scopeDropdown = null,
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
    const scopeBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.scopeField), textOverrides);
    const refreshButtonBounds = findTopRightRefreshButtonBounds(nodeIndex, textOverrides);
    const helpIconBounds = nodeBounds(nodeIndex.get(sharedShellSpec.sharedShellIds.helpIcon), textOverrides);
    const resolvedHelpContent = helpContent ?? defaultHelpContent(activeNavKey);
    const resolvedScopeDropdown = scopeDropdown ?? {
      label: "Header scope",
      value: session?.shell?.scope_title ?? "",
      options: [
        {
          id: session?.shell?.scope_title ?? "current",
          label: session?.shell?.scope_title ?? "Current scope",
        },
      ],
    };

    return [
      <SharedShellRefreshMetadataOverlay
        key="shared-shell-refresh-meta"
        buttonBounds={refreshButtonBounds}
        refreshMetadata={refreshMetadata}
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
