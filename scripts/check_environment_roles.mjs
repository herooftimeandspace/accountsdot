#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const REPO_ROOT = path.resolve(new URL("..", import.meta.url).pathname);

const ENVIRONMENT_CONTRACTS = [
  {
    role: "dev",
    appEnv: "development",
    dataMode: "mock",
    databaseMarker: "postgres-dev:5432/provisioner_dev",
    envFile: "deploy/env/dev.app.env.example",
    dbFile: "deploy/env/dev.db.env.example",
    dbName: "provisioner_dev",
    dbUser: "provisioner_dev",
    mockProviders: true,
  },
  {
    role: "staging",
    appEnv: "staging",
    dataMode: "masked-read-only",
    dataSource: "masked-production-derived",
    databaseMarker: "postgres-staging:5432/provisioner_staging",
    envFile: "deploy/env/staging.app.env.example",
    dbFile: "deploy/env/staging.db.env.example",
    dbName: "provisioner_staging",
    dbUser: "provisioner_staging",
    mockProviders: true,
  },
  {
    role: "main",
    appEnv: "production",
    dataMode: "production",
    databaseMarker: "postgres-main:5432/provisioner_main",
    envFile: "deploy/env/main.app.env.example",
    dbFile: "deploy/env/main.db.env.example",
    dbName: "provisioner_main",
    dbUser: "provisioner_main",
    mockProviders: false,
  },
];

const PROVIDER_MOCK_KEYS = [
  "USE_MOCK_ZOOM",
  "USE_MOCK_GOOGLE",
  "USE_MOCK_AERIES",
  "USE_MOCK_SFTP",
];

function readText(relativePath) {
  return fs.readFileSync(path.join(REPO_ROOT, relativePath), "utf8");
}

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

function requireValue(errors, entries, relativePath, key, expected) {
  const actual = entries.get(key);
  if (actual !== expected) {
    errors.push(`${relativePath} must set ${key}=${expected}; found ${actual || "<missing>"}`);
  }
}

function requireContains(errors, entries, relativePath, key, marker) {
  const actual = entries.get(key) || "";
  if (!actual.includes(marker)) {
    errors.push(`${relativePath} ${key} must contain ${marker}; found ${actual || "<missing>"}`);
  }
}

function validateEnvironmentContracts() {
  const errors = [];
  const compose = readText("docker-compose.deploy.yml");
  const seenDatabaseMarkers = new Set();

  for (const contract of ENVIRONMENT_CONTRACTS) {
    const app = parseEnvFile(contract.envFile);
    const db = parseEnvFile(contract.dbFile);

    requireValue(errors, app, contract.envFile, "ENVIRONMENT_ROLE", contract.role);
    requireValue(errors, app, contract.envFile, "APP_ENV", contract.appEnv);
    requireValue(errors, app, contract.envFile, "ENVIRONMENT_DATA_MODE", contract.dataMode);
    requireContains(errors, app, contract.envFile, "DATABASE_URL", contract.databaseMarker);
    requireValue(errors, db, contract.dbFile, "POSTGRES_DB", contract.dbName);
    requireValue(errors, db, contract.dbFile, "POSTGRES_USER", contract.dbUser);

    if (contract.dataSource) {
      requireValue(errors, app, contract.envFile, "ENVIRONMENT_DATA_SOURCE", contract.dataSource);
    }

    for (const key of PROVIDER_MOCK_KEYS) {
      requireValue(errors, app, contract.envFile, key, String(contract.mockProviders));
    }

    if (contract.role === "dev") {
      if (app.get("ENVIRONMENT_DATA_MODE") !== "mock") {
        errors.push(`${contract.envFile} must stay mock-backed so dev cannot assume production-only data`);
      }
      if (PROVIDER_MOCK_KEYS.some((key) => app.get(key) !== "true")) {
        errors.push(`${contract.envFile} must keep every provider mock enabled in dev`);
      }
    }

    if (contract.role === "staging" && app.get("ENVIRONMENT_DATA_SOURCE") !== "masked-production-derived") {
      errors.push(`${contract.envFile} must declare the masked production-derived staging data source`);
    }

    if (seenDatabaseMarkers.has(contract.databaseMarker)) {
      errors.push(`${contract.envFile} reuses database marker ${contract.databaseMarker}`);
    }
    seenDatabaseMarkers.add(contract.databaseMarker);

    for (const expectedComposeToken of [`app-${contract.role}`, `postgres-${contract.role}`]) {
      if (!compose.includes(expectedComposeToken)) {
        errors.push(`docker-compose.deploy.yml must define ${expectedComposeToken}`);
      }
    }
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
