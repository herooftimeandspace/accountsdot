import { useCallback, useEffect, useMemo, useState } from "react";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboards } from "../generated/artboards.generated.js";
import { PenArtboard } from "../lib/PenArtboard";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const DEPARTING_SENIORS_ENDPOINT = "/api/v1/dev/pages/departing-seniors";
const DEPARTING_SENIORS_RECORDS_ENDPOINT = "/api/v1/dev/departing-seniors/records";
const DEPARTING_SENIORS_HEADING_ID = "departing-seniors-heading";

/**
 * readJSON loads or decodes data for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * formatDate formats display data for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * deviceSummary documents runtime data flow for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function deviceSummary(row) {
  const devices = row.outstanding_devices ?? [];
  if (devices.length === 0) {
    return "No outstanding devices";
  }
  return devices.map((device) => `${device.type} ${device.serial} (${device.asset_id})`).join("; ");
}

/**
 * statusClass formats display data for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * collectGeneratedPaneNodeIds builds derived data for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function collectGeneratedPaneNodeIds(artboard) {
  const ids = [];
  /**
   * visit documents runtime data flow for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
  { key: "graduation_year", label: "Grad Year", value: (row) => row.graduation_year },
  { key: "student_id", label: "Student ID", value: (row) => row.student_id },
  { key: "end_date", label: "End Date", value: (row) => formatDate(row.end_date), sortValue: (row) => row.end_date || "" },
  {
    key: "devices",
    label: "IncidentIQ Devices",
    value: deviceSummary,
    searchValue: (row) => [
      deviceSummary(row),
      ...(row.outstanding_devices ?? []).flatMap((device) => [device.asset_id, device.serial]),
    ],
  },
  { key: "status", label: "Status", value: (row) => row.status },
];

/**
 * DepartingSeniorsTable renders the UI surface for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller. Pay special attention to side effects: this path may update React state, browser storage, cookies, or DEV mock APIs and should stay aligned with docs/external-write-inventory.md when it triggers mutations.
 */
function DepartingSeniorsTable({ rows, canManage, savingRowId, error, onSaveEndDate, onDeprovision }) {
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
          placeholder="Search by name, email, grad year, asset serial, asset ID, or student ID..."
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
        {table.visibleRows.map((row) => {
          const dateValue = dateDrafts[row.id] ?? row.end_date ?? "";
          const hasDevices = (row.outstanding_devices ?? []).length > 0;
          return (
            <div key={row.id} className="departing-seniors-runtime__row">
              <div>
                <strong>{row.display_name}</strong>
                <span>{row.site}</span>
              </div>
              <div>{row.email}</div>
              <div>{row.graduation_year}</div>
              <div>{row.student_id}</div>
              <div>{formatDate(row.end_date)}</div>
              <div className="departing-seniors-runtime__devices">
                {hasDevices ? (
                  row.outstanding_devices.map((device) => (
                    <span key={`${row.id}-${device.asset_id}`}>
                      {device.type}: {device.serial} / {device.asset_id}
                    </span>
                  ))
                ) : (
                  <span>No outstanding devices</span>
                )}
              </div>
              <div>
                <span className={statusClass(row.status)}>{row.status}</span>
                {row.notes?.length ? <small>{row.notes[0]}</small> : null}
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
 * DepartingSeniorsPage renders the UI surface for frontend/src/pages/DepartingSeniorsPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function DepartingSeniorsPage({ session, onNavigate, onSearch, searchQuery = "", onUnauthorized, onForbidden }) {
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [savingRowId, setSavingRowId] = useState("");
  const [error, setError] = useState("");

  const artboard = generatedArtboards.offboarding;

  const loadPage = useCallback(async () => {
    setPageState("loading");
    setError("");
    try {
      const nextPayload = await readJSON(
        await fetch(DEPARTING_SENIORS_ENDPOINT, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setPayload(nextPayload);
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
      await loadPage();
    } catch (saveError) {
      setError(saveError.payload?.errors?.end_date || saveError.message);
    } finally {
      setSavingRowId("");
    }
  }, [loadPage]);

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
      await loadPage();
    } catch (saveError) {
      setError(saveError.message);
    } finally {
      setSavingRowId("");
    }
  }, [loadPage]);

  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...collectGeneratedPaneNodeIds(artboard));
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: "departingSeniors",
    refreshMetadata: payload?.page?.last_refreshed,
  });

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <section className="departing-seniors-runtime__pane">
        <div className="departing-seniors-runtime__header">
          <div>
            <h1 id={DEPARTING_SENIORS_HEADING_ID}>{payload?.page?.title || "Departing Seniors"}</h1>
            <p>{payload?.page?.description || "Current senior class account retirement review."}</p>
          </div>
          <div className="departing-seniors-runtime__year">
            <span>School year</span>
            <strong>{payload?.page?.school_year || "Current"}</strong>
          </div>
        </div>
        <DepartingSeniorsTable
          rows={payload?.page?.rows ?? []}
          canManage={Boolean(payload?.page?.can_manage)}
          savingRowId={savingRowId}
          error={error}
          onSaveEndDate={handleSaveEndDate}
          onDeprovision={handleDeprovision}
        />
      </section>
    </>
  ), [error, handleDeprovision, handleSaveEndDate, payload, savingRowId, sharedShellRenderOverlay]);

  if (pageState === "loading") {
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
