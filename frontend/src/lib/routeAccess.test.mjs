import assert from "node:assert/strict";
import test from "node:test";

import { redirectTargetForRoute } from "./routeAccess.mjs";

test("redirectTargetForRoute lets logged-out users view explicit public error routes", () => {
  assert.equal(
    redirectTargetForRoute({
      sessionState: "ready",
      authenticated: false,
      currentPath: "/error/404",
      currentRoute: { path: "/error/404", kind: "error", public: true, code: 404 },
      session: null,
    }),
    null
  );
});

test("redirectTargetForRoute still sends logged-out protected routes to 401", () => {
  assert.equal(
    redirectTargetForRoute({
      sessionState: "ready",
      authenticated: false,
      currentPath: "/data-quality",
      currentRoute: { path: "/data-quality", kind: "data-quality", artboardKey: "data-quality" },
      session: null,
    }),
    "/error/401"
  );
});

test("redirectTargetForRoute keeps authenticated access checks intact", () => {
  const session = { allowed_routes: ["/data-quality"], landing_path: "/dashboard/it-admin" };

  assert.equal(
    redirectTargetForRoute({
      sessionState: "ready",
      authenticated: true,
      currentPath: "/data-quality",
      currentRoute: { path: "/data-quality", kind: "data-quality", artboardKey: "data-quality" },
      session,
    }),
    null
  );
  assert.equal(
    redirectTargetForRoute({
      sessionState: "ready",
      authenticated: true,
      currentPath: "/admin",
      currentRoute: { path: "/admin", kind: "static", artboardKey: "admin" },
      session,
    }),
    "/error/403"
  );
});
