import { PenArtboard } from "../lib/PenArtboard";
import { generatedArtboards } from "../generated/artboards.generated.js";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const ERROR_COPY = {
  401: {
    title: "not authorized",
    body: "You need to sign in before you can view this page.",
  },
  403: {
    title: "forbidden",
    body: "Your current role does not have permission to access this page.",
  },
  404: {
    title: "page not found",
    body: "The requested page could not be found.",
  },
  500: {
    title: "application error",
    body: "The application encountered an unexpected problem while handling this request.",
  },
  502: {
    title: "upstream error",
    body: "A dependent service returned an invalid response.",
  },
  503: {
    title: "service unavailable",
    body: "The application is temporarily unavailable. Please try again shortly.",
  },
};

/**
 * errorCopyFor documents runtime data flow for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function errorCopyFor(code, details) {
  const fallback = {
    title: "unexpected error",
    body: "The application could not complete the requested operation.",
  };
  const base = ERROR_COPY[code] ?? fallback;

  return {
    code: String(code),
    title: base.title,
    body: details ? `${base.body}\n\n${details}` : base.body,
  };
}

/**
 * cloneArtboard documents runtime data flow for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function cloneArtboard(artboard) {
  return JSON.parse(JSON.stringify(artboard));
}

/**
 * estimateWrappedTextHeight documents runtime data flow for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function estimateWrappedTextHeight(content, width, fontSize) {
  const normalized = String(content ?? "");
  const resolvedFontSize = fontSize ?? 14;
  const lineHeight = resolvedFontSize * 1.35;
  const hardLines = normalized.split("\n");
  const approxCharsPerLine = Math.max(
    1,
    Math.floor((width ?? Math.ceil(normalized.length * resolvedFontSize * 0.6)) / Math.max(resolvedFontSize * 0.58, 1))
  );
  const lineCount = hardLines.reduce(
    (total, line) => total + Math.max(1, Math.ceil(Math.max(line.length, 1) / approxCharsPerLine)),
    0
  );
  return Math.ceil(lineCount * lineHeight);
}

/**
 * updateErrorArtboard documents runtime data flow for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function updateErrorArtboard(baseArtboard, copy, loggedIn) {
  const artboard = cloneArtboard(baseArtboard);
  const codeIds = new Set(loggedIn ? ["error__t4"] : ["t4"]);
  const titleIds = new Set(loggedIn ? ["error__t5"] : ["t5"]);
  const bodyIds = new Set(loggedIn ? ["error__t6"] : ["t6"]);

  /**
   * visit documents runtime data flow for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
   */
  function visit(node) {
    if (node?.type === "text") {
      if (codeIds.has(node.id)) {
        node.content = copy.code;
      } else if (titleIds.has(node.id)) {
        node.content = copy.title;
      } else if (bodyIds.has(node.id)) {
        node.content = copy.body;
      }
      if (node.stroke?.fill) {
        node.stroke.thickness = 1;
      }
    }

    for (const child of node.children || []) {
      visit(child);
    }
  }

  visit(artboard);
  return artboard;
}

/**
 * ErrorPage renders the UI surface for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function ErrorPage({ code, session, onNavigate, onSearch, searchQuery = "", details = "" }) {
  const copy = errorCopyFor(code, details);
  const loggedIn = Boolean(session?.authenticated && session?.authorized);
  const hideVisibleBody = loggedIn && Number(code) === 403;
  const artboard = updateErrorArtboard(
    generatedArtboards[loggedIn ? "error-logged-in" : "error-logged-out"],
    copy,
    loggedIn
  );

  const textOverrides = loggedIn
    ? buildSharedShellTextOverrides(session)
    : {
        t4: copy.code,
        t5: copy.title,
        t6: copy.body,
      };

  const hiddenNodeIds = loggedIn
    ? buildSharedShellHiddenNodeIds(session, {
        hideNavHighlight: true,
        hideSearchPlaceholder: true,
        hideAllNavGroups: true,
      })
    : [];
  if (hideVisibleBody) {
    hiddenNodeIds.push("error__t6");
  }
  if (loggedIn) {
    hiddenNodeIds.push("error__t4", "error__t5", "error__t6");
  }
  const imageNodeOverrides = loggedIn ? buildSharedShellImageOverrides(session) : {};
  const sharedShellOverlay = loggedIn
    ? createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: null,
      })
    : null;
  const recoveryTarget = loggedIn ? "/dashboard" : "/login";
  const recoveryLabel = loggedIn ? "Return to Dashboard" : "Go to Login";
  const semanticTitleId = `error-page-${code}-title`;
  /**
   * renderOverlay documents runtime data flow for frontend/src/pages/ErrorPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
   */
  const renderOverlay = ({ nodeIndex, textOverrides: overlayTextOverrides }) => {
    const sharedShell = typeof sharedShellOverlay === "function"
      ? sharedShellOverlay({ nodeIndex, textOverrides: overlayTextOverrides })
      : sharedShellOverlay;
    const bodyNode = nodeIndex.get(loggedIn ? "error__t6" : "t6");
    const titleNode = nodeIndex.get(loggedIn ? "error__t5" : "t5");
    const anchorNode = hideVisibleBody ? titleNode ?? bodyNode : bodyNode ?? titleNode;

    if (!anchorNode) {
      return sharedShell;
    }

    const anchorContent = overlayTextOverrides[anchorNode.id] ?? anchorNode.content ?? "";
    const anchorHeight = estimateWrappedTextHeight(
      anchorContent,
      anchorNode.width ?? 780,
      anchorNode.fontSize ?? 24
    );
    const buttonWidth = loggedIn ? 260 : 220;
    const buttonHeight = 48;
    const buttonLeft = (anchorNode.x ?? 0) + ((anchorNode.width ?? buttonWidth) - buttonWidth) / 2;
    const buttonTop = (anchorNode.y ?? 0) + anchorHeight + (hideVisibleBody ? 42 : 28);

    return (
      <>
        {sharedShell}
        {loggedIn ? (
          <section
            className="error-page__runtime-copy"
            aria-hidden="true"
            style={{
              position: "absolute",
              left: 520,
              top: 340,
              width: 900,
            }}
          >
            <p className="error-page__runtime-code">{copy.code}</p>
            <h2>{copy.title}</h2>
            {!hideVisibleBody ? <p>{copy.body}</p> : null}
          </section>
        ) : null}
        <button
          type="button"
          className="error-page__cta"
          style={{
            position: "absolute",
            left: buttonLeft,
            top: buttonTop,
            width: buttonWidth,
            height: buttonHeight,
          }}
          onClick={() => onNavigate(recoveryTarget)}
        >
          {recoveryLabel}
        </button>
      </>
    );
  };

  return (
    <main
      id="main-content"
      className={`page-canvas ${loggedIn ? "page-canvas--error-shell" : "page-canvas--error"}`}
      aria-labelledby={semanticTitleId}
    >
      {/* WCAG 1.3.1/2.4.2: error artboards are visual-only, so expose the status copy semantically. */}
      <section className="sr-only" aria-labelledby={semanticTitleId}>
        <h1 id={semanticTitleId}>{copy.title}</h1>
        <p>{copy.body}</p>
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
