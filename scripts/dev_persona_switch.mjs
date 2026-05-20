#!/usr/bin/env node

const DEFAULT_BASE_URL = "http://localhost:5173";

function usage() {
  return [
    "Usage: npm run dev:persona -- <persona_id> [--base-url http://localhost:5173]",
    "",
    "Examples:",
    "  npm run dev:persona -- site_admin",
    "  npm run dev:persona -- no_access --base-url http://localhost:8080",
  ].join("\n");
}

function parseArgs(argv) {
  const args = [...argv];
  let personaId = "";
  let baseUrl = process.env.WIZARD_DEV_BASE_URL || DEFAULT_BASE_URL;

  while (args.length > 0) {
    const arg = args.shift();
    if (arg === "--help" || arg === "-h") {
      return { help: true, personaId, baseUrl };
    }
    if (arg === "--base-url") {
      baseUrl = args.shift() || "";
      continue;
    }
    if (!personaId) {
      personaId = arg;
      continue;
    }
    throw new Error(`Unexpected argument: ${arg}`);
  }

  return { help: false, personaId, baseUrl };
}

async function readJSONResponse(response) {
  const text = await response.text();
  if (!text) {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch (error) {
    throw new Error(`Response was not JSON: ${error.message}\n${text}`);
  }
}

function endpointFor(baseUrl) {
  const url = new URL("/api/v1/dev/login", baseUrl);
  return url.toString();
}

async function main() {
  const { help, personaId, baseUrl } = parseArgs(process.argv.slice(2));
  if (help) {
    console.log(usage());
    return;
  }
  if (!personaId) {
    throw new Error(`Missing persona_id.\n${usage()}`);
  }
  if (!baseUrl) {
    throw new Error(`Missing --base-url value.\n${usage()}`);
  }

  const response = await fetch(endpointFor(baseUrl), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({ persona_id: personaId, activate_mock_session: true }),
  });
  const payload = await readJSONResponse(response);
  console.log(JSON.stringify(payload, null, 2));
  if (!response.ok) {
    process.exitCode = 1;
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
