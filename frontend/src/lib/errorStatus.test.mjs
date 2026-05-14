import assert from "node:assert/strict";
import test from "node:test";

import { errorStatusCodeFor } from "./errorStatus.mjs";

test("errorStatusCodeFor preserves a valid request status", () => {
  assert.equal(errorStatusCodeFor({ status: 404 }), 404);
  assert.equal(errorStatusCodeFor({ status: "503" }), 503);
});

test("errorStatusCodeFor falls back when no valid HTTP status is attached", () => {
  assert.equal(errorStatusCodeFor(new Error("network failed")), 500);
  assert.equal(errorStatusCodeFor({ status: 0 }, 502), 502);
  assert.equal(errorStatusCodeFor({ status: 700 }, 502), 502);
});
