import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const projectRoot = path.resolve(path.dirname(__filename), "..");
const paletteColors = new Set([
  "#01161E",
  "#FFFFFF",
  "#CEB770",
  "#DDE2E3",
  "#FCD9E9",
  "#D73533",
  "#FE5E41",
  "#00A878",
  "#A6B0B5",
  "#CA8BB9",
  "#6E7E85",
  "#774936",
  "#89B6E7",
  "#5797DB",
  "#2D7ED2",
  "#194676",
  "#0E2843",
]);

const generatedArtboardDir = path.join(projectRoot, "frontend", "src", "generated");
const hotspotTargetsByArtboard = new Map([
  [
    "data-quality.artboard.json",
    {
      refresh: "f104",
      openMappingDashboard: "f183",
    },
  ],
  [
    "login.artboard.json",
    {
      loginWithGoogle: "f5",
    },
  ],
]);

const artboardPages = fs
  .readdirSync(generatedArtboardDir)
  .filter((fileName) => fileName.endsWith(".artboard.json"))
  .sort()
  .map((fileName) => ({
    name: fileName.replace(/\.artboard\.json$/, ""),
    artboardPath: path.join("frontend/src/generated", fileName),
    hotspots: hotspotTargetsByArtboard.get(fileName) || {},
  }));

const sourceChecks = [
  {
    name: "Data Quality",
    sourcePath: "frontend/src/pages/DataQualityPage.jsx",
    required: [
      "DataQualitySemanticContent",
      "data-quality-semantic__mobile-actions",
      "<table>",
      'scope="col"',
      'scope="row"',
      "WCAG 1.3.1/1.4.10/2.4.6",
    ],
  },
  {
    name: "App route status",
    sourcePath: "frontend/src/app.jsx",
    required: ["document.title", "WCAG 2.4.2", "WCAG 4.1.3", "WCAG 2.4.1/2.4.7", "skip-link"],
  },
  {
    name: "Login page",
    sourcePath: "frontend/src/pages/LoginPage.jsx",
    required: ["login-page-title", "WCAG 2.4.2/2.4.6"],
  },
  {
    name: "Error page",
    sourcePath: "frontend/src/pages/ErrorPage.jsx",
    required: ["semanticTitleId", "WCAG 1.3.1/2.4.2"],
  },
  {
    name: "Static PEN page",
    sourcePath: "frontend/src/pages/StaticPenPage.jsx",
    required: ["buildArtboardSemanticSummary", "WCAG 1.3.1/2.4.2", "aria-labelledby={semanticTitleId}"],
  },
  {
    name: "Phone Directory",
    sourcePath: "frontend/src/pages/PhoneDirectoryPage.jsx",
    required: [
      "PHONE_DIRECTORY_HEADING_ID",
      "WCAG 1.3.1/2.4.6",
      "WCAG 1.3.1/3.3.2/4.1.2",
      "aria-pressed",
      "aria-current",
    ],
  },
  {
    name: "Shared shell",
    sourcePath: "frontend/src/lib/sharedShellPresentation.jsx",
    required: ["WCAG 2.4.4/4.1.2", "role=\"search\"", "aria-label={`Open ${label}`}"],
  },
  {
    name: "PEN artboard bridge",
    sourcePath: "frontend/src/lib/PenArtboard.jsx",
    required: ["WCAG 1.3.1/4.1.2", "WCAG 2.1.1/2.4.7/4.1.2"],
  },
];

function fail(message) {
  throw new Error(message);
}

function readJSON(relativePath) {
  return JSON.parse(fs.readFileSync(path.join(projectRoot, relativePath), "utf8"));
}

function readText(relativePath) {
  return fs.readFileSync(path.join(projectRoot, relativePath), "utf8");
}

function listFiles(relativeDir, predicate) {
  const absoluteDir = path.join(projectRoot, relativeDir);
  const files = [];

  function walk(currentDir) {
    for (const entry of fs.readdirSync(currentDir, { withFileTypes: true })) {
      const absolutePath = path.join(currentDir, entry.name);
      if (entry.isDirectory()) {
        walk(absolutePath);
        continue;
      }
      const relativePath = path.relative(projectRoot, absolutePath);
      if (!predicate || predicate(relativePath)) {
        files.push(relativePath);
      }
    }
  }

  walk(absoluteDir);
  return files.sort();
}

function flattenNodes(root) {
  const nodes = [];
  let order = 0;

  function walk(node) {
    nodes.push({ ...node, order: order++ });
    for (const child of node.children || []) {
      walk(child);
    }
  }

  walk(root);
  return nodes;
}

function parseHexColor(value) {
  const match = /^#([0-9a-f]{6})$/i.exec(value || "");
  if (!match) {
    return null;
  }
  const number = Number.parseInt(match[1], 16);
  return [(number >> 16) & 255, (number >> 8) & 255, number & 255];
}

function relativeLuminance([red, green, blue]) {
  return [red, green, blue]
    .map((channel) => {
      const value = channel / 255;
      return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
    })
    .reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0);
}

function contrastRatio(foreground, background) {
  const lighter = Math.max(relativeLuminance(foreground), relativeLuminance(background));
  const darker = Math.min(relativeLuminance(foreground), relativeLuminance(background));
  return (lighter + 0.05) / (darker + 0.05);
}

function containsPoint(node, x, y) {
  return (
    (node.x ?? 0) <= x &&
    x <= (node.x ?? 0) + (node.width ?? 0) &&
    (node.y ?? 0) <= y &&
    y <= (node.y ?? 0) + (node.height ?? 0)
  );
}

function backgroundForText(textNode, nodes, artboardFill) {
  const centerX = (textNode.x ?? 0) + (textNode.width ?? 0) / 2;
  const centerY = (textNode.y ?? 0) + (textNode.height ?? textNode.fontSize ?? 14) / 2;
  const candidates = nodes.filter(
    (node) =>
      node.order < textNode.order &&
      node.type === "frame" &&
      typeof node.fill === "string" &&
      (node.width ?? 0) > 1 &&
      (node.height ?? 0) > 1 &&
      containsPoint(node, centerX, centerY)
  );
  return candidates.at(-1)?.fill || artboardFill || "#FFFFFF";
}

function assertHotspotTargets(page, nodes) {
  for (const [name, nodeId] of Object.entries(page.hotspots)) {
    const node = nodes.find((candidate) => candidate.id === nodeId);
    if (!node) {
      fail(`${page.name} hotspot ${name} points at missing node ${nodeId}`);
    }
    if ((node.width ?? 0) < 24 || (node.height ?? 0) < 24) {
      fail(`${page.name} hotspot ${name} must be at least 24 by 24 CSS pixels for WCAG 2.5.8`);
    }
  }
}

function assertTextContrast(page, artboard, nodes) {
  for (const node of nodes.filter((candidate) => candidate.type === "text")) {
    const foreground = parseHexColor(node.fill);
    const background = parseHexColor(backgroundForText(node, nodes, artboard.fill));
    if (!foreground || !background) {
      continue;
    }

    const ratio = contrastRatio(foreground, background);
    const fontSize = node.fontSize ?? 14;
    const fontWeight = Number.parseInt(String(node.fontWeight ?? 400), 10);
    const isLargeText = fontSize >= 24 || (fontSize >= 18.66 && fontWeight >= 700);
    const minimum = isLargeText ? 3 : 4.5;
    const strokeColor = parseHexColor(node.stroke?.fill);
    const strokeRatio = strokeColor ? contrastRatio(strokeColor, background) : 0;
    const hasReadableStroke = (node.stroke?.thickness ?? 0) >= 1 && strokeRatio >= minimum;

    if (ratio < minimum && !hasReadableStroke) {
      fail(
        `${page.name} text node ${node.id} contrast ${ratio.toFixed(2)} is below WCAG 1.4.3 minimum ${minimum}`
      );
    }
  }
}

function assertPaletteUsage(page, nodes) {
  for (const node of nodes) {
    for (const [property, color] of [
      ["fill", node.fill],
      ["stroke", node.stroke?.fill],
    ]) {
      if (typeof color === "string" && color.startsWith("#") && !paletteColors.has(color.toUpperCase())) {
        fail(`${page.name} node ${node.id} uses non-palette ${property} color ${color}`);
      }
    }
  }
}

function assertSourceMarkers(check) {
  const source = readText(check.sourcePath);
  for (const required of check.required) {
    if (!source.includes(required)) {
      fail(`${check.name} source is missing accessibility marker ${required}`);
    }
  }
}

function assertFocusStyles() {
  const css = readText("frontend/src/styles.css");
  for (const required of [
    "--color-text: #01161E",
    "--color-bg: #FFFFFF",
    "--color-highlight: #CEB770",
    "--color-canvas: #DDE2E3",
    ".sr-only",
    ".skip-link:focus-visible",
    ".pen-hotspot:focus-visible",
    ".shared-shell-search:focus-within",
    ".phone-directory-runtime__local-search:focus-within",
    ".phone-directory-runtime__row:focus-visible",
    ".phone-directory-runtime__mode-button:focus-visible",
    "@media (max-width: 900px)",
    ".data-quality-semantic",
  ]) {
    if (!css.includes(required)) {
      fail(`frontend styles are missing focus-visible rule ${required}`);
    }
  }

  const hexColors = css.match(/#[0-9a-f]{6}/gi) || [];
  for (const color of hexColors) {
    if (!paletteColors.has(color.toUpperCase())) {
      fail(`frontend styles use non-palette color ${color}`);
    }
  }
}

function assertFrontendSourcePaletteUsage() {
  const sourceFiles = listFiles("frontend/src", (relativePath) => {
    if (relativePath.includes("/generated/")) {
      return false;
    }
    return /\.(css|js|jsx)$/.test(relativePath);
  });

  for (const filePath of sourceFiles) {
    const hexColors = readText(filePath).match(/#[0-9a-f]{6}/gi) || [];
    for (const color of hexColors) {
      if (!paletteColors.has(color.toUpperCase())) {
        fail(`${filePath} uses non-palette color ${color}`);
      }
    }
  }
}

function assertPenSourcePaletteUsage() {
  const penFiles = listFiles("docs/mocks/wireframes", (relativePath) => relativePath.endsWith(".pen"));

  for (const filePath of penFiles) {
    const hexColors = readText(filePath).match(/#[0-9a-f]{6}/gi) || [];
    for (const color of hexColors) {
      if (!paletteColors.has(color.toUpperCase())) {
        fail(`${filePath} uses non-palette color ${color}`);
      }
    }
  }
}

for (const page of artboardPages) {
  const artboard = readJSON(page.artboardPath);
  const nodes = flattenNodes(artboard);
  assertPaletteUsage(page, nodes);
  assertHotspotTargets(page, nodes);
  assertTextContrast(page, artboard, nodes);
}

for (const check of sourceChecks) {
  assertSourceMarkers(check);
}

assertFocusStyles();
assertFrontendSourcePaletteUsage();
assertPenSourcePaletteUsage();
console.log("frontend accessibility checks passed");
