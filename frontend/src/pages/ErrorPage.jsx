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
    title: "Not Authorized",
    body: "You need to sign in before you can view this page.",
  },
  403: {
    title: "Forbidden",
    body: "Your current role does not have permission to access this page.",
  },
  404: {
    title: "Page Not Found",
    body: "The requested page could not be found.",
  },
  500: {
    title: "Application Error",
    body: "The application encountered an unexpected problem while handling this request.",
  },
  502: {
    title: "Upstream Error",
    body: "A dependent service returned an invalid response.",
  },
  503: {
    title: "Service Unavailable",
    body: "The application is temporarily unavailable. Please try again shortly.",
  },
};

const ERROR_CODE_IDS = new Set(["t4", "error__t4"]);
const ERROR_TITLE_IDS = new Set(["t5", "error__t5"]);
const ERROR_BODY_IDS = new Set(["t6", "error__t6"]);

function clampStrokeThickness(fontSize) {
  return Math.max(1, Math.min(3, Math.round((fontSize ?? 14) / 24)));
}

function errorCopyFor(code, details) {
  const fallback = {
    title: "Unexpected Error",
    body: "The application could not complete the requested operation.",
  };
  const base = ERROR_COPY[code] ?? fallback;

  return {
    code: String(code),
    title: base.title,
    body: details ? `${base.body}\n\n${details}` : base.body,
  };
}

function cloneArtboard(artboard) {
  return JSON.parse(JSON.stringify(artboard));
}

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

function updateErrorArtboard(baseArtboard, copy) {
  const artboard = cloneArtboard(baseArtboard);

  function visit(node) {
    if (node?.type === "text") {
      if (ERROR_CODE_IDS.has(node.id)) {
        node.content = copy.code;
      } else if (ERROR_TITLE_IDS.has(node.id)) {
        node.content = copy.title;
      } else if (ERROR_BODY_IDS.has(node.id)) {
        node.content = copy.body;
      }
      if (node.stroke?.fill) {
        node.stroke.thickness = clampStrokeThickness(node.fontSize);
      }
    }

    for (const child of node.children || []) {
      visit(child);
    }
  }

  visit(artboard);
  return artboard;
}

export function ErrorPage({ code, session, onNavigate, onSearch, searchQuery = "", details = "" }) {
  const copy = errorCopyFor(code, details);
  const loggedIn = Boolean(session?.authenticated && session?.authorized);
  const artboard = updateErrorArtboard(
    generatedArtboards[loggedIn ? "error-logged-in" : "error-logged-out"],
    copy
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
  const renderOverlay = ({ nodeIndex, textOverrides: overlayTextOverrides }) => {
    const sharedShell = typeof sharedShellOverlay === "function"
      ? sharedShellOverlay({ nodeIndex, textOverrides: overlayTextOverrides })
      : sharedShellOverlay;
    const bodyNode = nodeIndex.get(loggedIn ? "error__t6" : "t6");

    if (!bodyNode) {
      return sharedShell;
    }

    const bodyContent = overlayTextOverrides[bodyNode.id] ?? bodyNode.content ?? "";
    const bodyHeight = estimateWrappedTextHeight(
      bodyContent,
      bodyNode.width ?? 780,
      bodyNode.fontSize ?? 24
    );
    const buttonWidth = loggedIn ? 260 : 220;
    const buttonHeight = 48;
    const buttonLeft = (bodyNode.x ?? 0) + ((bodyNode.width ?? buttonWidth) - buttonWidth) / 2;
    const buttonTop = (bodyNode.y ?? 0) + bodyHeight + 28;

    return (
      <>
        {sharedShell}
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
