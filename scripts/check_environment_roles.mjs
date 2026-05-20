#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const STAGING_MASKED_DATA_SOURCE = "masked-production-derived";
const STAGING_SANDBOX_DATA_SOURCE = "documented-sandbox";

const ENVIRONMENT_CONTRACTS = [
  {
    role: "dev",
    appEnv: "development",
    dataModes: ["mock"],
    databaseMarker: "postgres-dev:5432/provisioner_dev",
    envFile: "deploy/env/dev.app.env.example",
    dbFile: "deploy/env/dev.db.env.example",
    dbName: "provisioner_dev",
    dbUser: "provisioner_dev",
  },
  {
    role: "staging",
    appEnv: "staging",
    dataModes: ["masked-read-only", "sandbox"],
    dataSources: [STAGING_MASKED_DATA_SOURCE, STAGING_SANDBOX_DATA_SOURCE],
    databaseMarker: "postgres-staging:5432/provisioner_staging",
    envFile: "deploy/env/staging.app.env.example",
    dbFile: "deploy/env/staging.db.env.example",
    dbName: "provisioner_staging",
    dbUser: "provisioner_staging",
  },
  {
    role: "main",
    appEnv: "production",
    dataModes: ["production"],
    databaseMarker: "postgres-main:5432/provisioner_main",
    envFile: "deploy/env/main.app.env.example",
    dbFile: "deploy/env/main.db.env.example",
    dbName: "provisioner_main",
    dbUser: "provisioner_main",
  },
];

const PROVIDER_MOCK_KEYS = [
  "USE_MOCK_ZOOM",
  "USE_MOCK_GOOGLE",
  "USE_MOCK_AERIES",
  "USE_MOCK_SFTP",
];

// readText keeps every validation path rooted in this checkout so the script can
// run from local CI, GitHub Actions, or a contributor's shell without relying on
// the current working directory.
function readText(relativePath) {
  return fs.readFileSync(path.join(REPO_ROOT, relativePath), "utf8");
}

// parseEnvFile reads the checked-in env examples into a Map and rejects syntax
// that would make a deployment role ambiguous before the examples reach Compose.
function parseEnvFile(relativePath) {
  const entries = new Map();
  for (const [index, rawLine] of readText(relativePath).split(/\r?\n/).entries()) {
    const line = rawLine.trim();
    if (line === "" || line.startsWith("#")) {
      continue;
    }
    const equalsIndex = line.indexOf("=");
    if (equalsIndex === -1) {
      throw new Error(`${relativePath}:${index + 1} is not KEY=value syntax`);
    }
    const key = line.slice(0, equalsIndex).trim();
    const value = line.slice(equalsIndex + 1).trim();
    if (entries.has(key)) {
      throw new Error(`${relativePath}:${index + 1} duplicates ${key}`);
    }
    entries.set(key, value);
  }
  return entries;
}

// parseComposeServices extracts only the service env_file wiring that this
// Phase 0 gate owns. It intentionally avoids a general YAML parser because the
// deploy file shape is small and the repository has no YAML dependency.
function parseComposeServices(composeText) {
  const services = new Map();
  const lines = composeText.split(/\r?\n/);
  let inServices = false;
  let currentService = null;
  let inEnvFile = false;

  for (const rawLine of lines) {
    if (/^services:\s*$/.test(rawLine)) {
      inServices = true;
      currentService = null;
      inEnvFile = false;
      continue;
    }
    if (!inServices) {
      continue;
    }
    if (/^[^\s].+:\s*$/.test(rawLine)) {
      break;
    }

    const serviceMatch = /^  ([A-Za-z0-9_-]+):\s*$/.exec(rawLine);
    if (serviceMatch) {
      currentService = { envFiles: [] };
      services.set(serviceMatch[1], currentService);
      inEnvFile = false;
      continue;
    }
    if (!currentService) {
      continue;
    }

    if (/^    env_file:\s*$/.test(rawLine)) {
      inEnvFile = true;
      continue;
    }
    if (inEnvFile) {
      const envFileMatch = /^      -\s+(.+?)\s*$/.exec(rawLine);
      if (envFileMatch) {
        currentService.envFiles.push(envFileMatch[1]);
        continue;
      }
      if (/^    \S/.test(rawLine) || /^  \S/.test(rawLine)) {
        inEnvFile = false;
      }
    }
  }

  return services;
}

function requireValue(errors, entries, relativePath, key, expected) {
  const actual = entries.get(key);
  if (actual !== expected) {
    errors.push(`${relativePath} must set ${key}=${expected}; found ${actual || "<missing>"}`);
  }
}

function requireOneOf(errors, entries, relativePath, key, expectedValues) {
  const actual = entries.get(key);
  if (!expectedValues.includes(actual)) {
    errors.push(
      `${relativePath} must set ${key} to one of ${expectedValues.join(", ")}; found ${actual || "<missing>"}`,
    );
  }
}

function requireContains(errors, entries, relativePath, key, marker) {
  const actual = entries.get(key) || "";
  if (!actual.includes(marker)) {
    errors.push(`${relativePath} ${key} must contain ${marker}; found ${actual || "<missing>"}`);
  }
}

function requireBooleanValue(errors, entries, relativePath, key) {
  const actual = entries.get(key);
  if (actual !== "true" && actual !== "false") {
    errors.push(`${relativePath} must set ${key} to true or false; found ${actual || "<missing>"}`);
  }
}

function deployedEnvFile(examplePath) {
  return examplePath.replace(/\.example$/, "");
}

// validateComposeServiceEnvFiles proves the deployment services load the env
// file for their own role. A bare service-name token check would miss a staging
// app accidentally wired to main secrets or a main database using staging state.
function validateComposeServiceEnvFiles(errors, composeServices, contract) {
  const expectedAppService = `app-${contract.role}`;
  const expectedDbService = `postgres-${contract.role}`;
  const expectedAppEnvFile = deployedEnvFile(contract.envFile);
  const expectedDbEnvFile = deployedEnvFile(contract.dbFile);

  const serviceExpectations = [
    [expectedAppService, expectedAppEnvFile],
    [expectedDbService, expectedDbEnvFile],
  ];

  for (const [serviceName, expectedEnvFile] of serviceExpectations) {
    const service = composeServices.get(serviceName);
    if (!service) {
      errors.push(`docker-compose.deploy.yml must define ${serviceName}`);
      continue;
    }
    if (!service.envFiles.includes(expectedEnvFile)) {
      errors.push(
        `docker-compose.deploy.yml service ${serviceName} must load ${expectedEnvFile}; found ${
          service.envFiles.join(", ") || "<none>"
        }`,
      );
    }

    for (const envFile of service.envFiles) {
      const roleEnvMatch = /^deploy\/env\/(dev|staging|main)\.(app|db)\.env$/.exec(envFile);
      if (roleEnvMatch && envFile !== expectedEnvFile) {
        errors.push(
          `docker-compose.deploy.yml service ${serviceName} must not load another role env file ${envFile}`,
        );
      }
    }
  }
}

// validateStagingProfile allows staging to graduate from masked read-only data
// to documented sandbox data without weakening the current Phase 0 default. The
// exact checked-in example still uses masked production-derived data today.
function validateStagingProfile(errors, app, contract) {
  const dataMode = app.get("ENVIRONMENT_DATA_MODE");
  const dataSource = app.get("ENVIRONMENT_DATA_SOURCE");

  requireOneOf(errors, app, contract.envFile, "ENVIRONMENT_DATA_SOURCE", contract.dataSources);

  if (dataMode === "masked-read-only" && dataSource !== STAGING_MASKED_DATA_SOURCE) {
    errors.push(
      `${contract.envFile} must pair ENVIRONMENT_DATA_MODE=masked-read-only with ENVIRONMENT_DATA_SOURCE=${STAGING_MASKED_DATA_SOURCE}`,
    );
  }
  if (dataMode === "sandbox" && dataSource !== STAGING_SANDBOX_DATA_SOURCE) {
    errors.push(
      `${contract.envFile} must pair ENVIRONMENT_DATA_MODE=sandbox with ENVIRONMENT_DATA_SOURCE=${STAGING_SANDBOX_DATA_SOURCE}`,
    );
  }
}

// validateEnvironmentContracts checks the checked-in deployment examples for
// distinct dev/staging/main roles, databases, data modes, provider safety flags,
// and Compose service-to-env-file wiring.
function validateEnvironmentContracts() {
  const errors = [];
  const composeServices = parseComposeServices(readText("docker-compose.deploy.yml"));
  const seenDatabaseMarkers = new Set();

  for (const contract of ENVIRONMENT_CONTRACTS) {
    const app = parseEnvFile(contract.envFile);
    const db = parseEnvFile(contract.dbFile);

    requireValue(errors, app, contract.envFile, "ENVIRONMENT_ROLE", contract.role);
    requireValue(errors, app, contract.envFile, "APP_ENV", contract.appEnv);
    requireOneOf(errors, app, contract.envFile, "ENVIRONMENT_DATA_MODE", contract.dataModes);
    requireContains(errors, app, contract.envFile, "DATABASE_URL", contract.databaseMarker);
    requireValue(errors, db, contract.dbFile, "POSTGRES_DB", contract.dbName);
    requireValue(errors, db, contract.dbFile, "POSTGRES_USER", contract.dbUser);

    if (contract.role === "dev") {
      if (app.get("ENVIRONMENT_DATA_MODE") !== "mock") {
        errors.push(`${contract.envFile} must stay mock-backed so dev cannot assume production-only data`);
      }
      if (PROVIDER_MOCK_KEYS.some((key) => app.get(key) !== "true")) {
        errors.push(`${contract.envFile} must keep every provider mock enabled in dev`);
      }
      for (const key of PROVIDER_MOCK_KEYS) {
        requireValue(errors, app, contract.envFile, key, "true");
      }
    }

    if (contract.role === "staging") {
      validateStagingProfile(errors, app, contract);
      for (const key of PROVIDER_MOCK_KEYS) {
        requireBooleanValue(errors, app, contract.envFile, key);
      }
    }

    if (contract.role === "main") {
      for (const key of PROVIDER_MOCK_KEYS) {
        requireValue(errors, app, contract.envFile, key, "false");
      }
    }

    if (seenDatabaseMarkers.has(contract.databaseMarker)) {
      errors.push(`${contract.envFile} reuses database marker ${contract.databaseMarker}`);
    }
    seenDatabaseMarkers.add(contract.databaseMarker);

    validateComposeServiceEnvFiles(errors, composeServices, contract);
  }

  return errors;
}

function main() {
  const errors = validateEnvironmentContracts();
  if (errors.length > 0) {
    for (const error of errors) {
      console.error(`environment-role check failed: ${error}`);
    }
    return 1;
  }
  console.log("environment-role check passed: dev, staging, and main deploy examples are distinct and safety-scoped.");
  return 0;
}

process.exitCode = main();
