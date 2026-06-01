import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const inventoryPath = path.join(repoRoot, "docs", "planning", "route-api-authorization-inventory.md");
const appGoPath = path.join(repoRoot, "internal", "web", "app.go");
const routeRegistryPath = path.join(repoRoot, "frontend", "src", "lib", "routeRegistry.js");

const routeInventory = [
  { route: "/login", apiPaths: [] },
  { route: "/dashboard", apiPaths: [] },
  { route: "/dashboard/it-admin", apiPaths: [] },
  { route: "/dashboard/hr-lifecycle", apiPaths: [] },
  { route: "/dashboard/site-admin", apiPaths: [] },
  { route: "/search", apiPaths: ["/api/v1/dev/search"] },
  {
    route: "/onboarding",
    apiPaths: [
      "/api/v1/dev/pages/onboarding",
      "/api/v1/dev/onboarding/manual-drafts",
      "/api/v1/dev/onboarding/manual-drafts/",
    ],
  },
  {
    route: "/offboarding",
    apiPaths: [
      "/api/v1/dev/pages/offboarding",
      "/api/v1/dev/offboarding/records/",
      "/api/v1/dev/offboarding/candidates",
      "/api/v1/dev/offboarding/emergency-deprovision",
      "/api/v1/dev/offboarding/contractor-offboarding",
    ],
  },
  {
    route: "/departing-seniors",
    apiPaths: [
      "/api/v1/dev/pages/departing-seniors",
      "/api/v1/dev/departing-seniors/records/",
    ],
  },
  {
    route: "/room-moves",
    apiPaths: [
      "/api/v1/dev/pages/room-moves",
      "/api/v1/dev/room-moves/drafts",
      "/api/v1/dev/room-moves/drafts/",
      "/api/v1/dev/room-moves/completed",
      "/api/v1/dev/room-moves/completed/",
    ],
  },
  {
    route: "/room-moves/bulk-draft",
    apiPaths: [
      "/api/v1/dev/pages/room-moves/bulk-draft",
      "/api/v1/dev/room-moves/drafts",
      "/api/v1/dev/room-moves/drafts/",
    ],
  },
  { route: "/phone-directory/by-person", apiPaths: ["/api/v1/dev/pages/phone-directory/by-person"] },
  { route: "/phone-directory/by-room", apiPaths: ["/api/v1/dev/pages/phone-directory/by-room"] },
  { route: "/phone-directory/by-department", apiPaths: ["/api/v1/dev/pages/phone-directory/by-department"] },
  { route: "/data-quality", apiPaths: ["/api/v1/dev/pages/data-quality"] },
  { route: "/frequent-fliers", apiPaths: [] },
  { route: "/meraki-last-seen", apiPaths: ["/api/v1/dev/pages/meraki-last-seen"] },
  { route: "/student-data-cleanup", apiPaths: [] },
  { route: "/reports", apiPaths: [] },
  { route: "/reports/security-issues", apiPaths: ["/api/v1/dev/pages/reports/security-issues"] },
  { route: "/reports/zoom-desk-phone-renames", apiPaths: ["/api/v1/dev/pages/reports/zoom-desk-phone-renames"] },
  { route: "/reports/sync-transparency", apiPaths: [] },
  { route: "/admin", apiPaths: [] },
  { route: "/admin/auth-settings", apiPaths: [] },
  { route: "/admin/feature-flags", apiPaths: ["/api/v1/dev/feature-flags", "/api/v1/dev/feature-flags/"] },
  { route: "/my-profile", apiPaths: ["/api/v1/dev/my-profile"] },
];

const fail = (message) => {
  console.error(message);
  process.exitCode = 1;
};

const routeRegistry = fs.readFileSync(routeRegistryPath, "utf8");
const appRoutesMatch = routeRegistry.match(/export const APP_ROUTES = \[([\s\S]*?)\];/);
if (!appRoutesMatch) {
  console.error("could not locate APP_ROUTES in frontend/src/lib/routeRegistry.js");
  process.exit(1);
}
const routeRegistryPaths = [...appRoutesMatch[1].matchAll(/path:\s*"([^"]+)"/g)].map((match) => match[1]).sort();
const inventoryRoutes = routeInventory.map((row) => row.route).sort();
const missingFromInventory = routeRegistryPaths.filter((route) => !inventoryRoutes.includes(route));
const staleInInventory = inventoryRoutes.filter((route) => !routeRegistryPaths.includes(route));

if (missingFromInventory.length > 0) {
  fail(`route inventory missing frontend routes: ${missingFromInventory.join(", ")}`);
}
if (staleInInventory.length > 0) {
  fail(`route inventory contains routes not in routeRegistry.js: ${staleInInventory.join(", ")}`);
}

const appGo = fs.readFileSync(appGoPath, "utf8");
const registeredAPIPaths = new Set(
  [...appGo.matchAll(/mux\.Handle\("([^"]+)"/g), ...appGo.matchAll(/mux\.HandleFunc\("([^"]+)"/g)]
    .map((match) => match[1])
    .filter((registeredPath) => registeredPath.startsWith("/api/v1/dev/")),
);

for (const row of routeInventory) {
  for (const apiPath of row.apiPaths) {
    if (!registeredAPIPaths.has(apiPath)) {
      fail(`${row.route} references unregistered DEV API path ${apiPath}`);
    }
  }
}

const inventoryDoc = fs.readFileSync(inventoryPath, "utf8");
for (const apiPath of registeredAPIPaths) {
  if (!inventoryDoc.includes(`\`${apiPath}\``)) {
    fail(`docs/planning/route-api-authorization-inventory.md missing registered DEV API path ${apiPath}`);
  }
}
for (const row of routeInventory) {
  if (!inventoryDoc.includes(`| \`${row.route}\` |`)) {
    fail(`docs/planning/route-api-authorization-inventory.md missing table row for ${row.route}`);
  }
  for (const apiPath of row.apiPaths) {
    if (!inventoryDoc.includes(`\`${apiPath}\``)) {
      fail(`docs/planning/route-api-authorization-inventory.md missing API path ${apiPath}`);
    }
  }
}

if (!process.exitCode) {
  console.log(`Route/API authorization inventory covers ${routeInventory.length} frontend routes and ${registeredAPIPaths.size} DEV API mux registrations.`);
}
