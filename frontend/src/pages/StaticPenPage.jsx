import { useEffect, useState } from "react";

import { RowHotspotOverlay, RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { PenArtboard } from "../lib/PenArtboard";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";

const ROOM_MOVES_COMPLETED_ENDPOINT = "/api/v1/dev/room-moves/completed";

const STATIC_PAGE_TITLES = {
  "dashboard-it-admin": "IT Admin Dashboard",
  "dashboard-hr-lifecycle": "Human Resources Dashboard",
  "dashboard-site-admin": "Site Admin Dashboard",
  onboarding: "Onboarding",
  offboarding: "Offboarding",
  "room-moves": "Room Moves",
  "frequent-fliers": "Frequent Fliers",
  "student-data-cleanup": "Student Data Cleanup",
  reports: "Reports",
  "reports-sync-transparency": "Sync Transparency",
  "reports-ticketing-human-work": "Ticketing Human Work",
  admin: "Admin",
  "my-profile": "My Profile",
};

const STATIC_PAGE_ACTIVE_ROUTE_PATHS = {
  "reports-sync-transparency": "/reports/sync-transparency",
  "reports-ticketing-human-work": "/reports/ticketing-human-work",
  admin: "/admin",
};

const STATIC_DRAWER_CONFIGS = {
  onboarding: {
    title: "Selected Workflow",
    ariaLabel: "Upcoming staff onboarding rows",
    rows: [
      {
        id: "jordan-miles",
        ariaLabel: "View workflow details for Jordan Miles",
        left: 306,
        top: 405,
        width: 1232,
        height: 54,
        details: [
          ["Person", "Jordan Miles"],
          ["Site", "Clover High School (CLA)"],
          ["Start", "May 6, 2025"],
          ["Assigned Email", "jordan.miles@wusd.org"],
          ["Workflow State", "Google pending"],
          ["Issue / Action", "Waiting Entra convergence"],
          ["Status", "In Progress"],
          ["IncidentIQ", "No local write owned by this app. User lookup retries at most once per hour."],
        ],
        sections: [
          ["Earliest matching Aeries ticket", "IT-12904 Open"],
          ["Earliest matching Verkada ticket", "MOT-4412 Waiting"],
        ],
      },
      {
        id: "nia-brooks",
        ariaLabel: "View workflow details for Nia Brooks",
        left: 306,
        top: 459,
        width: 1232,
        height: 54,
        details: [
          ["Person", "Nia Brooks"],
          ["Site", "District Office"],
          ["Start", "May 8, 2025"],
          ["Assigned Email", "nia.brooks@wusd.org"],
          ["Workflow State", "Sync dry-run"],
          ["Issue / Action", "Room mapping required"],
          ["Status", "Needs Review"],
          ["IncidentIQ", "Room assignment mismatch is waiting on district-office review before provisioning resumes."],
        ],
        sections: [
          ["Earliest matching Aeries ticket", "IT-12941 Needs room mapping"],
          ["Earliest matching Verkada ticket", "MOT-4420 Not started"],
        ],
      },
      {
        id: "evan-ruiz",
        ariaLabel: "View workflow details for Evan Ruiz",
        left: 306,
        top: 513,
        width: 1232,
        height: 54,
        details: [
          ["Person", "Evan Ruiz"],
          ["Site", "Franklin Middle School"],
          ["Start", "May 12, 2025"],
          ["Assigned Email", "evan.ruiz@wusd.org"],
          ["Workflow State", "HR intake"],
          ["Issue / Action", "Missing mandatory field"],
          ["Status", "Blocked"],
          ["IncidentIQ", "HR intake is missing a required employment field; downstream account work is blocked."],
        ],
        sections: [
          ["Earliest matching Aeries ticket", "IT-12988 Waiting on HR"],
          ["Earliest matching Verkada ticket", "MOT-4434 Waiting"],
        ],
      },
      {
        id: "mika-ito",
        ariaLabel: "View workflow details for Mika Ito",
        left: 306,
        top: 567,
        width: 1232,
        height: 54,
        details: [
          ["Person", "Mika Ito"],
          ["Site", "Desert View"],
          ["Start", "May 13, 2025"],
          ["Assigned Email", "mika.ito@wusd.org"],
          ["Workflow State", "Ready"],
          ["Issue / Action", "No blockers"],
          ["Status", "Ready"],
          ["IncidentIQ", "Ready for baseline provisioning. No external follow-up is currently required."],
        ],
        sections: [
          ["Earliest matching Aeries ticket", "IT-13002 Ready"],
          ["Earliest matching Verkada ticket", "MOT-4441 Ready"],
        ],
      },
    ],
  },
  "student-data-cleanup": {
    title: "Student Details",
    ariaLabel: "Aeries name correction queue rows",
    rows: [
      {
        id: "carlos-nuno",
        ariaLabel: "View student details for Carlos Nuno",
        left: 306,
        top: 443,
        width: 1232,
        height: 54,
        details: [
          ["Student", "Carlos Nuno"],
          ["Student ID", "0001021"],
          ["Grade", "11"],
          ["FirstName raw", "Carlos"],
          ["LastName raw", "Nuño"],
          ["FirstName normalized", "Carlos"],
          ["LastName normalized", "Nuno"],
          ["Issue Type", "Invalid character"],
          ["Detected", "May 2, 2025 8:58 AM PT"],
        ],
        sections: [["Source system", "Corrections must be made in Aeries. This dashboard cannot edit student data."]],
      },
      {
        id: "alex-oneil",
        ariaLabel: "View student details for Alex O'Neil",
        left: 306,
        top: 497,
        width: 1232,
        height: 54,
        details: [
          ["Student", "Alex O'Neil"],
          ["Student ID", "0001087"],
          ["Grade", "10"],
          ["FirstName raw", "Alex"],
          ["LastName raw", "O'Neil"],
          ["FirstName normalized", "Alex"],
          ["LastName normalized", "ONeil"],
          ["Issue Type", "Invalid character"],
          ["Detected", "May 2, 2025 8:56 AM PT"],
        ],
        sections: [["Source system", "Corrections must be made in Aeries. This dashboard cannot edit student data."]],
      },
      {
        id: "jose-martinez",
        ariaLabel: "View student details for Jose Martinez",
        left: 306,
        top: 551,
        width: 1232,
        height: 54,
        details: [
          ["Student", "Jose Martinez"],
          ["Student ID", "0001142"],
          ["Grade", "12"],
          ["FirstName raw", "Jose"],
          ["LastName raw", "Martinez"],
          ["FirstName normalized", "Jose"],
          ["LastName normalized", "Martinez"],
          ["Issue Type", "Invalid character"],
          ["Detected", "May 2, 2025 8:54 AM PT"],
        ],
      },
      {
        id: "taylor-smith-jones",
        ariaLabel: "View student details for Taylor Smith-Jones",
        left: 306,
        top: 605,
        width: 1232,
        height: 54,
        details: [
          ["Student", "Taylor Smith-Jones"],
          ["Student ID", "0001233"],
          ["Grade", "9"],
          ["FirstName raw", "Taylor"],
          ["LastName raw", "Smith-Jones"],
          ["FirstName normalized", "Taylor"],
          ["LastName normalized", "SmithJones"],
          ["Issue Type", "Invalid character"],
          ["Detected", "May 2, 2025 8:52 AM PT"],
        ],
      },
    ],
  },
  "reports-sync-transparency": {
    title: "Selected Sync Item",
    ariaLabel: "Sync transparency queue rows",
    rows: [
      {
        id: "alex-ramirez",
        ariaLabel: "View sync item for Alex Ramirez",
        left: 306,
        top: 321,
        width: 1232,
        height: 42,
        details: [
          ["User", "Alex Ramirez"],
          ["Type", "Staff"],
          ["Current Phase", "room_mapped"],
          ["Overall Status", "manual_action"],
          ["Queued At", "Apr 29, 2026 8:44 AM PT"],
          ["Errors / Warnings", "primary_conflict"],
          ["Action", "Open mapping"],
        ],
        sections: [["Warning", "Phone assignment needs a primary owner selection."]],
      },
      {
        id: "marisol-vega",
        ariaLabel: "View sync item for Marisol Vega",
        left: 306,
        top: 363,
        width: 1232,
        height: 42,
        details: [
          ["User", "Marisol Vega"],
          ["Type", "Student"],
          ["Current Phase", "iiq_matched"],
          ["Overall Status", "manual_action"],
          ["Queued At", "Apr 29, 2026 8:38 AM PT"],
          ["Errors / Warnings", "missing_asset"],
          ["Action", "Override"],
        ],
      },
      {
        id: "mika-ito-sync",
        ariaLabel: "View sync item for Mika Ito",
        left: 306,
        top: 405,
        width: 1232,
        height: 42,
        details: [
          ["User", "Mika Ito"],
          ["Type", "Staff"],
          ["Current Phase", "photo_processed"],
          ["Overall Status", "in_progress"],
          ["Queued At", "Apr 28, 2026 2:30 PM PT"],
          ["Errors / Warnings", "rollover_wait"],
          ["Action", "Recheck"],
        ],
        sections: [["Warning", "rollover_wait is an in-progress warning, not a manual-action block."]],
      },
      {
        id: "nia-brooks-sync",
        ariaLabel: "View sync item for Nia Brooks",
        left: 306,
        top: 447,
        width: 1232,
        height: 42,
        details: [
          ["User", "Nia Brooks"],
          ["Type", "Staff"],
          ["Current Phase", "ingested"],
          ["Overall Status", "manual_action"],
          ["Queued At", "Apr 28, 2026 12:12 PM PT"],
          ["Errors / Warnings", "room_mapping_required"],
          ["Action", "Open mapping"],
        ],
      },
    ],
  },
  "reports-ticketing-human-work": {
    title: "Selected Ticket Context",
    ariaLabel: "Human work queue rows",
    rows: [
      {
        id: "ticket-jordan-miles",
        ariaLabel: "View ticket context for Jordan Miles",
        left: 306,
        top: 407,
        width: 1232,
        height: 54,
        details: [
          ["Affected User", "Jordan Miles"],
          ["Current Workflow", "Onboarding"],
          ["Matching Rule", "Requestor email + category"],
          ["Displayed Ticket", "IT-12904"],
          ["Current Status", "Open"],
          ["Category", "Aeries Add User"],
        ],
        sections: [["Workflow", "Aeries (Asset Tag: AERIES) → User Rights → Add User"]],
      },
      {
        id: "ticket-nia-brooks",
        ariaLabel: "View ticket context for Nia Brooks",
        left: 306,
        top: 461,
        width: 1232,
        height: 42,
        details: [
          ["Affected User", "Nia Brooks"],
          ["Current Workflow", "Onboarding"],
          ["Matching Rule", "External IIQ config"],
          ["Displayed Ticket", "MOT-4412"],
          ["Current Status", "Waiting"],
          ["Category", "Alarm Code"],
        ],
        sections: [["Workflow", "Security Systems → Alarm Codes → Add Alarm Code"]],
      },
      {
        id: "ticket-morgan-lee",
        ariaLabel: "View ticket context for Morgan Lee",
        left: 306,
        top: 503,
        width: 1232,
        height: 42,
        details: [
          ["Affected User", "Morgan Lee"],
          ["Current Workflow", "Room Move"],
          ["Matching Rule", "Manual fallback"],
          ["Displayed Ticket", "IT-13012"],
          ["Current Status", "Open"],
          ["Category", "Phone conflict"],
        ],
      },
      {
        id: "ticket-chris-morgan",
        ariaLabel: "View ticket context for Chris Morgan",
        left: 306,
        top: 545,
        width: 1232,
        height: 42,
        details: [
          ["Affected User", "Chris Morgan"],
          ["Current Workflow", "Offboarding"],
          ["Matching Rule", "Linked to lifecycle"],
          ["Displayed Ticket", "IT-13044"],
          ["Current Status", "Closed"],
          ["Category", "Asset retrieval"],
        ],
      },
    ],
  },
};

/**
 * StaticDrawerOverlay renders the UI surface for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function StaticDrawerOverlay({ config, selectedRow, onSelectRow, onClose }) {
  return (
    <>
      <RowHotspotOverlay
        rows={config.rows}
        selectedId={selectedRow?.id}
        onSelect={onSelectRow}
        ariaLabel={config.ariaLabel}
      />
      {selectedRow ? (
        <RuntimeDrawer title={config.title} onClose={onClose}>
          <RuntimeDetailList
            items={selectedRow.details.map(([label, value]) => ({
              label,
              value,
            }))}
          />
          {selectedRow.sections?.length ? (
            <div className="runtime-drawer__section">
              {selectedRow.sections.map(([label, value]) => (
                <p key={label}>
                  <strong>{label}:</strong>
                  <span>{value}</span>
                </p>
              ))}
            </div>
          ) : null}
        </RuntimeDrawer>
      ) : null}
    </>
  );
}

/**
 * readJSON loads or decodes data for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * AdminRoomMoveRevertOverlay renders the UI surface for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller. Pay special attention to side effects: this path may update React state, browser storage, cookies, or DEV mock APIs and should stay aligned with docs/external-write-inventory.md when it triggers mutations.
 */
function AdminRoomMoveRevertOverlay({ session, onNavigate }) {
  const isItAdmin = session?.current_persona?.id === "it_admin";
  const [state, setState] = useState("idle");
  const [jobs, setJobs] = useState([]);
  const [message, setMessage] = useState("");
  const [busyJobId, setBusyJobId] = useState("");

  useEffect(() => {
    if (!isItAdmin) {
      setJobs([]);
      return undefined;
    }
    const controller = new AbortController();
    /**
     * loadCompletedJobs loads or decodes data for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
     */
    async function loadCompletedJobs() {
      setState("loading");
      setMessage("");
      try {
        const payload = await readJSON(
          await fetch(ROOM_MOVES_COMPLETED_ENDPOINT, {
            credentials: "same-origin",
            headers: { Accept: "application/json" },
            signal: controller.signal,
          })
        );
        setJobs(payload.jobs || []);
        setState("ready");
      } catch (error) {
        if (!controller.signal.aborted) {
          setState("error");
          setMessage(error.message);
        }
      }
    }
    void loadCompletedJobs();
    return () => controller.abort();
  }, [isItAdmin]);

  /**
   * revertJob documents runtime data flow for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller. Pay special attention to side effects: this path may update React state, browser storage, cookies, or DEV mock APIs and should stay aligned with docs/external-write-inventory.md when it triggers mutations.
   */
  async function revertJob(job) {
    const confirmed = window.confirm(
      "Reverting this completed room move schedules a new job that reverses every change from the selected job. IT can only fully revert a room move. To partially revert a room move, create a new Room Move draft for the affected employees."
    );
    if (!confirmed) {
      return;
    }
    setBusyJobId(job.id);
    setMessage("");
    try {
      const payload = await readJSON(
        await fetch(`${ROOM_MOVES_COMPLETED_ENDPOINT}/${encodeURIComponent(job.id)}/revert`, {
          method: "POST",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setJobs((current) =>
        current.map((candidate) =>
          candidate.id === job.id
            ? { ...candidate, revert_draft_id: payload.draft?.id, revert_status: payload.draft?.status || "scheduled" }
            : candidate
        )
      );
      setMessage(`Revert scheduled as ${payload.draft?.id || "a new room move draft"}.`);
    } catch (error) {
      setMessage(error.payload?.errors ? Object.values(error.payload.errors).join(" ") : error.message);
    } finally {
      setBusyJobId("");
    }
  }

  if (!isItAdmin) {
    return null;
  }

  return (
    <section className="admin-room-move-revert" aria-label="Completed room move reversals">
      <div>
        <h2>Room Move Reversal</h2>
        <p>IT Admins can schedule a full reversal for completed room move jobs.</p>
      </div>
      <button type="button" className="admin-room-move-revert__feature-flags" onClick={() => onNavigate("/admin/feature-flags")}>
        Open Feature Flags
      </button>
      {state === "loading" ? <p role="status">Loading completed room moves...</p> : null}
      {message ? <p className="admin-room-move-revert__message" role={state === "error" ? "alert" : "status"}>{message}</p> : null}
      {state === "ready" ? (
        <div className="admin-room-move-revert__table">
          <div className="admin-room-move-revert__header">
            <span>Job</span>
            <span>Scope</span>
            <span>Completed</span>
            <span>Rows</span>
            <span>Action</span>
          </div>
          {jobs.map((job) => (
            <div key={job.id} className="admin-room-move-revert__row">
              <span>{job.id}</span>
              <span>{job.scope_site}</span>
              <span>{job.completed_at}</span>
              <span>{job.row_count}</span>
              {job.revert_draft_id ? (
                <span className="admin-room-move-revert__scheduled">Revert scheduled: {job.revert_draft_id}</span>
              ) : (
                <button type="button" onClick={() => revertJob(job)} disabled={busyJobId === job.id}>
                  Revert
                </button>
              )}
            </div>
          ))}
        </div>
      ) : null}
    </section>
  );
}

/**
 * StaticPenPage renders the UI surface for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function StaticPenPage({ artboardKey, session, onNavigate, onSearch, searchQuery = "" }) {
  const [selectedStaticDrawerRow, setSelectedStaticDrawerRow] = useState(null);
  const { artboard, status: artboardStatus } = useGeneratedArtboard(artboardKey);
  const meta = generatedArtboardMeta[artboardKey];
  const staticDrawerConfig = STATIC_DRAWER_CONFIGS[artboardKey] ?? null;

  useEffect(() => {
    setSelectedStaticDrawerRow(null);
  }, [artboardKey]);

  const textOverrides = buildSharedShellTextOverrides(session);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? null,
    activeRoutePath: STATIC_PAGE_ACTIVE_ROUTE_PATHS[artboardKey] ?? null,
    refreshMetadata: staticRefreshMetadataForArtboard(artboardKey),
  });
  /**
   * renderOverlay documents runtime data flow for frontend/src/pages/StaticPenPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
   */
  const renderOverlay = (overlayProps) => (
    <>
      {sharedShellRenderOverlay(overlayProps)}
      {staticDrawerConfig ? (
        <StaticDrawerOverlay
          config={staticDrawerConfig}
          selectedRow={selectedStaticDrawerRow}
          onSelectRow={setSelectedStaticDrawerRow}
          onClose={() => setSelectedStaticDrawerRow(null)}
        />
      ) : null}
      {artboardKey === "admin" ? <AdminRoomMoveRevertOverlay session={session} onNavigate={onNavigate} /> : null}
    </>
  );
  const pageTitle = STATIC_PAGE_TITLES[artboardKey] || "Dashboard Page";
  const semanticTitleId = `static-page-${artboardKey}-title`;
  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading {pageTitle}</h1>
          <p>Preparing the generated page artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>{pageTitle} unavailable</h1></main>;
  }
  const semanticSummary = buildArtboardSemanticSummary(artboard, {
    fallbackTitle: pageTitle,
    textOverrides,
  });

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={semanticTitleId}>
      {/* WCAG 1.3.1/2.4.2: static PEN pages need semantic text while the artboard stays visual-only. */}
      <section className="sr-only" aria-labelledby={semanticTitleId}>
        <h1 id={semanticTitleId}>{semanticSummary.title}</h1>
        {semanticSummary.items.length > 0 ? (
          <ul>
            {semanticSummary.items.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        ) : null}
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
