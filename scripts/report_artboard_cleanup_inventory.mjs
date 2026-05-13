import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");
const manifestPath = path.join(repoRoot, "frontend", "src", "generated", "implemented-page-design-manifest.generated.json");
const pagesDir = path.join(repoRoot, "frontend", "src", "pages");
const LOGGED_IN_PANE_X = 264;
const LOGGED_IN_PANE_Y = 76;

function readJSON(relativeOrAbsolutePath) {
  const target = path.isAbsolute(relativeOrAbsolutePath)
    ? relativeOrAbsolutePath
    : path.join(repoRoot, relativeOrAbsolutePath);
  return JSON.parse(fs.readFileSync(target, "utf8"));
}

function flattenNodes(root) {
  const nodes = [];
  function visit(node, parent = null) {
    nodes.push({ ...node, parentId: parent?.id ?? null });
    for (const child of node.children ?? []) {
      visit(child, node);
    }
  }
  visit(root);
  return nodes;
}

function collectImageFills(root) {
  return flattenNodes(root)
    .filter((node) => node.fill && typeof node.fill === "object" && node.fill.type === "image" && node.fill.url)
    .map((node) => ({ id: node.id, url: node.fill.url }));
}

function runtimeHiddenDebt() {
  return fs.readdirSync(pagesDir)
    .filter((file) => file.endsWith(".jsx"))
    .sort()
    .map((file) => {
      const source = fs.readFileSync(path.join(pagesDir, file), "utf8");
      const hiddenConstants = [...new Set(source.match(/\b(?:HIDDEN|STATIC)_[A-Z0-9_]*NODE[A-Z0-9_]*\b/g) ?? [])];
      return {
        file: path.join("frontend/src/pages", file),
        hiddenConstants,
        hiddenPushes: (source.match(/hiddenNodeIds\.push/g) ?? []).length,
        paneCollectors: (source.match(/collect(?:Generated)?PaneNodeIds|collectPaneNodeIds/g) ?? []).length,
      };
    })
    .filter((entry) => entry.hiddenConstants.length || entry.hiddenPushes || entry.paneCollectors);
}

function sourcePenInventory(manifest) {
  return manifest.artboards
    .filter((page) => page.sourcePen)
    .map((page) => {
      const root = readJSON(page.sourcePen).children?.[0];
      const nodes = root ? flattenNodes(root) : [];
      const shellRegionNodeCount = page.mode === "merge-shell"
        ? nodes.filter((node) => node.id !== root.id && ((node.x ?? 0) < LOGGED_IN_PANE_X || (node.y ?? 0) < LOGGED_IN_PANE_Y)).length
        : 0;
      return {
        key: page.key,
        sourcePen: page.sourcePen,
        mode: page.mode,
        standardPrimitives: page.standardPrimitives ?? [],
        sourceShellRegionNodesIgnoredByMerge: shellRegionNodeCount,
        imageFills: root ? collectImageFills(root) : [],
      };
    });
}

function renderMarkdown({ hiddenDebt, penInventory }) {
  const lines = [
    "# Artboard Cleanup Inventory",
    "",
    "## Runtime Hidden Node Debt",
    "",
    "| Page source | Hidden constants | hiddenNodeIds.push | Pane collectors |",
    "| --- | ---: | ---: | ---: |",
    ...hiddenDebt.map((entry) =>
      `| \`${entry.file}\` | ${entry.hiddenConstants.length} | ${entry.hiddenPushes} | ${entry.paneCollectors} |`
    ),
    "",
    "## Source PEN Shell Overlap",
    "",
    "| Artboard | Source PEN | Ignored shell-region nodes | Standard primitives |",
    "| --- | --- | ---: | --- |",
    ...penInventory.map((entry) =>
      `| \`${entry.key}\` | \`${entry.sourcePen}\` | ${entry.sourceShellRegionNodesIgnoredByMerge} | ${entry.standardPrimitives.join(", ") || "none"} |`
    ),
    "",
    "## Source Image Fills",
    "",
    "| Artboard | Image fill URLs |",
    "| --- | --- |",
    ...penInventory.map((entry) =>
      `| \`${entry.key}\` | ${entry.imageFills.map((image) => `\`${image.url}\``).join("<br>") || "none"} |`
    ),
  ];
  return `${lines.join("\n")}\n`;
}

function main() {
  const manifest = readJSON(manifestPath);
  const report = {
    generatedAt: new Date().toISOString(),
    hiddenDebt: runtimeHiddenDebt(),
    penInventory: sourcePenInventory(manifest),
  };
  if (process.argv.includes("--json")) {
    console.log(JSON.stringify(report, null, 2));
    return;
  }
  console.log(renderMarkdown(report));
}

main();
