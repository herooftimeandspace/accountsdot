import { useCallback, useEffect, useMemo, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
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
} from "../lib/sharedShellPresentation";

const ARTBOARD_KEY = "reports";
const ZOOM_DESK_PHONE_RENAMES_ENDPOINT = "/api/v1/dev/pages/reports/zoom-desk-phone-renames";
const ZOOM_DESK_PHONE_RENAMES_HEADING_ID = "zoom-desk-phone-renames-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1260;
const DRAWER_BOUNDS = { left: 1278, top: 92, width: 390, height: 802 };
const ZOOM_DESK_PHONE_RENAME_COLUMNS = [
  { key: "serial_number", label: "Serial Number", value: (row) => row.serial_number },
  { key: "mac_address", label: "MAC Address", value: (row) => row.mac_address },
  { key: "current_name", label: "Current Name", value: (row) => row.current_name },
  { key: "new_name", label: "New Name", value: (row) => row.new_name },
  { key: "incidentiq_asset_label", label: "IncidentIQ Asset", value: (row) => row.incidentiq_asset_label },
];

async function readJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(payload?.message || `Request failed with ${response.status}`);
    error.status = response.status;
    error.payload = payload;
    throw error;
  }
  return payload;
}

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

function renameStatusClass(status) {
  if (["Error", "Failed"].includes(status)) {
    return "reports-runtime__status reports-runtime__status--critical";
  }
  if (["Pending manual adjustment", "Manual action", "Needs Review"].includes(status)) {
    return "reports-runtime__status reports-runtime__status--review";
  }
  return "reports-runtime__status reports-runtime__status--neutral";
}

function SummaryCards({ cards }) {
  if (!cards?.length) {
    return null;
  }
  return (
    <section className="reports-runtime__cards zoom-desk-phone-renames-runtime__cards" aria-label="Zoom desk phone rename summary cards">
      {cards.map((card) => (
        <article key={card.title} className="reports-runtime__card">
          <h2>{card.title}</h2>
          <strong>{card.count}</strong>
        </article>
      ))}
    </section>
  );
}

function AssetLink({ row }) {
  if (!row.incidentiq_asset_url) {
    return <span>{row.incidentiq_asset_label || "No IncidentIQ asset link"}</span>;
  }
  return (
    <a href={row.incidentiq_asset_url} target="_blank" rel="noreferrer" onClick={(event) => event.stopPropagation()}>
      {row.incidentiq_asset_label || "Open IncidentIQ asset"}
    </a>
  );
}

function ZoomDeskPhoneRenameTable({ rows, selectedId, onSelect }) {
  const columns = useMemo(() => ZOOM_DESK_PHONE_RENAME_COLUMNS, []);
  const table = useRuntimeTableData(rows, columns, {
    defaultSort: { key: "serial_number", direction: "asc" },
  });
  return (
    <section className="reports-runtime__table-card" aria-label="Zoom desk phone rename rows">
      <h2>Pending Zoom Desk Phone Renames</h2>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="reports-runtime__table-header zoom-desk-phone-renames-runtime__table-header">
        {columns.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
        <div>Status</div>
      </div>
      <div className="reports-runtime__table-body">
        {table.visibleRows.map((row) => (
          <div
            key={row.id}
            role="button"
            tabIndex={0}
            className={`reports-runtime__row zoom-desk-phone-renames-runtime__row ${
              selectedId === row.id ? "reports-runtime__row--selected" : ""
            }`}
            aria-label={`Open Zoom desk phone rename row for ${row.serial_number}`}
            aria-pressed={selectedId === row.id}
            onClick={() => onSelect(row)}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onSelect(row);
              }
            }}
          >
            <div>{row.serial_number}</div>
            <div>{row.mac_address}</div>
            <div>{row.current_name}</div>
            <div>{row.new_name}</div>
            <div><AssetLink row={row} /></div>
            <div><span className={renameStatusClass(row.status)}>{row.status}</span></div>
          </div>
        ))}
      </div>
    </section>
  );
}

function ZoomDeskPhoneRenameDrawer({ row, onClose }) {
  if (!row) {
    return null;
  }
  return (
    <RuntimeDrawer title={row.serial_number} bounds={DRAWER_BOUNDS} onClose={onClose} className="zoom-desk-phone-renames-runtime__drawer">
      <RuntimeDetailList
        items={[
          { label: "Status", value: row.status },
          { label: "MAC Address", value: row.mac_address },
          { label: "Current Name", value: row.current_name },
          { label: "New Name", value: row.new_name },
          { label: "Next Action", value: row.next_action },
          { label: "IncidentIQ Domain", value: row.incidentiq_asset_domain || "Not provided" },
        ]}
      />
      <div className="runtime-drawer__section">
        <p>
          <strong>IncidentIQ asset</strong>
          <span><AssetLink row={row} /></span>
        </p>
      </div>
    </RuntimeDrawer>
  );
}

function ZoomDeskPhoneRenamesOverlay({ payload, selectedRow, onSelect }) {
  const rows = payload?.page?.rows ?? [];
  return (
    <section
      className="reports-runtime zoom-desk-phone-renames-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        zIndex: 2,
      }}
      aria-labelledby={ZOOM_DESK_PHONE_RENAMES_HEADING_ID}
    >
      <header className="reports-runtime__header">
        <div>
          <h1 id={ZOOM_DESK_PHONE_RENAMES_HEADING_ID}>{payload?.page?.title || "Zoom Desk Phone Renames"}</h1>
          <p>{payload?.page?.description || "Desk phones that need IT Admin follow-up."}</p>
        </div>
      </header>
      {payload?.page?.help_text ? (
        <p className="zoom-desk-phone-renames-runtime__help">{payload.page.help_text}</p>
      ) : null}
      <SummaryCards cards={payload?.page?.summary_cards ?? []} />
      <ZoomDeskPhoneRenameTable rows={rows} selectedId={selectedRow?.id} onSelect={onSelect} />
    </section>
  );
}

export function ZoomDeskPhoneRenamesReportPage({ session, onNavigate, onSearch, searchQuery, onUnauthorized, onForbidden }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [selectedRow, setSelectedRow] = useState(null);

  const loadPage = useCallback(async () => {
    setPageState("loading");
    try {
      const nextPayload = await readJSON(
        await fetch(ZOOM_DESK_PHONE_RENAMES_ENDPOINT, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setPayload(nextPayload);
      setPageState("ready");
    } catch (error) {
      if (error.status === 401 && onUnauthorized) {
        onUnauthorized();
        return;
      }
      if (error.status === 403 && onForbidden) {
        onForbidden();
        return;
      }
      setPageState("error");
    }
  }, [onForbidden, onUnauthorized]);

  useEffect(() => {
    setSelectedRow(null);
    void loadPage();
  }, [loadPage]);

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
    activeNavKey: meta?.activeNav ?? "reports",
    activeRoutePath: "/reports/zoom-desk-phone-renames",
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Zoom Desk Phone Renames",
        textOverrides,
      })
    : { title: "Zoom Desk Phone Renames", items: [] };
  const rows = payload?.page?.rows ?? [];
  const selectedPayloadRow = selectedRow ? rows.find((row) => row.id === selectedRow.id) || selectedRow : null;

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <ZoomDeskPhoneRenamesOverlay payload={payload} selectedRow={selectedPayloadRow} onSelect={setSelectedRow} />
      <ZoomDeskPhoneRenameDrawer row={selectedPayloadRow} onClose={() => setSelectedRow(null)} />
    </>
  ), [payload, selectedPayloadRow, sharedShellRenderOverlay]);

  if (artboardStatus === "loading" || pageState === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Zoom Desk Phone Renames</h1>
          <p>Loading the DEV Zoom desk phone rename report.</p>
        </section>
      </main>
    );
  }

  if (pageState === "error") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Zoom Desk Phone Renames unavailable</h1>
          <p>The DEV Zoom desk phone rename report could not be loaded.</p>
        </section>
      </main>
    );
  }

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Zoom Desk Phone Renames unavailable</h1></main>;
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
      semanticSummary={semanticSummary}
    />
  );
}
