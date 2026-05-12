import { useCallback, useEffect, useMemo, useState } from "react";
import { AccessDenied } from "../components/AccessDenied";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import {
  DataQualityGeneratedView,
  dataQualityDesign,
} from "../generated/data-quality.generated.jsx";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const apiOrigin = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";
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

function readableLine(value) {
  return String(value ?? "").replaceAll("\n", " ");
}

function assignValue(overrides, slotId, value) {
  if (!slotId || value == null) {
    return;
  }
  overrides[slotId] = String(value);
}

function assignLines(overrides, slotIds, value) {
  const values = Array.isArray(value) ? value : String(value ?? "").split("\n");
  slotIds.forEach((slotId, index) => {
    overrides[slotId] = values[index] ?? "";
  });
}

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
  assignValue(overrides, dataQualityDesign.slots.page.description, page.description);
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

  assignValue(overrides, dataQualityDesign.slots.routingCard.title, page.routing_card.title);
  assignValue(overrides, dataQualityDesign.slots.routingCard.headline, page.routing_card.headline);
  assignValue(overrides, dataQualityDesign.slots.routingCard.body, page.routing_card.body);

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

  assignValue(overrides, dataQualityDesign.slots.routingRules.title, page.routing_rules.title);
  page.routing_rules.rules.forEach((rule, index) => {
    const slot = dataQualityDesign.slots.routingRules.rows[index];
    if (!slot) {
      return;
    }
    assignValue(overrides, slot.queue, rule.queue);
    assignValue(overrides, slot.description, rule.description);
  });
  assignValue(
    overrides,
    dataQualityDesign.slots.routingRules.primaryActionLabel,
    page.routing_rules.primary_action_label
  );

  return overrides;
}

function DataQualitySemanticContent({ payload, onRefresh, mappingHref, table }) {
  if (!payload) {
    return null;
  }

  const { page } = payload;

  return (
    <section className="data-quality-semantic" aria-labelledby={DATA_QUALITY_HEADING_ID}>
      <div className="data-quality-semantic__header">
        <div>
          <h1 id={DATA_QUALITY_HEADING_ID}>{page.title}</h1>
          <p>{page.description}</p>
          <p style={{ whiteSpace: "pre-line" }}>{page.last_refreshed}</p>
        </div>
        <div className="data-quality-semantic__mobile-actions">
          <button type="button" onClick={onRefresh}>
            Refresh Data Quality
          </button>
          <a href={mappingHref} target="_blank" rel="noopener noreferrer">
            Open Mapping Dashboard
          </a>
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

      <h2>{page.routing_rules.title}</h2>
      <ul>
        {page.routing_rules.rules.map((rule) => (
          <li key={rule.queue}>
            <strong>{rule.queue}</strong>: {readableLine(rule.description)}
          </li>
        ))}
      </ul>
    </section>
  );
}

export function DataQualityPage({
  session,
  onNavigate,
  onSearch,
  searchQuery = "",
  onUnauthorized,
  onForbidden,
}) {
  const [pageState, setPageState] = useState("loading");
  const [payload, setPayload] = useState(null);
  const [errorMessage, setErrorMessage] = useState("");
  const [reloadKey, setReloadKey] = useState(0);
  const table = useRuntimeTableData(payload?.page?.queue?.rows ?? [], DATA_QUALITY_QUEUE_COLUMNS, {
    defaultSort: { key: "issue", direction: "asc" },
  });

  useEffect(() => {
    if (!session?.authenticated || !session?.authorized) {
      return undefined;
    }

    const controller = new AbortController();

    async function loadPage() {
      setPageState("loading");
      setErrorMessage("");
      try {
        const response = await fetch(DATA_QUALITY_ENDPOINT, {
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
          throw new Error(`Data Quality request failed with ${response.status}`);
        }
        const nextPayload = await response.json();
        setPayload(nextPayload);
        setPageState("ready");
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        setPayload(null);
        setErrorMessage(
          error instanceof Error ? error.message : "Unable to load Data Quality mock data."
        );
        setPageState("error");
      }
    }

    void loadPage();
    return () => controller.abort();
  }, [onForbidden, onUnauthorized, reloadKey, session]);

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
  const mappingDashboardHref = `${apiOrigin}/sync-dashboard/mappings`;
  const refreshDataQuality = useCallback(() => setReloadKey((value) => value + 1), []);

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

    const mappingNodeId = viewPayload.hotspots.open_mapping_dashboard?.node_id;
    if (mappingNodeId) {
      mapping[mappingNodeId] = {
        label: viewPayload.hotspots.open_mapping_dashboard.label,
        href: mappingDashboardHref,
        target: "_blank",
        rel: "noopener noreferrer",
      };
    }

    Object.entries(dataQualityDesign.slots.queue.headers || {}).forEach(([key, nodeId]) => {
      mapping[nodeId] = {
        label: `Sort by ${QUEUE_SORT_HEADERS[key]?.label ?? key}`,
        onClick: () => table.toggleSort(key),
      };
    });

    return mapping;
  }, [mappingDashboardHref, pageState, refreshDataQuality, table, viewPayload]);

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
        refreshMetadata: viewPayload?.page?.last_refreshed ?? null,
      }),
    [onNavigate, onSearch, searchQuery, session, viewPayload?.page?.last_refreshed]
  );

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
          mappingHref={mappingDashboardHref}
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
