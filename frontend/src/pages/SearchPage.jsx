import { useEffect, useMemo, useState } from "react";
import { AccessDenied } from "../components/AccessDenied";
import { PenArtboard } from "../lib/PenArtboard";
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
 * clone documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

/**
 * uniquifyNodeIds documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function uniquifyNodeIds(artboard) {
  const seen = new Map();

  /**
   * visit documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * buildNodeIndex builds derived data for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function buildNodeIndex(node, map = new Map()) {
  map.set(node.id, node);
  for (const child of node.children || []) {
    buildNodeIndex(child, map);
  }
  return map;
}

/**
 * descendantIds documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function descendantIds(node) {
  const ids = [];
  for (const child of node?.children || []) {
    ids.push(child.id, ...descendantIds(child));
  }
  return ids;
}

/**
 * nodeBounds documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * contentHiddenNodeIds documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * GlobalSearchResults renders the UI surface for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * SearchPage renders the UI surface for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function SearchPage({
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

  const { artboard: baseArtboard, status: artboardStatus } = useGeneratedArtboard(SEARCH_ARTBOARD_KEY);
  const artboard = useMemo(() => baseArtboard ? uniquifyNodeIds(clone(baseArtboard)) : null, [baseArtboard]);
  const nodeIndex = useMemo(() => artboard ? buildNodeIndex(artboard) : new Map(), [artboard]);

  useEffect(() => {
    if (!session?.authenticated || !session?.authorized) {
      return undefined;
    }

    const controller = new AbortController();

    /**
     * loadSearch loads or decodes data for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
     */
    async function loadSearch() {
      setPageState("loading");
      setErrorMessage("");
      try {
        const requestUrl = new URL("/api/v1/dev/search", window.location.origin);
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
          throw new Error(`Search request failed with ${response.status}`);
        }
        setPayload(await response.json());
        setPageState("ready");
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        setPayload(null);
        setErrorMessage(error instanceof Error ? error.message : "Unable to load global search results.");
        setPageState("error");
      }
    }

    void loadSearch();
    return () => controller.abort();
  }, [onForbidden, onUnauthorized, searchQuery, session?.authenticated, session?.authorized]);

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
   * overlay documents runtime data flow for frontend/src/pages/SearchPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
