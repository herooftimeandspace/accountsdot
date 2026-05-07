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

export function StaticPenPage({ artboardKey, session, onNavigate, onSearch, searchQuery = "" }) {
  const artboard = generatedArtboards[artboardKey];
  const meta = generatedArtboardMeta[artboardKey];

  const textOverrides = buildSharedShellTextOverrides(session);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  const renderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? null,
    refreshMetadata: staticRefreshMetadataForArtboard(artboardKey),
  });
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
