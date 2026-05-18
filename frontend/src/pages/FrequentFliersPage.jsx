import { useCallback, useMemo, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { RuntimeSelectDropdown } from "../components/RuntimeDropdown";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { PenArtboard } from "../lib/PenArtboard";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";
import {
  DEFAULT_FREQUENT_FLIERS_FILTERS,
  FREQUENT_FLIER_ROWS,
  FREQUENT_FLIERS_RANGE_OPTIONS,
  frequentFliersRowsForFilters,
  linkForDevice,
  linkForTicket,
  metricCountForRange,
  rangeLabelForValue,
  trendClass,
} from "./frequentFliersModel.mjs";

const ARTBOARD_KEY = "frequent-fliers";
const FREQUENT_FLIERS_HEADING_ID = "frequent-fliers-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1260;
const PANE_HEIGHT = 730;
const DRAWER_BOUNDS = { left: 1280, top: 92, width: 388, height: 802 };

/**
 * buildFrequentFliersColumns creates the sortable/searchable table contract
 * after the user applies a lookback range, ensuring visible counts and sort
 * values use the same committed range.
 */
function buildFrequentFliersColumns(range) {
  return [
    { key: "student", label: "Student", value: (row) => row.student },
    { key: "studentId", label: "Student ID", value: (row) => row.studentId },
    { key: "grade", label: "Grade", value: (row) => row.grade },
    { key: "site", label: "Site", value: (row) => row.site },
    {
      key: "deviceAssignments",
      label: "Devices",
      value: (row) => metricCountForRange(row, "devices", range),
      sortValue: (row) => metricCountForRange(row, "devices", range),
    },
    {
      key: "linkedTickets",
      label: "Tickets",
      value: (row) => metricCountForRange(row, "tickets", range),
      sortValue: (row) => metricCountForRange(row, "tickets", range),
    },
    { key: "trend", label: "Trend", value: (row) => row.trend.join(" "), sortValue: (row) => row.trend.at(-1) ?? 0 },
  ];
}

/**
 * collectPaneNodeIds finds every generated artboard node inside the page pane
 * so FrequentFliersPage can hide the static mock table/detail layout and render
 * the runtime-owned filters, table, and drawer over the shared shell.
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
 * collectAllNodeIds appends a generated node and all descendants after a pane
 * root is found, preventing hidden static children from leaking underneath the
 * runtime Frequent Fliers controls.
 */
function collectAllNodeIds(node, ids) {
  ids.push(node.id);
  for (const child of node.children || []) {
    collectAllNodeIds(child, ids);
  }
}

/**
 * TrendGraph renders the compact Frequent Fliers sparkline for a table row. It
 * receives the already-selected threshold from FrequentFliersOverlay and emits
 * only visual bars with an accessible count summary.
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
 * FrequentFliersDrawer presents the selected student's range-scoped counts plus
 * all related mock device and ticket context. The drawer is read-only in this
 * DEV slice and uses deterministic IncidentIQ links instead of provider calls.
 */
function FrequentFliersDrawer({ row, threshold, metric, range, onClose }) {
  if (!row) {
    return null;
  }
  const metricLabel = metric === "tickets" ? "linked tickets" : "device assignments";
  const rangeLabel = rangeLabelForValue(range);
  return (
    <RuntimeDrawer title={row.student} bounds={DRAWER_BOUNDS} onClose={onClose}>
      <RuntimeDetailList
        items={[
          { label: "Student ID", value: row.studentId },
          { label: "Grade", value: row.grade },
          { label: "Site", value: row.site },
          { label: "Device Assignments", value: metricCountForRange(row, "devices", range) },
          { label: "Linked Tickets", value: metricCountForRange(row, "tickets", range) },
          { label: "Date Range", value: rangeLabel },
          { label: "Days Since Last Ticket", value: row.daysSinceLastTicket },
          { label: "Current Rule", value: `Show ${metricLabel} greater than or equal to ${threshold} in ${rangeLabel}` },
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
 * FrequentFliersOverlay owns the runtime-only filter form and table that replace
 * the hidden static artboard pane. Pending controls do not affect rows until
 * Apply commits threshold, metric, and range into parent state.
 */
function FrequentFliersOverlay({ rows, selectedRowId, filters, pendingFilters, onPendingChange, onApply, onSelectRow }) {
  const columns = useMemo(() => buildFrequentFliersColumns(filters.range), [filters.range]);
  const tableRows = useMemo(() => {
    return frequentFliersRowsForFilters(rows, filters);
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
          <p>Students with repeated device assignments or IncidentIQ tickets during the selected date range.</p>
        </div>
      </header>
      <form className="frequent-fliers-runtime__filters" onSubmit={onApply}>
        <span>Show students with</span>
        <span className="frequent-fliers-runtime__operator" aria-label="Comparison: greater than or equal to">
          &gt;=
        </span>
        <RuntimeSelectDropdown
          label="Threshold"
          value={pendingFilters.threshold}
          options={Array.from({ length: 10 }, (_, index) => {
            const optionValue = index + 1;
            return { value: optionValue, label: String(optionValue) };
          })}
          onChange={(threshold) => onPendingChange({ threshold: Number(threshold) })}
        />
        <RuntimeSelectDropdown
          label="Metric"
          value={pendingFilters.metric}
          options={[
            { value: "devices", label: "Devices" },
            { value: "tickets", label: "Tickets" },
          ]}
          onChange={(metric) => onPendingChange({ metric })}
        />
        <span>in the last</span>
        <RuntimeSelectDropdown
          label="Date range"
          value={pendingFilters.range}
          options={FREQUENT_FLIERS_RANGE_OPTIONS}
          onChange={(range) => onPendingChange({ range })}
        />
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
              <div>{metricCountForRange(row, "devices", filters.range)}</div>
              <div>{metricCountForRange(row, "tickets", filters.range)}</div>
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
 * FrequentFliersPage is the route component for `/frequent-fliers`. It loads the
 * generated shared-shell artboard, hides the static Frequent Fliers pane, and
 * keeps the selected row drawer synchronized with the applied runtime filters.
 */
export function FrequentFliersPage({ session, onNavigate, onSearch, searchQuery }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [filters, setFilters] = useState(DEFAULT_FREQUENT_FLIERS_FILTERS);
  const [pendingFilters, setPendingFilters] = useState(DEFAULT_FREQUENT_FLIERS_FILTERS);
  const [selectedRow, setSelectedRow] = useState(null);

  const textOverrides = buildSharedShellTextOverrides(session);
  const paneNodeIds = useMemo(() => artboard ? collectPaneNodeIds(artboard) : [], [artboard]);
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
    activeRoutePath: "/frequent-fliers",
    refreshMetadata: staticRefreshMetadataForArtboard(ARTBOARD_KEY),
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Frequent Fliers",
        textOverrides,
      })
    : { title: "Frequent Fliers", items: [] };
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
        range={filters.range}
        onClose={() => setSelectedRow(null)}
      />
    </>
  ), [filters, handleApply, handlePendingChange, pendingFilters, selectedPayloadRow, sharedShellRenderOverlay]);

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Frequent Fliers</h1>
          <p>Preparing the generated Frequent Fliers artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Frequent Fliers unavailable</h1></main>;
  }

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
