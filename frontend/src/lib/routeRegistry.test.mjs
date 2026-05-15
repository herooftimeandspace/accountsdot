import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";

const routeRegistryUrl = new URL("./routeRegistry.js", import.meta.url);
const routeRegistrySource = fs.readFileSync(routeRegistryUrl, "utf8");
const routeRegistryModule = await import(
  `data:text/javascript;base64,${Buffer.from(routeRegistrySource).toString("base64")}`
);

test("artboardKeysForAllowedRoutes only returns artboards for the active session routes", () => {
  const { artboardKeysForAllowedRoutes } = routeRegistryModule;

  const keys = artboardKeysForAllowedRoutes({
    allowed_routes: [
      "/my-profile",
      "/search",
      "/student-data-cleanup",
      "/room-moves",
      "/room-moves/bulk-draft",
      "/phone-directory/by-person",
      "/phone-directory/by-room",
      "/phone-directory/by-department",
    ],
  });

  assert.deepEqual(keys.sort(), [
    "my-profile",
    "phone-directory-by-department",
    "phone-directory-by-person",
    "phone-directory-by-room",
    "room-moves",
    "room-moves-bulk-draft",
    "student-data-cleanup",
  ]);
  assert.equal(keys.includes("admin-feature-flags"), false);
  assert.equal(keys.includes("reports-sync-transparency"), false);
  assert.equal(keys.includes("dashboard-it-admin"), false);
  assert.equal(keys.includes("frequent-fliers"), false);
});

test("artboardKeysForAllowedRoutes ignores public, redirect, and unknown routes without artboards", () => {
  const { artboardKeysForAllowedRoutes } = routeRegistryModule;

  assert.deepEqual(
    artboardKeysForAllowedRoutes({
      allowed_routes: ["/login", "/dashboard", "/error/403", "/search", "/not-a-real-route"],
    }),
    []
  );
});

test("artboardKeysForAllowedRoutes deduplicates shared artboards across allowed routes", () => {
  const { artboardKeysForAllowedRoutes } = routeRegistryModule;

  assert.deepEqual(
    artboardKeysForAllowedRoutes({
      allowed_routes: ["/offboarding", "/departing-seniors", "/reports", "/reports/security-issues", "/reports"],
    }).sort(),
    ["offboarding", "reports"]
  );
});

test("visibleNavChildrenForKey returns only documented nested routes allowed for the session", () => {
  const { visibleNavChildrenForKey } = routeRegistryModule;

  assert.deepEqual(
    visibleNavChildrenForKey("admin", {
      allowed_routes: ["/admin/feature-flags"],
    }),
    [{ path: "/admin/feature-flags", label: "Feature Flags" }]
  );
  assert.deepEqual(
    visibleNavChildrenForKey("reports", {
      allowed_routes: ["/reports", "/reports/security-issues"],
    }),
    [{ path: "/reports/security-issues", label: "Security Issues" }]
  );
  assert.deepEqual(
    visibleNavChildrenForKey("phoneDirectory", {
      allowed_routes: ["/phone-directory/by-person", "/phone-directory/by-room"],
    }),
    [
      { path: "/phone-directory/by-person", label: "By Person" },
      { path: "/phone-directory/by-room", label: "By Room" },
    ]
  );
  assert.deepEqual(
    visibleNavChildrenForKey("admin", {
      allowed_routes: ["/admin"],
    }),
    []
  );
});
