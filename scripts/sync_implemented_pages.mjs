import crypto from "node:crypto";
import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = process.cwd();
const generatedDir = path.join(repoRoot, "frontend", "src", "generated");
const publicAssetsDir = path.join(repoRoot, "frontend", "public", "pen-assets");

const STANDALONE_WIDTH = 1672;
const STANDALONE_HEIGHT = 1080;
const LOGGED_IN_PANE_X = 264;
const LOGGED_IN_PANE_Y = 76;
const SHARED_SHELL_SOURCE = "docs/design/mocks/wireframes/wireframe-shared-shell.pen";

const artboardSpecs = [
  {
    key: "data-quality",
    source: "docs/design/mocks/wireframes/wireframe-data-quality-dashboard.pen",
    mode: "merge-shell",
    paneIdPrefix: "",
  },
  {
    key: "dashboard-it-admin",
    source: "docs/design/mocks/wireframes/wireframe-it-admin-overview.pen",
    mode: "merge-shell",
  },
  {
    key: "dashboard-hr-lifecycle",
    source: "docs/design/mocks/wireframes/wireframe-hr-lifecycle-overview.pen",
    mode: "merge-shell",
  },
  {
    key: "dashboard-site-admin",
    source: "docs/design/mocks/wireframes/wireframe-site-admin-dashboard.pen",
    mode: "merge-shell",
    standardPrimitives: [],
  },
  {
    key: "onboarding",
    source: "docs/design/mocks/wireframes/wireframe-onboarding-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "offboarding",
    source: "docs/design/mocks/wireframes/wireframe-offboarding-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "room-moves",
    source: "docs/design/mocks/wireframes/wireframe-room-moves.pen",
    mode: "merge-shell",
  },
  {
    key: "room-moves-bulk-draft",
    source: "docs/design/mocks/wireframes/wireframe-room-moves-bulk-draft.pen",
    mode: "merge-shell",
    standardPrimitives: [],
  },
  {
    key: "phone-directory-by-person",
    source: "docs/design/mocks/wireframes/wireframe-phone-directory-by-person.pen",
    mode: "merge-shell",
  },
  {
    key: "phone-directory-by-room",
    source: "docs/design/mocks/wireframes/wireframe-phone-directory-by-room.pen",
    mode: "merge-shell",
  },
  {
    key: "phone-directory-by-department",
    source: "docs/design/mocks/wireframes/wireframe-phone-directory-by-department.pen",
    mode: "merge-shell",
  },
  {
    key: "frequent-fliers",
    source: "docs/design/mocks/wireframes/wireframe-device-wrangler-frequent-fliers.pen",
    mode: "merge-shell",
  },
  {
    key: "student-data-cleanup",
    source: "docs/design/mocks/wireframes/wireframe-site-secretary-student-data-cleanup.pen",
    mode: "merge-shell",
  },
  {
    key: "reports",
    source: "docs/design/mocks/wireframes/wireframe-it-admin-reports.pen",
    mode: "merge-shell",
  },
  {
    key: "reports-sync-transparency",
    source: "docs/design/mocks/wireframes/wireframe-sync-transparency-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "reports-ticketing-human-work",
    source: "docs/design/mocks/wireframes/wireframe-ticketing-human-work.pen",
    mode: "merge-shell",
  },
  {
    key: "admin",
    source: "docs/design/mocks/wireframes/wireframe-it-admin-admin-controls.pen",
    mode: "merge-shell",
    standardPrimitives: [],
  },
  {
    key: "admin-feature-flags",
    source: "docs/design/mocks/wireframes/wireframe-admin-feature-flags.pen",
    mode: "merge-shell",
    standardPrimitives: [],
  },
  {
    key: "my-profile",
    source: "docs/design/mocks/wireframes/wireframe-faculty-staff-my-profile.pen",
    mode: "merge-shell",
    standardPrimitives: [],
  },
  {
    key: "login",
    source: "docs/design/mocks/wireframes/wireframe-login.pen",
    mode: "passthrough",
  },
  {
    key: "error-logged-in",
    source: "docs/design/mocks/wireframes/wireframe-http-error.pen",
    mode: "error-logged-in",
  },
  {
    key: "error-logged-out",
    source: "docs/design/mocks/wireframes/wireframe-http-error.pen",
    mode: "error-logged-out",
  },
];

const stableAssets = new Map([
  [
    path.join(repoRoot, "docs", "reference-inputs", "branding", "Firefly.png"),
    "firefly.png",
  ],
  [
    path.join(repoRoot, "docs", "reference-inputs", "branding", "google-g.png"),
    "google-g.png",
  ],
  [
    path.join(repoRoot, "docs", "reference-inputs", "branding", "Wordmarks", "Gold W black outline.png"),
    "gold-w-black-outline.png",
  ],
  [
    path.join(repoRoot, "docs", "design", "mocks", "wireframes", "varsity_regular.ttf"),
    "varsity_regular.ttf",
  ],
]);

const dataQualityDesign = {
  key: "data-quality",
  width: 1672,
  height: 1080,
  hotspots: {
    refresh: "f104",
  },
  slots: {
    shell: {
      scopeTitle: "t12",
      scopeSubtitle: "t13",
      searchPlaceholder: "t25",
      notificationCount: "t31",
      userAvatar: "e36",
      userInitials: "t37",
      userName: "t38",
      userRole: "t39",
      platformStatus: "t98",
    },
    page: {
      title: "t99",
      lastRefreshed: "t103",
      refreshLabel: "t105",
    },
    summaryCards: [
      { title: "t107", count: "t108" },
      { title: "t111", count: "t112" },
      { title: "t115", count: "t116" },
      { title: "t119", count: "t120" },
    ],
    queue: {
      headers: {
        issue: "t131",
        source: "t132",
        owner: "t133",
        impact: "t134",
        nextAction: "t135",
      },
      rows: [
        { issue: "t137", source: "t138", owner: "t139", impact: ["t140"], nextAction: ["t141"] },
        { issue: "t143", source: "t144", owner: "t145", impact: ["t146"], nextAction: ["t147"] },
        { issue: "t149", source: "t150", owner: "t151", impact: ["t152"], nextAction: ["t153"] },
        { issue: "t155", source: "t156", owner: "t157", impact: ["t158"], nextAction: ["t159"] },
        { issue: "t161", source: "t162", owner: "t163", impact: ["t164"], nextAction: ["t166"] },
      ],
    },
  },
};

const shellSpec = {
  sharedShellIds: {
    scopeField: "f11",
    scopeTitle: "t12",
    scopeSubtitle: "t13",
    searchField: "f22",
    searchIcon: "p23",
    searchPlaceholder: "t25",
    notificationBubble: "f30",
    notificationCount: "t31",
    helpIcon: "p34",
    accountBox: "f35",
    avatar: "e36",
    initials: "t37",
    userName: "t38",
    userRole: "t39",
    navHighlight: "f64",
    supportIcon: "p92",
    supportLabel: "t95",
    platformStatusLabel: "t96",
    platformStatusDot: "e97",
    platformStatusValue: "t98",
  },
  navGroups: {
    dashboard: ["p41", "p42", "p43", "p44", "t45"],
    onboarding: ["p46", "p47", "p48", "p49", "t50", "p51"],
    offboarding: ["p52", "p53", "p54", "t55", "p56"],
    roomMoves: ["p57", "p58", "p59", "p60", "t61"],
    phoneDirectory: ["p62", "t63"],
    dataQuality: ["p65", "p66", "p67", "t68"],
    frequentFliers: ["p69", "p70", "p71", "p72", "t73"],
    studentDataCleanup: ["p74", "p75", "p76", "p77", "p78", "t79"],
    reports: ["p80", "p81", "p82", "p83", "t84", "p85"],
    admin: ["p86", "p87", "p88", "p89", "t90", "p91", "e92a", "t92a"],
  },
  navLabelIds: {
    dashboard: "t45",
    onboarding: "t50",
    offboarding: "t55",
    roomMoves: "t61",
    phoneDirectory: "t63",
    dataQuality: "t68",
    frequentFliers: "t73",
    studentDataCleanup: "t79",
    reports: "t84",
    admin: "t90",
  },
};

const standardPrimitiveSpec = {
  refresh: {
    label: "Refresh",
    role: "standard-header-action",
    frame: {
      x: 1540,
      y: 90,
      width: 112,
      height: 38,
      fill: "#CEB770",
      stroke: "#CEB770",
      cornerRadius: 8,
    },
    text: {
      y: 101,
      fontSize: 13,
      fontWeight: "700",
      fill: "#01161E",
      textAlign: "center",
    },
  },
};

const activeNavByArtboardKey = {
  "dashboard-it-admin": "dashboard",
  "dashboard-hr-lifecycle": "dashboard",
  "dashboard-site-admin": "dashboard",
  onboarding: "onboarding",
  offboarding: "offboarding",
  "room-moves": "roomMoves",
  "phone-directory-by-person": "phoneDirectory",
  "phone-directory-by-room": "phoneDirectory",
  "phone-directory-by-department": "phoneDirectory",
  "data-quality": "dataQuality",
  "frequent-fliers": "frequentFliers",
  "student-data-cleanup": "studentDataCleanup",
  reports: "reports",
  "reports-sync-transparency": "reports",
  "reports-ticketing-human-work": "reports",
  admin: "admin",
  "admin-feature-flags": "admin",
};

const loggedInArtboardKeys = artboardSpecs
  .filter((spec) => !["login", "error-logged-out"].includes(spec.key))
  .map((spec) => spec.key);

function generatedManifest() {
  return {
    schemaVersion: 1,
    sourceOfTruth: [
      "README.md",
      "docs/planning/implementation-plan.md",
      "docs/product/product-requirements.md",
      "docs/testing/test-matrix.md",
      ".agents/AGENTS.md",
      "docs/design/mocks/wireframes/implemented-page-design-contract.md",
    ],
    generatedBy: "scripts/sync_implemented_pages.mjs",
    generatedFileGlobs: [
      "frontend/src/generated/*.artboard.json",
      "frontend/src/generated/artboards.generated.js",
      "frontend/src/generated/data-quality.generated.jsx",
      "frontend/src/generated/implemented-page-design-manifest.generated.json",
    ],
    artboards: artboardSpecs.map((spec) => ({
      key: spec.key,
      sourcePen: spec.source,
      mode: spec.mode,
      activeNav: activeNavByArtboardKey[spec.key] ?? null,
      loggedInShell: loggedInArtboardKeys.includes(spec.key),
      standardPrimitives:
        spec.standardPrimitives ??
        (loggedInArtboardKeys.includes(spec.key) && !spec.key.startsWith("error-")
          ? ["refresh"]
          : []),
    })),
    sharedShell: {
      sourcePen: SHARED_SHELL_SOURCE,
      loggedInPane: {
        x: LOGGED_IN_PANE_X,
        y: LOGGED_IN_PANE_Y,
      },
      ...shellSpec,
    },
    standardPrimitives: standardPrimitiveSpec,
    lintPolicy: {
      initialPosture: "warn broadly, fail high-confidence regressions",
      warningPromotion: "Promote stable warning checks to failures after false positives are resolved.",
      minimumVisualGapPx: 5,
      recoveryLayers: ["pipeline", ".pen layout", "docs/new behavior", "runtime behavior", "review artifact"],
    },
  };
}

const assetCache = new Map();

async function assertFileExists(absolutePath, label) {
  try {
    await fs.access(absolutePath);
  } catch (error) {
    throw new Error(`${label} does not exist: ${path.relative(repoRoot, absolutePath)}`);
  }
}

function collectImageFillUrls(node, urls = []) {
  if (node?.fill && typeof node.fill === "object" && node.fill.type === "image" && node.fill.url) {
    urls.push(node.fill.url);
  }
  for (const child of node?.children ?? []) {
    collectImageFillUrls(child, urls);
  }
  return urls;
}

async function validateSourceImageFills() {
  const sources = new Set([SHARED_SHELL_SOURCE, ...artboardSpecs.map((spec) => spec.source)]);
  for (const source of sources) {
    const root = await readPenRoot(source);
    for (const url of collectImageFillUrls(root)) {
      const resolved = path.isAbsolute(url) ? url : path.join(repoRoot, url);
      await assertFileExists(resolved, `Image fill in ${source}`);
    }
  }
}

async function validateGeneratedImageFills(artboards) {
  for (const [key, artboard] of artboards.entries()) {
    for (const url of collectImageFillUrls(artboard)) {
      if (!url.startsWith("/pen-assets/")) {
        throw new Error(`Generated artboard ${key} has non-public image URL: ${url}`);
      }
      await assertFileExists(path.join(repoRoot, "frontend", "public", url.replace(/^\//, "")), `Generated image fill in ${key}`);
    }
  }
}

function deepClone(value) {
  return JSON.parse(JSON.stringify(value));
}

async function readPenRoot(relativePath) {
  const absolutePath = path.join(repoRoot, relativePath);
  const content = await fs.readFile(absolutePath, "utf8");
  const parsed = JSON.parse(content);
  if (!Array.isArray(parsed.children) || parsed.children.length === 0) {
    throw new Error(`${relativePath} does not contain a root artboard`);
  }
  return parsed.children[0];
}

function isShellNode(node) {
  return (node.x ?? 0) < LOGGED_IN_PANE_X || (node.y ?? 0) < LOGGED_IN_PANE_Y;
}

function isPaneNode(node) {
  return (node.x ?? 0) >= LOGGED_IN_PANE_X && (node.y ?? 0) >= LOGGED_IN_PANE_Y;
}

async function ensureAsset(sourcePath) {
  const resolved = path.isAbsolute(sourcePath) ? sourcePath : path.join(repoRoot, sourcePath);
  if (assetCache.has(resolved)) {
    return assetCache.get(resolved);
  }

  const stableName = stableAssets.get(resolved);
  const targetName =
    stableName ??
    `${path.parse(resolved).name.replace(/[^a-z0-9_-]+/gi, "-").toLowerCase()}-${crypto
      .createHash("sha1")
      .update(resolved)
      .digest("hex")
      .slice(0, 10)}${path.extname(resolved).toLowerCase()}`;

  await fs.mkdir(publicAssetsDir, { recursive: true });
  const targetPath = path.join(publicAssetsDir, targetName);
  await fs.copyFile(resolved, targetPath);
  const publicUrl = `/pen-assets/${targetName}`;
  assetCache.set(resolved, publicUrl);
  return publicUrl;
}

async function normalizeNode(node, offsetX = 0, offsetY = 0, idPrefix = "") {
  const clone = deepClone(node);
  if (typeof clone.id === "string" && idPrefix) {
    clone.id = `${idPrefix}${clone.id}`;
  }
  if (typeof clone.x === "number") {
    clone.x += offsetX;
  }
  if (typeof clone.y === "number") {
    clone.y += offsetY;
  }
  if (clone.fill && typeof clone.fill === "object" && clone.fill.type === "image" && clone.fill.url) {
    clone.fill.url = await ensureAsset(clone.fill.url);
  }
  if (Array.isArray(clone.children) && clone.children.length > 0) {
    const nextChildren = [];
    for (const child of clone.children) {
      nextChildren.push(await normalizeNode(child, 0, 0, idPrefix));
    }
    clone.children = nextChildren;
  }
  return clone;
}

async function buildArtboard(spec, shellRoot) {
  const sourceRoot = await readPenRoot(spec.source);

  if (spec.mode === "passthrough") {
    return normalizeNode(sourceRoot, 0, 0);
  }

  if (spec.mode === "merge-shell") {
    const children = [];
    const panePrefix = spec.paneIdPrefix ?? `${spec.key}__`;
    for (const child of shellRoot.children.filter(isShellNode)) {
      children.push(await normalizeNode(child, 0, 0));
    }
    for (const child of sourceRoot.children.filter(isPaneNode)) {
      children.push(await normalizeNode(child, 0, 0, panePrefix));
    }

    const root = await normalizeNode(shellRoot, 0, 0);
    root.name = sourceRoot.name || spec.key;
    // Merged logged-in pages inherit the shared shell frame, but page-local
    // panes may intentionally extend below the first viewport for runtime
    // overlays or long tables. Preserve the larger source bounds so generated
    // artboards do not clip valid page-local layout nodes.
    root.width = Math.max(root.width ?? 0, sourceRoot.width ?? 0);
    root.height = Math.max(root.height ?? 0, sourceRoot.height ?? 0);
    root.children = children;
    return root;
  }

  if (spec.mode === "error-logged-in") {
    const children = [];
    for (const child of shellRoot.children.filter(isShellNode)) {
      children.push(await normalizeNode(child, 0, 0));
    }
    for (const child of sourceRoot.children) {
      children.push(await normalizeNode(child, LOGGED_IN_PANE_X, LOGGED_IN_PANE_Y, "error__"));
    }

    const root = await normalizeNode(shellRoot, 0, 0);
    root.name = "HTTP Error (Logged In)";
    root.children = children;
    return root;
  }

  if (spec.mode === "error-logged-out") {
    const centeredOffsetX = Math.floor((STANDALONE_WIDTH - sourceRoot.width) / 2);
    const centeredOffsetY = Math.floor((STANDALONE_HEIGHT - sourceRoot.height) / 2);
    const backgroundRoot = {
      type: "frame",
      id: "root",
      name: "HTTP Error",
      x: 0,
      y: 0,
      width: STANDALONE_WIDTH,
      height: STANDALONE_HEIGHT,
      fill: "#DDE2E3",
      cornerRadius: 0,
      layout: "none",
      children: [
        {
          type: "frame",
          id: "error_background",
          name: "Background",
          x: 0,
          y: 0,
          width: STANDALONE_WIDTH,
          height: STANDALONE_HEIGHT,
          fill: "#FFFFFF",
          cornerRadius: 0,
          layout: "none",
          stroke: {
            thickness: 1,
            fill: "#FFFFFF",
            align: "inside",
          },
        },
      ],
    };

    for (const child of sourceRoot.children) {
      backgroundRoot.children.push(await normalizeNode(child, centeredOffsetX, centeredOffsetY));
    }
    return backgroundRoot;
  }

  throw new Error(`Unsupported artboard mode: ${spec.mode}`);
}

function toJsonFileName(key) {
  return `${key}.artboard.json`;
}

function generateArtboardsModule(specs) {
  const loaderEntries = specs
    .map((spec) => `  "${spec.key}": () => import("./${toJsonFileName(spec.key)}").then((module) => module.default),`)
    .join("\n");

  const metadata = specs
    .map(
      (spec) =>
        `  "${spec.key}": { key: "${spec.key}", sourcePen: "${spec.source}", activeNav: ${JSON.stringify(
          activeNavByArtboardKey[spec.key] ?? null
        )} },`
    )
    .join("\n");

  return `export const generatedArtboardLoaders = {
${loaderEntries}
};

const generatedArtboardCache = new Map();

export function loadGeneratedArtboard(key) {
  const loader = generatedArtboardLoaders[key];
  if (!loader) {
    return Promise.reject(new Error(\`Unknown generated artboard: \${key}\`));
  }
  if (!generatedArtboardCache.has(key)) {
    generatedArtboardCache.set(key, loader());
  }
  return generatedArtboardCache.get(key);
}

export const generatedArtboardMeta = {
${metadata}
};

export const sharedShellSpec = ${JSON.stringify(shellSpec, null, 2)};

export const implementedPageDesignManifest = ${JSON.stringify(generatedManifest(), null, 2)};
`;
}

function generateDataQualityViewModule() {
  return `import React from "react";
import artboard from "./data-quality.artboard.json";
import { PenArtboard } from "../lib/PenArtboard";

export const dataQualityDesign = ${JSON.stringify(dataQualityDesign, null, 2)};

export function DataQualityGeneratedView({ textOverrides = {}, hotspots = {}, hiddenNodeIds = [], imageNodeOverrides = {}, renderOverlay = null }) {
  return (
    <PenArtboard
      artboard={artboard}
      textOverrides={textOverrides}
      hotspots={hotspots}
      hiddenNodeIds={hiddenNodeIds}
      imageNodeOverrides={imageNodeOverrides}
      renderOverlay={renderOverlay}
    />
  );
}
`;
}

async function readFileIfExists(targetPath) {
  try {
    return await fs.readFile(targetPath, "utf8");
  } catch (error) {
    if (error && error.code === "ENOENT") {
      return null;
    }
    throw error;
  }
}

async function syncFile(targetPath, nextContent, checkOnly) {
  const normalizedContent = nextContent.replace(/\n+$/, "\n");
  const currentContent = await readFileIfExists(targetPath);
  if (currentContent === normalizedContent) {
    return false;
  }
  if (checkOnly) {
    throw new Error(`Generated file drift detected: ${path.relative(repoRoot, targetPath)}`);
  }
  await fs.mkdir(path.dirname(targetPath), { recursive: true });
  await fs.writeFile(targetPath, normalizedContent, "utf8");
  return true;
}

async function syncBinaryAsset(sourcePath, checkOnly) {
  const publicUrl = await ensureAsset(sourcePath);
  const targetPath = path.join(repoRoot, "frontend", "public", publicUrl.replace(/^\//, ""));
  const sourceBuffer = await fs.readFile(path.isAbsolute(sourcePath) ? sourcePath : path.join(repoRoot, sourcePath));
  let currentBuffer = null;
  try {
    currentBuffer = await fs.readFile(targetPath);
  } catch (error) {
    if (!error || error.code !== "ENOENT") {
      throw error;
    }
  }
  if (currentBuffer && Buffer.compare(currentBuffer, sourceBuffer) === 0) {
    return false;
  }
  if (checkOnly) {
    throw new Error(`Generated asset drift detected: ${path.relative(repoRoot, targetPath)}`);
  }
  await fs.mkdir(path.dirname(targetPath), { recursive: true });
  await fs.writeFile(targetPath, sourceBuffer);
  return true;
}

async function main() {
  const checkOnly = process.argv.includes("--check");
  const thisFile = fileURLToPath(import.meta.url);
  await validateSourceImageFills();
  const shellRoot = await readPenRoot(SHARED_SHELL_SOURCE);
  const artboards = new Map();

  for (const spec of artboardSpecs) {
    artboards.set(spec.key, await buildArtboard(spec, shellRoot));
  }
  await validateGeneratedImageFills(artboards);

  const filesToWrite = [];
  for (const spec of artboardSpecs) {
    const artboard = artboards.get(spec.key);
    filesToWrite.push({
      path: path.join(generatedDir, toJsonFileName(spec.key)),
      content: `${JSON.stringify(artboard, null, 2)}\n`,
    });
  }

  filesToWrite.push({
    path: path.join(generatedDir, "artboards.generated.js"),
    content: `${generateArtboardsModule(artboardSpecs)}\n`,
  });
  filesToWrite.push({
    path: path.join(generatedDir, "data-quality.generated.jsx"),
    content: `${generateDataQualityViewModule()}\n`,
  });
  filesToWrite.push({
    path: path.join(generatedDir, "implemented-page-design-manifest.generated.json"),
    content: `${JSON.stringify(generatedManifest(), null, 2)}\n`,
  });

  let changed = false;
  for (const assetSource of stableAssets.keys()) {
    changed = (await syncBinaryAsset(assetSource, checkOnly)) || changed;
  }
  for (const file of filesToWrite) {
    changed = (await syncFile(file.path, file.content, checkOnly)) || changed;
  }

  if (!checkOnly) {
    console.log(changed ? "Implemented-page artboards refreshed." : "Implemented-page artboards already up to date.");
  } else {
    console.log("Implemented-page artboards match committed output.");
  }
}

main().catch((error) => {
  console.error(error?.stack || String(error));
  process.exitCode = 1;
});
