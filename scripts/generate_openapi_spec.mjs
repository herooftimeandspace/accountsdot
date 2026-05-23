#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const sourcePath = path.join(repoRoot, "docs", "api", "openapi-source.json");
const specPath = path.join(repoRoot, "docs", "api", "openapi.json");
const goSpecPath = path.join(repoRoot, "internal", "web", "openapi_spec_gen.go");
const appGoPath = path.join(repoRoot, "internal", "web", "app.go");
const checkMode = process.argv.includes("--check");

function sorted(value) {
  return [...value].sort();
}

function readSource() {
  return JSON.parse(fs.readFileSync(sourcePath, "utf8"));
}

function registeredApiPaths() {
  const appGo = fs.readFileSync(appGoPath, "utf8");
  return sorted(
    [...appGo.matchAll(/mux\.Handle\("([^"]+)"/g), ...appGo.matchAll(/mux\.HandleFunc\("([^"]+)"/g)]
      .map((match) => match[1])
      .filter((registeredPath) => registeredPath.startsWith("/api/v1/")),
  );
}

function registeredPathForOperation(operationPath) {
  const templatedSegment = operationPath.indexOf("{");
  if (templatedSegment === -1) {
    return operationPath;
  }
  return operationPath.slice(0, templatedSegment);
}

function validateOperationCoverage(source) {
  const registered = sorted(source.registeredApiPaths);
  const operationRegisteredPaths = sorted([...new Set(source.operations.map((operation) => registeredPathForOperation(operation.path)))]);
  const failures = diffLists("OpenAPI operation route coverage", registered, operationRegisteredPaths);
  const operationIds = source.operations.map((operation) => operation.operationId);
  const duplicateOperationIds = sorted(operationIds.filter((operationId, index) => operationIds.indexOf(operationId) !== index));
  if (duplicateOperationIds.length > 0) {
    failures.push(`duplicate OpenAPI operationId values: ${[...new Set(duplicateOperationIds)].join(", ")}`);
  }
  return failures;
}

function pathParameters(operationPath) {
  return [...operationPath.matchAll(/\{([^}]+)\}/g)].map((match) => ({
    name: match[1],
    in: "path",
    required: true,
    schema: { type: "string" },
  }));
}

function queryParameters(operation) {
  if (operation.path !== "/api/v1/room-mappings" && operation.path !== "/api/v1/sync-status/{tab}") {
    return [];
  }
  if (operation.path === "/api/v1/room-mappings") {
    return [{ name: "query", in: "query", required: false, schema: { type: "string" } }];
  }
  return [
    { name: "site_code", in: "query", required: false, schema: { type: "string" } },
    { name: "user_type", in: "query", required: false, schema: { type: "string" } },
    { name: "school_year", in: "query", required: false, schema: { type: "string" } },
  ];
}

function responseFor(schemaName) {
  if (!schemaName) {
    return { description: "Successful response with no response body." };
  }
  return {
    description: "Successful response.",
    content: {
      "application/json": {
        schema: { $ref: `#/components/schemas/${schemaName}` },
      },
    },
  };
}

function defaultSuccessStatus(operation) {
  if (operation.method === "GET") {
    return "200";
  }
  if (operation.successStatus) {
    return String(operation.successStatus);
  }
  throw new Error(`${operation.operationId} must declare successStatus so OpenAPI does not infer write success from HTTP method.`);
}

function operationSecurity(operation) {
  const cookieBackedAuth = new Set([
    "staff-session-required-planned",
    "it-admin-required-planned",
    "dev-staff-session-required",
    "it-admin-required",
    "dev-hr-or-it-required",
    "dev-onboarding-room-permission-required",
    "dev-room-move-author-required",
    "dev-it-admin-required",
    "dev-it-or-device-wrangler-required",
    "dev-it-site-admin-or-device-wrangler-required",
    "dev-route-permission-required",
  ]);
  return cookieBackedAuth.has(operation.auth) ? [{ sessionCookie: [] }] : [];
}

function operationDBAccess(operation) {
  if (operation.writeBoundary === "dev-memory-or-local-db-write") {
    return "conditional-local-db-write";
  }
  if (operation.writeBoundary === "session-and-audit-write") {
    return "conditional-audit-log-write";
  }
  if (operation.writeBoundary === "planned-db-write-boundary") {
    return "planned-db-write";
  }
  if (operation.surface === "dev-mock-or-local-db") {
    return "conditional-local-db-read";
  }
  if (operation.surface === "db-backed-runtime-planned") {
    return "planned-db-read";
  }
  if (operation.surface === "db-backed-runtime") {
    return "none";
  }
  return "none";
}

function operationRetryExpectation(operation) {
  switch (operationDBAccess(operation)) {
    case "planned-db-write":
      return "future durable implementation must run serializable transaction work through internal/db.WithRetry before exposing retryable mutations";
    case "conditional-local-db-write":
      return "configured local-database path uses internal/db.WithRetry for feature-flag target and audit updates; memory-only DEV path has no transaction retry";
    case "conditional-local-db-read":
      return "configured local-database path may refresh feature-flag state before returning; memory-only DEV path has no transaction retry";
    case "conditional-audit-log-write":
      return "configured audit storage must fail closed if the audit write cannot be recorded; no provider retry occurs";
    case "planned-db-read":
      return "future database read implementation must enforce auth, site scope, feature flags, and field visibility before querying";
    default:
      return "no database transaction retry";
  }
}

function operationIdempotencyExpectation(operation) {
  if (operation.writeBoundary === "planned-db-write-boundary") {
    return "future durable implementation must define deterministic idempotency keys and audit/request-log behavior before writing state";
  }
  if (operation.writeBoundary === "dev-memory-or-local-db-write") {
    return "repeating an unchanged feature-flag target update must not create duplicate audit rows";
  }
  if (operation.writeBoundary === "session-and-audit-write") {
    return "repeating a valid breakglass login creates a fresh local session and records sanitized audit evidence; provider state is unchanged";
  }
  if (operation.writeBoundary === "dev-memory-write" || operation.writeBoundary === "cookie-and-dev-memory-write") {
    return "DEV mock mutation only; repeated requests follow the handler's in-memory mock-store semantics and never write providers";
  }
  return "read-only operation; idempotency key not required";
}

function errorResponseRef(statusCode) {
  const responseByStatus = {
    "400": "BadRequest",
    "401": "Unauthorized",
    "403": "Forbidden",
    "404": "NotFound",
    "409": "Conflict",
    "413": "PayloadTooLarge",
    "422": "ValidationError",
    "500": "InternalServerError",
    "503": "ServiceUnavailable",
  };
  const responseName = responseByStatus[String(statusCode)];
  if (!responseName) {
    throw new Error(`unsupported OpenAPI error status ${statusCode}`);
  }
  return { $ref: `#/components/responses/${responseName}` };
}

function requestBodyFor(schemaName) {
  if (!schemaName) {
    return undefined;
  }
  return {
    required: true,
    content: {
      "application/json": {
        schema: { $ref: `#/components/schemas/${schemaName}` },
      },
    },
  };
}

function buildSpec(source) {
  const spec = {
    openapi: "3.1.0",
    info: {
      title: "The WIZARD Callable API",
      version: source.version,
      summary: "Generated contract for callable runtime, planned DB-backed, accepted no-op, and DEV mock APIs.",
      description:
        "This specification is generated from docs/api/openapi-source.json. DEV mock operations and accepted no-op placeholders are labeled with x-wizard-surface so they are not mistaken for production-ready DB-backed APIs.",
    },
    servers: [{ url: "/" }],
    tags: [
      { name: "API Contract", description: "Contract discovery and generated OpenAPI metadata." },
      { name: "Session", description: "Runtime session and permission introspection." },
      { name: "Workflow Runtime", description: "Workflow, approval, retry, and annual-reset boundaries." },
      { name: "Sync Status", description: "Sync-status read and planned override boundaries." },
      { name: "Room Mapping", description: "Room-mapping search and planned persistence boundaries." },
      { name: "Breakglass", description: "Local non-production emergency access boundary." },
      { name: "DEV Mock APIs", description: "Development-only mock endpoints used by the React DEV frontend." },
    ],
    paths: {},
    components: {
      securitySchemes: {
        sessionCookie: {
          type: "apiKey",
          in: "cookie",
          name: "wizard_session",
          description: "Local session cookie used by DEV and breakglass flows; production SAML wiring remains environment-managed.",
        },
      },
      schemas: {
        AcceptedStatusResponse: objectSchema({
          status: { type: "string", examples: ["accepted"] },
        }),
        AcceptedWorkflowResponse: objectSchema({
          status: { type: "string", examples: ["accepted"] },
          workflow_run_id: { type: "string" },
        }),
        AnnualResetRequest: objectSchema({}),
        AnnualResetResponse: objectSchema({
          status: { type: "string", examples: ["accepted"] },
          workflow_type: { type: "string", examples: ["annual_reset_archive"] },
        }),
        ApprovalDecisionRequest: objectSchema({ reason: { type: "string" } }),
        ApprovalDecisionResponse: objectSchema({
          status: { type: "string", examples: ["accepted"] },
          approval_id: { type: "string" },
          decision: { type: "string", enum: ["approve", "reject"] },
        }),
        BreakglassLoginRequest: objectSchema({
          account_id: { type: "string" },
          token: { type: "string", format: "password" },
        }),
        CollectionResponse: objectSchema({ items: { type: "array", items: {} } }),
        DevEndDateRequest: objectSchema({ end_date: { type: "string", format: "date" } }),
        DevFeatureFlagResponse: objectSchema({ key: { type: "string" }, targets: { type: "array", items: {} } }),
        DevFeatureFlagsResponse: objectSchema({ items: { type: "array", items: {} } }),
        DevFeatureFlagUpdateRequest: objectSchema({ targets: { type: "array", items: {} } }),
        DevLoginRequest: objectSchema({
          persona_id: { type: "string" },
          activate_mock_session: { type: "boolean" },
        }),
        DevMyProfileResponse: objectSchema({ profile: {} }),
        DevMyProfileUpdateRequest: objectSchema({
          preferred_first_name: { type: "string" },
          preferred_last_name: { type: "string" },
          pronouns: { type: "string" },
        }),
        DevOffboardingActionRequest: objectSchema({ target_id: { type: "string" }, scheduled_for: { type: "string" } }),
        DevOnboardingManualDraftRequest: objectSchema({ draft: {} }),
        DevOnboardingManualDraftResponse: objectSchema({ draft: {} }),
        DevPageResponse: objectSchema({ page: { type: "string" }, payload: {} }),
        DevRoomMoveDraftRequest: objectSchema({ draft: {} }),
        DevRoomMoveDraftResponse: objectSchema({ draft: {} }),
        DevRoomMoveRevertRequest: objectSchema({ reason: { type: "string" } }),
        DevRoomMoveScheduleRequest: objectSchema({ scheduled_for: { type: "string" } }),
        DevRoomUpdateRequest: objectSchema({ room_id: { type: "string" } }),
        DevSessionResponse: objectSchema({
          authenticated: { type: "boolean" },
          authorized: { type: "boolean" },
          current_persona: {},
          allowed_routes: { type: "array", items: { type: "string" } },
        }),
        ErrorResponse: objectSchema({
          code: { type: "string" },
          message: { type: "string" },
        }),
        OpenAPISpec: { type: "object", description: "The generated OpenAPI document." },
        RoomMappingRequest: objectSchema({ query: { type: "string" }, mapping: {} }),
        RoomMappingSearchResponse: objectSchema({ query: { type: "string" }, items: { type: "array", items: {} } }),
        SessionResponse: objectSchema({
          authenticated: { type: "boolean" },
          mode: { type: "string" },
        }),
        SyncOverrideRequest: objectSchema({ reason: { type: "string" } }),
        SyncOverrideResponse: objectSchema({
          status: { type: "string", examples: ["accepted"] },
          user_type: { type: "string" },
          user_id: { type: "string" },
        }),
        SyncStatusResponse: objectSchema({
          tab: { type: "string" },
          filters: {},
          items: { type: "array", items: {} },
        }),
        WorkflowResponse: objectSchema({
          workflow_run_id: { type: "string" },
          status: { type: "string" },
          items: { type: "array", items: {} },
        }),
        WorkflowRetryRequest: objectSchema({ reason: { type: "string" } }),
      },
      responses: {
        BadRequest: {
          description: "The request body, query string, or path parameter was invalid for this operation.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        Unauthorized: {
          description: "The caller is not authenticated.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        Forbidden: {
          description: "The caller is authenticated but lacks the required role, site scope, feature flag, or field-level permission.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        ValidationError: {
          description: "The request payload failed validation.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        NotFound: {
          description: "The requested resource or route was not found.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        Conflict: {
          description: "The request conflicts with the current resource state.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        PayloadTooLarge: {
          description: "The request payload is larger than this operation accepts.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        InternalServerError: {
          description: "The server could not complete the operation.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
        ServiceUnavailable: {
          description: "The operation is disabled or the required backing service is unavailable.",
          content: { "application/json": { schema: { $ref: "#/components/schemas/ErrorResponse" } } },
        },
      },
    },
  };

  for (const operation of source.operations) {
    spec.paths[operation.path] ??= {};
    const responses = {
      [defaultSuccessStatus(operation)]: responseFor(operation.response),
    };
    for (const statusCode of operation.errorStatuses ?? []) {
      responses[String(statusCode)] = errorResponseRef(statusCode);
    }
    spec.paths[operation.path][operation.method.toLowerCase()] = {
      tags: [operation.tag],
      operationId: operation.operationId,
      summary: operation.summary,
      description: operation.summary,
      parameters: [...pathParameters(operation.path), ...queryParameters(operation)],
      ...(operation.request ? { requestBody: requestBodyFor(operation.request) } : {}),
      responses,
      security: operationSecurity(operation),
      "x-wizard-surface": operation.surface,
      "x-wizard-phase": operation.phase,
      "x-wizard-auth": operation.auth,
      "x-wizard-write-boundary": operation.writeBoundary,
      "x-wizard-db-access": operationDBAccess(operation),
      "x-wizard-transaction-retry": operationRetryExpectation(operation),
      "x-wizard-idempotency": operationIdempotencyExpectation(operation),
    };
  }

  return spec;
}

function objectSchema(properties) {
  return {
    type: "object",
    additionalProperties: true,
    properties,
  };
}

function stableJSON(value) {
  return `${JSON.stringify(value, null, 2)}\n`;
}

function buildGoSpec(specJSON) {
  return `// Code generated by scripts/generate_openapi_spec.mjs; DO NOT EDIT.\n\npackage web\n\nconst openAPISpecJSON = ${JSON.stringify(specJSON)}\n`;
}

function diffLists(label, expected, actual) {
  const missing = expected.filter((value) => !actual.includes(value));
  const extra = actual.filter((value) => !expected.includes(value));
  const failures = [];
  if (missing.length > 0) {
    failures.push(`${label} missing: ${missing.join(", ")}`);
  }
  if (extra.length > 0) {
    failures.push(`${label} extra: ${extra.join(", ")}`);
  }
  return failures;
}

function main() {
  const source = readSource();
  const failures = [
    ...diffLists("registered /api/v1 routes", sorted(source.registeredApiPaths), registeredApiPaths()),
    ...validateOperationCoverage(source),
  ];
  const spec = buildSpec(source);
  const specJSON = stableJSON(spec);
  const goSpec = buildGoSpec(specJSON);

  if (checkMode) {
    if (fs.readFileSync(specPath, "utf8") !== specJSON) {
      failures.push("docs/api/openapi.json is stale; run npm run openapi:generate");
    }
    if (fs.readFileSync(goSpecPath, "utf8") !== goSpec) {
      failures.push("internal/web/openapi_spec_gen.go is stale; run npm run openapi:generate");
    }
    if (failures.length > 0) {
      console.error(failures.join("\n"));
      process.exit(1);
    }
    console.log(`OpenAPI spec covers ${source.operations.length} operations and ${source.registeredApiPaths.length} registered /api/v1 mux paths.`);
    return;
  }

  if (failures.length > 0) {
    console.error(failures.join("\n"));
    process.exit(1);
  }
  fs.writeFileSync(specPath, specJSON);
  fs.writeFileSync(goSpecPath, goSpec);
  console.log(`Generated ${path.relative(repoRoot, specPath)} and ${path.relative(repoRoot, goSpecPath)}.`);
}

main();
