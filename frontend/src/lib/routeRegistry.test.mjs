import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";

const routeRegistryUrl = new URL("./routeRegistry.js", import.meta.url);
const routeRegistrySource = fs.readFileSync(routeRegistryUrl, "utf8");
const routeRegistryModule = await import(
  `data:text/javascript;base64,${Buffer.from(routeRegistrySource).toString("base64")}`
);
const routeHelpContentUrl = new URL("./routeHelpContent.js", import.meta.url);
const routeHelpContentSource = fs.readFileSync(routeHelpContentUrl, "utf8");
const routeHelpContentModule = await import(
  `data:text/javascript;base64,${Buffer.from(routeHelpContentSource).toString("base64")}`
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
      allowed_routes: [
        "/offboarding",
        "/departing-seniors",
        "/reports",
        "/reports/security-issues",
        "/reports/zoom-desk-phone-renames",
        "/reports",
      ],
    }).sort(),
    ["offboarding", "reports"]
  );
});

test("visibleNavChildrenForKey returns only documented nested routes allowed for the session", () => {
  const { visibleNavChildrenForKey } = routeRegistryModule;

  assert.deepEqual(
    visibleNavChildrenForKey("phoneDirectory", {
      allowed_routes: ["/phone-directory/by-person", "/phone-directory/by-room", "/phone-directory/by-department"],
    }),
    []
  );
  assert.deepEqual(
    visibleNavChildrenForKey("admin", {
      allowed_routes: ["/admin/feature-flags"],
    }),
    [{ path: "/admin/feature-flags", label: "Feature Flags" }]
  );
  assert.deepEqual(
    visibleNavChildrenForKey("reports", {
      allowed_routes: ["/reports", "/reports/security-issues", "/reports/zoom-desk-phone-renames"],
    }),
    [
      { path: "/reports/security-issues", label: "Security Issues" },
      { path: "/reports/zoom-desk-phone-renames", label: "Zoom Desk Phone Renames" },
    ]
  );
  assert.deepEqual(
    visibleNavChildrenForKey("admin", {
      allowed_routes: ["/admin"],
    }),
    []
  );
});

test("route help content covers every implemented logged-in route with training-oriented copy", () => {
  const { APP_ROUTES } = routeRegistryModule;
  const { helpContentForRoute, helpSourceNoteForRoute } = routeHelpContentModule;
  const implementedLoggedInRoutes = APP_ROUTES.filter(
    (route) => !route.public && route.kind !== "dashboard-redirect"
  );

  assert.ok(implementedLoggedInRoutes.length > 0);
  for (const route of implementedLoggedInRoutes) {
    const helpContent = helpContentForRoute(route.path, null);
    const allText = [
      helpContent.title,
      ...helpContent.sections.flatMap((section) => [
        section.heading,
        ...(section.paragraphs || []),
      ]),
    ].join(" ");

    assert.notEqual(helpContent.title, "Page help", `${route.path} must not use generic page help`);
    assert.ok(helpContent.sections.length >= 3, `${route.path} needs enough sections to train operators`);
    assert.match(allText, /(control|filter|search|sort|drawer|table|status|correction|source)/i);
    assert.ok(helpSourceNoteForRoute(route.path), `${route.path} should record source documents`);
  }
});

test("route help content keeps child routes distinct from their parent sections", () => {
  const { helpContentForRoute } = routeHelpContentModule;

  assert.notDeepEqual(
    helpContentForRoute("/reports/sync-transparency", "reports"),
    helpContentForRoute("/reports", "reports")
  );
  assert.match(
    helpContentForRoute("/reports/sync-transparency", "reports").title,
    /Sync Transparency/
  );
  assert.match(
    helpContentForRoute("/reports/security-issues", "reports").title,
    /Security Issues/
  );
  assert.match(
    helpContentForRoute("/reports/zoom-desk-phone-renames", "reports").title,
    /Zoom Desk Phone Renames/
  );
  assert.match(
    helpContentForRoute("/admin/feature-flags", "admin").title,
    /Feature Flags/
  );
});

test("route help content preserves documented correction-path warnings", () => {
  const { helpContentForRoute } = routeHelpContentModule;
  const roomMovesText = helpContentForRoute("/room-moves", "roomMoves")
    .sections.flatMap((section) => section.paragraphs)
    .join(" ");
  const studentDataText = helpContentForRoute("/student-data-cleanup", "studentDataCleanup")
    .sections.flatMap((section) => section.paragraphs)
    .join(" ");
  const dataQualityText = helpContentForRoute("/data-quality", "dataQuality")
    .sections.flatMap((section) => section.paragraphs)
    .join(" ");

  assert.match(
    roomMovesText,
    /IT can only fully revert a room move\. To partially revert a room move, create a new Room Move draft/
  );
  assert.match(studentDataText, /search by the displayed Student ID/);
  assert.doesNotMatch(dataQualityText, /Open Mapping Dashboard/);
});

test("ticketing human work report route is retired from runtime routing", () => {
  const { resolveRoute, visibleNavChildrenForKey } = routeRegistryModule;

  assert.equal(resolveRoute("/reports/ticketing-human-work"), null);
  assert.deepEqual(
    visibleNavChildrenForKey("reports", {
      allowed_routes: [
        "/reports/security-issues",
        "/reports/zoom-desk-phone-renames",
        "/reports/sync-transparency",
        "/reports/ticketing-human-work",
      ],
    }),
    [
      { path: "/reports/security-issues", label: "Security Issues" },
      { path: "/reports/zoom-desk-phone-renames", label: "Zoom Desk Phone Renames" },
      { path: "/reports/sync-transparency", label: "Sync Transparency" },
    ]
  );
});
