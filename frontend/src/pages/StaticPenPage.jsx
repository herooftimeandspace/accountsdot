import { useState } from "react";

import { PenArtboard } from "../lib/PenArtboard";
import { generatedArtboards, generatedArtboardMeta } from "../generated/artboards.generated.js";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";

const STATIC_PAGE_TITLES = {
  "dashboard-it-admin": "IT Admin Dashboard",
  "dashboard-hr-lifecycle": "Human Resources Dashboard",
  "dashboard-site-admin": "Site Admin Dashboard",
  onboarding: "Staff Onboarding",
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

const ONBOARDING_WORKFLOW_ROWS = [
  {
    id: "jordan-miles",
    person: "Jordan Miles",
    site: "Clover High School (CLA)",
    start: "May 6, 2025",
    currentStep: "Google pending",
    issueAction: "Waiting Entra convergence",
    status: "In Progress",
    email: "jordan.miles@wusd.org",
    incidentIq: "No local write owned by this app. User lookup retries at most once per hour.",
    aeriesTicket: "IT-12904 Open",
    verkadaTicket: "MOT-4412 Waiting",
    top: 405,
  },
  {
    id: "nia-brooks",
    person: "Nia Brooks",
    site: "District Office",
    start: "May 8, 2025",
    currentStep: "Sync dry-run",
    issueAction: "Room mapping required",
    status: "Needs Review",
    email: "nia.brooks@wusd.org",
    incidentIq: "Room assignment mismatch is waiting on district-office review before provisioning resumes.",
    aeriesTicket: "IT-12941 Needs room mapping",
    verkadaTicket: "MOT-4420 Not started",
    top: 459,
  },
  {
    id: "evan-ruiz",
    person: "Evan Ruiz",
    site: "Franklin Middle School",
    start: "May 12, 2025",
    currentStep: "HR intake",
    issueAction: "Missing mandatory field",
    status: "Blocked",
    email: "evan.ruiz@wusd.org",
    incidentIq: "HR intake is missing a required employment field; downstream account work is blocked.",
    aeriesTicket: "IT-12988 Waiting on HR",
    verkadaTicket: "MOT-4434 Waiting",
    top: 513,
  },
  {
    id: "mika-ito",
    person: "Mika Ito",
    site: "Desert View",
    start: "May 13, 2025",
    currentStep: "Ready",
    issueAction: "No blockers",
    status: "Ready",
    email: "mika.ito@wusd.org",
    incidentIq: "Ready for baseline provisioning. No external follow-up is currently required.",
    aeriesTicket: "IT-13002 Ready",
    verkadaTicket: "MOT-4441 Ready",
    top: 567,
  },
];

function OnboardingWorkflowDrawerOverlay({ selectedWorkflow, onSelectWorkflow, onClose }) {
  return (
    <>
      <div className="onboarding-row-hotspots" aria-label="Upcoming staff onboarding rows">
        {ONBOARDING_WORKFLOW_ROWS.map((row) => (
          <button
            key={row.id}
            type="button"
            className="onboarding-row-hotspot"
            aria-label={`View workflow details for ${row.person}`}
            aria-pressed={selectedWorkflow?.id === row.id}
            onClick={() => onSelectWorkflow(row)}
            style={{ top: row.top }}
          />
        ))}
      </div>
      {selectedWorkflow ? (
        <aside
          className="onboarding-workflow-drawer"
          aria-labelledby="onboarding-workflow-drawer-title"
          aria-live="polite"
        >
          <div className="onboarding-workflow-drawer__header">
            <h2 id="onboarding-workflow-drawer-title">Selected Workflow</h2>
            <button
              type="button"
              className="onboarding-workflow-drawer__close"
              aria-label="Close selected workflow drawer"
              onClick={onClose}
            >
              ×
            </button>
          </div>
          <dl className="onboarding-workflow-drawer__details">
            <div>
              <dt>Person</dt>
              <dd>{selectedWorkflow.person}</dd>
            </div>
            <div>
              <dt>Site</dt>
              <dd>{selectedWorkflow.site}</dd>
            </div>
            <div>
              <dt>Start</dt>
              <dd>{selectedWorkflow.start}</dd>
            </div>
            <div>
              <dt>Assigned Email</dt>
              <dd>{selectedWorkflow.email}</dd>
            </div>
            <div>
              <dt>Workflow State</dt>
              <dd>{selectedWorkflow.currentStep}</dd>
            </div>
            <div>
              <dt>Issue / Action</dt>
              <dd>{selectedWorkflow.issueAction}</dd>
            </div>
            <div>
              <dt>Status</dt>
              <dd>{selectedWorkflow.status}</dd>
            </div>
            <div>
              <dt>IncidentIQ</dt>
              <dd>{selectedWorkflow.incidentIq}</dd>
            </div>
          </dl>
          <div className="onboarding-workflow-drawer__tickets">
            <p>
              <strong>Earliest matching Aeries ticket:</strong>
              <span>{selectedWorkflow.aeriesTicket}</span>
            </p>
            <p>
              <strong>Earliest matching Verkada ticket:</strong>
              <span>{selectedWorkflow.verkadaTicket}</span>
            </p>
          </div>
        </aside>
      ) : null}
    </>
  );
}

export function StaticPenPage({ artboardKey, session, onNavigate, onSearch, searchQuery = "" }) {
  const [selectedOnboardingWorkflow, setSelectedOnboardingWorkflow] = useState(null);
  const artboard = generatedArtboards[artboardKey];
  const meta = generatedArtboardMeta[artboardKey];

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
    refreshMetadata: staticRefreshMetadataForArtboard(artboardKey),
  });
  const renderOverlay = (overlayProps) => (
    <>
      {sharedShellRenderOverlay(overlayProps)}
      {artboardKey === "onboarding" ? (
        <OnboardingWorkflowDrawerOverlay
          selectedWorkflow={selectedOnboardingWorkflow}
          onSelectWorkflow={setSelectedOnboardingWorkflow}
          onClose={() => setSelectedOnboardingWorkflow(null)}
        />
      ) : null}
    </>
  );
  const pageTitle = STATIC_PAGE_TITLES[artboardKey] || "Dashboard Page";
  const semanticTitleId = `static-page-${artboardKey}-title`;
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
