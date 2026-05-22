import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { nextRuntimeDrawerSelectionForId } from "../components/runtimeDrawerController.mjs";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { fetchDevApiJSON, handleDevApiAuthError } from "../lib/devApi";
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
const SECURITY_ISSUES_ENDPOINT = "/api/v1/dev/pages/reports/security-issues";
const SECURITY_ISSUES_HEADING_ID = "security-issues-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1260;
const SECURITY_ISSUE_COLUMNS = [
  { key: "status", label: "Status", value: (row) => row.status },
  { key: "person", label: "Person / Account", value: (row) => row.person },
  { key: "email", label: "Email", value: (row) => row.email },
  { key: "site", label: "Site", value: (row) => row.site },
  { key: "next_action", label: "Next Action", value: (row) => row.next_action },
  { key: "asset_work", label: "Asset Work", value: (row) => row.asset_work },
  { key: "reference", label: "Reference", value: (row) => row.external_reference || "None" },
];

/**
 * collectAllNodeIds recursively collects generated report-pane node ids so this
 * page can hide the static Reports review artifacts and place its runtime table
 * over the shared Reports shell without hand-editing generated artboard JSON.
 */
function collectAllNodeIds(node, ids) {
  ids.push(node.id);
  for (const child of node.children || []) {
    collectAllNodeIds(child, ids);
  }
}

/**
 * collectPaneNodeIds finds generated nodes in the report content pane. The
 * Security Issues route reuses the Reports artboard shell while React owns the
 * live report content, so these static pane nodes are hidden at render time.
 */
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

/**
 * statusClass maps migrated Offboarding security issue statuses to the shared
 * severity colors already used by runtime tables. Security-risk rows stay
 * critical so IT Admin can distinguish them from ordinary report freshness.
 */
function statusClass(status) {
  if (["Blocked", "Invalid", "Failed", "Error", "Incomplete Data", "Warning", "Security risk"].includes(status)) {
    return "reports-runtime__status reports-runtime__status--critical";
  }
  if (["Needs Review", "Review", "Manual action", "External action"].includes(status)) {
    return "reports-runtime__status reports-runtime__status--review";
  }
  if (["Ready", "Healthy", "Complete", "Allowed"].includes(status)) {
    return "reports-runtime__status reports-runtime__status--ready";
  }
  return "reports-runtime__status reports-runtime__status--neutral";
}

/**
 * SummaryCards renders the small read-only Security Issues counters returned by
 * the DEV endpoint. The values are informational only; all actionable detail is
 * still in the searchable table and row drawer.
 */
function SummaryCards({ cards }) {
  if (!cards?.length) {
    return null;
  }
  return (
    <section className="reports-runtime__cards security-issues-runtime__cards" aria-label="Security issue summary cards">
      {cards.map((card) => (
        <article key={card.title} className="reports-runtime__card">
          <h2>{card.title}</h2>
          <strong>{card.count}</strong>
        </article>
      ))}
    </section>
  );
}

/**
 * SecurityIssueTable renders the migrated account-security rows with local
 * search and three-way sort. It intentionally does not expose Offboarding date
 * editing because the report is IT Admin review-only in this slice.
 */
function SecurityIssueTable({ rows, selectedId, onSelect }) {
  const columns = useMemo(() => SECURITY_ISSUE_COLUMNS, []);
  const table = useRuntimeTableData(rows, columns, {
    defaultSort: { key: "person", direction: "asc" },
  });
  return (
    <section className="reports-runtime__table-card" aria-label="Security issue accounts">
      <h2>Security Issue Accounts</h2>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="reports-runtime__table-header security-issues-runtime__table-header">
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
            className={`reports-runtime__row security-issues-runtime__row ${
              selectedId === row.id ? "reports-runtime__row--selected" : ""
            }`}
            aria-label={`Open security issue row for ${row.person}`}
            aria-pressed={selectedId === row.id}
            onClick={() => onSelect(nextRuntimeDrawerSelectionForId(selectedId, row))}
          >
            <div><span className={statusClass(row.status)}>{row.status}</span></div>
            <div>{row.person}</div>
            <div>{row.email}</div>
            <div>{row.site}</div>
            <div>{row.next_action}</div>
            <div>{row.asset_work}</div>
            <div>{row.external_reference || "None"}</div>
          </button>
        ))}
      </div>
    </section>
  );
}

/**
 * SecurityIssueDrawer shows the migrated detail/action context from the old
 * Offboarding security-risk row. Links remain deterministic DEV mock URLs, and
 * there are no mutation controls because this report is read-only for now.
 */
function SecurityIssueDrawer({ row, onClose }) {
  if (!row) {
    return null;
  }
  return (
    <RuntimeDrawer title={row.person} onClose={onClose} className="security-issues-runtime__drawer">
      {row.warning ? (
        <div className="security-issues-runtime__warning">
          <strong>Security issue</strong>
          <p>{row.warning}</p>
        </div>
      ) : null}
      <RuntimeDetailList
        items={[
          { label: "Status", value: row.status },
          { label: "Email", value: row.email },
          { label: "Site", value: row.site },
          { label: "End date", value: row.end_date || "Not set" },
          { label: "End date source", value: row.end_date_source },
          { label: "Next Action", value: row.next_action },
          { label: "Asset Work", value: row.asset_work },
          { label: "Reference", value: row.external_reference || "None" },
        ]}
      />
      {row.details?.length ? (
        <div className="runtime-drawer__section">
          <RuntimeDetailList items={row.details} />
        </div>
      ) : null}
      {row.actions?.length ? (
        <div className="security-issues-runtime__actions">
          <h3>Review Actions</h3>
          {row.actions.map((action) => (
            <section key={action.name} className="security-issues-runtime__action">
              <div>
                <strong>{action.name}</strong>
                <span className={statusClass(action.status)}>{action.status}</span>
              </div>
              <p><strong>Owner:</strong> {action.owner}</p>
              <p>{action.detail}</p>
              <p>{action.resolution}</p>
              {action.links?.length ? (
                <ul>
                  {action.links.map((link) => (
                    <li key={`${action.name}-${link.label}`}>
                      <a href={link.href} target="_blank" rel="noreferrer">{link.label}</a>
                      <span>{link.system}</span>
                    </li>
                  ))}
                </ul>
              ) : null}
            </section>
          ))}
        </div>
      ) : null}
    </RuntimeDrawer>
  );
}

/**
 * SecurityIssuesOverlay positions the report content over the Reports artboard
 * pane. React owns the live security report behavior, while the generated PEN
 * source still owns the shared Reports shell geometry.
 */
function SecurityIssuesOverlay({ payload, selectedRow, onSelect }) {
  const rows = payload?.page?.rows ?? [];
  return (
    <section
      className="reports-runtime security-issues-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        zIndex: 2,
      }}
      aria-labelledby={SECURITY_ISSUES_HEADING_ID}
    >
      <header className="reports-runtime__header">
        <div>
          <h1 id={SECURITY_ISSUES_HEADING_ID}>{payload?.page?.title || "Security Issues Report"}</h1>
          <p>{payload?.page?.description || "Account-security issues that need IT Admin review."}</p>
        </div>
      </header>
      <SummaryCards cards={payload?.page?.summary_cards ?? []} />
      <SecurityIssueTable rows={rows} selectedId={selectedRow?.id} onSelect={onSelect} />
    </section>
  );
}

/**
 * SecurityIssuesReportPage renders /reports/security-issues. The page fetches
 * its DEV read model, reuses the Reports generated shell, and shows migrated
 * security-risk rows without exposing Offboarding's HR-oriented mutation path.
 */
export function SecurityIssuesReportPage({ session, onNavigate, onSearch, searchQuery, onUnauthorized, onForbidden }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [selectedRow, setSelectedRow] = useState(null);
  const securityIssuesQuery = useQuery({
    queryKey: ["dev-page", "reports", "security-issues", session?.current_persona?.id ?? ""],
    enabled: Boolean(session?.authenticated && session?.authorized),
    queryFn: ({ signal }) => fetchDevApiJSON(SECURITY_ISSUES_ENDPOINT, { signal }),
  });
  const payload = securityIssuesQuery.data ?? null;
  const pageState = securityIssuesQuery.isLoading || securityIssuesQuery.isFetching ? "loading" : securityIssuesQuery.isError ? "error" : "ready";

  useEffect(() => {
    setSelectedRow(null);
  }, [payload]);

  useEffect(() => {
    if (securityIssuesQuery.isError) {
      handleDevApiAuthError(securityIssuesQuery.error, { onUnauthorized, onForbidden });
    }
  }, [onForbidden, onUnauthorized, securityIssuesQuery.error, securityIssuesQuery.isError]);

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
    activeRoutePath: "/reports/security-issues",
    refreshMetadata: payload?.page?.last_refreshed ?? staticRefreshMetadataForArtboard(ARTBOARD_KEY),
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Security Issues Report",
        textOverrides,
      })
    : { title: "Security Issues Report", items: [] };
  const rows = payload?.page?.rows ?? [];
  const selectedPayloadRow = selectedRow ? rows.find((row) => row.id === selectedRow.id) || selectedRow : null;

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <SecurityIssuesOverlay payload={payload} selectedRow={selectedPayloadRow} onSelect={setSelectedRow} />
      <SecurityIssueDrawer row={selectedPayloadRow} onClose={() => setSelectedRow(null)} />
    </>
  ), [payload, selectedPayloadRow, sharedShellRenderOverlay]);

  if (artboardStatus === "loading" || pageState === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Security Issues</h1>
          <p>Loading the DEV security issues report.</p>
        </section>
      </main>
    );
  }

  if (pageState === "error") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Security Issues unavailable</h1>
          <p>The DEV security issues report could not be loaded.</p>
        </section>
      </main>
    );
  }

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Security Issues unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={SECURITY_ISSUES_HEADING_ID}>
      <section className="sr-only" aria-labelledby={`${SECURITY_ISSUES_HEADING_ID}-summary`}>
        <h1 id={`${SECURITY_ISSUES_HEADING_ID}-summary`}>{payload?.page?.title || semanticSummary.title}</h1>
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
