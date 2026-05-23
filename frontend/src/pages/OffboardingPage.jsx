import { useCallback, useEffect, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { nextRuntimeDrawerSelectionForId } from "../components/runtimeDrawerController.mjs";
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

const OFFBOARDING_ENDPOINT = "/api/v1/dev/pages/offboarding";
const OFFBOARDING_RECORDS_ENDPOINT = "/api/v1/dev/offboarding/records";
const OFFBOARDING_CANDIDATES_ENDPOINT = "/api/v1/dev/offboarding/candidates";
const OFFBOARDING_EMERGENCY_ENDPOINT = "/api/v1/dev/offboarding/emergency-deprovision";
const OFFBOARDING_CONTRACTOR_ENDPOINT = "/api/v1/dev/offboarding/contractor-offboarding";
const OFFBOARDING_HEADING_ID = "offboarding-heading";
const OFFBOARDING_TABLE_FRAME_NODE_ID = "offboarding__f115";
const OFFBOARDING_TABLE_COLUMNS = [
  { key: "status", label: "Status", value: (row) => row.status },
  { key: "person", label: "Person / Account", value: (row) => row.person },
  { key: "email", label: "Email", value: (row) => row.email },
  { key: "employee_id", label: "Employee ID", value: (row) => row.employee_id || "None" },
  { key: "site", label: "Site", value: (row) => row.site },
  { key: "end_date", label: "End", value: (row) => formatDate(row.end_date), sortValue: (row) => row.end_date || "" },
  { key: "next_action", label: "Next Action", value: (row) => row.next_action },
  { key: "asset_work", label: "Asset Work", value: (row) => row.asset_work },
];
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

/**
 * readJSON turns DEV Offboarding API responses into page state. It preserves
 * HTTP status and backend error payloads so page loaders, end-date saves, and
 * manual offboarding drawers can route 401/403 through the app-level guards.
 */
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

/**
 * nodeBox converts a generated artboard node into absolute overlay bounds for
 * the runtime Offboarding table. Missing nodes return null so the page can
 * render safely if the generated frame contract drifts.
 */
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

/**
 * formatDate presents DEV ISO dates for Offboarding tables and drawers. Empty
 * dates remain explicit as Not set, and invalid seed values are returned
 * unchanged to avoid hiding mock-data mistakes during review.
 */
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

/**
 * formatDateTime presents scheduled Emergency Offboarding effective timestamps
 * returned by the DEV mock schedule API. It preserves immediate actions as a
 * distinct label so operators can distinguish destructive-now work from future
 * scheduled work in drawer confirmation text.
 */
function formatDateTime(value) {
  if (!value || value === "immediate") {
    return "immediate DEV mock deprovisioning";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/**
 * scheduledEmergencyTimestamp converts the datetime-local control value into
 * an explicit instant before the DEV mock API sees it. The browser interprets
 * the control value in the operator's local timezone, and toISOString preserves
 * that intended instant with a UTC offset marker instead of making the Go
 * handler guess a timezone.
 */
function scheduledEmergencyTimestamp(value) {
  if (!value) {
    return "";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toISOString();
}

/**
 * formatScheduledActionConfirmation keeps contractor date-only confirmations on
 * the date formatter while Emergency Offboarding confirmations use timestamp
 * formatting. Contractor actions still return YYYY-MM-DD dates from the DEV
 * shortcut endpoint, so parsing them as datetimes would add an unintended time
 * and can shift the displayed day for some timezones.
 */
function formatScheduledActionConfirmation(action) {
  if (!action) {
    return "";
  }
  if (action.kind === "contractor_scheduled_deprovision") {
    return formatDate(action.scheduled_for);
  }
  return formatDateTime(action.scheduled_for);
}

/**
 * statusClass maps Offboarding workflow states onto the shared runtime severity
 * palette. It keeps row badges, drawer action badges, and mock schedule status
 * styling consistent without changing the source status text.
 */
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

/**
 * OffboardingWarning renders the focusable warning marker used by table rows
 * that need HR/IT review. The tooltip carries only backend-authorized row text,
 * so personas without employee-id access do not receive hidden detail here.
 */
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

/**
 * OffboardingTableOverlay places the live sortable/searchable table over the
 * hidden static .pen table frame. It uses the backend's showEmployeeIDs flag to
 * remove the employee-id column entirely for site-scoped viewers.
 */
function OffboardingTableOverlay({ bounds, rows, selectedRowId, showEmployeeIDs, onSelectRow }) {
  const columns = showEmployeeIDs
    ? OFFBOARDING_TABLE_COLUMNS
    : OFFBOARDING_TABLE_COLUMNS.filter((column) => column.key !== "employee_id");
  const table = useRuntimeTableData(rows, columns, {
    defaultSort: { key: "end_date", direction: "asc" },
  });

  if (!bounds) {
    return null;
  }
  const visibleRows = table.visibleRows;
  return (
    <section
      className={`offboarding-runtime__table ${showEmployeeIDs ? "offboarding-runtime__table--with-ids" : ""}`}
      style={{
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: Math.max(0, bounds.width),
        height: Math.max(0, bounds.height + 96),
        zIndex: 2,
      }}
      aria-labelledby={OFFBOARDING_HEADING_ID}
    >
      <div className="offboarding-runtime__table-title">Upcoming Offboarding</div>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="offboarding-runtime__table-header">
        {columns.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
      </div>
      <div className="offboarding-runtime__table-body">
        {visibleRows.map((row) => (
          <button
            key={row.id}
            type="button"
            className={`offboarding-runtime__row ${
              selectedRowId === row.id ? "offboarding-runtime__row--selected" : ""
            }`}
            aria-label={`Open offboarding row for ${row.person}`}
            aria-pressed={selectedRowId === row.id}
            onClick={() => onSelectRow(nextRuntimeDrawerSelectionForId(selectedRowId, row))}
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

/**
 * candidateMatches filters the HR/IT-only drawer search results already loaded
 * from /api/v1/dev/offboarding/candidates. It searches only fields that the
 * backend has authorized for this drawer: display name, district email, and
 * employee ID.
 */
function candidateMatches(candidate, query) {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return true;
  }
  return [candidate.person, candidate.email, candidate.employee_id]
    .filter(Boolean)
    .some((value) => value.toLowerCase().includes(normalizedQuery));
}

/**
 * OffboardingActionBar renders the runtime-owned page actions for issue #161.
 * The backend-provided canManageManual flag hides the controls for non-HR and
 * non-IT personas; the schedule APIs repeat that authorization server-side. The
 * buttons stay in one vertical right-side group so the header action area does
 * not spread these related manual workflows across the page.
 */
function OffboardingActionBar({ canManageManual, onEmergency, onContractor }) {
  if (!canManageManual) {
    return null;
  }
  return (
    <div className="offboarding-runtime__action-bar">
      <button
        type="button"
        className="offboarding-runtime__page-action offboarding-runtime__page-action--danger"
        onClick={onEmergency}
      >
        Emergency Offboarding
      </button>
      <button
        type="button"
        className="offboarding-runtime__page-action offboarding-runtime__page-action--gold"
        onClick={onContractor}
      >
        Offboard Contractor
      </button>
    </div>
  );
}

/**
 * OffboardingManualActionDrawer owns the HR/IT emergency and contractor
 * offboarding workflows. It loads candidates only after the drawer opens and
 * submits explicit schedule payloads to DEV mock APIs so date edits alone never
 * mutate state or imply live provider write approval.
 */
function OffboardingManualActionDrawer({ mode, onClose, onUnauthorized, onForbidden }) {
  const [candidates, setCandidates] = useState([]);
  const [query, setQuery] = useState("");
  const [selectedCandidate, setSelectedCandidate] = useState(null);
  const [emergencyExecutionMode, setEmergencyExecutionMode] = useState("immediate");
  const [scheduledFor, setScheduledFor] = useState("");
  const [terminationDate, setTerminationDate] = useState("");
  const [state, setState] = useState("loading");
  const [error, setError] = useState("");
  const [scheduledAction, setScheduledAction] = useState(null);

  useEffect(() => {
    if (!mode) {
      return;
    }
    const controller = new AbortController();
    const loadCandidates = async () => {
      setState("loading");
      setError("");
      setScheduledAction(null);
      setSelectedCandidate(null);
      setEmergencyExecutionMode("immediate");
      setScheduledFor("");
      setTerminationDate("");
      try {
        const payload = await readJSON(
          await fetch(`${OFFBOARDING_CANDIDATES_ENDPOINT}?mode=${encodeURIComponent(mode)}`, {
            credentials: "same-origin",
            headers: { Accept: "application/json" },
            signal: controller.signal,
          })
        );
        setCandidates(payload?.candidates ?? []);
        setState("ready");
      } catch (loadError) {
        if (loadError.name === "AbortError") {
          return;
        }
        if (loadError.status === 401 && onUnauthorized) {
          onUnauthorized();
          return;
        }
        if (loadError.status === 403 && onForbidden) {
          onForbidden();
          return;
        }
        setState("error");
        setError(loadError.message);
      }
    };
    void loadCandidates();
    return () => controller.abort();
  }, [mode, onForbidden, onUnauthorized]);

  if (!mode) {
    return null;
  }

  const isEmergency = mode === "emergency";
  const title = isEmergency ? "Emergency Offboarding" : "Offboard Contractor";
  const visibleCandidates = candidates.filter((candidate) => candidateMatches(candidate, query));

  const handleSelectCandidate = (candidate) => {
    setSelectedCandidate(candidate);
    setScheduledAction(null);
    setError("");
    setTerminationDate(candidate.termination_date || "");
  };

  const handleSubmit = async () => {
    if (!selectedCandidate) {
      setError(isEmergency ? "Select an employee or contractor first." : "Select a contractor first.");
      return;
    }
    setState("saving");
    setError("");
    try {
      const endpoint = isEmergency ? OFFBOARDING_EMERGENCY_ENDPOINT : OFFBOARDING_CONTRACTOR_ENDPOINT;
      const body = isEmergency
        ? {
            person_id: selectedCandidate.id,
            execution_mode: emergencyExecutionMode,
            scheduled_for: scheduledEmergencyTimestamp(scheduledFor),
          }
        : { person_id: selectedCandidate.id, end_date: terminationDate };
      const payload = await readJSON(
        await fetch(endpoint, {
          method: "POST",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify(body),
        })
      );
      setScheduledAction(payload.action);
      setState("ready");
    } catch (submitError) {
      if (submitError.status === 401 && onUnauthorized) {
        onUnauthorized();
        return;
      }
      if (submitError.status === 403 && onForbidden) {
        onForbidden();
        return;
      }
      setError(
        submitError.payload?.errors?.scheduled_for ||
          submitError.payload?.errors?.execution_mode ||
          submitError.payload?.errors?.end_date ||
          submitError.payload?.errors?.person_id ||
          submitError.message
      );
      setState("ready");
    }
  };

  return (
    <RuntimeDrawer title={title} onClose={onClose} variant="modal">
      <div className={isEmergency ? "offboarding-runtime__manual-callout offboarding-runtime__manual-callout--danger" : "offboarding-runtime__manual-callout offboarding-runtime__manual-callout--gold"}>
        {isEmergency
          ? "Use Emergency Offboarding for urgent termination now or a scheduled future termination/deprovisioning action after HR or IT verifies the selected person."
          : "This form is to offboard a manually created contractor. To offboard an employee, update the Escape record(s)."}
      </div>
      <div className="runtime-drawer__section">
        <label className="offboarding-runtime__candidate-search" htmlFor={`offboarding-${mode}-search`}>
          <span>{isEmergency ? "Search active employees and contractors" : "Search active contractors"}</span>
          <input
            id={`offboarding-${mode}-search`}
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search by name, email, or employee ID"
          />
        </label>
        {state === "loading" ? <p>Loading candidates...</p> : null}
        {state === "error" ? <p role="alert">{error || "Candidate search could not be loaded."}</p> : null}
        {state !== "loading" && state !== "error" ? (
          <div className="offboarding-runtime__candidate-list" role="listbox" aria-label={isEmergency ? "Active offboarding candidates" : "Active contractor candidates"}>
            {visibleCandidates.map((candidate) => (
              <button
                key={candidate.id}
                type="button"
                role="option"
                aria-selected={selectedCandidate?.id === candidate.id}
                className={selectedCandidate?.id === candidate.id ? "offboarding-runtime__candidate offboarding-runtime__candidate--selected" : "offboarding-runtime__candidate"}
                onClick={() => handleSelectCandidate(candidate)}
              >
                <span>{candidate.person}</span>
                <span>{candidate.email}</span>
                <span>{candidate.employee_id}</span>
              </button>
            ))}
            {visibleCandidates.length === 0 ? <p>No matching candidates.</p> : null}
          </div>
        ) : null}
      </div>
      {selectedCandidate ? (
        <div className="runtime-drawer__section">
          <RuntimeDetailList
            items={[
              { label: "Name", value: selectedCandidate.person },
              { label: "Email", value: selectedCandidate.email },
              { label: "Employee ID", value: selectedCandidate.employee_id },
              { label: "Current termination date", value: formatDate(selectedCandidate.termination_date) },
            ]}
          />
        </div>
      ) : null}
      {isEmergency ? (
        <div className="runtime-drawer__section">
          <fieldset className="offboarding-runtime__mode-group">
            <legend>Execution mode</legend>
            <label>
              <input
                type="radio"
                name="offboarding-emergency-mode"
                value="immediate"
                checked={emergencyExecutionMode === "immediate"}
                onChange={() => {
                  setEmergencyExecutionMode("immediate");
                  setScheduledFor("");
                  setScheduledAction(null);
                }}
              />
              <span>Immediate termination</span>
            </label>
            <label>
              <input
                type="radio"
                name="offboarding-emergency-mode"
                value="scheduled"
                checked={emergencyExecutionMode === "scheduled"}
                onChange={() => {
                  setEmergencyExecutionMode("scheduled");
                  setScheduledAction(null);
                }}
              />
              <span>Scheduled termination</span>
            </label>
          </fieldset>
          {emergencyExecutionMode === "scheduled" ? (
            <label className="offboarding-runtime__candidate-search" htmlFor="offboarding-emergency-scheduled-for">
              <span>Effective date and time</span>
              <input
                id="offboarding-emergency-scheduled-for"
                type="datetime-local"
                value={scheduledFor}
                onChange={(event) => setScheduledFor(event.target.value)}
              />
            </label>
          ) : (
            <div className="offboarding-runtime__manual-callout offboarding-runtime__manual-callout--danger">
              Clicking this button will remove all access for this person. This action cannot be undone.
            </div>
          )}
          <div className="offboarding-runtime__drawer-actions">
            <button
              type="button"
              className={emergencyExecutionMode === "immediate" ? "offboarding-runtime__drawer-action offboarding-runtime__drawer-action--danger" : "offboarding-runtime__drawer-action offboarding-runtime__drawer-action--gold"}
              disabled={!selectedCandidate || state === "saving"}
              onClick={handleSubmit}
            >
              {state === "saving"
                ? "Scheduling..."
                : emergencyExecutionMode === "immediate"
                  ? "Deprovision Now"
                  : "Schedule Emergency Offboarding"}
            </button>
            <button
              type="button"
              className="offboarding-runtime__drawer-action offboarding-runtime__drawer-action--gold"
              onClick={onClose}
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <div className="runtime-drawer__section">
          <label className="offboarding-runtime__candidate-search" htmlFor="offboarding-contractor-date">
            <span>Termination date</span>
            <input
              id="offboarding-contractor-date"
              type="date"
              value={terminationDate}
              onChange={(event) => setTerminationDate(event.target.value)}
            />
          </label>
          <div className="offboarding-runtime__drawer-actions">
            <button
              type="button"
              className="offboarding-runtime__drawer-action offboarding-runtime__drawer-action--gold"
              disabled={!selectedCandidate || state === "saving"}
              onClick={handleSubmit}
            >
              {state === "saving" ? "Scheduling..." : "Schedule Offboarding"}
            </button>
            <button
              type="button"
              className="offboarding-runtime__drawer-action offboarding-runtime__drawer-action--gold"
              onClick={onClose}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
      {error && state !== "error" ? <p className="offboarding-runtime__form-error" role="alert">{error}</p> : null}
      {scheduledAction ? (
        <p className="offboarding-runtime__form-success" role="status">
          {scheduledAction.person} scheduled for {formatScheduledActionConfirmation(scheduledAction)}.
        </p>
      ) : null}
    </RuntimeDrawer>
  );
}

/**
 * OffboardingDrawer shows the selected row's source ownership, workflow steps,
 * and optional local end-date editor. Saving calls the DEV end-date API only
 * for rows the backend marked editable, keeping Escape-owned dates read-only.
 */
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
  /**
   * handleSave submits an explicit end-date update for the selected row. It
   * leaves the edited date in local form state until the backend accepts it and
   * then reloads the page payload so role-scoped table data stays authoritative.
   */
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

/**
 * OffboardingPage coordinates the generated Offboarding artboard, backend page
 * payload, row drawer, and HR/IT manual-action drawers. Its fetch handlers send
 * 401/403 responses back to the app router so direct unauthorized navigation
 * renders the documented login or access-denied state.
 */
export function OffboardingPage({ session, onNavigate, onSearch, searchQuery = "", onUnauthorized, onForbidden }) {
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [selectedRow, setSelectedRow] = useState(null);
  const [manualDrawerMode, setManualDrawerMode] = useState(null);

  const { artboard, status: artboardStatus } = useGeneratedArtboard("offboarding");
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
    setManualDrawerMode(null);
    void loadPage();
  }, [loadPage]);

  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(OFFBOARDING_TABLE_FRAME_NODE_ID, ...STATIC_OFFBOARDING_NODE_IDS);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? null,
    activeRoutePath: "/offboarding",
    refreshMetadata: payload?.page?.last_refreshed ?? staticRefreshMetadataForArtboard("offboarding"),
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Offboarding Dashboard",
        textOverrides,
      })
    : { title: "Offboarding Dashboard", items: [] };
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
        <OffboardingActionBar
          canManageManual={Boolean(payload?.page?.can_manage_manual)}
          onEmergency={() => {
            setSelectedRow(null);
            setManualDrawerMode("emergency");
          }}
          onContractor={() => {
            setSelectedRow(null);
            setManualDrawerMode("contractor");
          }}
        />
        <OffboardingDrawer
          row={selectedPayloadRow}
          canManageEndDates={Boolean(payload?.page?.can_manage_end_dates)}
          onClose={() => setSelectedRow(null)}
          onSaveEndDate={handleSaveEndDate}
        />
        <OffboardingManualActionDrawer
          mode={manualDrawerMode}
          onClose={() => setManualDrawerMode(null)}
          onUnauthorized={onUnauthorized}
          onForbidden={onForbidden}
        />
      </>
    );
  }, [handleSaveEndDate, manualDrawerMode, onForbidden, onUnauthorized, payload, rows, selectedPayloadRow, sharedShellRenderOverlay]);

  if (artboardStatus === "loading" || pageState === "loading") {
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

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Offboarding unavailable</h1></main>;
  }

  return (
    <main
      id="main-content"
      className="page-canvas page-canvas--static"
      aria-labelledby={OFFBOARDING_HEADING_ID}
    >
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
