import test from "node:test";
import assert from "node:assert/strict";

import { nextRuntimeDrawerSelection, nextRuntimeDrawerSelectionForId, runtimeDrawerItemId } from "./runtimeDrawerController.mjs";

test("runtimeDrawerItemId normalizes row identifiers for selection comparisons", () => {
  assert.equal(runtimeDrawerItemId({ id: 42 }), "42");
  assert.equal(runtimeDrawerItemId(null), "");
});

test("nextRuntimeDrawerSelection opens, toggles, and replaces row drawer selection", () => {
  const rowA = { id: "a", label: "A" };
  const rowB = { id: "b", label: "B" };

  assert.equal(nextRuntimeDrawerSelection(null, rowA), rowA);
  assert.equal(nextRuntimeDrawerSelection(rowA, rowA), null);
  assert.equal(nextRuntimeDrawerSelection(rowA, rowB), rowB);
  assert.equal(nextRuntimeDrawerSelection(rowA, null), null);
});

test("nextRuntimeDrawerSelectionForId toggles from selected row ids used by table overlays", () => {
  const rowA = { id: "a" };
  const rowB = { id: "b" };

  assert.equal(nextRuntimeDrawerSelectionForId("a", rowA), null);
  assert.equal(nextRuntimeDrawerSelectionForId("a", rowB), rowB);
});
