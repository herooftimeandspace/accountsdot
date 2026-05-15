import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";

const displayHelperUrl = new URL("./studentDataCleanupDisplay.js", import.meta.url);
const displayHelperSource = fs.readFileSync(displayHelperUrl, "utf8");
const displayHelperModule = await import(
  `data:text/javascript;base64,${Buffer.from(displayHelperSource).toString("base64")}`
);

const {
  markEdgeWhitespaceForStudentData,
  shouldShowSuggestedStudentNameValue,
  VISIBLE_SPACE_MARKER,
} = displayHelperModule;

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
