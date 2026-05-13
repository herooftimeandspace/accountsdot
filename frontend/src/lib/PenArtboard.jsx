import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { devMark } from "./devPerformance";

const ACCESSIBLE_UI_FONT_STACK = '"Atkinson Hyperlegible", Arial, Helvetica, sans-serif';

/**
 * clone documents runtime data flow for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

/**
 * uniquifyNodeIds documents runtime data flow for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function uniquifyNodeIds(artboard) {
  const seen = new Map();

  /**
   * visit documents runtime data flow for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
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
 * resolveFontFamily builds derived data for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function resolveFontFamily(fontFamily) {
  if (!fontFamily || /arial|helvetica/i.test(fontFamily)) {
    return ACCESSIBLE_UI_FONT_STACK;
  }
  return `${fontFamily}, ${ACCESSIBLE_UI_FONT_STACK}`;
}

/**
 * buildNodeIndex builds derived data for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function buildNodeIndex(node, map = new Map()) {
  map.set(node.id, node);
  for (const child of node.children || []) {
    buildNodeIndex(child, map);
  }
  return map;
}

/**
 * resolveTextContent builds derived data for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function resolveTextContent(node, textOverrides) {
  return Object.prototype.hasOwnProperty.call(textOverrides, node.id)
    ? textOverrides[node.id]
    : node.content ?? "";
}

/**
 * estimateTextNodeBox documents runtime data flow for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function estimateTextNodeBox(node, textOverrides) {
  const content = String(resolveTextContent(node, textOverrides) ?? "");
  const fontSize = node.fontSize ?? 14;
  const lineHeight = fontSize * 1.35;
  const width = node.width ?? Math.ceil(content.length * fontSize * 0.6);
  const hardLines = content.split("\n");
  const approxCharsPerLine = Math.max(1, Math.floor(width / Math.max(fontSize * 0.58, 1)));
  const lineCount = hardLines.reduce(
    (total, line) => total + Math.max(1, Math.ceil(Math.max(line.length, 1) / approxCharsPerLine)),
    0
  );

  return {
    width,
    height: node.height ?? Math.ceil(lineCount * lineHeight),
  };
}

/**
 * drawPath documents runtime data flow for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function drawPath(node) {
  const stroke = node.stroke?.fill;
  const strokeWidth = node.stroke?.thickness ?? 0;
  const lineJoin = node.stroke?.join ?? "round";
  const lineCap = node.stroke?.cap ?? "round";
  const fill = typeof node.fill === "string" ? node.fill : "none";
  return (
    <svg
      key={node.id}
      aria-hidden="true"
      style={{
        position: "absolute",
        left: node.x ?? 0,
        top: node.y ?? 0,
        width: node.width ?? 0,
        height: node.height ?? 0,
        overflow: "visible",
        opacity: node.opacity ?? 1,
      }}
      viewBox={Array.isArray(node.viewBox) ? node.viewBox.join(" ") : "0 0 24 24"}
    >
      <path
        d={node.geometry ?? ""}
        fill={fill}
        stroke={stroke}
        strokeWidth={strokeWidth}
        strokeLinejoin={lineJoin}
        strokeLinecap={lineCap}
      />
    </svg>
  );
}

/**
 * renderNode documents runtime data flow for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
function renderNode(node, textOverrides, hiddenNodeIds, imageNodeOverrides) {
  if (hiddenNodeIds.has(node.id)) {
    return null;
  }

  if (node.type === "frame") {
    const imageFill =
      node.fill && typeof node.fill === "object" && node.fill.type === "image" && node.fill.url
        ? node.fill
        : null;
    return (
      <div
        key={node.id}
        data-node-id={node.id}
        aria-hidden="true"
        style={{
          position: "absolute",
          left: node.x ?? 0,
          top: node.y ?? 0,
          width: node.width ?? 0,
          height: node.height ?? 0,
          background: typeof node.fill === "string" ? node.fill : undefined,
          border: node.stroke?.fill ? `${node.stroke.thickness ?? 1}px solid ${node.stroke.fill}` : undefined,
          borderRadius: node.cornerRadius ?? 0,
          overflow: imageFill || node.cornerRadius ? "hidden" : "visible",
          boxSizing: "border-box",
          opacity: node.opacity ?? 1,
        }}
      >
        {imageFill ? (
          <img
            src={imageFill.url}
            alt=""
            aria-hidden="true"
            style={{ width: "100%", height: "100%", objectFit: imageFill.mode === "fit" ? "contain" : "cover" }}
          />
        ) : null}
        {(node.children || []).map((child) =>
          renderNode(child, textOverrides, hiddenNodeIds, imageNodeOverrides)
        )}
      </div>
    );
  }

  if (node.type === "text") {
    const content = resolveTextContent(node, textOverrides);
    if (content === "") {
      return null;
    }
    return (
      <div
        key={node.id}
        data-node-id={node.id}
        aria-hidden="true"
        style={{
          position: "absolute",
          left: node.x ?? 0,
          top: node.y ?? 0,
          width: node.width ?? "auto",
          fontFamily: resolveFontFamily(node.fontFamily),
          fontSize: node.fontSize ?? 14,
          fontWeight: node.fontWeight ?? 400,
          color: node.fill ?? "var(--color-text)",
          lineHeight: 1.35,
          whiteSpace: "pre-wrap",
          textAlign: node.textAlign ?? "left",
          opacity: node.opacity ?? 1,
          WebkitTextStroke:
            node.stroke?.fill && node.stroke?.thickness
              ? `${node.stroke.thickness}px ${node.stroke.fill}`
              : undefined,
        }}
      >
        {content}
      </div>
    );
  }

  if (node.type === "ellipse") {
    const imageUrl = imageNodeOverrides[node.id];
    return (
      <div
        key={node.id}
        data-node-id={node.id}
        aria-hidden="true"
        style={{
          position: "absolute",
          left: node.x ?? 0,
          top: node.y ?? 0,
          width: node.width ?? 0,
          height: node.height ?? 0,
          borderRadius: "999px",
          background: imageUrl ? `center / cover no-repeat url("${imageUrl}")` : typeof node.fill === "string" ? node.fill : undefined,
          border: node.stroke?.fill ? `${node.stroke.thickness ?? 1}px solid ${node.stroke.fill}` : undefined,
          boxSizing: "border-box",
          opacity: node.opacity ?? 1,
        }}
      />
    );
  }

  if (node.type === "path") {
    return drawPath(node);
  }

  return null;
}

/**
 * PenArtboard renders the UI surface for frontend/src/lib/PenArtboard.jsx. Implemented pages call this renderer helper to draw .pen-derived nodes; debug it with artboard node data, text overrides, hidden node ids, and image overrides. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function PenArtboard({
  artboard,
  textOverrides = {},
  hotspots = {},
  hiddenNodeIds = [],
  imageNodeOverrides = {},
  renderOverlay = null,
}) {
  const containerRef = useRef(null);
  const normalizedArtboard = useMemo(() => uniquifyNodeIds(clone(artboard)), [artboard]);
  const [containerWidth, setContainerWidth] = useState(normalizedArtboard.width);
  const hiddenNodeIdSet = useMemo(() => new Set(hiddenNodeIds), [hiddenNodeIds]);
  const nodeIndex = useMemo(() => buildNodeIndex(normalizedArtboard), [normalizedArtboard]);

  useEffect(() => {
    devMark("artboard-render-commit", {
      artboardId: normalizedArtboard.id ?? null,
      nodeCount: nodeIndex.size,
    });
  }, [nodeIndex.size, normalizedArtboard.id]);

  useLayoutEffect(() => {
    const element = containerRef.current;
    if (!element) {
      return undefined;
    }

    const observer = new ResizeObserver(([entry]) => {
      setContainerWidth(entry.contentRect.width);
    });
    observer.observe(element);
    setContainerWidth(element.getBoundingClientRect().width || normalizedArtboard.width);

    return () => observer.disconnect();
  }, [normalizedArtboard.width]);

  const scale = Math.min(1, containerWidth / normalizedArtboard.width || 1);
  const scaledHeight = normalizedArtboard.height * scale;

  return (
    <div ref={containerRef} className="pen-stage" style={{ height: scaledHeight }}>
      <div
        className="pen-stage__artboard"
        style={{
          width: normalizedArtboard.width,
          height: normalizedArtboard.height,
          transform: `scale(${scale})`,
          transformOrigin: "top left",
          background:
            typeof normalizedArtboard.fill === "string"
              ? normalizedArtboard.fill
              : "var(--color-bg)",
        }}
      >
        {/* WCAG 1.3.1/4.1.2: PEN nodes are visual-only; page containers provide semantic structure. */}
        {(normalizedArtboard.children || []).map((child) =>
          renderNode(child, textOverrides, hiddenNodeIdSet, imageNodeOverrides)
        )}
        {Object.entries(hotspots).map(([nodeId, hotspot]) => {
          const node = nodeIndex.get(nodeId);
          if (!node || hiddenNodeIdSet.has(nodeId)) {
            return null;
          }

          const hotspotBox =
            node.type === "text"
              ? estimateTextNodeBox(node, textOverrides)
              : { width: node.width ?? 0, height: node.height ?? 0 };

          const baseStyle = {
            position: "absolute",
            left: node.x ?? 0,
            top: node.y ?? 0,
            width: hotspotBox.width,
            height: hotspotBox.height,
            border: 0,
            background: "transparent",
            padding: 0,
            cursor: "pointer",
          };
          const label = hotspot.label || "Page action";

          if (hotspot.href) {
            return (
              <a
                key={nodeId}
                className={hotspot.className || "pen-hotspot"}
                data-hotspot-id={nodeId}
                href={hotspot.href}
                onClick={hotspot.onClick}
                target={hotspot.target}
                rel={hotspot.rel}
                aria-label={label}
                title={label}
                style={baseStyle}
              />
            );
          }

          // WCAG 2.1.1/2.4.7/4.1.2: visual PEN controls become named, focus-visible native buttons.
          return (
            <button
              key={nodeId}
              className={hotspot.className || "pen-hotspot"}
              data-hotspot-id={nodeId}
              type="button"
              aria-label={label}
              title={label}
              onClick={hotspot.onClick}
              style={baseStyle}
            />
          );
        })}
        {typeof renderOverlay === "function" ? renderOverlay({ nodeIndex, textOverrides }) : null}
      </div>
    </div>
  );
}
