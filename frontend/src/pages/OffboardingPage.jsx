import { useCallback, useEffect, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
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

const OFFBOARDING_ENDPOINT = "/api/v1/dev/pages/offboarding";
const OFFBOARDING_RECORDS_ENDPOINT = "/api/v1/dev/offboarding/records";
const OFFBOARDING_HEADING_ID = "offboarding-heading";
const OFFBOARDING_TABLE_FRAME_NODE_ID = "offboarding__f115";
const STATIC_OFFBOARDING_NODE_IDS = [
  "t116", "t117", "t118", "t119", "t120", "t121", "t122", "l123",
  "t124", "t125", "t126", "f128", "t129", "t130", "t131", "l132",
  "t133", "t134", "t135", "f137", "t138", "t139", "t140", "l141",
  "t142", "t143", "t144", "f146", "t147", "t148", "t149", "l150",
  "t151", "t152", "t153", "f155", "t156", "t157", "t158", "l159",
  "f161", "t162", "p163", "p164", "t165", "t166", "t167", "t168",
  "t169", "t170", "t172", "t173", "l175", "f176", "t177", "f178",
  "t179", "f180", "t181",
].map((id) => `offboarding__${id}`);

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

function nodeBox(node) {
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

function formatDate(value) {
  if (!value) {
    return "Not set";
  }
  const parsed = new Date(`${value}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

function statusClass(status) {
  if (["Ready", "Ready to Provision", "Healthy", "Complete", "Allowed"].includes(status)) {
    return "offboarding-runtime__status offboarding-runtime__status--ready";
  }
  if (["Blocked", "Invalid", "Failed", "Error", "Incomplete Data", "Warning", "Security risk"].includes(status)) {
    return "offboarding-runtime__status offboarding-runtime__status--critical";
  }
  if (["Needs Review", "Review", "Manual action", "External action"].includes(status)) {
    return "offboarding-runtime__status offboarding-runtime__status--review";
  }
  if (["Queued", "Scheduled", "Waiting"].includes(status)) {
    return "offboarding-runtime__status offboarding-runtime__status--waiting";
  }
  if (["In Progress", "Running"].includes(status)) {
    return "offboarding-runtime__status offboarding-runtime__status--active";
  }
  return "offboarding-runtime__status offboarding-runtime__status--neutral";
}

function OffboardingWarning({ id, text }) {
  if (!text) {
    return null;
  }
  return (
    <span className="offboarding-runtime__warning" tabIndex={0} aria-describedby={id}>
      <span aria-hidden="true">!</span>
      <span id={id} role="tooltip" className="offboarding-runtime__warning-tooltip">
        {text}
      </span>
    </span>
  );
}

function OffboardingTableOverlay({ bounds, rows, selectedRowId, showEmployeeIDs, onSelectRow }) {
  if (!bounds) {
    return null;
  }
  return (
    <section
      className={`offboarding-runtime__table ${showEmployeeIDs ? "offboarding-runtime__table--with-ids" : ""}`}
      style={{
        position: "absolute",
        left: bounds.left + 18,
        top: bounds.top + 14,
        width: Math.max(0, bounds.width - 36),
        height: Math.max(0, bounds.height - 28),
        zIndex: 2,
      }}
      aria-labelledby={OFFBOARDING_HEADING_ID}
    >
      <div className="offboarding-runtime__table-title">Upcoming Offboarding</div>
      <div className="offboarding-runtime__table-header">
        <div>Status</div>
        <div>Person / Account</div>
        <div>Email</div>
        {showEmployeeIDs ? <div>Employee ID</div> : null}
        <div>Site</div>
        <div>End</div>
        <div>Next Action</div>
        <div>Asset Work</div>
      </div>
      <div className="offboarding-runtime__table-body">
        {rows.map((row) => (
          <button
            key={row.id}
            type="button"
            className={`offboarding-runtime__row ${
              selectedRowId === row.id ? "offboarding-runtime__row--selected" : ""
            }`}
            aria-label={`Open offboarding row for ${row.person}`}
            aria-pressed={selectedRowId === row.id}
            onClick={() => onSelectRow(row)}
          >
            <div className="offboarding-runtime__status-cell">
              <span className={statusClass(row.status)}>{row.status}</span>
            </div>
            <div className="offboarding-runtime__person-cell">
              <span>{row.person}</span>
              <OffboardingWarning id={`offboarding-warning-${row.id}`} text={row.warning} />
            </div>
            <div>{row.email}</div>
            {showEmployeeIDs ? <div>{row.employee_id || "None"}</div> : null}
            <div>{row.site}</div>
            <div>{formatDate(row.end_date)}</div>
            <div>{row.next_action}</div>
            <div>{row.asset_work}</div>
          </button>
        ))}
      </div>
    </section>
  );
}

function OffboardingDrawer({ row, canManageEndDates, onClose, onSaveEndDate }) {
  const [endDate, setEndDate] = useState(row?.end_date || "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setEndDate(row?.end_date || "");
    setError("");
    setSaving(false);
  }, [row]);

  if (!row) {
    return null;
  }

  const editable = row.end_date_editable && canManageEndDates;
  const handleSave = async () => {
    setSaving(true);
    setError("");
    try {
      await onSaveEndDate(row, endDate);
    } catch (saveError) {
      setError(saveError.payload?.errors?.end_date || saveError.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <RuntimeDrawer title={row.person} onClose={onClose}>
      {row.warning ? (
        <div className="offboarding-runtime__drawer-warning">
          <strong>Flagged account</strong>
          <p>{row.warning}</p>
        </div>
      ) : null}
      <RuntimeDetailList
        items={[
          { label: "Status", value: row.status },
          { label: "Email", value: row.email },
          { label: "Employee ID", value: row.employee_id },
          { label: "Site", value: row.site },
          { label: "End date", value: formatDate(row.end_date) },
          { label: "End date source", value: row.end_date_source },
          { label: "Next Action", value: row.next_action },
          { label: "Asset Work", value: row.asset_work },
          { label: "Reference", value: row.external_reference },
        ]}
      />
      <div className="runtime-drawer__section">
        {editable ? (
          <form className="offboarding-runtime__end-date-form" onSubmit={(event) => event.preventDefault()}>
            <label htmlFor="offboarding-end-date">
              <span>End date</span>
              <input
                id="offboarding-end-date"
                type="date"
                value={endDate}
                onChange={(event) => setEndDate(event.target.value)}
              />
            </label>
            <button type="button" onClick={handleSave} disabled={saving}>
              {saving ? "Saving..." : "Save End Date"}
            </button>
            {error ? <p role="alert">{error}</p> : null}
          </form>
        ) : (
          <p>
            <strong>End date:</strong>
            <span>
              {row.end_date_source === "Escape"
                ? " Escape is authoritative. Correct this date upstream in Escape."
                : " This end date is read-only for your current role."}
            </span>
          </p>
        )}
      </div>
      {row.details?.length ? (
        <div className="runtime-drawer__section">
          <RuntimeDetailList items={row.details} />
        </div>
      ) : null}
      {row.actions?.length ? (
        <div className="offboarding-runtime__actions">
          <h3>Offboarding Actions</h3>
          {row.actions.map((action) => (
            <section key={action.name} className="offboarding-runtime__action">
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

export function OffboardingPage({ session, onNavigate, onSearch, searchQuery = "", onUnauthorized, onForbidden }) {
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [selectedRow, setSelectedRow] = useState(null);

  const artboard = generatedArtboards.offboarding;
  const meta = generatedArtboardMeta.offboarding;
  const personaId = session?.current_persona?.id ?? "";

  const loadPage = useCallback(async () => {
    setPageState("loading");
    try {
      const nextPayload = await readJSON(
        await fetch(OFFBOARDING_ENDPOINT, {
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
  }, [onForbidden, onUnauthorized, personaId]);

  useEffect(() => {
    setSelectedRow(null);
    void loadPage();
  }, [loadPage]);

  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...STATIC_OFFBOARDING_NODE_IDS);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? null,
    refreshMetadata: payload?.page?.last_refreshed ?? staticRefreshMetadataForArtboard("offboarding"),
  });
  const semanticSummary = buildArtboardSemanticSummary(artboard, {
    fallbackTitle: "Offboarding Dashboard",
    textOverrides,
  });
  const rows = payload?.page?.rows ?? [];
  const selectedPayloadRow = selectedRow ? rows.find((row) => row.id === selectedRow.id) || selectedRow : null;

  const handleSaveEndDate = useCallback(async (row, endDate) => {
    const updated = await readJSON(
      await fetch(`${OFFBOARDING_RECORDS_ENDPOINT}/${row.id}/end-date`, {
        method: "PUT",
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify({ end_date: endDate }),
      })
    );
    setSelectedRow(updated.row);
    await loadPage();
  }, [loadPage]);

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => {
    const tableBounds = nodeBox(nodeIndex.get(OFFBOARDING_TABLE_FRAME_NODE_ID));
    return (
      <>
        {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
        <OffboardingTableOverlay
          bounds={tableBounds}
          rows={rows}
          selectedRowId={selectedPayloadRow?.id}
          showEmployeeIDs={Boolean(payload?.page?.show_employee_ids)}
          onSelectRow={setSelectedRow}
        />
        <OffboardingDrawer
          row={selectedPayloadRow}
          canManageEndDates={Boolean(payload?.page?.can_manage_end_dates)}
          onClose={() => setSelectedRow(null)}
          onSaveEndDate={handleSaveEndDate}
        />
      </>
    );
  }, [handleSaveEndDate, payload, rows, selectedPayloadRow, sharedShellRenderOverlay]);

  if (pageState === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Offboarding</h1>
          <p>Loading the DEV offboarding dashboard.</p>
        </section>
      </main>
    );
  }

  if (pageState === "error") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Offboarding unavailable</h1>
          <p>The DEV offboarding dashboard could not be loaded.</p>
        </section>
      </main>
    );
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={OFFBOARDING_HEADING_ID}>
      <section className="sr-only" aria-labelledby={OFFBOARDING_HEADING_ID}>
        <h1 id={OFFBOARDING_HEADING_ID}>{payload?.page?.title || semanticSummary.title}</h1>
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
