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

const ARTBOARD_KEY = "reports";
const REPORTS_HEADING_ID = "reports-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1260;
const DRAWER_BOUNDS = { left: 1278, top: 92, width: 390, height: 802 };

const REPORT_ROWS = [
  {
    id: "onboarding-status",
    report: "Onboarding Status Report",
    scope: "(All Sites)",
    sourceSystems: "Aeries SIS",
    openItems: 186,
    status: "Up to date",
    destination: "/onboarding",
    summary: "Shows all new hires pending provision and their progress through the employee reprovisioning workflow.",
    dataIncluded: "New hires not yet fully provisioned, license status, account provisioning state, and device-stage status.",
    refreshFrequency: "Every 15 minutes",
    lastRun: "May 2, 2025 9:05 AM PT",
  },
  {
    id: "offboarding-status",
    report: "Offboarding Status Report",
    scope: "(All Sites)",
    sourceSystems: "Aeries SIS",
    openItems: 142,
    status: "Up to date",
    destination: "/offboarding",
    summary: "Summarizes pending offboarding work, asset retrieval status, and accounts still waiting on manual review.",
    dataIncluded: "Scheduled leaves, immediate terms, security-risk rows, and asset retrieval counts.",
    refreshFrequency: "Every 15 minutes",
    lastRun: "May 2, 2025 9:05 AM PT",
  },
  {
    id: "room-move-status",
    report: "Room Move Status Report",
    scope: "(All Sites)",
    sourceSystems: "Aeries SIS",
    openItems: 64,
    status: "Up to date",
    destination: "/room-moves",
    summary: "Tracks room and phone move batches, warnings, and scheduled cutovers.",
    dataIncluded: "Draft moves, warning states, immediate moves, batch cutovers, and execution-rule exceptions.",
    refreshFrequency: "Every 15 minutes",
    lastRun: "May 2, 2025 9:05 AM PT",
  },
  {
    id: "phone-directory-coverage",
    report: "Phone Directory Coverage Report",
    scope: "(All Sites)",
    sourceSystems: "Aeries, AD, Telephony",
    openItems: 2146,
    status: "Up to date",
    destination: "/phone-directory/by-person",
    summary: "Shows whether people, rooms, and shared lines have usable district phone-directory coverage.",
    dataIncluded: "Person lines, rooms, departments, call queues, extension values, and site coverage.",
    refreshFrequency: "Hourly",
    lastRun: "May 2, 2025 9:05 AM PT",
  },
  {
    id: "student-data-cleanup",
    report: "Student Data Cleanup Queue Report",
    scope: "(All Sites)",
    sourceSystems: "Aeries SIS",
    openItems: 7,
    status: "Up to date",
    destination: "/student-data-cleanup",
    summary: "Lists active unresolved student-name issues that must be corrected in Aeries.",
    dataIncluded: "Student ID, raw name values, cleaned suggestions, issue type, grade, and submitted time.",
    refreshFrequency: "Hourly",
    lastRun: "May 2, 2025 9:05 AM PT",
  },
  {
    id: "frequent-fliers",
    report: "Frequent Fliers Report",
    scope: "This Site",
    sourceSystems: "IncidentIQ",
    openItems: 142,
    status: "Up to date",
    destination: "/frequent-fliers",
    summary: "Highlights students with repeated device assignments or IncidentIQ tickets in the current lookback window.",
    dataIncluded: "Student context, device assignment history, ticket history, and trend signals.",
    refreshFrequency: "Daily",
    lastRun: "May 2, 2025 9:04 AM PT",
  },
  {
    id: "sync-transparency",
    report: "Sync Transparency Report",
    scope: "(All Sites)",
    sourceSystems: "All providers",
    openItems: 37,
    status: "Up to date",
    destination: "/reports/sync-transparency",
    summary: "Shows provider sync stages, warnings, manual-action states, and retry context.",
    dataIncluded: "Sync items, providers, phases, warnings, and next actions.",
    refreshFrequency: "Every 15 minutes",
    lastRun: "May 2, 2025 9:03 AM PT",
  },
  {
    id: "ticketing-human-work",
    report: "Ticketing Human Work Report",
    scope: "(All Sites)",
    sourceSystems: "IncidentIQ",
    openItems: 9,
    status: "Up to date",
    destination: "/reports/ticketing-human-work",
    summary: "Collects human-owned tickets that unblock lifecycle, room, or account workflows.",
    dataIncluded: "Ticket category, owner context, workflow, matching rule, and required action.",
    refreshFrequency: "Every 15 minutes",
    lastRun: "May 2, 2025 9:02 AM PT",
  },
  {
    id: "data-quality",
    report: "Data Quality Summary",
    scope: "(All Sites)",
    sourceSystems: "All providers",
    openItems: 237,
    status: "Up to date",
    destination: "/data-quality",
    summary: "Summarizes provider data-quality exceptions and cleanup queues across source systems.",
    dataIncluded: "Google-active/Aeries-inactive users, orphaned Zoom users, and sync health signals.",
    refreshFrequency: "Hourly",
    lastRun: "May 2, 2025 9:00 AM PT",
  },
];

const REFRESH_ROWS = [
  {
    id: "aeries",
    source: "Aeries SIS",
    lastRefresh: "May 2, 2025 9:05 AM PT",
    status: "Healthy",
    details: "Student, staff, room, and lifecycle extracts completed without provider errors.",
  },
  {
    id: "google",
    source: "Google Workspace",
    lastRefresh: "May 2, 2025 9:04 AM PT",
    status: "Healthy",
    details: "User, group, and license reconciliation completed with no blocked writeback.",
  },
  {
    id: "zoom",
    source: "Zoom",
    lastRefresh: "May 2, 2025 9:02 AM PT",
    status: "Healthy",
    details: "License and orphaned-user cleanup projections refreshed successfully.",
  },
  {
    id: "incidentiq",
    source: "IncidentIQ",
    lastRefresh: "May 2, 2025 8:59 AM PT",
    status: "Healthy",
    details: "Ticket, device, site, and room references refreshed successfully.",
  },
];

const REPORT_COLUMNS = [
  { key: "report", label: "Report", value: (row) => row.report },
  { key: "scope", label: "Scope", value: (row) => row.scope },
  { key: "sourceSystems", label: "Source Systems", value: (row) => row.sourceSystems },
  { key: "lastRun", label: "Last Run", value: (row) => row.lastRun },
  { key: "openItems", label: "Open Items", value: (row) => row.openItems, sortValue: (row) => row.openItems },
  { key: "status", label: "Status", value: (row) => row.status },
];

const REFRESH_COLUMNS = [
  { key: "source", label: "Source System", value: (row) => row.source },
  { key: "lastRefresh", label: "Last Refresh", value: (row) => row.lastRefresh },
  { key: "status", label: "Status", value: (row) => row.status },
];

function collectAllNodeIds(node, ids) {
  ids.push(node.id);
  for (const child of node.children || []) {
    collectAllNodeIds(child, ids);
  }
}

function collectPaneNodeIds(node, ids = []) {
  const isPaneNode = (node.x ?? 0) >= 280 && (node.y ?? 0) >= 88;
  if (isPaneNode) {
    collectAllNodeIds(node, ids);
    return ids;
  }
  for (const child of node.children || []) {
    collectPaneNodeIds(child, ids);
  }
  return ids;
}

function reportStatusClass(status) {
  if (status === "Healthy" || status === "Up to date") {
    return "reports-runtime__status reports-runtime__status--ready";
  }
  if (status === "Warning" || status === "Needs Review") {
    return "reports-runtime__status reports-runtime__status--review";
  }
  return "reports-runtime__status reports-runtime__status--neutral";
}

function ReportsDrawer({ item, onClose, onNavigate }) {
  if (!item) {
    return null;
  }
  const isReport = item.kind === "report";
  return (
    <RuntimeDrawer title={isReport ? item.report : item.source} bounds={DRAWER_BOUNDS} onClose={onClose}>
      <RuntimeDetailList
        items={
          isReport
            ? [
                { label: "Scope", value: item.scope },
                { label: "Source System", value: item.sourceSystems },
                { label: "Data Included", value: item.dataIncluded },
                { label: "Open Items", value: item.openItems },
                { label: "Last Run", value: item.lastRun },
                { label: "Refresh Frequency", value: item.refreshFrequency },
                { label: "Status", value: item.status },
              ]
            : [
                { label: "Source System", value: item.source },
                { label: "Last Refresh", value: item.lastRefresh },
                { label: "Status", value: item.status },
                { label: "Details", value: item.details },
              ]
        }
      />
      <div className="runtime-drawer__section">
        <p>
          <strong>{isReport ? "Report details" : "Refresh details"}</strong>
          <span>{isReport ? item.summary : item.details}</span>
        </p>
        {isReport ? (
          <button type="button" className="reports-runtime__drawer-action" onClick={() => onNavigate(item.destination)}>
            Open Report
          </button>
        ) : null}
      </div>
    </RuntimeDrawer>
  );
}

function SummaryCards() {
  const cards = [
    ["Onboarding", "186", "Pending"],
    ["Offboarding", "142", "Pending"],
    ["Room Moves", "64", "In Progress"],
    ["Phone Directory", "98%", "Coverage"],
    ["Student Data Cleanup", "7", "Active Issues"],
    ["Frequent Fliers", "142", "This Site"],
    ["Google-active / Aeries-inactive", "237", "Users"],
    ["Orphaned Zoom Cleanup", "89", "Orphaned Users"],
    ["Sync Health", "Healthy", "All Providers"],
  ];
  return (
    <section className="reports-runtime__cards" aria-label="Report summary cards">
      {cards.map(([label, value, helper]) => (
        <article key={label} className="reports-runtime__card">
          <h2>{label}</h2>
          <strong>{value}</strong>
          <span>{helper}</span>
        </article>
      ))}
    </section>
  );
}

function ReportsTable({ selectedId, onSelect }) {
  const columns = useMemo(() => REPORT_COLUMNS, []);
  const table = useRuntimeTableData(REPORT_ROWS, columns, {
    defaultSort: { key: "report", direction: "asc" },
  });
  return (
    <section className="reports-runtime__table-card" aria-label="Available reports">
      <h2>Available Reports</h2>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="reports-runtime__table-header reports-runtime__table-header--reports">
        {columns.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
        <div>Actions</div>
      </div>
      <div className="reports-runtime__table-body">
        {table.visibleRows.map((row) => (
          <button
            key={row.id}
            type="button"
            className={`reports-runtime__row reports-runtime__row--reports ${
              selectedId === row.id ? "reports-runtime__row--selected" : ""
            }`}
            aria-label={`Open report details for ${row.report}`}
            aria-pressed={selectedId === row.id}
            onClick={() => onSelect({ ...row, kind: "report" })}
          >
            <div>{row.report}</div>
            <div>{row.scope}</div>
            <div>{row.sourceSystems}</div>
            <div>{row.lastRun}</div>
            <div>{row.openItems}</div>
            <div><span className={reportStatusClass(row.status)}>{row.status}</span></div>
            <div>Open</div>
          </button>
        ))}
      </div>
    </section>
  );
}

function RefreshTable({ selectedId, onSelect }) {
  const columns = useMemo(() => REFRESH_COLUMNS, []);
  const table = useRuntimeTableData(REFRESH_ROWS, columns, {
    defaultSort: { key: "lastRefresh", direction: "desc" },
  });
  return (
    <section className="reports-runtime__table-card" aria-label="Recent refreshes">
      <h2>Recent Refreshes</h2>
      <div className="reports-runtime__table-header reports-runtime__table-header--refreshes">
        {columns.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
      </div>
      <div className="reports-runtime__table-body">
        {table.visibleRows.map((row) => (
          <button
            key={row.id}
            type="button"
            className={`reports-runtime__row reports-runtime__row--refreshes ${
              selectedId === row.id ? "reports-runtime__row--selected" : ""
            }`}
            aria-label={`Open refresh details for ${row.source}`}
            aria-pressed={selectedId === row.id}
            onClick={() => onSelect({ ...row, kind: "refresh" })}
          >
            <div>{row.source}</div>
            <div>{row.lastRefresh}</div>
            <div><span className={reportStatusClass(row.status)}>{row.status}</span></div>
          </button>
        ))}
      </div>
    </section>
  );
}

function ReportsOverlay({ selectedItem, onSelect }) {
  return (
    <section
      className="reports-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        zIndex: 2,
      }}
      aria-labelledby={REPORTS_HEADING_ID}
    >
      <header className="reports-runtime__header">
        <div>
          <h1 id={REPORTS_HEADING_ID}>Reports</h1>
          <p>Operational reports and queue summaries across all systems and workflows.</p>
        </div>
      </header>
      <SummaryCards />
      <ReportsTable selectedId={selectedItem?.id} onSelect={onSelect} />
      <RefreshTable selectedId={selectedItem?.id} onSelect={onSelect} />
    </section>
  );
}

export function ReportsPage({ session, onNavigate, onSearch, searchQuery }) {
  const artboard = generatedArtboards[ARTBOARD_KEY];
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [selectedItem, setSelectedItem] = useState(null);
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
    activeNavKey: meta?.activeNav ?? "reports",
    refreshMetadata: staticRefreshMetadataForArtboard(ARTBOARD_KEY),
  });
  const semanticSummary = buildArtboardSemanticSummary(artboard, {
    fallbackTitle: "Reports",
    textOverrides,
  });
  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <ReportsOverlay selectedItem={selectedItem} onSelect={setSelectedItem} />
      <ReportsDrawer item={selectedItem} onClose={() => setSelectedItem(null)} onNavigate={onNavigate} />
    </>
  ), [onNavigate, selectedItem, sharedShellRenderOverlay]);

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={REPORTS_HEADING_ID}>
      <section className="sr-only" aria-labelledby={`${REPORTS_HEADING_ID}-summary`}>
        <h1 id={`${REPORTS_HEADING_ID}-summary`}>{semanticSummary.title}</h1>
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
