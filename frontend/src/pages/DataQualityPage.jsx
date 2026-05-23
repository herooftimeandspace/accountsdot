import { useCallback, useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { AccessDenied } from "../components/AccessDenied";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import {
  DataQualityGeneratedView,
  dataQualityDesign,
} from "../generated/data-quality.generated.jsx";
import { fetchDevApiJSON, handleDevApiAuthError } from "../lib/devApi";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const DATA_QUALITY_ENDPOINT = "/api/v1/dev/pages/data-quality";
const MAIN_CONTENT_ID = "main-content";
const DATA_QUALITY_HEADING_ID = "data-quality-heading";
const QUEUE_SORT_HEADERS = {
  issue: { label: "Issue", field: "issue" },
  source: { label: "Source", field: "source" },
  owner: { label: "Owner", field: "owner" },
  impact: { label: "Impact", field: "impact" },
  nextAction: { label: "Next Action", field: "next_action" },
};
const DATA_QUALITY_QUEUE_COLUMNS = Object.entries(QUEUE_SORT_HEADERS).map(([key, config]) => ({
  key,
  label: config.label,
  value: (row) => readableLine(row?.[config.field]),
}));

/**
 * readableLine keeps multi-line DEV queue fields readable in the generated
 * artboard and semantic fallback table by flattening backend newline-separated
 * labels into a single row-cell string.
 */
function readableLine(value) {
  return String(value ?? "").replaceAll("\n", " ");
}

/**
 * assignValue writes a string override for one generated PEN node id. It skips
 * missing slot ids so Data Quality can remove obsolete static nodes from the
 * authoritative PEN without forcing runtime callers to keep stale mappings.
 */
function assignValue(overrides, slotId, value) {
  if (!slotId || value == null) {
    return;
  }
  overrides[slotId] = String(value);
}

/**
 * assignLines maps newline-separated backend cell text onto the fixed set of
 * generated text nodes available for one PEN table cell. Missing lines become
 * empty strings so old content is cleared when the visible queue has fewer rows.
 */
function assignLines(overrides, slotIds, value) {
  const values = Array.isArray(value) ? value : String(value ?? "").split("\n");
  slotIds.forEach((slotId, index) => {
    overrides[slotId] = values[index] ?? "";
  });
}

/**
 * queueHeaderLabel adds the current sort indicator to generated table headers.
 * RuntimeTableControls owns the sort state; this helper only turns that state
 * into operator-visible text for the PEN artboard layer.
 */
function queueHeaderLabel(key, sortState) {
  const config = QUEUE_SORT_HEADERS[key];
  if (!config) {
    return "";
  }
  if (sortState?.key !== key || sortState.direction === "none") {
    return `${config.label} ↕`;
  }
  return `${config.label} ${sortState.direction === "asc" ? "↑" : "↓"}`;
}

/**
 * buildDataQualityTextOverrides combines DEV page JSON, shell persona labels,
 * and runtime table sort/filter state into PEN node text overrides. It only
 * targets documented Data Quality slots, leaving help text and unsupported
 * mapping navigation out of the visible page pane.
 */
function buildDataQualityTextOverrides(session, payload, sortState) {
  const overrides = buildSharedShellTextOverrides(session);
  if (!payload) {
    return overrides;
  }

  const { shell, page } = payload;

  assignValue(overrides, dataQualityDesign.slots.shell.scopeTitle, shell.scope_title);
  assignValue(overrides, dataQualityDesign.slots.shell.scopeSubtitle, shell.scope_subtitle);
  assignValue(overrides, dataQualityDesign.slots.shell.searchPlaceholder, shell.search_placeholder);
  assignValue(overrides, dataQualityDesign.slots.shell.notificationCount, shell.notification_count);
  assignValue(overrides, dataQualityDesign.slots.shell.platformStatus, shell.platform_status);

  assignValue(overrides, dataQualityDesign.slots.page.title, page.title);
  assignValue(overrides, dataQualityDesign.slots.page.lastRefreshed, page.last_refreshed);
  assignValue(overrides, dataQualityDesign.slots.page.refreshLabel, page.refresh_label);

  page.summary_cards.forEach((card, index) => {
    const slot = dataQualityDesign.slots.summaryCards[index];
    if (!slot) {
      return;
    }
    assignValue(overrides, slot.title, card.title);
    assignValue(overrides, slot.count, card.count);
  });

  Object.entries(dataQualityDesign.slots.queue.headers || {}).forEach(([key, slotId]) => {
    assignValue(overrides, slotId, queueHeaderLabel(key, sortState));
  });

  dataQualityDesign.slots.queue.rows.forEach((slot, index) => {
    const row = page.queue.rows[index];
    if (!row) {
      assignValue(overrides, slot.issue, "");
      assignValue(overrides, slot.source, "");
      assignValue(overrides, slot.owner, "");
      assignLines(overrides, slot.impact, "");
      assignLines(overrides, slot.nextAction, "");
      return;
    }
    assignValue(overrides, slot.issue, row.issue);
    assignValue(overrides, slot.source, row.source);
    assignValue(overrides, slot.owner, row.owner);
    assignLines(overrides, slot.impact, row.impact);
    assignLines(overrides, slot.nextAction, row.next_action);
  });
  return overrides;
}

/**
 * DataQualitySemanticContent provides the accessible semantic mirror for the
 * generated artboard. It exposes refresh, summary cards, search, sorting, and
 * queue rows without rendering the removed mapping-dashboard shortcut.
 */
function DataQualitySemanticContent({ payload, onRefresh, table }) {
  if (!payload) {
    return null;
  }

  const { page } = payload;

  return (
    <section className="data-quality-semantic" aria-labelledby={DATA_QUALITY_HEADING_ID}>
      <div className="data-quality-semantic__header">
        <div>
          <h1 id={DATA_QUALITY_HEADING_ID}>{page.title}</h1>
          <p style={{ whiteSpace: "pre-line" }}>{page.last_refreshed}</p>
        </div>
        <div className="data-quality-semantic__mobile-actions">
          <button type="button" onClick={onRefresh}>
            Refresh Data Quality
          </button>
        </div>
      </div>

      <h2>Summary</h2>
      <dl className="data-quality-semantic__summary">
        {page.summary_cards.map((card) => (
          <div key={card.title}>
            <dt>{card.title}</dt>
            <dd>{card.count}</dd>
          </div>
        ))}
      </dl>

      <h2>Data Quality Queue</h2>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <table>
        <thead>
          <tr>
            {DATA_QUALITY_QUEUE_COLUMNS.map((column) => {
              const activeDirection = table.sortState?.key === column.key ? table.sortState.direction : "none";
              const ariaSort =
                activeDirection === "asc"
                  ? "ascending"
                  : activeDirection === "desc"
                    ? "descending"
                    : "none";
              return (
                <th key={column.key} scope="col" aria-sort={ariaSort}>
                  <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {page.queue.rows.map((row) => (
            <tr key={`${row.issue}-${row.source}`}>
              <th scope="row">{row.issue}</th>
              <td data-label="Source">{row.source}</td>
              <td data-label="Owner">{row.owner}</td>
              <td data-label="Impact">{readableLine(row.impact)}</td>
              <td data-label="Next Action">{readableLine(row.next_action)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

/**
 * DataQualityPage is the `/data-quality` React route. It fetches the DEV mock
 * payload, handles auth errors by delegating to the app-level router, wires
 * refresh and table sorting hotspots, and renders shared-shell overlays around
 * the PEN-generated Data Quality artboard.
 */
export function DataQualityPage({
  session,
  onNavigate,
  onSearch,
  searchQuery = "",
  onUnauthorized,
  onForbidden,
}) {
  const dataQualityQuery = useQuery({
    queryKey: ["dev-page", "data-quality", session?.current_persona?.id ?? "", session?.current_site_id ?? ""],
    enabled: Boolean(session?.authenticated && session?.authorized),
    queryFn: ({ signal }) => fetchDevApiJSON(DATA_QUALITY_ENDPOINT, { signal }),
  });
  const payload = dataQualityQuery.data ?? null;
  const pageState = dataQualityQuery.isLoading || dataQualityQuery.isFetching ? "loading" : dataQualityQuery.isError ? "error" : "ready";
  const errorMessage =
    dataQualityQuery.error instanceof Error ? dataQualityQuery.error.message : "Unable to load Data Quality mock data.";
  const table = useRuntimeTableData(payload?.page?.queue?.rows ?? [], DATA_QUALITY_QUEUE_COLUMNS, {
    defaultSort: { key: "issue", direction: "asc" },
  });

  useEffect(() => {
    if (dataQualityQuery.isError) {
      handleDevApiAuthError(dataQualityQuery.error, { onUnauthorized, onForbidden });
    }
  }, [dataQualityQuery.error, dataQualityQuery.isError, onForbidden, onUnauthorized]);

  const viewPayload = useMemo(() => {
    if (!payload) {
      return null;
    }

    return {
      ...payload,
      page: {
        ...payload.page,
        queue: {
          ...payload.page.queue,
          rows: table.visibleRows,
        },
      },
    };
  }, [payload, table.visibleRows]);

  const textOverrides = useMemo(
    () => buildDataQualityTextOverrides(session, viewPayload, table.sortState),
    [session, table.sortState, viewPayload]
  );
  const refreshDataQuality = useCallback(() => {
    void dataQualityQuery.refetch();
  }, [dataQualityQuery]);

  const hotspots = useMemo(() => {
    if (!viewPayload?.hotspots || pageState !== "ready") {
      return {};
    }

    const mapping = {};
    const refreshNodeId = viewPayload.hotspots.refresh?.node_id;
    if (refreshNodeId) {
      mapping[refreshNodeId] = {
        label: viewPayload.hotspots.refresh.label,
        onClick: refreshDataQuality,
      };
    }

    Object.entries(dataQualityDesign.slots.queue.headers || {}).forEach(([key, nodeId]) => {
      mapping[nodeId] = {
        label: `Sort by ${QUEUE_SORT_HEADERS[key]?.label ?? key}`,
        onClick: () => table.toggleSort(key),
      };
    });

    return mapping;
  }, [pageState, refreshDataQuality, table, viewPayload]);

  const imageNodeOverrides = useMemo(
    () => buildSharedShellImageOverrides(session),
    [session]
  );
  const hiddenNodeIds = useMemo(
    () => [
      ...buildSharedShellHiddenNodeIds(session, {
        hideNavHighlight: true,
        hideSearchPlaceholder: true,
        hideAllNavGroups: true,
      }),
      dataQualityDesign.slots.page.lastRefreshed,
    ],
    [session]
  );
  const renderOverlay = useMemo(
    () =>
      createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: "dataQuality",
        activeRoutePath: "/data-quality",
        pageSyncControl: {
          label: "Refresh",
          loadingLabel: "Refreshing",
          lastRefreshed: viewPayload?.page?.last_refreshed ?? null,
          disabled: pageState === "loading",
          loading: pageState === "loading",
          onAction: refreshDataQuality,
        },
      }),
    [onNavigate, onSearch, pageState, refreshDataQuality, searchQuery, session, viewPayload?.page?.last_refreshed]
  );

  /**
   * overlay renders transient loading/error/empty states above the generated
   * artboard while preserving the shared shell and accessible status messages.
   */
  const overlay = (() => {
    if (pageState === "loading") {
      return (
        <AccessDenied
          role="status"
          title="Loading Data Quality"
          message="Loading the DEV mock payload from the Go backend."
        />
      );
    }
    if (pageState === "error") {
      return (
        <AccessDenied
          title="Unable to load Data Quality"
          message={errorMessage || "Request failed."}
        />
      );
    }
    return null;
  })();

  return (
    <main
      id={MAIN_CONTENT_ID}
      className="page-canvas page-canvas--semantic"
      tabIndex="-1"
      aria-busy={pageState === "loading" ? "true" : undefined}
      aria-labelledby={payload ? DATA_QUALITY_HEADING_ID : undefined}
    >
      <div className="page-canvas__frame">
        {/* WCAG 1.3.1/1.4.10/2.4.6: semantic mirror preserves structure and becomes the reflow UI. */}
        <DataQualitySemanticContent
          payload={viewPayload}
          onRefresh={refreshDataQuality}
          table={table}
        />
        <DataQualityGeneratedView
          textOverrides={textOverrides}
          hotspots={hotspots}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
        {overlay}
      </div>
    </main>
  );
}
