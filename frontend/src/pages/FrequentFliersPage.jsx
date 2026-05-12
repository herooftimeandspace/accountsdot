import { useCallback, useMemo, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboards, generatedArtboardMeta } from "../generated/artboards.generated.js";
import { PenArtboard } from "../lib/PenArtboard";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";

const ARTBOARD_KEY = "frequent-fliers";
const FREQUENT_FLIERS_HEADING_ID = "frequent-fliers-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1260;
const PANE_HEIGHT = 730;
const DRAWER_BOUNDS = { left: 1280, top: 92, width: 388, height: 802 };
const DEVICE_LINK_BASE = "https://mock.wusd.local/incidentiq/assets";
const TICKET_LINK_BASE = "https://mock.wusd.local/incidentiq/tickets";

const FREQUENT_FLIERS_HELP_CONTENT = {
  title: "Frequent Fliers help",
  sections: [
    {
      heading: "What this page shows",
      body:
        "This page helps staff find students with repeated device assignments or IncidentIQ tickets during the last 90 days. Use it to plan support, repairs, and follow-up before the pattern becomes harder to resolve.",
    },
    {
      heading: "How to use it",
      body:
        "Choose Devices or Tickets, pick the threshold, and select Apply. The table shows matching students. Select a row to open the details drawer with device history, recent tickets, and context for the follow-up.",
    },
    {
      heading: "Links",
      body:
        "Device serial numbers and IncidentIQ ticket numbers open deterministic DEV links so the demo behaves like the production workflow without connecting to live provider records.",
    },
  ],
};

const FREQUENT_FLIER_ROWS = [
  {
    id: "jason-rodriguez",
    student: "Jason Rodriguez",
    studentId: "3504011",
    grade: "10",
    site: "Desert View",
    deviceAssignments: 4,
    linkedTickets: 3,
    daysSinceLastTicket: 42,
    trend: [1, 2, 3, 4, 4, 3],
    note: "Multiple physical damage incidents within 90 days. Tech Support follow-up scheduled for May 2, 2025.",
    devices: [
      { serial: "CLA-24-27891", type: "Chromebook", status: "Active" },
      { serial: "CLA-24-26412", type: "Chromebook", status: "Returned" },
      { serial: "CLA-24-25103", type: "Chromebook", status: "Returned" },
      { serial: "CLA-24-23987", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1782345", summary: "Broken Screen", status: "Closed" },
      { id: "INC-1758991", summary: "Broken Hinge", status: "Closed" },
      { id: "INC-1741123", summary: "Keyboard Not Working", status: "Closed" },
    ],
  },
  {
    id: "maria-nguyen",
    student: "Maria Nguyen",
    studentId: "3501187",
    grade: "9",
    site: "Clover High School",
    deviceAssignments: 3,
    linkedTickets: 2,
    daysSinceLastTicket: 18,
    trend: [0, 1, 1, 2, 3, 3],
    note: "Two recent device exchanges are tied to keyboard damage. Library staff should confirm storage expectations.",
    devices: [
      { serial: "CLA-24-19912", type: "Chromebook", status: "Active" },
      { serial: "CLA-24-18801", type: "Chromebook", status: "Returned" },
      { serial: "CLA-24-17144", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1780042", summary: "Keyboard Damage", status: "Closed" },
      { id: "INC-1776620", summary: "Loaner Exchange", status: "Closed" },
    ],
  },
  {
    id: "devon-price",
    student: "Devon Price",
    studentId: "3502235",
    grade: "11",
    site: "Franklin Middle School",
    deviceAssignments: 2,
    linkedTickets: 4,
    daysSinceLastTicket: 9,
    trend: [1, 1, 2, 2, 3, 4],
    note: "Ticket volume is higher than assignment count. Review charger and case notes before assigning a replacement.",
    devices: [
      { serial: "FMS-24-11098", type: "Chromebook", status: "Active" },
      { serial: "FMS-24-10870", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1784210", summary: "Screen Flicker", status: "Open" },
      { id: "INC-1783022", summary: "Missing Charger", status: "Closed" },
      { id: "INC-1779234", summary: "Case Damage", status: "Closed" },
      { id: "INC-1774122", summary: "Replacement Request", status: "Closed" },
    ],
  },
  {
    id: "sophia-patel",
    student: "Sophia Patel",
    studentId: "3503407",
    grade: "8",
    site: "Canyon Ridge",
    deviceAssignments: 2,
    linkedTickets: 1,
    daysSinceLastTicket: 61,
    trend: [0, 0, 1, 1, 2, 2],
    note: "Meets the assignment threshold only. No recent ticket pattern is visible.",
    devices: [
      { serial: "CRM-24-09418", type: "Chromebook", status: "Active" },
      { serial: "CRM-24-08064", type: "Chromebook", status: "Returned" },
    ],
    tickets: [{ id: "INC-1768019", summary: "Cracked Bezel", status: "Closed" }],
  },
  {
    id: "eli-washington",
    student: "Eli Washington",
    studentId: "3505128",
    grade: "12",
    site: "District Office",
    deviceAssignments: 1,
    linkedTickets: 2,
    daysSinceLastTicket: 27,
    trend: [0, 0, 1, 1, 1, 2],
    note: "Ticket threshold match without repeated assignments. Review before escalating.",
    devices: [{ serial: "DO-24-03117", type: "Chromebook", status: "Active" }],
    tickets: [
      { id: "INC-1779901", summary: "Power Issue", status: "Closed" },
      { id: "INC-1774488", summary: "Trackpad Issue", status: "Closed" },
    ],
  },
];

const FREQUENT_FLIERS_COLUMNS = [
  { key: "student", label: "Student", value: (row) => row.student },
  { key: "studentId", label: "Student ID", value: (row) => row.studentId },
  { key: "grade", label: "Grade", value: (row) => row.grade },
  { key: "site", label: "Site", value: (row) => row.site },
  {
    key: "deviceAssignments",
    label: "Devices",
    value: (row) => row.deviceAssignments,
    sortValue: (row) => row.deviceAssignments,
  },
  {
    key: "linkedTickets",
    label: "Tickets",
    value: (row) => row.linkedTickets,
    sortValue: (row) => row.linkedTickets,
  },
  { key: "trend", label: "Trend", value: (row) => row.trend.join(" "), sortValue: (row) => row.trend.at(-1) ?? 0 },
];

/**
 * collectPaneNodeIds builds derived data for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function collectPaneNodeIds(node, ids = []) {
  const children = node.children || [];
  const isPaneNode = (node.x ?? 0) >= 280 && (node.y ?? 0) >= 88;
  if (isPaneNode) {
    ids.push(node.id);
    for (const child of children) {
      collectAllNodeIds(child, ids);
    }
    return ids;
  }
  for (const child of children) {
    collectPaneNodeIds(child, ids);
  }
  return ids;
}

/**
 * collectAllNodeIds builds derived data for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function collectAllNodeIds(node, ids) {
  ids.push(node.id);
  for (const child of node.children || []) {
    collectAllNodeIds(child, ids);
  }
}

/**
 * linkForDevice formats display data for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function linkForDevice(serial) {
  return `${DEVICE_LINK_BASE}/${encodeURIComponent(serial)}`;
}

/**
 * linkForTicket formats display data for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function linkForTicket(ticketId) {
  return `${TICKET_LINK_BASE}/${encodeURIComponent(ticketId)}`;
}

/**
 * trendClass documents runtime data flow for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function trendClass(value, threshold) {
  if (value >= threshold + 2) {
    return "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--critical";
  }
  if (value >= threshold) {
    return "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--review";
  }
  return "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--ready";
}

/**
 * TrendGraph renders the UI surface for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function TrendGraph({ values, threshold }) {
  const maxValue = Math.max(threshold, ...values, 1);
  return (
    <div className="frequent-fliers-runtime__trend" aria-label={`Trend counts: ${values.join(", ")}`}>
      {values.map((value, index) => (
        <span
          // The mock trend series is stable and intentionally ordered by lookback bucket.
          key={`${index}-${value}`}
          className={trendClass(value, threshold)}
          style={{ height: `${Math.max(18, (value / maxValue) * 44)}px` }}
          title={`${value} event${value === 1 ? "" : "s"}`}
        />
      ))}
    </div>
  );
}

/**
 * FrequentFliersDrawer renders the UI surface for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function FrequentFliersDrawer({ row, threshold, metric, onClose }) {
  if (!row) {
    return null;
  }
  const metricLabel = metric === "tickets" ? "linked tickets" : "device assignments";
  return (
    <RuntimeDrawer title={row.student} bounds={DRAWER_BOUNDS} onClose={onClose}>
      <RuntimeDetailList
        items={[
          { label: "Student ID", value: row.studentId },
          { label: "Grade", value: row.grade },
          { label: "Site", value: row.site },
          { label: "Device Assignments", value: row.deviceAssignments },
          { label: "Linked Tickets", value: row.linkedTickets },
          { label: "Days Since Last Ticket", value: row.daysSinceLastTicket },
          { label: "Current Rule", value: `Show ${metricLabel} greater than or equal to ${threshold}` },
        ]}
      />
      <div className="runtime-drawer__section">
        <h3>Device Assignment History</h3>
        <ul className="frequent-fliers-runtime__link-list">
          {row.devices.map((device) => (
            <li key={device.serial}>
              <a href={linkForDevice(device.serial)} target="_blank" rel="noreferrer">
                {device.serial} · {device.type}
              </a>
              <span>{device.status}</span>
            </li>
          ))}
        </ul>
      </div>
      <div className="runtime-drawer__section">
        <h3>Recent Tickets</h3>
        <ul className="frequent-fliers-runtime__link-list">
          {row.tickets.map((ticket) => (
            <li key={ticket.id}>
              <a href={linkForTicket(ticket.id)} target="_blank" rel="noreferrer">
                {ticket.id} {ticket.summary}
              </a>
              <span>{ticket.status}</span>
            </li>
          ))}
        </ul>
      </div>
      <div className="frequent-fliers-runtime__drawer-note">
        <strong>Follow-up context</strong>
        <p>{row.note}</p>
      </div>
    </RuntimeDrawer>
  );
}

/**
 * FrequentFliersOverlay renders the UI surface for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller. Pay special attention to side effects: this path may update React state, browser storage, cookies, or DEV mock APIs and should stay aligned with docs/external-write-inventory.md when it triggers mutations.
 */
function FrequentFliersOverlay({ rows, selectedRowId, filters, pendingFilters, onPendingChange, onApply, onSelectRow }) {
  const columns = useMemo(() => FREQUENT_FLIERS_COLUMNS, []);
  const tableRows = useMemo(() => {
    const metricKey = filters.metric === "tickets" ? "linkedTickets" : "deviceAssignments";
    return rows.filter((row) => row[metricKey] >= filters.threshold);
  }, [filters, rows]);
  const table = useRuntimeTableData(tableRows, columns, {
    defaultSort: { key: filters.metric === "tickets" ? "linkedTickets" : "deviceAssignments", direction: "desc" },
  });
  return (
    <section
      className="frequent-fliers-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        minHeight: PANE_HEIGHT,
        zIndex: 2,
      }}
      aria-labelledby={FREQUENT_FLIERS_HEADING_ID}
    >
      <header className="frequent-fliers-runtime__header">
        <div>
          <h1 id={FREQUENT_FLIERS_HEADING_ID}>Frequent Fliers</h1>
          <p>Students with repeated device assignments or IncidentIQ tickets in the last 90 days.</p>
        </div>
      </header>
      <form className="frequent-fliers-runtime__filters" onSubmit={onApply}>
        <span>Show students with</span>
        <span className="frequent-fliers-runtime__operator" aria-label="Comparison: greater than or equal to">
          &gt;=
        </span>
        <label>
          <span className="sr-only">Threshold</span>
          <select
            value={pendingFilters.threshold}
            onChange={(event) => onPendingChange({ threshold: Number(event.target.value) })}
          >
            {Array.from({ length: 10 }, (_, index) => index + 1).map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span className="sr-only">Metric</span>
          <select
            value={pendingFilters.metric}
            onChange={(event) => onPendingChange({ metric: event.target.value })}
          >
            <option value="devices">Devices</option>
            <option value="tickets">Tickets</option>
          </select>
        </label>
        <span>in the last 90 days</span>
        <button type="submit">Apply</button>
      </form>
      <div className="frequent-fliers-runtime__table-card">
        <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
        <div className="frequent-fliers-runtime__table-header">
          {columns.map((column) => (
            <div key={column.key}>
              <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
            </div>
          ))}
        </div>
        <div className="frequent-fliers-runtime__table-body">
          {table.visibleRows.map((row) => (
            <button
              key={row.id}
              type="button"
              className={`frequent-fliers-runtime__row ${
                selectedRowId === row.id ? "frequent-fliers-runtime__row--selected" : ""
              }`}
              aria-label={`Open frequent flier details for ${row.student}`}
              aria-pressed={selectedRowId === row.id}
              onClick={() => onSelectRow(row)}
            >
              <div>{row.student}</div>
              <div>{row.studentId}</div>
              <div>{row.grade}</div>
              <div>{row.site}</div>
              <div>{row.deviceAssignments}</div>
              <div>{row.linkedTickets}</div>
              <div>
                <TrendGraph values={row.trend} threshold={filters.threshold} />
              </div>
            </button>
          ))}
          {!table.visibleRows.length ? (
            <div className="frequent-fliers-runtime__empty">No students match the current Frequent Fliers filters.</div>
          ) : null}
        </div>
      </div>
    </section>
  );
}

/**
 * FrequentFliersPage renders the UI surface for frontend/src/pages/FrequentFliersPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function FrequentFliersPage({ session, onNavigate, onSearch, searchQuery }) {
  const artboard = generatedArtboards[ARTBOARD_KEY];
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [filters, setFilters] = useState({ threshold: 2, metric: "devices" });
  const [pendingFilters, setPendingFilters] = useState({ threshold: 2, metric: "devices" });
  const [selectedRow, setSelectedRow] = useState(null);

  const textOverrides = buildSharedShellTextOverrides(session);
  const paneNodeIds = useMemo(() => collectPaneNodeIds(artboard), [artboard]);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...paneNodeIds);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? "frequentFliers",
    refreshMetadata: staticRefreshMetadataForArtboard(ARTBOARD_KEY),
    helpContent: FREQUENT_FLIERS_HELP_CONTENT,
  });
  const semanticSummary = buildArtboardSemanticSummary(artboard, {
    fallbackTitle: "Frequent Fliers",
    textOverrides,
  });
  const selectedPayloadRow = selectedRow
    ? FREQUENT_FLIER_ROWS.find((row) => row.id === selectedRow.id) || selectedRow
    : null;

  const handlePendingChange = useCallback((change) => {
    setPendingFilters((current) => ({ ...current, ...change }));
  }, []);
  const handleApply = useCallback((event) => {
    event.preventDefault();
    setFilters(pendingFilters);
    setSelectedRow(null);
  }, [pendingFilters]);
  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <FrequentFliersOverlay
        rows={FREQUENT_FLIER_ROWS}
        selectedRowId={selectedPayloadRow?.id}
        filters={filters}
        pendingFilters={pendingFilters}
        onPendingChange={handlePendingChange}
        onApply={handleApply}
        onSelectRow={setSelectedRow}
      />
      <FrequentFliersDrawer
        row={selectedPayloadRow}
        threshold={filters.threshold}
        metric={filters.metric}
        onClose={() => setSelectedRow(null)}
      />
    </>
  ), [filters, handleApply, handlePendingChange, pendingFilters, selectedPayloadRow, sharedShellRenderOverlay]);

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={FREQUENT_FLIERS_HEADING_ID}>
      <section className="sr-only" aria-labelledby={FREQUENT_FLIERS_HEADING_ID}>
        <h1 id={`${FREQUENT_FLIERS_HEADING_ID}-summary`}>{semanticSummary.title}</h1>
        <ul>
          {semanticSummary.items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      </section>
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
      </div>
    </main>
  );
}
