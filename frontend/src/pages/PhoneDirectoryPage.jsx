import { useEffect, useMemo, useRef, useState } from "react";
import { AccessDenied } from "../components/AccessDenied";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import {
  RuntimeSortableHeader,
  RuntimeTableSearch,
  runtimeTableColumnValue,
  useRuntimeTableData,
} from "../components/RuntimeTableControls";
import { sharedShellSpec } from "../generated/artboards.generated.js";
import { PenArtboard } from "../lib/PenArtboard";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const MODE_CONFIG = {
  person: {
    endpoint: "/api/v1/dev/pages/phone-directory/by-person",
    artboardKey: "phone-directory-by-person",
    descriptionId: "t84",
    lastRefreshedId: "t85",
    searchFieldId: "f95",
    searchIconId: "p97",
    searchPlaceholderId: "t98",
    resultsFrameId: "f110",
    detailRailId: "f159",
    hiddenStaticNodeIds: [
      "f99",
      "t100",
      "f102",
      "t103",
      "f105",
      "t106",
      "f108",
      "t109",
      "t111",
      "t112",
      "t113",
      "t114",
      "t115",
      "t116",
      "t117",
      "t118",
      "l119",
      "t120",
      "t121",
      "t122",
      "t123",
      "t124",
      "f125",
      "t126",
      "t127",
      "t128",
      "l129",
      "t130",
      "t131",
      "t132",
      "t133",
      "t134",
      "f135",
      "t136",
      "t137",
      "l138",
      "t139",
      "t140",
      "t141",
      "t142",
      "t143",
      "f144",
      "t145",
      "t146",
      "l147",
      "t148",
      "t149",
      "t150",
      "t151",
      "t152",
      "f153",
      "t154",
      "t155",
      "t156",
      "l157",
      "t158",
      "t160",
      "t163",
      "t164",
      "t165",
      "t166",
      "t167",
      "t168",
      "t169",
      "t170",
      "t171",
      "t172",
      "l173",
      "f174",
      "t175",
      "f176",
      "t177",
      "f178",
      "t179",
      "t180",
      "t181",
      "t182",
      "t183",
    ],
  },
  room: {
    endpoint: "/api/v1/dev/pages/phone-directory/by-room",
    artboardKey: "phone-directory-by-room",
    descriptionId: "t84",
    lastRefreshedId: "t85",
    searchFieldId: "f95",
    searchIconId: "p97",
    searchPlaceholderId: "t98",
    resultsFrameId: "f110",
    detailRailId: "f153",
    hiddenStaticNodeIds: [
      "f99",
      "t100",
      "f102",
      "t103",
      "f105",
      "t106",
      "f108",
      "t109",
      "t111",
      "t112",
      "t113",
      "t114",
      "t115",
      "t116",
      "t117",
      "l118",
      "t119",
      "t120",
      "t121",
      "t122",
      "f123",
      "t124",
      "t125",
      "t126",
      "l127",
      "t128",
      "t129",
      "t130",
      "t131",
      "f132",
      "t133",
      "t134",
      "l135",
      "t136",
      "t137",
      "t138",
      "t139",
      "f140",
      "t141",
      "t142",
      "l143",
      "t144",
      "t145",
      "t146",
      "t147",
      "f148",
      "t149",
      "t150",
      "l151",
      "t152",
      "t154",
      "t157",
      "t158",
      "t159",
      "t160",
      "t161",
      "t162",
      "t163",
      "t164",
      "t165",
      "t166",
      "l167",
      "f168",
      "t169",
      "f170",
      "t171",
      "f172",
      "t173",
      "t174",
      "t175",
      "t176",
    ],
  },
  department: {
    endpoint: "/api/v1/dev/pages/phone-directory/by-department",
    artboardKey: "phone-directory-by-department",
    descriptionId: "t84",
    lastRefreshedId: "t85",
    searchFieldId: "f101",
    searchIconId: "p103",
    searchPlaceholderId: "t104",
    resultsFrameId: "f116",
    detailRailId: "f166",
    hiddenStaticNodeIds: [
      "f95",
      "p96",
      "p97",
      "t99",
      "f101",
      "p103",
      "t104",
      "f105",
      "t106",
      "f108",
      "t109",
      "f111",
      "t112",
      "f114",
      "t115",
      "t117",
      "t118",
      "t119",
      "t120",
      "t121",
      "t122",
      "t123",
      "l124",
      "t125",
      "t126",
      "t127",
      "t128",
      "t129",
      "t130",
      "t131",
      "f132",
      "t133",
      "l134",
      "t135",
      "t136",
      "t137",
      "t138",
      "t139",
      "t140",
      "t141",
      "t142",
      "f143",
      "t144",
      "l145",
      "t146",
      "t147",
      "t148",
      "t149",
      "t150",
      "f151",
      "t152",
      "l153",
      "t154",
      "t155",
      "t156",
      "t157",
      "t158",
      "t159",
      "t160",
      "t161",
      "f162",
      "t163",
      "l164",
      "t165",
      "t167",
      "f168",
      "t169",
      "t172",
      "t173",
      "t174",
      "t175",
      "t176",
      "t177",
      "t178",
      "t179",
      "t180",
      "t181",
      "t182",
      "t183",
      "t184",
      "t185",
      "t186",
      "l187",
      "t188",
      "t189",
      "t190",
      "l191",
      "f192",
      "t193",
      "f194",
      "t195",
      "f196",
      "t197",
      "t198",
      "t199",
      "t200",
    ],
  },
};

const MODE_BUTTONS = [
  { mode: "person", label: "By Person", buttonId: "f89", labelId: "t90" },
  { mode: "room", label: "By Room", buttonId: "f91", labelId: "t92" },
  { mode: "department", label: "By Department", buttonId: "f93", labelId: "t94" },
];
const PHONE_DIRECTORY_HEADING_ID = "phone-directory-heading";
const PHONE_DIRECTORY_RESULTS_TOP = 236;
const PHONE_DIRECTORY_RESULTS_BOTTOM_PADDING = 64;
const MODE_PAGE_TITLES = {
  person: "Phone Directory by Person",
  room: "Phone Directory by Room",
  department: "Phone Directory by Department",
};
const PHONE_COLUMN = {
  key: "phone",
  label: "Phone",
  value: (result) => result.phone || "—",
  render: (result) => renderTelLink(result.phone),
};
const EXTENSION_COLUMN = {
  key: "extension",
  label: "Extension",
  value: (result) => result.extension || "—",
  render: (result) => renderTelLink(result.extension),
};
const SHARED_LINE_RESULT_COLUMNS = [
  PHONE_COLUMN,
  EXTENSION_COLUMN,
  { key: "site", label: "Site", render: (result) => result.site_name || "—" },
  { key: "type", label: "Type", pill: true, render: (result) => result.type_label },
];

/**
 * telHrefForDirectoryValue creates the dialable href used by Phone Directory table cells and the detail drawer. Project docs require numeric-only extensions and allow 4-, 5-, and 6-digit internal dialing, so the safest documented extension format is `tel:<digits>`; formatted phone numbers use the same digit-only href form after punctuation is stripped.
 */
function telHrefForDirectoryValue(value) {
  const digits = String(value ?? "").replace(/\D/g, "");
  return digits ? `tel:${digits}` : "";
}

/**
 * renderTelLink turns a phone or extension display value into an accessible link while preserving the source display text. Empty values stay as the muted dash used by other directory cells, and click/keyboard activation does not select the surrounding result row.
 */
function renderTelLink(value) {
  const displayValue = String(value ?? "").trim();
  const href = telHrefForDirectoryValue(displayValue);
  if (!displayValue || !href) {
    return "—";
  }

  return (
    <a
      className="phone-directory-runtime__tel-link"
      href={href}
      onClick={(event) => event.stopPropagation()}
      onKeyDown={(event) => event.stopPropagation()}
    >
      {displayValue}
    </a>
  );
}

/**
 * paneNodeId documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function paneNodeId(artboardKey, baseId) {
  return `${artboardKey}__${baseId}`;
}

/**
 * clone documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

/**
 * uniquifyNodeIds documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function uniquifyNodeIds(artboard) {
  const seen = new Map();

  /**
   * visit documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
   */
  function visit(node) {
    const count = (seen.get(node.id) ?? 0) + 1;
    seen.set(node.id, count);
    if (count > 1) {
      node.id = `${node.id}__dup${count}`;
    }
    for (const child of node.children || []) {
      visit(child);
    }
  }

  visit(artboard);
  return artboard;
}

/**
 * buildNodeIndex builds derived data for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function buildNodeIndex(node, map = new Map()) {
  map.set(node.id, node);
  for (const child of node.children || []) {
    buildNodeIndex(child, map);
  }
  return map;
}

/**
 * descendantIds documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function descendantIds(node) {
  const ids = [];
  for (const child of node?.children || []) {
    ids.push(child.id, ...descendantIds(child));
  }
  return ids;
}

/**
 * pushDuplicateIds documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function pushDuplicateIds(target, nodeIndex, baseId) {
  if (!baseId) {
    return;
  }
  for (const id of nodeIndex.keys()) {
    if (id === baseId || id.startsWith(`${baseId}__dup`)) {
      target.push(id);
    }
  }
}

/**
 * resolvePaneId builds derived data for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function resolvePaneId(nodeIndex, config, baseId) {
  const prefixedId = paneNodeId(config.artboardKey, baseId);
  if (nodeIndex.has(prefixedId)) {
    return prefixedId;
  }
  return baseId;
}

/**
 * nodeBounds documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function nodeBounds(node) {
  if (!node) {
    return null;
  }
  return {
    left: node.x ?? 0,
    top: node.y ?? 0,
    width: node.width ?? 0,
    height: node.height ?? 0,
  };
}

/**
 * phoneDirectoryResultsBounds documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function phoneDirectoryResultsBounds(bounds) {
  if (!bounds) {
    return null;
  }
  return {
    ...bounds,
    top: PHONE_DIRECTORY_RESULTS_TOP,
    height: Math.max(bounds.height, bounds.top + bounds.height - PHONE_DIRECTORY_RESULTS_TOP),
  };
}

/**
 * phoneDirectoryExpandedArtboardHeight documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function phoneDirectoryExpandedArtboardHeight(baseHeight, resultCount) {
  const rows = Math.max(1, resultCount || 0);
  const tableChromeHeight = 104;
  const rowHeight = 48;
  return Math.max(
    baseHeight,
    PHONE_DIRECTORY_RESULTS_TOP +
      14 +
      tableChromeHeight +
      rows * rowHeight +
      PHONE_DIRECTORY_RESULTS_BOTTOM_PADDING
  );
}

/**
 * boundsIntersect documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function boundsIntersect(a, b, tolerance = 1) {
  if (!a || !b) {
    return false;
  }
  return !(
    a.left + a.width < b.left - tolerance ||
    b.left + b.width < a.left - tolerance ||
    a.top + a.height < b.top - tolerance ||
    b.top + b.height < a.top - tolerance
  );
}

/**
 * canViewEmployeeId documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function canViewEmployeeId(session) {
  const personaId = session?.current_persona?.id;
  return (
    personaId === "it_admin" ||
    personaId === "site_admin" ||
    personaId === "human_resources"
  );
}

/**
 * sharedLineColumns documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function sharedLineColumns(titleLabel) {
  return [
    { key: "title", label: titleLabel, render: (result) => result.title, primary: true },
    ...SHARED_LINE_RESULT_COLUMNS,
  ];
}

/**
 * resultsColumnsForMode documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function resultsColumnsForMode(mode) {
  switch (mode) {
    case "room":
      return sharedLineColumns("Room or Line");
    case "department":
      return sharedLineColumns("Department or Line");
    default:
      return [
        { key: "title", label: "Name or Line", render: (result) => result.title, primary: true },
        {
          key: "details",
          label: "Details",
          render: (result) => result.email || result.location || result.department || "—",
          secondary: (result) => result.role || "",
        },
        { key: "room", label: "Room", render: (result) => result.location || "—" },
        PHONE_COLUMN,
        EXTENSION_COLUMN,
        { key: "site", label: "Site", render: (result) => result.site_name || "—" },
        { key: "type", label: "Type", pill: true, render: (result) => result.type_label },
      ];
  }
}

/**
 * resultSummary documents runtime data flow for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function resultSummary(result, columns) {
  return columns
    .map((column) => `${column.label}: ${runtimeTableColumnValue(result, column, "search") || "none"}`)
    .join("; ");
}

/**
 * PhoneDirectoryResultsOverlay renders the UI surface for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function PhoneDirectoryResultsOverlay({
  bounds,
  mode,
  results,
  selectedResultId,
  onSelect,
}) {
  if (!bounds) {
    return null;
  }

  const columns = resultsColumnsForMode(mode);
  const table = useRuntimeTableData(results, columns, {
    defaultSort: { key: null, direction: "none" },
  });
  const resultsTitleId = `phone-directory-${mode}-results-title`;

  return (
    <section
      className="phone-directory-runtime__results"
      style={{
        position: "absolute",
        left: bounds.left + 18,
        top: bounds.top + 14,
        width: Math.max(0, bounds.width - 36),
        height: Math.max(0, bounds.height - 28),
        zIndex: 2,
      }}
      aria-live="polite"
      aria-labelledby={resultsTitleId}
    >
      {/* WCAG 1.3.1/2.4.6/4.1.3: dynamic results keep a programmatic heading and polite updates. */}
      <h2 id={resultsTitleId} className="sr-only">
        Phone Directory Results
      </h2>
      {results.length > 0 ? (
        <div className={`phone-directory-runtime__table phone-directory-runtime__table--${mode}`}>
          <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
          <div className="phone-directory-runtime__table-header">
            {columns.map((column) => (
              <div key={column.key}>
                <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
              </div>
            ))}
          </div>
          {table.visibleRows.map((result) => (
            <div
              key={result.id}
              className={`phone-directory-runtime__row phone-directory-runtime__row--${mode} ${
                result.id === selectedResultId ? "phone-directory-runtime__row--selected" : ""
              }`}
              data-selected={result.id === selectedResultId ? "true" : "false"}
            >
              {columns.map((column) => {
                const primaryValue = column.render(result);
                if (column.pill) {
                  return (
                    <div key={column.key}>
                      <span className="phone-directory-runtime__pill">{primaryValue}</span>
                    </div>
                  );
                }
                if (column.primary) {
                  return (
                    <div key={column.key}>
                      <button
                        type="button"
                        className="phone-directory-runtime__row-action"
                        aria-label={`View details for ${resultSummary(result, columns)}`}
                        aria-pressed={result.id === selectedResultId}
                        onClick={() => onSelect(result.id)}
                      >
                        <span className="phone-directory-runtime__primary">{primaryValue}</span>
                      </button>
                    </div>
                  );
                }

                const secondaryValue = typeof column.secondary === "function" ? column.secondary(result) : "";
                return (
                  <div key={column.key}>
                    <div>{primaryValue}</div>
                    {secondaryValue ? (
                      <div className="phone-directory-runtime__secondary">{secondaryValue}</div>
                    ) : null}
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      ) : (
        <p className="phone-directory-runtime__empty">
          No directory matches were found for this search.
        </p>
      )}
    </section>
  );
}

/**
 * PhoneDirectoryDetailOverlay renders the UI surface for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function PhoneDirectoryDetailOverlay({ bounds, mode, result, session, onClose }) {
  if (!bounds || !result) {
    return null;
  }

  return (
    <RuntimeDrawer title={result.title} bounds={bounds} onClose={onClose} className="phone-directory-runtime__drawer">
      <div className="phone-directory-runtime__detail">
        <p>{result.site_name}</p>
        <div className="phone-directory-runtime__detail-card">
          <RuntimeDetailList
            items={[
              { label: "Type", value: result.type_label },
              { label: "Role", value: result.role },
              { label: "Department", value: result.department },
              { label: mode === "room" ? "Area" : "Room", value: result.location },
              { label: "Phone", value: renderTelLink(result.phone) },
              { label: "Extension", value: renderTelLink(result.extension) },
              { label: "Email", value: result.email },
              canViewEmployeeId(session) ? { label: "ID", value: result.identifier } : null,
            ]}
          />
        </div>
      </div>
    </RuntimeDrawer>
  );
}

/**
 * PhoneDirectoryModeToggleOverlay renders the UI surface for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function PhoneDirectoryModeToggleOverlay({ nodeIndex, config, activeMode, searchQuery, onNavigate }) {
  const buttons = MODE_BUTTONS.map((button) => {
    const bounds = nodeBounds(nodeIndex.get(resolvePaneId(nodeIndex, config, button.buttonId)));
    if (!bounds) {
      return null;
    }

    const href = `/phone-directory/by-${button.mode}${searchQuery.trim() ? `?q=${encodeURIComponent(searchQuery.trim())}` : ""}`;
    return (
      <button
        key={button.mode}
        type="button"
        className={`phone-directory-runtime__mode-button ${
          activeMode === button.mode ? "phone-directory-runtime__mode-button--active" : ""
        }`}
        aria-current={activeMode === button.mode ? "page" : undefined}
        aria-label={`Show phone directory ${button.label.toLowerCase()}`}
        style={{
          position: "absolute",
          left: bounds.left,
          top: bounds.top,
          width: bounds.width,
          height: bounds.height,
          zIndex: 3,
        }}
        onClick={() => onNavigate(href)}
      >
        {button.label}
      </button>
    );
  });

  return buttons.filter(Boolean);
}

/**
 * buildTextOverrides builds derived data for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function buildTextOverrides(session, payload, config, searchQuery) {
  const overrides = buildSharedShellTextOverrides(session);
  if (!payload) {
    return overrides;
  }

  overrides[paneNodeId(config.artboardKey, config.descriptionId)] = payload.page.description;
  overrides[paneNodeId(config.artboardKey, config.lastRefreshedId)] = payload.page.last_refreshed;
  overrides[paneNodeId(config.artboardKey, config.searchPlaceholderId)] =
    searchQuery.trim() || "Search by name, classification, or extension...";
  return overrides;
}

/**
 * buildHiddenNodeIds builds derived data for frontend/src/pages/PhoneDirectoryPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function buildHiddenNodeIds(session, artboard, nodeIndex, config) {
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });

  if (!artboard || !config || nodeIndex.size === 0) {
    return hiddenNodeIds;
  }

  for (const button of MODE_BUTTONS) {
    pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, button.buttonId));
    pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, button.labelId));
  }

  pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, config.descriptionId));
  pushDuplicateIds(hiddenNodeIds, nodeIndex, sharedShellSpec.sharedShellIds.scopeField);
  pushDuplicateIds(hiddenNodeIds, nodeIndex, sharedShellSpec.sharedShellIds.scopeTitle);
  pushDuplicateIds(hiddenNodeIds, nodeIndex, sharedShellSpec.sharedShellIds.scopeSubtitle);
  pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, config.searchFieldId));
  pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, config.searchIconId));
  pushDuplicateIds(
    hiddenNodeIds,
    nodeIndex,
    resolvePaneId(nodeIndex, config, config.searchPlaceholderId)
  );
  pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, config.lastRefreshedId));
  pushDuplicateIds(hiddenNodeIds, nodeIndex, resolvePaneId(nodeIndex, config, config.detailRailId));

  const resultsFrame = nodeIndex.get(resolvePaneId(nodeIndex, config, config.resultsFrameId));
  const detailRail = nodeIndex.get(resolvePaneId(nodeIndex, config, config.detailRailId));
  const searchField = nodeIndex.get(resolvePaneId(nodeIndex, config, config.searchFieldId));
  const modeButtonBounds = MODE_BUTTONS.map((button) =>
    nodeBounds(nodeIndex.get(resolvePaneId(nodeIndex, config, button.buttonId)))
  ).filter(Boolean);
  if (resultsFrame?.id) {
    hiddenNodeIds.push(resultsFrame.id);
  }
  hiddenNodeIds.push(...descendantIds(resultsFrame), ...descendantIds(detailRail));
  hiddenNodeIds.push(
    ...(config.hiddenStaticNodeIds ?? []).map((nodeId) => resolvePaneId(nodeIndex, config, nodeId))
  );

  const resultsBounds = nodeBounds(resultsFrame);
  const detailBounds = nodeBounds(detailRail);
  const searchBounds = nodeBounds(searchField);
  const preservedRootIds = new Set([resolvePaneId(nodeIndex, config, config.resultsFrameId)]);
  for (const child of artboard.children || []) {
    if (preservedRootIds.has(child.id)) {
      continue;
    }
    const childBounds = nodeBounds(child);
    if (
      boundsIntersect(childBounds, resultsBounds) ||
      boundsIntersect(childBounds, detailBounds) ||
      boundsIntersect(childBounds, searchBounds) ||
      modeButtonBounds.some((bounds) => boundsIntersect(childBounds, bounds))
    ) {
      hiddenNodeIds.push(child.id, ...descendantIds(child));
    }
  }

  return hiddenNodeIds;
}

/**
 * selectedResultForPayload returns the row object that should populate the right-hand directory detail rail. It reads the current API payload plus the selected row id and returns null when the mode changes, the payload is empty, or the previous selection is no longer present.
 */
function selectedResultForPayload(payload, selectedResultId) {
  const results = payload?.page?.results ?? [];
  if (!results.length || !selectedResultId) {
    return null;
  }
  return results.find((result) => result.id === selectedResultId) ?? null;
}

/**
 * PhoneDirectoryPage owns the runtime search/sort/detail overlays for the three generated Phone Directory artboards. It combines route mode, DEV session context, query parameters, generated artboard readiness, and `/api/v1/dev/phone-directory/*` payloads into the table controls and selected-row detail rail.
 */
export function PhoneDirectoryPage({
  session,
  mode,
  artboardKey,
  onNavigate,
  onSearch,
  searchQuery = "",
  currentSearch = "",
  onUnauthorized,
  onForbidden,
}) {
  const modeConfig = MODE_CONFIG[mode];
  const [pageState, setPageState] = useState("loading");
  const [payload, setPayload] = useState(null);
  const [errorMessage, setErrorMessage] = useState("");
  const [selectedResultId, setSelectedResultId] = useState("");
  const activeRequestKeyRef = useRef("");
  const { artboard: baseArtboard, status: artboardStatus } = useGeneratedArtboard(artboardKey);

  useEffect(() => {
    setSelectedResultId("");
  }, [mode]);

  const resultCount = payload?.page?.results?.length ?? 0;
  const artboard = useMemo(() => {
    if (!baseArtboard) {
      return null;
    }
    const nextArtboard = uniquifyNodeIds(clone(baseArtboard));
    nextArtboard.height = phoneDirectoryExpandedArtboardHeight(nextArtboard.height, resultCount);
    return nextArtboard;
  }, [baseArtboard, resultCount]);
  const nodeIndex = useMemo(() => artboard ? buildNodeIndex(artboard) : new Map(), [artboard]);
  const requestKey = useMemo(() => {
    const params = new URLSearchParams(currentSearch);
    return JSON.stringify({
      endpoint: modeConfig?.endpoint ?? "",
      personaId: session?.current_persona?.id ?? "",
      q: searchQuery.trim(),
      siteId: params.get("site_id")?.trim() ?? "",
    });
  }, [currentSearch, modeConfig?.endpoint, searchQuery, session?.current_persona?.id]);

  useEffect(() => {
    if (!session?.authenticated || !session?.authorized || !modeConfig) {
      return undefined;
    }

    const controller = new AbortController();
    activeRequestKeyRef.current = requestKey;
    const request = JSON.parse(requestKey);

    /**
     * loadPage fetches the active directory mode JSON payload and ignores responses that belong to stale mode/site/query requests. It translates authentication failures into shared app navigation callbacks and stores successful payloads for the runtime table overlay.
     */
    async function loadPage() {
      setPageState("loading");
      setErrorMessage("");
      try {
        const requestUrl = new URL(modeConfig.endpoint, window.location.origin);
        if (request.q) {
          requestUrl.searchParams.set("q", request.q);
        }
        if (request.siteId) {
          requestUrl.searchParams.set("site_id", request.siteId);
        }

        const response = await fetch(requestUrl, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          signal: controller.signal,
        });
        if (response.status === 401) {
          onUnauthorized?.();
          return;
        }
        if (response.status === 403) {
          onForbidden?.();
          return;
        }
        if (!response.ok) {
          throw new Error(`Phone Directory request failed with ${response.status}`);
        }
        const nextPayload = await response.json();
        if (controller.signal.aborted || activeRequestKeyRef.current !== requestKey) {
          return;
        }
        setPayload(nextPayload);
        setSelectedResultId((current) => {
          const results = nextPayload?.page?.results ?? [];
          if (!results.length) {
            return "";
          }

          if (current && results.some((result) => result.id === current)) {
            return current;
          }

          const preferredResultId = nextPayload?.page?.selected_result?.id ?? "";
          if (preferredResultId && results.some((result) => result.id === preferredResultId)) {
            return preferredResultId;
          }

          return "";
        });
        setPageState("ready");
      } catch (error) {
        if (controller.signal.aborted || activeRequestKeyRef.current !== requestKey) {
          return;
        }
        setPayload(null);
        setErrorMessage(
          error instanceof Error ? error.message : "Unable to load Phone Directory results."
        );
        setPageState("error");
      }
    }

    void loadPage();
    return () => controller.abort();
  }, [modeConfig, onForbidden, onUnauthorized, requestKey, session?.authenticated, session?.authorized]);

  const textOverrides = useMemo(
    () => buildTextOverrides(session, payload, modeConfig, searchQuery),
    [modeConfig, payload, searchQuery, session]
  );
  const hiddenNodeIds = useMemo(
    () => buildHiddenNodeIds(session, artboard, nodeIndex, modeConfig),
    [artboard, modeConfig, nodeIndex, session]
  );
  const imageNodeOverrides = useMemo(() => buildSharedShellImageOverrides(session), [session]);
  const sharedShellOverlay = useMemo(
    () =>
      createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: "phoneDirectory",
        refreshMetadata: payload?.page?.last_refreshed ?? null,
        // WCAG 1.3.1/3.3.2/4.1.2: the DEV directory scope selector reuses the shared native dropdown primitive with an accessible label.
        scopeDropdown: {
          label: "Directory scope",
          value: payload?.page?.directory_scope_id ?? "district-wide",
          options: payload?.page?.directory_scope_options ?? [
            { id: "district-wide", label: "District-wide" },
          ],
          onChange: (nextScope) => {
            const params = new URLSearchParams();
            if (searchQuery.trim()) {
              params.set("q", searchQuery.trim());
            }
            params.set("site_id", nextScope);
            onNavigate(`/phone-directory/by-${mode}?${params.toString()}`);
          },
        },
      }),
    [
      mode,
      onNavigate,
      onSearch,
      payload?.page?.directory_scope_id,
      payload?.page?.directory_scope_options,
      payload?.page?.last_refreshed,
      searchQuery,
      session,
    ]
  );

  const renderOverlay = useMemo(() => {
    return (args) => {
      const shellElements =
        typeof sharedShellOverlay === "function" ? sharedShellOverlay(args) ?? [] : [];
      if (pageState !== "ready") {
        return shellElements;
      }
      const detailBounds = nodeBounds(
        nodeIndex.get(resolvePaneId(nodeIndex, modeConfig, modeConfig.detailRailId))
      );
      const resultsBounds = nodeBounds(
        nodeIndex.get(resolvePaneId(nodeIndex, modeConfig, modeConfig.resultsFrameId))
      );
      const runtimeResultsBounds = phoneDirectoryResultsBounds(resultsBounds);
      const results = payload?.page?.results ?? [];
      const selected = selectedResultForPayload(payload, selectedResultId);

      return [
        ...shellElements,
        ...PhoneDirectoryModeToggleOverlay({
          nodeIndex,
          config: modeConfig,
          activeMode: mode,
          searchQuery,
          onNavigate,
        }),
        <PhoneDirectoryResultsOverlay
          key="phone-directory-results"
          bounds={runtimeResultsBounds}
          mode={mode}
          results={results}
          selectedResultId={selectedResultId}
          onSelect={setSelectedResultId}
        />,
        <PhoneDirectoryDetailOverlay
          key="phone-directory-detail"
          bounds={detailBounds}
          mode={mode}
          result={selected}
          session={session}
          onClose={() => setSelectedResultId("")}
        />,
      ];
    };
  }, [
    mode,
    modeConfig.detailRailId,
    modeConfig.resultsFrameId,
    nodeIndex,
    onNavigate,
    payload,
    pageState,
    searchQuery,
    selectedResultId,
    session?.current_site_name,
    sharedShellOverlay,
  ]);

  /**
   * overlay selects the temporary loading/error surface or the runtime directory controls layered over the generated artboard. It keeps hidden-node computation behind artboard readiness so route transitions cannot dereference missing generated nodes.
   */
  const overlay = (() => {
    if (artboardStatus === "loading") {
      return (
        <AccessDenied
          role="status"
          title="Loading Phone Directory"
          message="Preparing the generated Phone Directory artboard."
        />
      );
    }
    if (pageState === "loading") {
      return (
        <AccessDenied
          role="status"
          title="Loading Phone Directory"
          message="Loading the DEV phone directory results."
        />
      );
    }
    if (pageState === "error") {
      return (
        <AccessDenied
          title="Unable to load Phone Directory"
          message={errorMessage || "Request failed."}
        />
      );
    }
    return null;
  })();
  const pageTitle = MODE_PAGE_TITLES[mode] || "Phone Directory";

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Phone Directory</h1>
          <p>Preparing the generated Phone Directory artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Phone Directory unavailable</h1></main>;
  }

  return (
    <main
      id="main-content"
      className="page-canvas page-canvas--static phone-directory-runtime"
      aria-labelledby={PHONE_DIRECTORY_HEADING_ID}
    >
      {/* WCAG 1.3.1/2.4.6: routed phone-directory modes need a semantic h1 outside the visual artboard. */}
      <h1 id={PHONE_DIRECTORY_HEADING_ID} className="sr-only">
        {pageTitle}
      </h1>
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
        {overlay}
      </div>
    </main>
  );
}
