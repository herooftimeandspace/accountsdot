import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";

const displayHelperUrl = new URL("./studentDataCleanupDisplay.js", import.meta.url);
const displayHelperSource = fs.readFileSync(displayHelperUrl, "utf8");
const displayHelperModule = await import(
  `data:text/javascript;base64,${Buffer.from(displayHelperSource).toString("base64")}`
);
const modelHelperModule = await import("./studentDataCleanupModel.mjs");

const {
  markEdgeWhitespaceForStudentData,
  shouldShowSuggestedStudentNameValue,
  VISIBLE_SPACE_MARKER,
} = displayHelperModule;
const {
  STUDENT_DATA_CLEANUP_ROWS,
  effectiveStudentDataCleanupSiteId,
  studentDataCleanupRowsForSession,
} = modelHelperModule;

function sessionForPersona(personaId, overrides = {}) {
  return {
    current_persona: { id: personaId },
    current_site_id: "clover-hs",
    default_site_id: "clover-hs",
    visible_sites: [
      { id: "clover-hs", name: "Clover High School" },
      { id: "desert-view", name: "Desert View" },
    ],
    ...overrides,
  };
}

test("markEdgeWhitespaceForStudentData renders leading and trailing spaces as visible markers", () => {
  assert.equal(markEdgeWhitespaceForStudentData(" Erin"), `${VISIBLE_SPACE_MARKER}Erin`);
  assert.equal(markEdgeWhitespaceForStudentData("Stone "), `Stone${VISIBLE_SPACE_MARKER}`);
  assert.equal(
    markEdgeWhitespaceForStudentData("  Ivy  "),
    `${VISIBLE_SPACE_MARKER}${VISIBLE_SPACE_MARKER}Ivy${VISIBLE_SPACE_MARKER}${VISIBLE_SPACE_MARKER}`
  );
  assert.equal(markEdgeWhitespaceForStudentData("Ana Maria"), "Ana Maria");
});

test("shouldShowSuggestedStudentNameValue suppresses suggestions that display identically", () => {
  assert.equal(shouldShowSuggestedStudentNameValue("Ivy", "Ivy"), false);
  assert.equal(shouldShowSuggestedStudentNameValue("Stone ", "Stone"), true);
  assert.equal(shouldShowSuggestedStudentNameValue("Nuño", "Nuno"), true);
});

test("studentDataCleanupRowsForSession keeps district-wide personas district-wide", () => {
  const rows = studentDataCleanupRowsForSession(STUDENT_DATA_CLEANUP_ROWS, sessionForPersona("it_admin"));
  assert.equal(rows.length, STUDENT_DATA_CLEANUP_ROWS.length);
  assert.ok(rows.some((row) => row.siteId === "desert-view"));
  assert.ok(rows.some((row) => row.siteId === "franklin-ms"));
});

test("studentDataCleanupRowsForSession scopes Site Admin and Site Secretary rows to active site", () => {
  for (const personaId of ["site_admin", "site_secretary"]) {
    const rows = studentDataCleanupRowsForSession(STUDENT_DATA_CLEANUP_ROWS, sessionForPersona(personaId));
    assert.ok(rows.length > 0, `${personaId} should retain assigned-site cleanup rows`);
    assert.ok(rows.every((row) => row.siteId === "clover-hs"), `${personaId} leaked rows outside Clover High School`);
  }
});

test("studentDataCleanupRowsForSession follows the active DEV header site and rejects unknown site params", () => {
  const originalWindow = globalThis.window;
  globalThis.window = { location: { search: "?site_id=desert-view" } };
  assert.equal(effectiveStudentDataCleanupSiteId(sessionForPersona("site_admin")), "desert-view");
  assert.deepEqual(
    studentDataCleanupRowsForSession(STUDENT_DATA_CLEANUP_ROWS, sessionForPersona("site_admin")).map(
      (row) => row.siteId
    ),
    ["desert-view", "desert-view", "desert-view", "desert-view"]
  );

  globalThis.window = { location: { search: "?site_id=unknown-site" } };
  assert.equal(effectiveStudentDataCleanupSiteId(sessionForPersona("site_secretary")), "clover-hs");
  if (originalWindow) {
    globalThis.window = originalWindow;
  } else {
    delete globalThis.window;
  }
});
