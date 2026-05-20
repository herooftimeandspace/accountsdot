import { useCallback, useEffect, useMemo, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { nextRuntimeDrawerSelectionForId } from "../components/runtimeDrawerController.mjs";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { PenArtboard } from "../lib/PenArtboard";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";
import { devMeasureAsync } from "../lib/devPerformance";

const DEPARTING_SENIORS_ENDPOINT = "/api/v1/dev/pages/departing-seniors";
const DEPARTING_SENIORS_RECORDS_ENDPOINT = "/api/v1/dev/departing-seniors/records";
const DEPARTING_SENIORS_HEADING_ID = "departing-seniors-heading";

/**
 * readJSON normalizes DEV API fetch responses for this page. It returns the
 * decoded payload for successful requests and attaches backend error payloads
 * to thrown Error objects so page loaders and mutation handlers can route 401,
 * 403, and field-validation failures correctly.
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
 * formatDate renders DEV date strings as district-readable calendar dates for
 * table cells and drawer detail rows. Invalid fixture values are returned
 * unchanged so bad mock data remains visible during debugging.
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
 * deviceSummary provides the searchable plain-text representation of a row's
 * IncidentIQ devices. The rendered table uses serial links, but the shared
 * search primitive needs a stable string containing serial and asset ids.
 */
function deviceSummary(row) {
  const devices = row.outstanding_devices ?? [];
  if (devices.length === 0) {
    return "No outstanding devices";
  }
  return devices.map((device) => `${device.type} ${device.serial} (${device.asset_id})`).join("; ");
}

/**
 * shouldSelectRowFromTarget keeps the row drawer trigger off nested controls.
 * Departing Seniors rows are clickable for drawer details, while device links,
 * date inputs, and mutation buttons keep their own browser behavior.
 */
function shouldSelectRowFromTarget(target) {
  return !target?.closest?.("a, button, input, select, textarea, label");
}

/**
 * statusClass maps Departing Seniors status labels to the shared severity
 * palette used by runtime table badges.
 */
function statusClass(status) {
  if (["Ready", "Complete"].includes(status)) {
    return "departing-seniors-runtime__status departing-seniors-runtime__status--ready";
  }
  if (["Device return required", "Blocked", "Error"].includes(status)) {
    return "departing-seniors-runtime__status departing-seniors-runtime__status--critical";
  }
  return "departing-seniors-runtime__status departing-seniors-runtime__status--neutral";
}

/**
 * collectGeneratedPaneNodeIds hides the offboarding artboard body underneath
 * the Departing Seniors runtime pane while leaving the shared shell visible.
 */
function collectGeneratedPaneNodeIds(artboard) {
  const ids = [];
  /**
   * visit walks generated nodes recursively because the .pen-derived artboard
   * stores pane children at multiple depths.
   */
  const visit = (node) => {
    if (!node) {
      return;
    }
    if ((node.x ?? 0) >= 264 && (node.y ?? 0) >= 76) {
      ids.push(node.id);
    }
    (node.children ?? []).forEach(visit);
  };
  (artboard?.children ?? []).forEach(visit);
  return ids;
}

const TABLE_COLUMNS = [
  { key: "last_name", label: "Student", value: (row) => row.display_name, sortValue: (row) => `${row.last_name} ${row.first_name}` },
  { key: "email", label: "Email", value: (row) => row.email },
  { key: "student_id", label: "Student ID", value: (row) => row.student_id },
  { key: "end_date", label: "End Date", value: (row) => formatDate(row.end_date), sortValue: (row) => row.end_date || "" },
  {
    key: "devices",
    label: "IncidentIQ Devices",
    value: deviceSummary,
    searchValue: (row) => [
      row.school_year,
      deviceSummary(row),
      ...(row.outstanding_devices ?? []).flatMap((device) => [device.asset_id, device.serial]),
    ],
  },
  { key: "status", label: "Status", value: (row) => row.status },
];

/**
 * DepartingSeniorsTable renders the searchable row list and local DEV controls.
 * The page passes DEV API mutation callbacks for end-date overrides and
 * deprovisioning; selecting row whitespace opens the shared detail drawer.
 */
function DepartingSeniorsTable({ rows, canManage, savingRowId, selectedRowId, error, onSelectRow, onSaveEndDate, onDeprovision }) {
  const [dateDrafts, setDateDrafts] = useState({});
  const columns = useMemo(() => TABLE_COLUMNS, []);
  const table = useRuntimeTableData(rows, columns, {
    defaultSort: { key: "last_name", direction: "asc" },
  });

  useEffect(() => {
    setDateDrafts((current) => {
      const next = { ...current };
      rows.forEach((row) => {
        if (!Object.prototype.hasOwnProperty.call(next, row.id)) {
          next[row.id] = row.end_date || "";
        }
      });
      return next;
    });
  }, [rows]);

  return (
    <section className="departing-seniors-runtime__table" aria-labelledby={DEPARTING_SENIORS_HEADING_ID}>
      <div className="departing-seniors-runtime__toolbar">
        <RuntimeTableSearch
          value={table.searchQuery}
          onChange={table.setSearchQuery}
          label="Search departing seniors"
          placeholder="Search by name, email, school year, asset serial, asset ID, or student ID..."
        />
      </div>
      {error ? <p className="departing-seniors-runtime__error" role="alert">{error}</p> : null}
      <div className="departing-seniors-runtime__table-header">
        {columns.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
        <div>End Override</div>
        <div>Action</div>
      </div>
      <div className="departing-seniors-runtime__table-body">
        {table.visibleRows.length === 0 ? (
          <p className="departing-seniors-runtime__empty" role="status">
            No departing seniors match the current search.
          </p>
        ) : null}
        {table.visibleRows.map((row) => {
          const dateValue = dateDrafts[row.id] ?? row.end_date ?? "";
          const hasDevices = (row.outstanding_devices ?? []).length > 0;
          return (
            <div
              key={row.id}
              className={`departing-seniors-runtime__row ${
                selectedRowId === row.id ? "departing-seniors-runtime__row--selected" : ""
              }`}
              onClick={(event) => {
                if (shouldSelectRowFromTarget(event.target)) {
                  onSelectRow(nextRuntimeDrawerSelectionForId(selectedRowId, row));
                }
              }}
            >
              <div>
                <button
                  type="button"
                  className="departing-seniors-runtime__row-open"
                  aria-label={`Open departing senior details for ${row.display_name}`}
                  aria-pressed={selectedRowId === row.id}
                  onClick={() => onSelectRow(nextRuntimeDrawerSelectionForId(selectedRowId, row))}
                >
                  <strong>{row.display_name}</strong>
                </button>
                <span>{row.site}</span>
              </div>
              <div>{row.email}</div>
              <div>{row.student_id}</div>
              <div>{formatDate(row.end_date)}</div>
              <div className="departing-seniors-runtime__devices">
                {hasDevices ? (
                  row.outstanding_devices.map((device) => (
                    <span key={`${row.id}-${device.asset_id}`}>
                      {device.asset_url ? (
                        <a href={device.asset_url} target="_blank" rel="noreferrer">
                          {device.serial}
                        </a>
                      ) : (
                        device.serial
                      )}
                    </span>
                  ))
                ) : (
                  <span>No outstanding devices</span>
                )}
              </div>
              <div>
                <span className={statusClass(row.status)}>{row.status}</span>
              </div>
              <div className="departing-seniors-runtime__date-control">
                <input
                  type="date"
                  aria-label={`End date override for ${row.display_name}`}
                  value={dateValue}
                  disabled={!canManage}
                  onChange={(event) => setDateDrafts((current) => ({ ...current, [row.id]: event.target.value }))}
                />
                <button
                  type="button"
                  disabled={!canManage || savingRowId === row.id}
                  onClick={() => onSaveEndDate(row, dateValue)}
                >
                  {savingRowId === row.id ? "Saving..." : "Save"}
                </button>
              </div>
              <div>
                <button
                  type="button"
                  className="departing-seniors-runtime__deprovision"
                  disabled={!canManage || !row.can_deprovision || savingRowId === row.id}
                  onClick={() => onDeprovision(row)}
                >
                  {row.deprovisioned ? "Account deprovisioned" : "Deprovision account"}
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}

/**
 * DepartingSeniorsDrawer shows the selected row's account-retirement and
 * device-return detail in the shared right-hand drawer. Device serials link to
 * IncidentIQ only when the DEV payload includes a concrete asset URL.
 */
function DepartingSeniorsDrawer({ row, schoolYear, onClose }) {
  if (!row) {
    return null;
  }
  const devices = row.outstanding_devices ?? [];
  return (
    <RuntimeDrawer title={row.display_name} onClose={onClose}>
      <RuntimeDetailList
        items={[
          { label: "Status", value: row.status },
          { label: "Email", value: row.email },
          { label: "Student ID", value: row.student_id },
          { label: "School year", value: schoolYear },
          { label: "Graduation year", value: row.graduation_year },
          { label: "Site", value: row.site },
          { label: "End date", value: formatDate(row.end_date) },
          { label: "End date source", value: row.end_date_source },
        ]}
      />
      <div className="runtime-drawer__section">
        <h3>Device return</h3>
        {devices.length ? (
          <ul className="departing-seniors-runtime__drawer-devices">
            {devices.map((device) => (
              <li key={`${row.id}-${device.asset_id || device.serial}`}>
                <strong>{device.type}</strong>
                {device.asset_url ? (
                  <a href={device.asset_url} target="_blank" rel="noreferrer">
                    {device.serial}
                  </a>
                ) : (
                  <span>{device.serial}</span>
                )}
                {device.asset_id ? <span>{device.asset_id}</span> : null}
              </li>
            ))}
          </ul>
        ) : (
          <p>No outstanding devices</p>
        )}
      </div>
      {row.notes?.length ? (
        <div className="runtime-drawer__section">
          <h3>Follow-up</h3>
          {row.notes.map((note) => (
            <p key={note}>{note}</p>
          ))}
        </div>
      ) : null}
    </RuntimeDrawer>
  );
}

/**
 * DepartingSeniorsPage owns the DEV data load, selected school-year state, and
 * row drawer state for /departing-seniors. It reads the mock page payload from
 * internal/web/dev_departing_seniors.go and sends only local DEV mutations for
 * date overrides or mock deprovisioning.
 */
export function DepartingSeniorsPage({ session, onNavigate, onSearch, searchQuery = "", onUnauthorized, onForbidden }) {
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [savingRowId, setSavingRowId] = useState("");
  const [error, setError] = useState("");
  const [selectedSchoolYear, setSelectedSchoolYear] = useState("");
  const [selectedRow, setSelectedRow] = useState(null);

  const { artboard, status: artboardStatus } = useGeneratedArtboard("offboarding");

  const loadPage = useCallback(async (schoolYear = "") => {
    setPageState("loading");
    setError("");
    try {
      const endpoint = schoolYear
        ? `${DEPARTING_SENIORS_ENDPOINT}?school_year=${encodeURIComponent(schoolYear)}`
        : DEPARTING_SENIORS_ENDPOINT;
      const nextPayload = await devMeasureAsync("page-payload-fetch", { page: "departing-seniors" }, async () =>
        readJSON(
          await fetch(endpoint, {
            credentials: "same-origin",
            headers: { Accept: "application/json" },
          })
        )
      );
      setPayload(nextPayload);
      setSelectedSchoolYear(nextPayload.page?.school_year || "");
      setPageState("ready");
    } catch (loadError) {
      if (loadError.status === 401 && onUnauthorized) {
        onUnauthorized();
        return;
      }
      if (loadError.status === 403 && onForbidden) {
        onForbidden();
        return;
      }
      setPageState("error");
    }
  }, [onForbidden, onUnauthorized]);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

  const handleSaveEndDate = useCallback(async (row, endDate) => {
    setSavingRowId(row.id);
    setError("");
    try {
      await readJSON(
        await fetch(`${DEPARTING_SENIORS_RECORDS_ENDPOINT}/${row.id}/end-date`, {
          method: "PUT",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify({ end_date: endDate }),
        })
      );
      await loadPage(selectedSchoolYear);
    } catch (saveError) {
      setError(saveError.payload?.errors?.end_date || saveError.message);
    } finally {
      setSavingRowId("");
    }
  }, [loadPage, selectedSchoolYear]);

  const handleDeprovision = useCallback(async (row) => {
    setSavingRowId(row.id);
    setError("");
    try {
      await readJSON(
        await fetch(`${DEPARTING_SENIORS_RECORDS_ENDPOINT}/${row.id}/deprovision`, {
          method: "POST",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      await loadPage(selectedSchoolYear);
    } catch (saveError) {
      setError(saveError.message);
    } finally {
      setSavingRowId("");
    }
  }, [loadPage, selectedSchoolYear]);

  const handleSchoolYearChange = useCallback((event) => {
    const nextSchoolYear = event.target.value;
    setSelectedSchoolYear(nextSchoolYear);
    setSelectedRow(null);
    void loadPage(nextSchoolYear);
  }, [loadPage]);

  const selectedPayloadRow = selectedRow
    ? (payload?.page?.rows ?? []).find((row) => row.id === selectedRow.id) || selectedRow
    : null;

  const generatedPaneNodeIds = useMemo(() => artboard ? collectGeneratedPaneNodeIds(artboard) : [], [artboard]);
  const textOverrides = useMemo(() => buildSharedShellTextOverrides(session), [session]);
  const hiddenNodeIds = useMemo(() => {
    const ids = buildSharedShellHiddenNodeIds(session, {
      hideNavHighlight: true,
      hideSearchPlaceholder: true,
      hideAllNavGroups: true,
    });
    ids.push(...generatedPaneNodeIds);
    return ids;
  }, [generatedPaneNodeIds, session]);
  const imageNodeOverrides = useMemo(() => buildSharedShellImageOverrides(session), [session]);
  const sharedShellRenderOverlay = useMemo(
    () =>
      createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: "departingSeniors",
        activeRoutePath: "/departing-seniors",
        refreshMetadata: payload?.page?.last_refreshed,
      }),
    [onNavigate, onSearch, payload?.page?.last_refreshed, searchQuery, session]
  );

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <section className="departing-seniors-runtime__pane">
        <div className="departing-seniors-runtime__header">
          <div>
            <h1 id={DEPARTING_SENIORS_HEADING_ID}>{payload?.page?.title || "Departing Seniors"}</h1>
          </div>
          <div className="departing-seniors-runtime__year">
            <label htmlFor="departing-seniors-school-year">School year</label>
            <select
              id="departing-seniors-school-year"
              value={selectedSchoolYear || payload?.page?.school_year || ""}
              onChange={handleSchoolYearChange}
            >
              {(payload?.page?.school_year_options ?? []).map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}{option.current ? " (current)" : ""}
                </option>
              ))}
            </select>
          </div>
        </div>
        <DepartingSeniorsTable
          rows={payload?.page?.rows ?? []}
          canManage={Boolean(payload?.page?.can_manage)}
          savingRowId={savingRowId}
          selectedRowId={selectedPayloadRow?.id}
          error={error}
          onSelectRow={setSelectedRow}
          onSaveEndDate={handleSaveEndDate}
          onDeprovision={handleDeprovision}
        />
        <DepartingSeniorsDrawer
          row={selectedPayloadRow}
          schoolYear={payload?.page?.school_year}
          onClose={() => setSelectedRow(null)}
        />
      </section>
    </>
  ), [error, handleDeprovision, handleSaveEndDate, handleSchoolYearChange, payload, savingRowId, selectedPayloadRow, selectedSchoolYear, sharedShellRenderOverlay]);

  if (artboardStatus === "loading" || pageState === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Departing Seniors</h1>
          <p>Loading the DEV departing seniors list.</p>
        </section>
      </main>
    );
  }

  if (pageState === "error") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Departing Seniors unavailable</h1>
          <p>The DEV departing seniors page could not be loaded.</p>
        </section>
      </main>
    );
  }

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Departing Seniors unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={DEPARTING_SENIORS_HEADING_ID}>
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
