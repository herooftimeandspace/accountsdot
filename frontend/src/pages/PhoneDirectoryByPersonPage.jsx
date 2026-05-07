import { useEffect, useMemo, useState } from "react";
import { AccessDenied } from "../components/AccessDenied";
import { generatedArtboards } from "../generated/artboards.generated.js";
import { PenArtboard } from "../lib/PenArtboard";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const PHONE_DIRECTORY_ENDPOINT = "/api/v1/dev/pages/phone-directory/by-person";
const PHONE_DIRECTORY_FRAME_ID = "f110";
const DETAIL_RAIL_ID = "f159";
const LOCAL_SEARCH_PLACEHOLDER_ID = "t98__dup2";
const PAGE_DESCRIPTION_ID = "t84__dup2";
const PAGE_LAST_REFRESHED_ID = "t85";

const HIDDEN_RESULT_NODE_IDS = [
  "t111",
  "t112",
  "t113",
  "t114",
  "t115",
  "t116",
  "t117",
  "t118",
  "l119",
  "t120",
  "t121",
  "t122",
  "t123",
  "t124",
  "f125",
  "t126",
  "t127",
  "t128",
  "l129",
  "t130",
  "t131",
  "t132",
  "t133",
  "t134",
  "f135",
  "t136",
  "t137",
  "l138",
  "t139",
  "t140",
  "t141",
  "t142",
  "t143",
  "f144",
  "t145",
  "t146",
  "l147",
  "t148",
  "t149",
  "t150",
  "t151",
  "t152",
  "f153",
  "t154",
  "t155",
  "t156",
  "l157",
  "t158",
  "t160",
  "p161",
  "p162",
  "t163",
  "t164",
  "t165",
  "t166",
  "t167",
  "t168",
  "t169",
  "t170",
  "t171",
  "t172",
  "l173",
  "f174",
  "t175",
  "f176",
  "t177",
  "f178",
  "t179",
  "t180",
  "t181",
  "t182",
  "t183",
];

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function uniquifyNodeIds(artboard) {
  const seen = new Map();

  function visit(node) {
    const count = (seen.get(node.id) ?? 0) + 1;
    seen.set(node.id, count);
    if (count > 1) {
      node.id = `${node.id}__dup${count}`;
    }
    for (const child of node.children || []) {
      visit(child);
    }
  }

  visit(artboard);
  return artboard;
}

function buildNodeIndex(node, map = new Map()) {
  map.set(node.id, node);
  for (const child of node.children || []) {
    buildNodeIndex(child, map);
  }
  return map;
}

function nodeBounds(node) {
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

function detailField(label, value) {
  if (!value) {
    return null;
  }
  return (
    <>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </>
  );
}

function PhoneDirectoryResultsOverlay({ bounds, currentSiteName, query, results }) {
  if (!bounds) {
    return null;
  }

  return (
    <section
      className="phone-directory-runtime__results"
      style={{
        position: "absolute",
        left: bounds.left + 18,
        top: bounds.top + 14,
        width: Math.max(0, bounds.width - 36),
        height: Math.max(0, bounds.height - 28),
        zIndex: 2,
      }}
      aria-live="polite"
    >
      <div className="phone-directory-runtime__results-header">
        <h2>Directory Results</h2>
        <p>
          {query
            ? `Showing ${results.length} matches for “${query}”. ${currentSiteName} results appear first.`
            : `Showing ${results.length} visible directory entries. ${currentSiteName} results appear first.`}
        </p>
      </div>
      {results.length > 0 ? (
        <div className="phone-directory-runtime__table">
          <div className="phone-directory-runtime__table-header">
            <div>Name or Line</div>
            <div>Details</div>
            <div>Room</div>
            <div>Extension</div>
            <div>Phone</div>
            <div>Site</div>
            <div>Type</div>
          </div>
          {results.map((result, index) => (
            <div
              key={result.id}
              className={`phone-directory-runtime__row ${index === 0 ? "phone-directory-runtime__row--selected" : ""}`}
            >
              <div>
                <div className="phone-directory-runtime__primary">{result.title}</div>
                {result.subtitle ? (
                  <div className="phone-directory-runtime__secondary">{result.subtitle}</div>
                ) : null}
              </div>
              <div>
                <div>{result.email || result.department || result.identifier || "—"}</div>
                {result.role ? (
                  <div className="phone-directory-runtime__secondary">{result.role}</div>
                ) : null}
              </div>
              <div>{result.location || "—"}</div>
              <div>{result.extension || "—"}</div>
              <div>{result.phone || "—"}</div>
              <div>{result.site_name}</div>
              <div>
                <span className="phone-directory-runtime__pill">{result.type_label}</span>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <p className="phone-directory-runtime__empty">
          No directory matches were found for this search.
        </p>
      )}
    </section>
  );
}

function PhoneDirectoryDetailOverlay({ bounds, result }) {
  if (!bounds) {
    return null;
  }

  return (
    <aside
      className="phone-directory-runtime__detail"
      style={{
        position: "absolute",
        left: bounds.left + 18,
        top: bounds.top + 14,
        width: Math.max(0, bounds.width - 36),
        height: Math.max(0, bounds.height - 28),
        zIndex: 2,
      }}
      aria-live="polite"
    >
      {result ? (
        <>
          <h2>{result.title}</h2>
          <p>{result.site_name}</p>
          <div className="phone-directory-runtime__detail-card">
            <dl>
              {detailField("Type", result.type_label)}
              {detailField("Role", result.role)}
              {detailField("Department", result.department)}
              {detailField("Room", result.location)}
              {detailField("Extension", result.extension)}
              {detailField("Phone", result.phone)}
              {detailField("Email", result.email)}
              {detailField("ID", result.identifier)}
            </dl>
          </div>
        </>
      ) : (
        <>
          <h2>No Result Selected</h2>
          <p>Run a directory search to view person, room, and shared-line details here.</p>
        </>
      )}
    </aside>
  );
}

function buildTextOverrides(session, payload, searchQuery) {
  const overrides = buildSharedShellTextOverrides(session);
  if (!payload) {
    return overrides;
  }

  overrides[PAGE_DESCRIPTION_ID] = payload.page.description;
  overrides[PAGE_LAST_REFRESHED_ID] = payload.page.last_refreshed;
  overrides[LOCAL_SEARCH_PLACEHOLDER_ID] =
    searchQuery.trim() || "Search by name, classification, or extension...";
  return overrides;
}

export function PhoneDirectoryByPersonPage({
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

  const artboard = useMemo(() => uniquifyNodeIds(clone(generatedArtboards["phone-directory-by-person"])), []);
  const nodeIndex = useMemo(() => buildNodeIndex(artboard), [artboard]);

  useEffect(() => {
    if (!session?.authenticated || !session?.authorized) {
      return undefined;
    }

    const controller = new AbortController();

    async function loadPage() {
      setPageState("loading");
      setErrorMessage("");
      try {
        const requestUrl = new URL(PHONE_DIRECTORY_ENDPOINT, window.location.origin);
        if (searchQuery.trim()) {
          requestUrl.searchParams.set("q", searchQuery.trim());
        }

        const response = await fetch(requestUrl, {
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
          throw new Error(`Phone Directory request failed with ${response.status}`);
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
          error instanceof Error ? error.message : "Unable to load Phone Directory results."
        );
        setPageState("error");
      }
    }

    void loadPage();
    return () => controller.abort();
  }, [onForbidden, onUnauthorized, searchQuery, session]);

  const textOverrides = useMemo(
    () => buildTextOverrides(session, payload, searchQuery),
    [payload, searchQuery, session]
  );
  const hiddenNodeIds = useMemo(
    () =>
      buildSharedShellHiddenNodeIds(session, {
        hideNavHighlight: true,
        hideSearchPlaceholder: true,
      }).concat(HIDDEN_RESULT_NODE_IDS),
    [session]
  );
  const imageNodeOverrides = useMemo(() => buildSharedShellImageOverrides(session), [session]);
  const sharedShellOverlay = useMemo(
    () =>
      createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: "phoneDirectory",
      }),
    [onNavigate, onSearch, searchQuery, session]
  );

  const renderOverlay = useMemo(() => {
    return (args) => {
      const shellElements =
        typeof sharedShellOverlay === "function" ? sharedShellOverlay(args) ?? [] : [];
      const resultsBounds = nodeBounds(nodeIndex.get(PHONE_DIRECTORY_FRAME_ID));
      const detailBounds = nodeBounds(nodeIndex.get(DETAIL_RAIL_ID));
      const results = payload?.page?.results ?? [];
      const selected = results[0] ?? null;

      return [
        ...shellElements,
        <PhoneDirectoryResultsOverlay
          key="phone-directory-results"
          bounds={resultsBounds}
          currentSiteName={payload?.page?.current_site_name ?? session?.current_site_name ?? ""}
          query={payload?.page?.query ?? searchQuery}
          results={results}
        />,
        <PhoneDirectoryDetailOverlay
          key="phone-directory-detail"
          bounds={detailBounds}
          result={selected}
        />,
      ];
    };
  }, [nodeIndex, payload, searchQuery, session?.current_site_name, sharedShellOverlay]);

  const overlay = (() => {
    if (pageState === "loading") {
      return (
        <AccessDenied
          role="status"
          title="Loading Phone Directory"
          message="Loading the DEV phone directory search results."
        />
      );
    }
    if (pageState === "error") {
      return (
        <AccessDenied
          title="Unable to load Phone Directory"
          message={errorMessage || "Request failed."}
        />
      );
    }
    return null;
  })();

  return (
    <main id="main-content" className="page-canvas page-canvas--static phone-directory-runtime">
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
        {overlay}
      </div>
    </main>
  );
}
