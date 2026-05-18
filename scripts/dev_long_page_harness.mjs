#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { LONG_PAGE_ROUTE_HARNESS } from "../frontend/src/lib/devLongPageHarness.mjs";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const routeRegistryPath = path.join(repoRoot, "frontend/src/lib/routeRegistry.js");

export function routesThatNeedLongPageHarness() {
  const source = fs.readFileSync(routeRegistryPath, "utf8");
  const routes = [...source.matchAll(/\{\s*path:\s*"([^"]+)"([^}]*)\}/g)].map((match) => ({
    path: match[1],
    source: match[0],
  }));
  return routes
    .filter((route) => !route.source.includes("public: true") && !route.source.includes('kind: "dashboard-redirect"'))
    .map((route) => route.path);
}

export function validateLongPageHarnessCoverage() {
  const expected = new Set(routesThatNeedLongPageHarness());
  const covered = new Set(LONG_PAGE_ROUTE_HARNESS.map((entry) => entry.path));
  const missing = [...expected].filter((path) => !covered.has(path));
  const stale = [...covered].filter((path) => !expected.has(path));
  const invalid = LONG_PAGE_ROUTE_HARNESS.filter((entry) => !Array.isArray(entry.selectors) || !entry.fallback);
  return { missing, stale, invalid };
}

function browserHarnessUrls(baseUrl) {
  return LONG_PAGE_ROUTE_HARNESS.map((entry) => `${baseUrl}${entry.path}?longPageHarness=1`);
}

function printUsage() {
  console.log("Usage:");
  console.log("  node scripts/dev_long_page_harness.mjs --self-test");
  console.log("  node scripts/dev_long_page_harness.mjs --print-browser-urls [base-url]");
}

if (import.meta.url === `file://${process.argv[1]}`) {
  if (process.argv.includes("--self-test")) {
    const result = validateLongPageHarnessCoverage();
    if (result.missing.length || result.stale.length || result.invalid.length) {
      console.error(JSON.stringify(result, null, 2));
      process.exit(1);
    }
    console.log(`long-page harness covers ${LONG_PAGE_ROUTE_HARNESS.length} implemented routes`);
  } else if (process.argv.includes("--print-browser-urls")) {
    const flagIndex = process.argv.indexOf("--print-browser-urls");
    const baseUrl = process.argv[flagIndex + 1] || "http://127.0.0.1:5173";
    console.log(browserHarnessUrls(baseUrl).join("\n"));
  } else {
    printUsage();
  }
}
