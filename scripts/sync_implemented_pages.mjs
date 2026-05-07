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

const artboardSpecs = [
  {
    key: "data-quality",
    source: "docs/mocks/wireframes/wireframe-data-quality-dashboard.pen",
    mode: "passthrough",
  },
  {
    key: "dashboard-it-admin",
    source: "docs/mocks/wireframes/wireframe-it-admin-overview.pen",
    mode: "merge-shell",
  },
  {
    key: "dashboard-hr-lifecycle",
    source: "docs/mocks/wireframes/wireframe-hr-lifecycle-overview.pen",
    mode: "merge-shell",
  },
  {
    key: "dashboard-site-admin",
    source: "docs/mocks/wireframes/wireframe-site-admin-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "onboarding",
    source: "docs/mocks/wireframes/wireframe-onboarding-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "offboarding",
    source: "docs/mocks/wireframes/wireframe-offboarding-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "room-moves",
    source: "docs/mocks/wireframes/wireframe-room-moves.pen",
    mode: "merge-shell",
  },
  {
    key: "phone-directory-by-person",
    source: "docs/mocks/wireframes/wireframe-phone-directory-by-person.pen",
    mode: "merge-shell",
  },
  {
    key: "phone-directory-by-room",
    source: "docs/mocks/wireframes/wireframe-phone-directory-by-room.pen",
    mode: "merge-shell",
  },
  {
    key: "phone-directory-by-department",
    source: "docs/mocks/wireframes/wireframe-phone-directory-by-department.pen",
    mode: "merge-shell",
  },
  {
    key: "frequent-fliers",
    source: "docs/mocks/wireframes/wireframe-device-wrangler-frequent-fliers.pen",
    mode: "merge-shell",
  },
  {
    key: "student-data-cleanup",
    source: "docs/mocks/wireframes/wireframe-site-secretary-student-data-cleanup.pen",
    mode: "merge-shell",
  },
  {
    key: "reports",
    source: "docs/mocks/wireframes/wireframe-it-admin-reports.pen",
    mode: "merge-shell",
  },
  {
    key: "reports-sync-transparency",
    source: "docs/mocks/wireframes/wireframe-sync-transparency-dashboard.pen",
    mode: "merge-shell",
  },
  {
    key: "reports-ticketing-human-work",
    source: "docs/mocks/wireframes/wireframe-ticketing-human-work.pen",
    mode: "merge-shell",
  },
  {
    key: "admin",
    source: "docs/mocks/wireframes/wireframe-it-admin-admin-controls.pen",
    mode: "merge-shell",
  },
  {
    key: "my-profile",
    source: "docs/mocks/wireframes/wireframe-faculty-staff-my-profile.pen",
    mode: "merge-shell",
  },
  {
    key: "login",
    source: "docs/mocks/wireframes/wireframe-login.pen",
    mode: "passthrough",
  },
  {
    key: "error-logged-in",
    source: "docs/mocks/wireframes/wireframe-http-error.pen",
    mode: "error-logged-in",
  },
  {
    key: "error-logged-out",
    source: "docs/mocks/wireframes/wireframe-http-error.pen",
    mode: "error-logged-out",
  },
];

const stableAssets = new Map([
  [
    path.join(repoRoot, "docs", "reference-inputs", "branding", "Firefly.png"),
    "firefly.png",
  ],
  [
    path.join(repoRoot, "docs", "reference-inputs", "branding", "google-g.svg"),
    "google-g.svg",
  ],
  [
    path.join(repoRoot, "docs", "mocks", "wireframes", "varsity_regular.ttf"),
    "varsity_regular.ttf",
  ],
]);

const dataQualityDesign = {
  key: "data-quality",
  width: 1672,
  height: 1080,
  hotspots: {
    refresh: "f104",
    openMappingDashboard: "f183",
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
      description: "t102",
      lastRefreshed: "t103",
      refreshLabel: "t105",
    },
    summaryCards: [
      { title: "t107", count: "t108" },
      { title: "t111", count: "t112" },
      { title: "t115", count: "t116" },
      { title: "t119", count: "t120" },
    ],
    routingCard: {
      title: "t123",
      headline: "t124",
      body: "t125",
    },
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
    routingRules: {
      title: "t170",
      rows: [
        { queue: "t173", description: "t174" },
        { queue: "t176", description: "t177" },
        { queue: "t179", description: "t180" },
      ],
      primaryActionLabel: "t184",
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
    admin: ["p86", "p87", "p88", "p89", "t90", "p91"],
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
};

const assetCache = new Map();

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
    const panePrefix = `${spec.key}__`;
    for (const child of shellRoot.children.filter(isShellNode)) {
      children.push(await normalizeNode(child, 0, 0));
    }
    for (const child of sourceRoot.children.filter(isPaneNode)) {
      children.push(await normalizeNode(child, 0, 0, panePrefix));
    }

    const root = await normalizeNode(shellRoot, 0, 0);
    root.name = sourceRoot.name || spec.key;
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
  const imports = specs
    .map((spec, index) => `import artboard${index} from "./${toJsonFileName(spec.key)}";`)
    .join("\n");

  const entries = specs
    .map((spec, index) => `  "${spec.key}": artboard${index},`)
    .join("\n");

  const metadata = specs
    .map(
      (spec) =>
        `  "${spec.key}": { key: "${spec.key}", sourcePen: "${spec.source}", activeNav: ${JSON.stringify(
          activeNavByArtboardKey[spec.key] ?? null
        )} },`
    )
    .join("\n");

  return `${imports}

export const generatedArtboards = {
${entries}
};

export const generatedArtboardMeta = {
${metadata}
};

export const sharedShellSpec = ${JSON.stringify(shellSpec, null, 2)};
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
  const shellRoot = await readPenRoot("docs/mocks/wireframes/wireframe-data-quality-dashboard.pen");
  const artboards = new Map();

  for (const spec of artboardSpecs) {
    artboards.set(spec.key, await buildArtboard(spec, shellRoot));
  }

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
