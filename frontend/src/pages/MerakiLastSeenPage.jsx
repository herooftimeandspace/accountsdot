import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { RuntimeSelectDropdown } from "../components/RuntimeDropdown";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { handleDevApiAuthError, fetchDevApiJSON } from "../lib/devApi";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { PenArtboard } from "../lib/PenArtboard";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";
import {
  DEFAULT_MERAKI_LAST_SEEN_FILTER,
  MERAKI_LAST_SEEN_FILTERS,
  merakiLastSeenAssignmentLabel,
  merakiLastSeenRowsForAssignmentFilter,
  merakiLastSeenStatusClass,
  merakiLastSeenStudentLabel,
} from "./merakiLastSeenModel.mjs";

const ARTBOARD_KEY = "reports";
const MERAKI_LAST_SEEN_ENDPOINT = "/api/v1/dev/pages/meraki-last-seen";
const MERAKI_LAST_SEEN_HEADING_ID = "meraki-last-seen-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1260;
const DRAWER_BOUNDS = { left: 1278, top: 92, width: 390, height: 802 };
const COLUMNS = [
  { key: "student", label: "Student", value: merakiLastSeenStudentLabel },
  { key: "device", label: "Device", value: (row) => row.device },
  { key: "assignment_type", label: "Assignment Type", value: merakiLastSeenAssignmentLabel },
  { key: "site", label: "Site", value: (row) => row.site },
  { key: "last_seen", label: "Date Last Seen", value: (row) => row.last_seen },
  {
    key: "match",
    label: "Match",
    value: (row) => `${row.match_state} ${row.match_confidence} ${(row.source_systems ?? []).join(" ")}`,
  },
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

function SourceList({ systems }) {
  return <span>{(systems ?? []).join(", ") || "Not provided"}</span>;
}

function MerakiLastSeenTable({ rows, selectedId, onSelect }) {
  const columns = useMemo(() => COLUMNS, []);
  const table = useRuntimeTableData(rows, columns, {
    defaultSort: { key: "last_seen", direction: "desc" },
  });
  return (
    <section className="reports-runtime__table-card meraki-last-seen-runtime__table-card" aria-label="Meraki last-seen rows">
      <h2>Device Last Seen</h2>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="reports-runtime__table-header meraki-last-seen-runtime__table-header">
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
            className={`reports-runtime__row meraki-last-seen-runtime__row ${
              selectedId === row.id ? "reports-runtime__row--selected" : ""
            }`}
            aria-label={`Open Meraki last-seen details for ${row.device}`}
            aria-pressed={selectedId === row.id}
            onClick={() => onSelect(row)}
          >
            <div>{merakiLastSeenStudentLabel(row)}</div>
            <div>
              <strong>{row.device}</strong>
              <span>{row.serial_number}</span>
            </div>
            <div><span className={merakiLastSeenStatusClass(row)}>{merakiLastSeenAssignmentLabel(row)}</span></div>
            <div>{row.site}</div>
            <div>{row.last_seen}</div>
            <div>{row.match_confidence}</div>
          </button>
        ))}
        {!table.visibleRows.length ? (
          <div className="frequent-fliers-runtime__empty">No devices match the current assignment filter or search.</div>
        ) : null}
      </div>
    </section>
  );
}

function MerakiLastSeenDrawer({ row, onClose }) {
  if (!row) {
    return null;
  }
  return (
    <RuntimeDrawer title={row.device} bounds={DRAWER_BOUNDS} onClose={onClose} className="meraki-last-seen-runtime__drawer">
      <RuntimeDetailList
        items={[
          { label: "Student", value: merakiLastSeenStudentLabel(row) },
          { label: "Assignment Type", value: merakiLastSeenAssignmentLabel(row) },
          { label: "Site", value: row.site },
          { label: "Date Last Seen", value: row.last_seen },
          { label: "Serial Number", value: row.serial_number },
          { label: "Asset Tag", value: row.asset_tag },
          { label: "MAC Address", value: row.mac_address },
          { label: "Hostname", value: row.hostname },
          { label: "Match State", value: row.match_state },
          { label: "Match Confidence", value: row.match_confidence },
          { label: "Source Systems", value: <SourceList systems={row.source_systems} /> },
        ]}
      />
      <div className="runtime-drawer__section">
        <p>{row.match_explanation}</p>
        {row.review_reason ? <p><strong>Review reason:</strong> {row.review_reason}</p> : null}
      </div>
    </RuntimeDrawer>
  );
}

function SummaryCards({ cards }) {
  if (!cards?.length) {
    return null;
  }
  return (
    <section className="reports-runtime__cards" aria-label="Meraki last-seen summary cards">
      {cards.map((card) => (
        <article key={card.title} className="reports-runtime__card">
          <h2>{card.title}</h2>
          <strong>{card.count}</strong>
        </article>
      ))}
    </section>
  );
}

function MerakiLastSeenOverlay({ payload, assignmentFilter, onFilterChange, selectedRow, onSelect }) {
  const rows = useMemo(
    () => merakiLastSeenRowsForAssignmentFilter(payload?.page?.rows ?? [], assignmentFilter),
    [assignmentFilter, payload?.page?.rows]
  );
  const filterOptions = payload?.page?.filters?.length ? payload.page.filters : MERAKI_LAST_SEEN_FILTERS;
  return (
    <section
      className="reports-runtime meraki-last-seen-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        zIndex: 2,
      }}
      aria-labelledby={MERAKI_LAST_SEEN_HEADING_ID}
    >
      <header className="reports-runtime__header">
        <div>
          <h1 id={MERAKI_LAST_SEEN_HEADING_ID}>{payload?.page?.title || "Meraki Last Seen"}</h1>
          <p>{payload?.page?.description || "Matched Meraki client last-seen records."}</p>
        </div>
      </header>
      {payload?.page?.help_text ? <p className="meraki-last-seen-runtime__help">{payload.page.help_text}</p> : null}
      <div className="frequent-fliers-runtime__filters meraki-last-seen-runtime__filters">
        <span>Assignment type</span>
        <RuntimeSelectDropdown
          label="Assignment type"
          value={assignmentFilter}
          options={filterOptions}
          onChange={onFilterChange}
        />
      </div>
      <SummaryCards cards={payload?.page?.summary_cards ?? []} />
      <MerakiLastSeenTable rows={rows} selectedId={selectedRow?.id} onSelect={onSelect} />
    </section>
  );
}

/**
 * MerakiLastSeenPage renders the read-only `/meraki-last-seen` DEV route. It
 * loads the scoped backend payload, keeps assignment-type filtering local to
 * the table, and relies on the Go handler for persona/site authorization.
 */
export function MerakiLastSeenPage({ session, onNavigate, onSearch, searchQuery, onUnauthorized, onForbidden }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [assignmentFilter, setAssignmentFilter] = useState(DEFAULT_MERAKI_LAST_SEEN_FILTER);
  const [selectedRow, setSelectedRow] = useState(null);
  const pageQuery = useQuery({
    queryKey: ["dev-page", "meraki-last-seen", session?.current_persona?.id ?? "", session?.current_site_id ?? ""],
    enabled: Boolean(session?.authenticated && session?.authorized),
    queryFn: ({ signal }) => fetchDevApiJSON(MERAKI_LAST_SEEN_ENDPOINT, { signal }),
  });

  useEffect(() => {
    if (pageQuery.isError) {
      handleDevApiAuthError(pageQuery.error, { onUnauthorized, onForbidden });
    }
  }, [onForbidden, onUnauthorized, pageQuery.error, pageQuery.isError]);

  const payload = pageQuery.data ?? null;
  const pageState = pageQuery.isLoading || pageQuery.isFetching ? "loading" : pageQuery.isError ? "error" : "ready";
  const textOverrides = buildSharedShellTextOverrides(session);
  const paneNodeIds = useMemo(() => artboard ? collectPaneNodeIds(artboard) : [], [artboard]);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...paneNodeIds);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const refreshPage = useCallback(() => {
    void pageQuery.refetch();
  }, [pageQuery]);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: "merakiLastSeen",
    activeRoutePath: "/meraki-last-seen",
    pageSyncControl: {
      label: "Refresh",
      loadingLabel: "Refreshing",
      lastRefreshed: payload?.page?.last_refreshed ?? null,
      disabled: pageState === "loading",
      loading: pageState === "loading",
      onAction: refreshPage,
    },
  });
  const rows = payload?.page?.rows ?? [];
  const selectedPayloadRow = selectedRow ? rows.find((row) => row.id === selectedRow.id) || selectedRow : null;
  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <MerakiLastSeenOverlay
        payload={payload}
        assignmentFilter={assignmentFilter}
        onFilterChange={(value) => {
          setAssignmentFilter(value);
          setSelectedRow(null);
        }}
        selectedRow={selectedPayloadRow}
        onSelect={setSelectedRow}
      />
      <MerakiLastSeenDrawer row={selectedPayloadRow} onClose={() => setSelectedRow(null)} />
    </>
  ), [assignmentFilter, payload, selectedPayloadRow, sharedShellRenderOverlay]);

  if (artboardStatus === "loading" || pageState === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Meraki Last Seen</h1>
          <p>Loading the scoped DEV Meraki last-seen dashboard.</p>
        </section>
      </main>
    );
  }
  if (pageState === "error") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Meraki Last Seen unavailable</h1>
          <p>The DEV Meraki last-seen dashboard could not be loaded.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Meraki Last Seen unavailable</h1></main>;
  }

  return (
    <PenArtboard
      artboard={artboard}
      meta={meta}
      textOverrides={textOverrides}
      hiddenNodeIds={hiddenNodeIds}
      imageNodeOverrides={imageNodeOverrides}
      renderOverlay={renderOverlay}
      mainId="main-content"
      semanticSummary={{
        title: "Meraki Last Seen",
        items: [
          "Read-only Meraki, IncidentIQ, and Google device matching dashboard.",
          `${rows.length} devices visible for the active persona and site scope.`,
        ],
      }}
    />
  );
}
