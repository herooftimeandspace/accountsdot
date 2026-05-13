#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");
const inventoryPath = path.join(repoRoot, "docs", "external-write-inventory.md");

const mutatingMethods = new Set(["POST", "PUT", "DELETE", "PATCH"]);

const routeInventory = [
  {
    method: "POST",
    path: "/api/v1/workflows/{workflow_run_id}/retry",
    source: "internal/web/app.go",
    owner: "workflow retry",
  },
  {
    method: "POST",
    path: "/api/v1/approvals/{approval_id}/approve",
    source: "internal/web/app.go",
    owner: "approval decision",
  },
  {
    method: "POST",
    path: "/api/v1/approvals/{approval_id}/reject",
    source: "internal/web/app.go",
    owner: "approval decision",
  },
  {
    method: "POST",
    path: "/api/v1/sync-status/{user_type}/{user_id}/override",
    source: "internal/web/app.go",
    owner: "sync override",
  },
  {
    method: "POST",
    path: "/api/v1/room-mappings",
    source: "internal/web/app.go",
    owner: "room mappings",
  },
  {
    method: "POST",
    path: "/api/v1/annual-reset",
    source: "internal/web/app.go",
    owner: "annual reset",
  },
  {
    method: "POST",
    path: "/api/v1/dev/login",
    source: "internal/web/dev_frontend.go",
    owner: "DEV session mock",
  },
  {
    method: "POST",
    path: "/api/v1/dev/logout",
    source: "internal/web/dev_frontend.go",
    owner: "DEV session mock",
  },
  {
    method: "PUT",
    path: "/api/v1/dev/feature-flags/{key}",
    source: "internal/web/dev_frontend.go",
    owner: "DEV feature flags",
  },
  {
    method: "POST",
    path: "/api/v1/dev/onboarding/manual-drafts",
    source: "internal/web/dev_onboarding.go",
    owner: "DEV drafts",
  },
  {
    method: "PUT",
    path: "/api/v1/dev/onboarding/manual-drafts/{id}",
    source: "internal/web/dev_onboarding.go",
    owner: "DEV drafts",
  },
  {
    method: "POST",
    path: "/api/v1/dev/onboarding/manual-drafts/{id}/finalize",
    source: "internal/web/dev_onboarding.go",
    owner: "DEV drafts",
  },
  {
    method: "DELETE",
    path: "/api/v1/dev/onboarding/manual-drafts/{id}",
    source: "internal/web/dev_onboarding.go",
    owner: "DEV drafts",
  },
  {
    method: "PUT",
    path: "/api/v1/dev/offboarding/records/{id}/end-date",
    source: "internal/web/dev_offboarding.go",
    owner: "offboarding",
  },
  {
    method: "PUT",
    path: "/api/v1/dev/departing-seniors/records/{id}/end-date",
    source: "internal/web/dev_departing_seniors.go",
    owner: "departing seniors",
  },
  {
    method: "POST",
    path: "/api/v1/dev/departing-seniors/records/{id}/deprovision",
    source: "internal/web/dev_departing_seniors.go",
    owner: "departing seniors",
  },
  {
    method: "POST",
    path: "/api/v1/dev/room-moves/drafts",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
  {
    method: "PUT",
    path: "/api/v1/dev/room-moves/drafts/{id}",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
  {
    method: "POST",
    path: "/api/v1/dev/room-moves/drafts/{id}/cancel",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
  {
    method: "POST",
    path: "/api/v1/dev/room-moves/drafts/{id}/schedule",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
  {
    method: "POST",
    path: "/api/v1/dev/room-moves/drafts/{id}/apply",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
  {
    method: "DELETE",
    path: "/api/v1/dev/room-moves/drafts/{id}",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
  {
    method: "POST",
    path: "/api/v1/dev/room-moves/completed/{id}/revert",
    source: "internal/web/dev_room_moves.go",
    owner: "room moves",
  },
];

function routeKey(route) {
  return `${route.method} ${route.path}`;
}

function parseInventoryRoutes(markdown) {
  const routePattern = /(?:^|\n)-\s+`(POST|PUT|DELETE|PATCH)\s+([^`]+)`/g;
  const exceptionPattern =
    /<!--\s*external-write-inventory-exception:\s*(POST|PUT|DELETE|PATCH)\s+([^>]+?)\s*-->/g;
  const routes = new Set();
  const exceptions = new Set();
  let match;

  while ((match = routePattern.exec(markdown)) !== null) {
    routes.add(`${match[1]} ${match[2].trim()}`);
  }

  while ((match = exceptionPattern.exec(markdown)) !== null) {
    exceptions.add(`${match[1]} ${match[2].trim()}`);
  }

  return { routes, exceptions };
}

function findDuplicateRoutes(routes) {
  const seen = new Set();
  const duplicates = new Set();
  for (const route of routes) {
    const key = routeKey(route);
    if (seen.has(key)) {
      duplicates.add(key);
    }
    seen.add(key);
  }
  return [...duplicates].sort();
}

function checkInventory(markdown, routes = routeInventory) {
  const { routes: documentedRoutes, exceptions } = parseInventoryRoutes(markdown);
  const expected = new Map(routes.map((route) => [routeKey(route), route]));
  const expectedKeys = new Set(expected.keys());
  const failures = [];
  const warnings = [];

  for (const route of routes) {
    if (!mutatingMethods.has(route.method)) {
      failures.push(`${routeKey(route)} uses unsupported method ${route.method}`);
      continue;
    }
    const key = routeKey(route);
    if (!documentedRoutes.has(key) && !exceptions.has(key)) {
      failures.push(`${key} (${route.owner}, ${route.source}) is missing from docs/external-write-inventory.md`);
    }
  }

  for (const documented of documentedRoutes) {
    if (!expectedKeys.has(documented)) {
      warnings.push(`${documented} is documented but not enumerated in scripts/check_external_write_inventory.mjs`);
    }
  }

  for (const exception of exceptions) {
    if (!expectedKeys.has(exception)) {
      warnings.push(`${exception} has an exception comment but is not enumerated in scripts/check_external_write_inventory.mjs`);
    }
  }

  for (const duplicate of findDuplicateRoutes(routes)) {
    failures.push(`${duplicate} is duplicated in scripts/check_external_write_inventory.mjs`);
  }

  return { failures, warnings };
}

function runSelfTest() {
  const sample = [
    "- `POST /api/example`",
    "- `PATCH /api/future`",
    "<!-- external-write-inventory-exception: DELETE /api/mock-only -->",
  ].join("\n");
  const { routes, exceptions } = parseInventoryRoutes(sample);
  const assertions = [
    [routes.has("POST /api/example"), "parses POST route bullets"],
    [routes.has("PATCH /api/future"), "parses future PATCH route bullets"],
    [exceptions.has("DELETE /api/mock-only"), "parses documented exception comments"],
  ];

  const failureCheck = checkInventory("- `POST /api/example`", [
    { method: "POST", path: "/api/example", source: "test", owner: "test" },
    { method: "PATCH", path: "/api/future", source: "test", owner: "test" },
  ]);
  assertions.push([
    failureCheck.failures.some((failure) => failure.includes("PATCH /api/future")),
    "fails when a mutating route is not documented",
  ]);

  const exceptionCheck = checkInventory("<!-- external-write-inventory-exception: PATCH /api/future -->", [
    { method: "PATCH", path: "/api/future", source: "test", owner: "test" },
  ]);
  assertions.push([
    exceptionCheck.failures.length === 0,
    "accepts documented no-op/mock exceptions",
  ]);

  const failed = assertions.filter(([passed]) => !passed).map(([, message]) => message);
  if (failed.length > 0) {
    throw new Error(`Self-test failed:\n- ${failed.join("\n- ")}`);
  }

  console.log("External write inventory self-test passed.");
}

function main() {
  if (process.argv.includes("--self-test")) {
    runSelfTest();
    return;
  }

  const markdown = fs.readFileSync(inventoryPath, "utf8");
  const { failures, warnings } = checkInventory(markdown);

  for (const warning of warnings) {
    console.warn(`warning: ${warning}`);
  }

  if (failures.length > 0) {
    console.error("External write inventory drift detected:");
    for (const failure of failures) {
      console.error(`- ${failure}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log(`External write inventory check passed for ${routeInventory.length} mutating routes.`);
}

main();
