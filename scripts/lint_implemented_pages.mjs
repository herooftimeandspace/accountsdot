import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");
const generatedDir = path.join(repoRoot, "frontend", "src", "generated");
const manifestPath = path.join(generatedDir, "implemented-page-design-manifest.generated.json");
const minimumGapPx = 5;
const maxWarningsPerCheck = 12;

const warningChecks = [
  "fragmented-paragraph",
  "text-divider-gap",
  "bordered-wrapper-gap",
  "text-overflow",
  "table-baseline",
];

function readJSON(absolutePath) {
  return JSON.parse(fs.readFileSync(absolutePath, "utf8"));
}

function readText(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), "utf8");
}

function flattenNodes(root) {
  const nodes = [];

  function walk(node, parent = null) {
    nodes.push({ ...node, parentId: parent?.id ?? null, parent });
    for (const child of node.children || []) {
      walk(child, node);
    }
  }

  walk(root);
  return nodes;
}

function nodeRight(node) {
  return (node.x ?? 0) + (node.width ?? 0);
}

function nodeBottom(node) {
  const fallbackHeight = node.type === "text" ? Math.ceil((node.fontSize ?? 14) * 1.35) : 0;
  return (node.y ?? 0) + (node.height ?? fallbackHeight);
}

function nodeCenterX(node) {
  return (node.x ?? 0) + (node.width ?? 0) / 2;
}

function nodeCenterY(node) {
  return (node.y ?? 0) + (node.height ?? Math.ceil((node.fontSize ?? 14) * 1.35)) / 2;
}

function contains(outer, inner) {
  return (
    (outer.x ?? 0) <= (inner.x ?? 0) &&
    (outer.y ?? 0) <= (inner.y ?? 0) &&
    nodeRight(inner) <= nodeRight(outer) &&
    nodeBottom(inner) <= nodeBottom(outer)
  );
}

function frameContainsPoint(frame, x, y) {
  return (
    frame.type === "frame" &&
    (frame.x ?? 0) <= x &&
    x <= nodeRight(frame) &&
    (frame.y ?? 0) <= y &&
    y <= nodeBottom(frame)
  );
}

function isLoggedInPage(page) {
  return page.loggedInShell && !["login", "error-logged-out"].includes(page.key);
}

function headerRefreshTextNodes(nodes) {
  return nodes.filter(
    (node) =>
      node.type === "text" &&
      String(node.content ?? "").trim() === "Refresh" &&
      (node.y ?? 0) >= 75 &&
      (node.y ?? 0) <= 130 &&
      (node.x ?? 0) >= 1500
  );
}

function findRefreshFrame(nodes, refreshText, primitive) {
  return nodes.find(
    (node) =>
      node.type === "frame" &&
      typeof node.fill === "string" &&
      Math.abs((node.x ?? 0) - primitive.frame.x) <= 8 &&
      Math.abs((node.y ?? 0) - primitive.frame.y) <= 4 &&
      Math.abs((node.width ?? 0) - primitive.frame.width) <= 8 &&
      Math.abs((node.height ?? 0) - primitive.frame.height) <= 6 &&
      frameContainsPoint(node, nodeCenterX(refreshText), nodeCenterY(refreshText))
  );
}

function assertGeneratedOutputCurrent(failures) {
  try {
    execFileSync("node", ["scripts/sync_implemented_pages.mjs", "--check"], {
      cwd: repoRoot,
      encoding: "utf8",
      stdio: "pipe",
    });
  } catch (error) {
    failures.push(
      `generated-output: ${String(error.stdout || error.stderr || error.message).trim()}`
    );
  }
}

function assertManifestExists(failures) {
  if (!fs.existsSync(manifestPath)) {
    failures.push(
      "generated-manifest: missing frontend/src/generated/implemented-page-design-manifest.generated.json; run npm run pen:sync"
    );
  }
}

function assertSharedShell(nodes, page, manifest, failures) {
  if (!isLoggedInPage(page)) {
    return;
  }
  for (const [name, nodeId] of Object.entries(manifest.sharedShell.sharedShellIds)) {
    const count = nodes.filter((node) => node.id === nodeId).length;
    if (count !== 1) {
      failures.push(`${page.key}: shared shell node ${name} (${nodeId}) appears ${count} times`);
    }
  }
}

function assertRoleNavReflowSource(failures) {
  const source = readText("frontend/src/lib/sharedShellPresentation.jsx");
  const required = ["buildVisibleNavGroups(session)", "sidebarRowMetrics(index)", "shared-shell-nav__highlight"];
  for (const token of required) {
    if (!source.includes(token)) {
      failures.push(`shared-shell-role-reflow: missing source token ${token}`);
    }
  }
}

function assertStandardRefresh(nodes, page, manifest, failures) {
  if (!page.standardPrimitives?.includes("refresh")) {
    return;
  }
  const primitive = manifest.standardPrimitives.refresh;
  const refreshTexts = headerRefreshTextNodes(nodes);
  if (refreshTexts.length !== 1) {
    failures.push(`${page.key}: expected exactly one standard header Refresh label, found ${refreshTexts.length}`);
    return;
  }
  const [refreshText] = refreshTexts;
  const refreshFrame = findRefreshFrame(nodes, refreshText, primitive);
  if (!refreshFrame) {
    failures.push(`${page.key}: standard header Refresh frame is missing or off-contract`);
    return;
  }
  const expected = primitive.frame;
  const expectedText = primitive.text;
  if (
    refreshFrame.fill !== expected.fill ||
    refreshFrame.stroke?.fill !== expected.stroke ||
    refreshFrame.cornerRadius !== expected.cornerRadius ||
    refreshText.fill !== expectedText.fill ||
    String(refreshText.fontWeight) !== String(expectedText.fontWeight)
  ) {
    failures.push(`${page.key}: standard header Refresh style drifted from the design contract`);
  }
}

function warnFragmentedParagraphs(nodes, page, warnings) {
  const textNodes = nodes
    .filter((node) => node.type === "text" && String(node.content ?? "").trim().length > 12)
    .sort((left, right) => (left.y ?? 0) - (right.y ?? 0) || (left.x ?? 0) - (right.x ?? 0));
  let emitted = 0;
  for (let index = 0; index < textNodes.length - 1 && emitted < maxWarningsPerCheck; index += 1) {
    const current = textNodes[index];
    const next = textNodes[index + 1];
    const sameColumn = Math.abs((current.x ?? 0) - (next.x ?? 0)) <= 6;
    const similarWidth = Math.abs((current.width ?? 0) - (next.width ?? 0)) <= 24;
    const verticalGap = (next.y ?? 0) - nodeBottom(current);
    const currentLooksSentence = /[a-z0-9,;:]$/i.test(String(current.content ?? "").trim());
    const nextLooksContinuation = /^[a-z(]/.test(String(next.content ?? "").trim());
    if (sameColumn && similarWidth && verticalGap >= -2 && verticalGap <= 8 && currentLooksSentence && nextLooksContinuation) {
      warnings.push(
        `${page.key}: likely fragmented paragraph near ${current.id}/${next.id}; prefer one wrapping text node`
      );
      emitted += 1;
    }
  }
}

function warnTextDividerGap(nodes, page, warnings) {
  const textNodes = nodes.filter((node) => node.type === "text");
  const dividerNodes = nodes.filter(
    (node) =>
      (node.type === "line" || node.type === "frame") &&
      (node.height ?? 0) <= 2 &&
      (node.width ?? 0) >= 80 &&
      (node.stroke?.fill || typeof node.fill === "string")
  );
  let emitted = 0;
  for (const textNode of textNodes) {
    if (emitted >= maxWarningsPerCheck) {
      return;
    }
    const textBottom = nodeBottom(textNode);
    const divider = dividerNodes.find(
      (candidate) =>
        (candidate.y ?? 0) > textBottom &&
        (candidate.y ?? 0) - textBottom < minimumGapPx &&
        nodeRight(textNode) > (candidate.x ?? 0) &&
        (textNode.x ?? 0) < nodeRight(candidate)
    );
    if (divider) {
      warnings.push(
        `${page.key}: text ${textNode.id} is within ${minimumGapPx}px of divider ${divider.id}`
      );
      emitted += 1;
    }
  }
}

function warnBorderedWrapperGap(nodes, page, warnings) {
  const bordered = nodes
    .filter((node) => node.type === "frame" && node.stroke?.fill && (node.width ?? 0) > 20 && (node.height ?? 0) > 20)
    .sort((left, right) => (left.y ?? 0) - (right.y ?? 0) || (left.x ?? 0) - (right.x ?? 0));
  let emitted = 0;
  for (let leftIndex = 0; leftIndex < bordered.length && emitted < maxWarningsPerCheck; leftIndex += 1) {
    for (let rightIndex = leftIndex + 1; rightIndex < bordered.length && emitted < maxWarningsPerCheck; rightIndex += 1) {
      const left = bordered[leftIndex];
      const right = bordered[rightIndex];
      if (left.parentId !== right.parentId || contains(left, right) || contains(right, left)) {
        continue;
      }
      const horizontalOverlap = nodeRight(left) > (right.x ?? 0) && (left.x ?? 0) < nodeRight(right);
      const verticalGap = Math.abs((right.y ?? 0) - nodeBottom(left));
      const verticalOverlap = nodeBottom(left) > (right.y ?? 0) && (left.y ?? 0) < nodeBottom(right);
      const horizontalGap = Math.abs((right.x ?? 0) - nodeRight(left));
      if ((horizontalOverlap && verticalGap > 0 && verticalGap < minimumGapPx) || (verticalOverlap && horizontalGap > 0 && horizontalGap < minimumGapPx)) {
        warnings.push(`${page.key}: bordered wrappers ${left.id}/${right.id} have less than ${minimumGapPx}px buffer`);
        emitted += 1;
      }
    }
  }
}

function warnTextOverflow(nodes, page, artboard, warnings) {
  let emitted = 0;
  for (const node of nodes.filter((candidate) => candidate.type === "text")) {
    if (emitted >= maxWarningsPerCheck) {
      return;
    }
    if ((node.x ?? 0) < 0 || (node.y ?? 0) < 0 || nodeRight(node) > artboard.width || nodeBottom(node) > artboard.height) {
      warnings.push(`${page.key}: text ${node.id} falls outside the artboard bounds`);
      emitted += 1;
    }
    if (node.parent && node.parent.type === "frame" && !contains(node.parent, node)) {
      warnings.push(`${page.key}: text ${node.id} may overflow parent frame ${node.parent.id}`);
      emitted += 1;
    }
  }
}

function warnTableBaseline(nodes, page, warnings) {
  const textNodes = nodes.filter((node) => node.type === "text" && (node.x ?? 0) > 260 && (node.y ?? 0) > 120);
  const buckets = new Map();
  for (const node of textNodes) {
    const yBucket = Math.round((node.y ?? 0) / 24) * 24;
    if (!buckets.has(yBucket)) {
      buckets.set(yBucket, []);
    }
    buckets.get(yBucket).push(node);
  }
  let emitted = 0;
  for (const [bucket, row] of buckets) {
    if (emitted >= maxWarningsPerCheck) {
      return;
    }
    if (row.length < 3) {
      continue;
    }
    const minY = Math.min(...row.map((node) => node.y ?? 0));
    const maxY = Math.max(...row.map((node) => node.y ?? 0));
    if (maxY - minY > 5) {
      warnings.push(`${page.key}: possible table baseline drift near y=${bucket}; row text spans ${maxY - minY}px`);
      emitted += 1;
    }
  }
}

function runArtboardWarnings(nodes, page, artboard, warnings) {
  warnFragmentedParagraphs(nodes, page, warnings);
  warnTextDividerGap(nodes, page, warnings);
  warnBorderedWrapperGap(nodes, page, warnings);
  warnTextOverflow(nodes, page, artboard, warnings);
  warnTableBaseline(nodes, page, warnings);
}

function loadManifest() {
  return readJSON(manifestPath);
}

function runLint({ includeDriftCheck = true } = {}) {
  const failures = [];
  const warnings = [];

  if (includeDriftCheck) {
    assertGeneratedOutputCurrent(failures);
  }
  assertManifestExists(failures);
  if (!fs.existsSync(manifestPath)) {
    return { failures, warnings };
  }

  const manifest = loadManifest();
  assertRoleNavReflowSource(failures);

  for (const page of manifest.artboards) {
    const artboardPath = path.join(generatedDir, `${page.key}.artboard.json`);
    if (!fs.existsSync(artboardPath)) {
      failures.push(`${page.key}: missing generated artboard JSON`);
      continue;
    }
    const artboard = readJSON(artboardPath);
    const nodes = flattenNodes(artboard);
    assertSharedShell(nodes, page, manifest, failures);
    assertStandardRefresh(nodes, page, manifest, failures);
    runArtboardWarnings(nodes, page, artboard, warnings);
  }

  return { failures, warnings };
}

function selfTest() {
  const manifest = {
    sharedShell: { sharedShellIds: {} },
    standardPrimitives: {
      refresh: {
        frame: { x: 1540, y: 90, width: 112, height: 38, fill: "#CEB770", stroke: "#CEB770", cornerRadius: 8 },
        text: { fill: "#01161E", fontWeight: "700", textAlign: "center" },
      },
    },
  };
  const page = { key: "fixture", standardPrimitives: ["refresh"], loggedInShell: true };
  const passingNodes = [
    { type: "frame", id: "f1", x: 1540, y: 90, width: 112, height: 38, fill: "#CEB770", cornerRadius: 8, stroke: { fill: "#CEB770" } },
    { type: "text", id: "t1", content: "Refresh", x: 1574, y: 101, width: 50, fill: "#01161E", fontWeight: "700", textAlign: "center" },
  ];
  const passingFailures = [];
  assertStandardRefresh(passingNodes, page, manifest, passingFailures);
  if (passingFailures.length > 0) {
    throw new Error(`self-test expected standard refresh to pass: ${passingFailures.join("; ")}`);
  }

  const failingFailures = [];
  assertStandardRefresh([{ ...passingNodes[1], x: 1200 }], page, manifest, failingFailures);
  if (failingFailures.length === 0) {
    throw new Error("self-test expected missing standard refresh to fail");
  }

  const paragraphWarnings = [];
  warnFragmentedParagraphs(
    [
      { type: "text", id: "t2", content: "This helper sentence continues", x: 10, y: 10, width: 200, fontSize: 14 },
      { type: "text", id: "t3", content: "onto another fragment.", x: 10, y: 31, width: 200, fontSize: 14 },
    ],
    { key: "fixture" },
    paragraphWarnings
  );
  if (paragraphWarnings.length === 0) {
    throw new Error("self-test expected fragmented paragraph warning");
  }
}

function printResults({ failures, warnings }) {
  if (warnings.length > 0) {
    console.log("Implemented-page design lint warnings:");
    for (const warning of warnings) {
      console.log(`- [warn] ${warning}`);
    }
    console.log(`Warning checks are non-blocking in phase 1: ${warningChecks.join(", ")}.`);
  } else {
    console.log("Implemented-page design lint warnings: none.");
  }

  if (failures.length > 0) {
    console.error("Implemented-page design lint failures:");
    for (const failure of failures) {
      console.error(`- [fail] ${failure}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log("Implemented-page design lint passed high-confidence checks.");
}

const isCli = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;

if (isCli) {
  if (process.argv.includes("--self-test")) {
    selfTest();
    console.log("Implemented-page design lint self-tests passed.");
  } else {
    printResults(runLint());
  }
}

export {
  assertStandardRefresh,
  flattenNodes,
  runLint,
  warnFragmentedParagraphs,
  warnTextDividerGap,
};
