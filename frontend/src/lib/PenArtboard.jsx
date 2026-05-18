import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { devMark } from "./devPerformance";
import { artboardHasSharedShell } from "./sharedShellArtboard.mjs";
import { sharedShellSpec } from "../generated/artboards.generated.js";

const ACCESSIBLE_UI_FONT_STACK = '"Atkinson Hyperlegible", Arial, Helvetica, sans-serif';
const SHARED_SHELL_HEADER_NODE_IDS = new Set([
  "f3",
  sharedShellSpec.sharedShellIds.scopeField,
  sharedShellSpec.sharedShellIds.scopeTitle,
  sharedShellSpec.sharedShellIds.scopeSubtitle,
  sharedShellSpec.sharedShellIds.searchField,
  sharedShellSpec.sharedShellIds.searchIcon,
  sharedShellSpec.sharedShellIds.searchPlaceholder,
  sharedShellSpec.sharedShellIds.notificationBubble,
  sharedShellSpec.sharedShellIds.notificationCount,
  sharedShellSpec.sharedShellIds.helpIcon,
  sharedShellSpec.sharedShellIds.accountBox,
  sharedShellSpec.sharedShellIds.avatar,
  sharedShellSpec.sharedShellIds.initials,
  sharedShellSpec.sharedShellIds.userName,
  sharedShellSpec.sharedShellIds.userRole,
  "p14",
  "p15",
  "p16",
  "p17",
  "p18",
  "p19",
  "p20",
  "p21",
  "p24",
  "p28",
  "p29",
  "p32",
  "p33",
  "p40",
]);
const SHARED_SHELL_SIDEBAR_NODE_IDS = new Set([
  "f4",
  "img5",
  "t6",
  "t7",
  "t9",
  "l10",
  sharedShellSpec.sharedShellIds.navHighlight,
  sharedShellSpec.sharedShellIds.supportIcon,
  sharedShellSpec.sharedShellIds.supportLabel,
  sharedShellSpec.sharedShellIds.platformStatusLabel,
  sharedShellSpec.sharedShellIds.platformStatusDot,
  sharedShellSpec.sharedShellIds.platformStatusValue,
  "p93",
  "p94",
  ...Object.values(sharedShellSpec.navGroups).flat(),
]);
const SHARED_SHELL_STICKY_NODE_IDS = new Set([
  ...SHARED_SHELL_HEADER_NODE_IDS,
  ...SHARED_SHELL_SIDEBAR_NODE_IDS,
]);

function sharedShellStickyLayer(node) {
  if (!SHARED_SHELL_STICKY_NODE_IDS.has(node.id)) {
    return null;
  }
  return SHARED_SHELL_HEADER_NODE_IDS.has(node.id) ? "header" : "sidebar";
}

/**
 * clone copies generated artboard JSON before the renderer adds duplicate-id suffixes. Page components receive immutable module-level artboard objects from generatedArtboards, so cloning keeps one route render from changing the shared definition used by later route transitions or prefetches.
 */
function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

/**
 * uniquifyNodeIds preserves generated artboard rendering when Pencil exports repeated child ids. React keys and DOM data-node-id attributes must be stable inside one rendered artboard, so later duplicates receive deterministic suffixes while the first occurrence keeps the canonical id used by shell overlays and runtime hotspots.
 */
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

/**
 * resolveFontFamily keeps generated text readable when a `.pen` file names a display or system font. Arial-like source fonts collapse to the accessible UI stack, while display fonts keep their requested family with the UI stack as fallback.
 */
function resolveFontFamily(fontFamily) {
  if (!fontFamily || /arial|helvetica/i.test(fontFamily)) {
    return ACCESSIBLE_UI_FONT_STACK;
  }
  return `${fontFamily}, ${ACCESSIBLE_UI_FONT_STACK}`;
}

/**
 * buildNodeIndex gives runtime overlays O(1) access to generated nodes by id. Page overlays use the map to align native controls and row hotspots to the authoritative `.pen` geometry without hard-coding duplicated layout measurements in each page.
 */
function buildNodeIndex(node, map = new Map()) {
  map.set(node.id, node);
  for (const child of node.children || []) {
    buildNodeIndex(child, map);
  }
  return map;
}

/**
 * resolveTextContent applies runtime-safe text overrides to generated visual nodes. Shared shell and page components use this to replace persona, scope, refresh, and mock payload labels while leaving all other `.pen` text faithful to the generated source.
 */
function resolveTextContent(node, textOverrides) {
  return Object.prototype.hasOwnProperty.call(textOverrides, node.id)
    ? textOverrides[node.id]
    : node.content ?? "";
}

/**
 * estimateTextNodeBox approximates the clickable box for text-backed hotspots. It mirrors the renderer's wrapping assumptions closely enough for native buttons and links to cover generated labels when the `.pen` node has no fixed exported height.
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
 * drawPath renders Pencil path geometry as visual-only SVG. Icons, line art, and small controls remain non-semantic here because page containers and runtime overlays provide the accessible names and interaction roles.
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
 * renderNode converts generated `.pen` nodes into absolutely positioned visual DOM. It also tags shared-shell header/sidebar nodes so the shared CSS can anchor chrome while the generated page pane scrolls; runtime overlays remain responsible for actual form controls, links, row selection, and drawer state.
 */
function renderNode(node, textOverrides, hiddenNodeIds, imageNodeOverrides, enableSharedShellSticky) {
  if (hiddenNodeIds.has(node.id)) {
    return null;
  }
  const stickyLayer = enableSharedShellSticky ? sharedShellStickyLayer(node) : null;
  const stickyClassName = stickyLayer ? `pen-node--shared-shell-sticky pen-node--shared-shell-${stickyLayer}` : undefined;

  if (node.type === "frame") {
    const imageFill =
      node.fill && typeof node.fill === "object" && node.fill.type === "image" && node.fill.url
        ? node.fill
        : null;
    return (
      <div
        key={node.id}
        data-node-id={node.id}
        data-shared-shell-sticky={stickyLayer ?? undefined}
        aria-hidden="true"
        className={stickyClassName}
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
          renderNode(child, textOverrides, hiddenNodeIds, imageNodeOverrides, enableSharedShellSticky)
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
        data-shared-shell-sticky={stickyLayer ?? undefined}
        aria-hidden="true"
        className={stickyClassName}
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
        data-shared-shell-sticky={stickyLayer ?? undefined}
        aria-hidden="true"
        className={stickyClassName}
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
    const path = drawPath(node);
    if (!stickyLayer) {
      return path;
    }
    return (
      <span
        key={node.id}
        data-node-id={node.id}
        data-shared-shell-sticky={stickyLayer}
        className={stickyClassName}
        aria-hidden="true"
        style={{
          position: "absolute",
          left: node.x ?? 0,
          top: node.y ?? 0,
          width: node.width ?? 0,
          height: node.height ?? 0,
        }}
      >
        {drawPath({ ...node, x: 0, y: 0 })}
      </span>
    );
  }

  return null;
}

/**
 * PenArtboard is the shared renderer for implemented `.pen` pages. Static and runtime-backed pages pass generated artboard JSON, hidden-node rules, text/image overrides, and an optional overlay callback; this component measures the available width, scales the visual artboard with CSS zoom so anchored shell layers keep their viewport behavior, and exposes the node index to overlays for precise native controls.
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
  const enableSharedShellSticky = useMemo(() => artboardHasSharedShell(normalizedArtboard), [normalizedArtboard]);

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
          zoom: scale,
          background:
            typeof normalizedArtboard.fill === "string"
              ? normalizedArtboard.fill
              : "var(--color-bg)",
        }}
      >
        {/* WCAG 1.3.1/4.1.2: PEN nodes are visual-only; page containers provide semantic structure. */}
        {(normalizedArtboard.children || []).map((child) =>
          renderNode(child, textOverrides, hiddenNodeIdSet, imageNodeOverrides, enableSharedShellSticky)
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
