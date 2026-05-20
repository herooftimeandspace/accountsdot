import test from "node:test";
import assert from "node:assert/strict";

import {
  SHARED_HEADER_HEIGHT,
  resolveRuntimeDrawerPlacement,
  runtimeDrawerHeightForViewport,
} from "./runtimeDrawerGeometry.mjs";

test("runtime drawer height is fixed to the viewport instead of document content", () => {
  assert.equal(runtimeDrawerHeightForViewport(1000), 600);
  assert.equal(runtimeDrawerHeightForViewport(500), 340);
});

test("bounded runtime drawer stays inside the artboard and viewport right edge", () => {
  const placement = resolveRuntimeDrawerPlacement({
    artboardRect: { left: 280, right: 1600, width: 1320 },
    artboardOffsetWidth: 1680,
    bounds: { left: 1278, width: 390 },
    viewportWidth: 1365,
    viewportHeight: 964,
  });

  assert.equal(placement.position, "fixed");
  assert.equal(placement.top, SHARED_HEADER_HEIGHT);
  assert.equal(placement.height, 578);
  assert.ok(placement.left >= 280);
  assert.ok(placement.left + placement.width <= 1365);
});

test("artboard-mounted drawer compensates for zoom so its visual right edge pins to the header", () => {
  const placement = resolveRuntimeDrawerPlacement({
    artboardRect: { left: 0, top: 0, right: 1265, bottom: 819, width: 1265, height: 819 },
    artboardOffsetWidth: 1672,
    headerRect: { left: 200, top: 0, right: 1265, bottom: 58, width: 1065, height: 58 },
    mountedInArtboard: true,
    viewportWidth: 1280,
    viewportHeight: 720,
  });
  const scale = 1265 / 1672;

  assert.equal(Math.round((placement.left + placement.width) * scale), 1265);
  assert.equal(Math.round(placement.top * scale), 58);
  assert.equal(Math.round(placement.height * scale), 432);
});

test("narrow viewport uses full-width sheet fallback below the shared header", () => {
  const placement = resolveRuntimeDrawerPlacement({
    artboardRect: { left: 280, right: 1600, width: 1320 },
    artboardOffsetWidth: 1680,
    viewportWidth: 640,
    viewportHeight: 780,
  });

  assert.deepEqual(placement, {
    position: "fixed",
    left: 0,
    top: SHARED_HEADER_HEIGHT,
    width: 640,
    height: 468,
    zIndex: 80,
  });
});
