#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");
const inventoryPath = path.join(repoRoot, "docs", "external-write-inventory.md");
const appPath = path.join(repoRoot, "internal", "web", "app.go");
const webPath = path.join(repoRoot, "internal", "web");

const mutatingMethods = new Set(["POST", "PUT", "DELETE", "PATCH"]);
const goMethodNames = new Map([
  ["MethodPost", "POST"],
  ["MethodPut", "PUT"],
  ["MethodDelete", "DELETE"],
  ["MethodPatch", "PATCH"],
]);

const routeMetadata = new Map([
  ["POST /api/v1/workflows/{workflow_run_id}/retry", { owner: "workflow retry" }],
  ["POST /api/v1/approvals/{approval_id}/approve", { owner: "approval decision" }],
  ["POST /api/v1/approvals/{approval_id}/reject", { owner: "approval decision" }],
  ["POST /api/v1/sync-status/{user_type}/{user_id}/override", { owner: "sync override" }],
  ["POST /api/v1/room-mappings", { owner: "room mappings" }],
  ["POST /api/v1/annual-reset", { owner: "annual reset" }],
  ["POST /api/v1/dev/login", { owner: "DEV session mock" }],
  ["POST /api/v1/dev/logout", { owner: "DEV session mock" }],
  ["PUT /api/v1/dev/my-profile", { owner: "DEV My Profile mock" }],
  ["PUT /api/v1/dev/feature-flags/{key}", { owner: "DEV feature flags" }],
  ["POST /api/v1/dev/onboarding/manual-drafts", { owner: "DEV drafts" }],
  ["PUT /api/v1/dev/onboarding/manual-drafts/{id}", { owner: "DEV drafts" }],
  ["POST /api/v1/dev/onboarding/manual-drafts/{id}/finalize", { owner: "DEV drafts" }],
  ["DELETE /api/v1/dev/onboarding/manual-drafts/{id}", { owner: "DEV drafts" }],
  ["PUT /api/v1/dev/offboarding/records/{id}/end-date", { owner: "offboarding" }],
  ["POST /api/v1/dev/offboarding/emergency-deprovision", { owner: "offboarding" }],
  ["POST /api/v1/dev/offboarding/contractor-offboarding", { owner: "offboarding" }],
  ["PUT /api/v1/dev/departing-seniors/records/{id}/end-date", { owner: "departing seniors" }],
  ["POST /api/v1/dev/departing-seniors/records/{id}/deprovision", { owner: "departing seniors" }],
  ["POST /api/v1/dev/room-moves/drafts", { owner: "room moves" }],
  ["PUT /api/v1/dev/room-moves/drafts/{id}", { owner: "room moves" }],
  ["POST /api/v1/dev/room-moves/drafts/{id}/cancel", { owner: "room moves" }],
  ["POST /api/v1/dev/room-moves/drafts/{id}/schedule", { owner: "room moves" }],
  ["POST /api/v1/dev/room-moves/drafts/{id}/apply", { owner: "room moves" }],
  ["DELETE /api/v1/dev/room-moves/drafts/{id}", { owner: "room moves" }],
  ["POST /api/v1/dev/room-moves/completed/{id}/revert", { owner: "room moves" }],
]);

const dynamicRouteResolvers = {
  handleWorkflowRoutes(_body, registeredPath) {
    return [{ method: "POST", path: `${stripTrailingSlash(registeredPath)}/{workflow_run_id}/retry` }];
  },
  handleApprovalRoutes(_body, registeredPath) {
    const base = stripTrailingSlash(registeredPath);
    return [
      { method: "POST", path: `${base}/{approval_id}/approve` },
      { method: "POST", path: `${base}/{approval_id}/reject` },
    ];
  },
  handleSyncStatusRoutes(_body, registeredPath) {
    return [{ method: "POST", path: `${stripTrailingSlash(registeredPath)}/{user_type}/{user_id}/override` }];
  },
  handleDevFeatureFlag(_body, registeredPath) {
    return [{ method: "PUT", path: `${stripTrailingSlash(registeredPath)}/{key}` }];
  },
  handleDevOnboardingManualDraft(_body, registeredPath) {
    const base = stripTrailingSlash(registeredPath);
    return [
      { method: "PUT", path: `${base}/{id}` },
      { method: "POST", path: `${base}/{id}/finalize` },
      { method: "DELETE", path: `${base}/{id}` },
    ];
  },
  handleDevOffboardingRecord(_body, registeredPath) {
    return [{ method: "PUT", path: `${stripTrailingSlash(registeredPath)}/{id}/end-date` }];
  },
  handleDevDepartingSeniorRecord(_body, registeredPath) {
    const base = stripTrailingSlash(registeredPath);
    return [
      { method: "PUT", path: `${base}/{id}/end-date` },
      { method: "POST", path: `${base}/{id}/deprovision` },
    ];
  },
  handleDevRoomMoveDraft(_body, registeredPath) {
    const base = stripTrailingSlash(registeredPath);
    return [
      { method: "PUT", path: `${base}/{id}` },
      { method: "POST", path: `${base}/{id}/cancel` },
      { method: "POST", path: `${base}/{id}/schedule` },
      { method: "POST", path: `${base}/{id}/apply` },
      { method: "DELETE", path: `${base}/{id}` },
    ];
  },
  handleDevRoomMoveCompletedJob(_body, registeredPath) {
    return [{ method: "POST", path: `${stripTrailingSlash(registeredPath)}/{id}/revert` }];
  },
};

function routeKey(route) {
  return `${route.method} ${route.path}`;
}

function stripTrailingSlash(value) {
  return value.endsWith("/") ? value.slice(0, -1) : value;
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

function listGoSources(root = webPath) {
  return fs
    .readdirSync(root)
    .filter((name) => name.endsWith(".go") && !name.endsWith("_test.go"))
    .map((name) => path.join(root, name))
    .sort();
}

function parseRegisteredHandlers(source) {
  const pattern = /mux\.Handle\("([^"]+)",\s*http\.HandlerFunc\((\w+)\)\)/g;
  const handlers = [];
  let match;
  while ((match = pattern.exec(source)) !== null) {
    handlers.push({ path: match[1], handler: match[2] });
  }
  return handlers;
}

function findFunctionBody(source, functionName) {
  const signature = `func ${functionName}(`;
  const signatureIndex = source.indexOf(signature);
  if (signatureIndex === -1) {
    return null;
  }
  const openIndex = source.indexOf("{", signatureIndex);
  if (openIndex === -1) {
    return null;
  }

  let depth = 0;
  for (let index = openIndex; index < source.length; index += 1) {
    const char = source[index];
    if (char === "{") {
      depth += 1;
    } else if (char === "}") {
      depth -= 1;
      if (depth === 0) {
        return source.slice(openIndex + 1, index);
      }
    }
  }
  return null;
}

function loadHandlerBodies(sourcePaths = listGoSources()) {
  const bodies = new Map();
  for (const sourcePath of sourcePaths) {
    const source = fs.readFileSync(sourcePath, "utf8");
    const functionPattern = /^func\s+(\w+)\(/gm;
    let match;
    while ((match = functionPattern.exec(source)) !== null) {
      const body = findFunctionBody(source, match[1]);
      if (body !== null) {
        bodies.set(match[1], { body, source: path.relative(repoRoot, sourcePath) });
      }
    }
  }
  return bodies;
}

function mutatingMethodsInBody(body) {
  const methods = new Set();
  const pattern = /http\.(MethodPost|MethodPut|MethodDelete|MethodPatch)\b/g;
  let match;
  while ((match = pattern.exec(body)) !== null) {
    methods.add(goMethodNames.get(match[1]));
  }
  return [...methods].sort();
}

function deriveRoutesFromHandler(registered, handlerInfo) {
  const methods = mutatingMethodsInBody(handlerInfo.body);
  if (methods.length === 0) {
    return [];
  }

  const resolver = dynamicRouteResolvers[registered.handler];
  const resolved = resolver ? resolver(handlerInfo.body, registered.path) : methods.map((method) => ({ method, path: registered.path }));

  return resolved.map((route) => ({
    ...route,
    source: handlerInfo.source,
    handler: registered.handler,
  }));
}

function deriveLiveMutatingRoutes({
  appSource = fs.readFileSync(appPath, "utf8"),
  handlerBodies = loadHandlerBodies(),
} = {}) {
  const failures = [];
  const routes = [];

  for (const registered of parseRegisteredHandlers(appSource)) {
    const handlerInfo = handlerBodies.get(registered.handler);
    if (!handlerInfo) {
      failures.push(`${registered.path} is registered to ${registered.handler}, but that handler body was not found`);
      continue;
    }
    const derivedRoutes = deriveRoutesFromHandler(registered, handlerInfo);
    const methods = mutatingMethodsInBody(handlerInfo.body);
    if (methods.length > 0 && derivedRoutes.length === 0) {
      failures.push(`${registered.handler} uses ${methods.join(", ")} but no live route could be derived`);
    }
    routes.push(...derivedRoutes);
  }

  return { routes, failures };
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

function checkInventory(markdown, routes = deriveLiveMutatingRoutes().routes, deriveFailures = []) {
  const { routes: documentedRoutes, exceptions } = parseInventoryRoutes(markdown);
  const expected = new Map(routes.map((route) => [routeKey(route), route]));
  const expectedKeys = new Set(expected.keys());
  const failures = [...deriveFailures];

  for (const route of routes) {
    const key = routeKey(route);
    if (!mutatingMethods.has(route.method)) {
      failures.push(`${key} uses unsupported method ${route.method}`);
      continue;
    }
    if (!routeMetadata.has(key)) {
      failures.push(`${key} (${route.handler}, ${route.source}) is live but missing metadata in scripts/check_external_write_inventory.mjs`);
    }
    if (!documentedRoutes.has(key) && !exceptions.has(key)) {
      const owner = routeMetadata.get(key)?.owner ?? "unknown owner";
      failures.push(`${key} (${owner}, ${route.source}) is missing from docs/external-write-inventory.md`);
    }
  }

  for (const documented of documentedRoutes) {
    if (!expectedKeys.has(documented)) {
      failures.push(`${documented} is documented but was not derived from live internal/web route handlers`);
    }
  }

  for (const exception of exceptions) {
    if (!expectedKeys.has(exception)) {
      failures.push(`${exception} has an exception comment but was not derived from live internal/web route handlers`);
    }
  }

  for (const duplicate of findDuplicateRoutes(routes)) {
    failures.push(`${duplicate} is duplicated in the live mutating route inventory`);
  }

  return { failures, routes };
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
    { method: "POST", path: "/api/example", source: "test", handler: "test" },
    { method: "PATCH", path: "/api/future", source: "test", handler: "test" },
  ]);
  assertions.push([
    failureCheck.failures.some((failure) => failure.includes("PATCH /api/future")),
    "fails when a live mutating route is not documented",
  ]);

  const staleDocumentationCheck = checkInventory("- `POST /api/stale`", [
    { method: "POST", path: "/api/example", source: "test", handler: "test" },
  ]);
  assertions.push([
    staleDocumentationCheck.failures.some((failure) => failure.includes("POST /api/stale")),
    "fails when documentation includes stale routes",
  ]);

  const exceptionCheck = checkInventory("<!-- external-write-inventory-exception: PATCH /api/future -->", [
    { method: "PATCH", path: "/api/future", source: "test", handler: "test" },
  ]);
  assertions.push([
    !exceptionCheck.failures.some((failure) => failure.includes("missing from docs")),
    "accepts documented no-op/mock exceptions",
  ]);

  const appSource = [
    "func NewAppHandler(deps HealthDependencies) http.Handler {",
    '  mux.Handle("/api/example", http.HandlerFunc(handleExample))',
    "}",
  ].join("\n");
  const handlerBodies = new Map([
    ["handleExample", { body: "if r.Method != http.MethodPost { return }", source: "internal/web/example.go" }],
  ]);
  const liveRoutes = deriveLiveMutatingRoutes({ appSource, handlerBodies }).routes;
  assertions.push([
    liveRoutes.some((route) => routeKey(route) === "POST /api/example"),
    "derives exact mutating routes from registered handlers",
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
  const { routes, failures: deriveFailures } = deriveLiveMutatingRoutes();
  const { failures } = checkInventory(markdown, routes, deriveFailures);

  if (failures.length > 0) {
    console.error("External write inventory drift detected:");
    for (const failure of failures) {
      console.error(`- ${failure}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log(`External write inventory check passed for ${routes.length} live mutating routes.`);
}

main();
