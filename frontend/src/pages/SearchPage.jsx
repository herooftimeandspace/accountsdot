import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { AccessDenied } from "../components/AccessDenied";
import { PenArtboard } from "../lib/PenArtboard";
import { fetchDevApiJSON, handleDevApiAuthError } from "../lib/devApi";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const SEARCH_HEADING_ID = "global-search-heading";
const SEARCH_ARTBOARD_KEY = "dashboard-it-admin";
const CONTENT_LEFT = 306;
const CONTENT_TOP = 118;
const CONTENT_WIDTH = 1232;

/**
 * clone gives SearchPage a mutable copy of the generated IT Admin dashboard
 * artboard. The search route reuses that shell as a visual base, then rewrites
 * duplicated node ids before layering live search results over the content pane.
 */
function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

/**
 * uniquifyNodeIds prevents repeated generated ids from colliding after the
 * dashboard artboard is reused for the Search route. PenArtboard and overlay
 * lookups depend on stable unique ids when hiding static nodes.
 */
function uniquifyNodeIds(artboard) {
  const seen = new Map();

  /**
   * visit walks the copied artboard tree in place and suffixes only duplicate
   * ids. The first occurrence keeps the PEN id so existing shared-shell slot
   * references continue to resolve.
   */
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

/**
 * buildNodeIndex creates the id lookup used by contentHiddenNodeIds and the
 * PenArtboard overlay callback. Search needs this index to hide the static
 * dashboard pane without changing generated artboard files.
 */
function buildNodeIndex(node, map = new Map()) {
  map.set(node.id, node);
  for (const child of node.children || []) {
    buildNodeIndex(child, map);
  }
  return map;
}

/**
 * descendantIds returns every generated child id below a hidden content node.
 * Hiding descendants with their parent prevents dashboard text and controls
 * from remaining clickable underneath the live search results panel.
 */
function descendantIds(node) {
  const ids = [];
  for (const child of node?.children || []) {
    ids.push(child.id, ...descendantIds(child));
  }
  return ids;
}

/**
 * nodeBounds normalizes PEN node geometry for the Search route's pane-hiding
 * heuristic. Missing generated nodes return null so route startup can wait for
 * artboard readiness instead of failing on undefined coordinates.
 */
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

/**
 * contentHiddenNodeIds removes the generated dashboard content pane while
 * preserving the shared shell. The global search route then renders live query
 * results in the cleared pane without altering sidebar, header, drawer, or
 * persona-switch behavior owned by sharedShellPresentation.
 */
function contentHiddenNodeIds(artboard, nodeIndex, session) {
  const hidden = new Set(
    buildSharedShellHiddenNodeIds(session, {
      hideNavHighlight: true,
      hideSearchPlaceholder: true,
      hideAllNavGroups: true,
    })
  );

  if (!artboard || nodeIndex.size === 0) {
    return hidden;
  }

  for (const [id, node] of nodeIndex.entries()) {
    const bounds = nodeBounds(node);
    if (!bounds) {
      continue;
    }
    const isMainContent =
      bounds.left >= 270 &&
      bounds.top >= 90 &&
      !(bounds.left >= 1180 && bounds.top <= 120);
    if (isMainContent) {
      hidden.add(id);
      for (const descendantId of descendantIds(node)) {
        hidden.add(descendantId);
      }
    }
  }

  hidden.delete(artboard.id);
  return hidden;
}

/**
 * GlobalSearchResults renders grouped search results returned by the DEV search API and forwards result destinations into the shared app navigation callback. Empty query text shows the operator search scope instead of rendering stale result groups.
 */
function GlobalSearchResults({ payload, query, onNavigate }) {
  const groups = payload?.page?.groups ?? [];
  const resultCount = groups.reduce((total, group) => total + (group.results?.length ?? 0), 0);
  const trimmedQuery = query.trim();

  return (
    <section
      className="global-search-runtime__panel"
      style={{ left: CONTENT_LEFT, top: CONTENT_TOP, width: CONTENT_WIDTH }}
      aria-labelledby={SEARCH_HEADING_ID}
    >
      <div className="global-search-runtime__header">
        <div>
          <h2 id={SEARCH_HEADING_ID}>Global Search</h2>
          <p>
            {trimmedQuery
              ? `${resultCount} result${resultCount === 1 ? "" : "s"} for "${trimmedQuery}"`
              : "Search from the header to find people, rooms, extensions, workflows, students, devices, and actions."}
          </p>
        </div>
      </div>

      {trimmedQuery && groups.length === 0 ? (
        <div className="global-search-runtime__empty">
          <h3>No results found</h3>
          <p>Try a name, email, phone number, extension, employee ID, student ID, asset ID, or serial number.</p>
        </div>
      ) : null}

      <div className="global-search-runtime__groups">
        {groups.map((group) => (
          <section key={group.id} className="global-search-runtime__group">
            <h3>{group.title}</h3>
            <div className="global-search-runtime__results">
              {(group.results ?? []).map((result) => (
                <button
                  key={result.id}
                  type="button"
                  className="global-search-runtime__result"
                  onClick={() => onNavigate(result.destination)}
                >
                  <span className="global-search-runtime__result-main">
                    <strong>{result.title}</strong>
                    <span>{result.type}</span>
                  </span>
                  <span className="global-search-runtime__result-context">
                    {result.subtitle ? `${result.subtitle} · ` : ""}
                    {result.context}
                  </span>
                  <span className="global-search-runtime__result-source">{result.source}</span>
                </button>
              ))}
            </div>
          </section>
        ))}
      </div>
    </section>
  );
}

/**
 * SearchPage combines the generated search artboard, shared shell overlays, and `/api/v1/dev/search` results for the global search route. It reacts to header query changes, handles DEV auth failures, and keeps runtime result cards layered over the generated page pane.
 */
export function SearchPage({
  session,
  onNavigate,
  onSearch,
  searchQuery = "",
  onUnauthorized,
  onForbidden,
}) {
  const trimmedSearchQuery = searchQuery.trim();
  const searchResultsQuery = useQuery({
    queryKey: ["dev-search", session?.current_persona?.id ?? "", trimmedSearchQuery],
    enabled: Boolean(session?.authenticated && session?.authorized),
    queryFn: ({ signal }) => {
      const requestUrl = new URL("/api/v1/dev/search", window.location.origin);
      if (trimmedSearchQuery) {
        requestUrl.searchParams.set("q", trimmedSearchQuery);
      }
      return fetchDevApiJSON(requestUrl, { signal });
    },
  });
  const payload = searchResultsQuery.data ?? null;
  const pageState = searchResultsQuery.isLoading || searchResultsQuery.isFetching ? "loading" : searchResultsQuery.isError ? "error" : "ready";
  const errorMessage =
    searchResultsQuery.error instanceof Error ? searchResultsQuery.error.message : "Unable to load global search results.";

  const { artboard: baseArtboard, status: artboardStatus } = useGeneratedArtboard(SEARCH_ARTBOARD_KEY);
  const artboard = useMemo(() => baseArtboard ? uniquifyNodeIds(clone(baseArtboard)) : null, [baseArtboard]);
  const nodeIndex = useMemo(() => artboard ? buildNodeIndex(artboard) : new Map(), [artboard]);

  useEffect(() => {
    if (searchResultsQuery.isError) {
      handleDevApiAuthError(searchResultsQuery.error, { onUnauthorized, onForbidden });
    }
  }, [onForbidden, onUnauthorized, searchResultsQuery.error, searchResultsQuery.isError]);

  const textOverrides = useMemo(() => buildSharedShellTextOverrides(session), [session]);
  const hiddenNodeIds = useMemo(
    () => contentHiddenNodeIds(artboard, nodeIndex, session),
    [artboard, nodeIndex, session]
  );
  const imageNodeOverrides = useMemo(() => buildSharedShellImageOverrides(session), [session]);
  const sharedShellOverlay = useMemo(
    () =>
      createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: null,
        activeRoutePath: "/search",
        refreshMetadata: payload?.page?.last_refreshed ?? null,
      }),
    [onNavigate, onSearch, payload?.page?.last_refreshed, searchQuery, session]
  );

  const renderOverlay = useMemo(() => {
    return (args) => {
      const shellElements =
        typeof sharedShellOverlay === "function" ? sharedShellOverlay(args) ?? [] : [];
      const resultElements =
        pageState === "ready"
          ? [
              <GlobalSearchResults
                key="global-search-results"
                payload={payload}
                query={searchQuery}
                onNavigate={onNavigate}
              />,
            ]
          : [];
      return [...shellElements, ...resultElements];
    };
  }, [onNavigate, pageState, payload, searchQuery, sharedShellOverlay]);

  /**
   * overlay chooses between loading/error states and the runtime search results layered on top of the generated artboard. It waits for the generated artboard before computing hidden-node ids so initial route loads cannot read missing artboard metadata.
   */
  const overlay = (() => {
    if (artboardStatus === "loading") {
      return (
        <AccessDenied
          role="status"
          title="Loading Search"
          message="Preparing the generated Search artboard."
        />
      );
    }
    if (pageState === "loading") {
      return (
        <AccessDenied
          role="status"
          title="Loading Search"
          message="Loading DEV global search results."
        />
      );
    }
    if (pageState === "error") {
      return (
        <AccessDenied
          title="Unable to load Search"
          message={errorMessage || "Request failed."}
        />
      );
    }
    return null;
  })();

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Search</h1>
          <p>Preparing the generated Search artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Search unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static global-search-runtime" aria-labelledby={SEARCH_HEADING_ID}>
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
