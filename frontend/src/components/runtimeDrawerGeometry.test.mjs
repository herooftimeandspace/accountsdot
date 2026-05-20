import assert from "node:assert/strict";
import test from "node:test";
import {
  DEFAULT_RUNTIME_DRAWER_RIGHT_INSET,
  DEFAULT_RUNTIME_DRAWER_WIDTH,
  SHARED_HEADER_HEIGHT,
  resolveArtboardDrawerStyle,
  resolveFallbackFixedDrawerStyle,
} from "./runtimeDrawerGeometry.mjs";

test("resolveArtboardDrawerStyle returns artboard-local geometry so long drawers extend page height", () => {
  const style = resolveArtboardDrawerStyle({
    bounds: { left: 1278, width: 390 },
    artboardWidth: 1672,
    scale: 1,
  });

  assert.equal(style.position, "absolute");
  assert.equal(style.top, SHARED_HEADER_HEIGHT);
  assert.equal(style.left, 1278);
  assert.equal(style.width, 390);
});

test("resolveArtboardDrawerStyle keeps unbounded drawers inside the artboard right edge", () => {
  const style = resolveArtboardDrawerStyle({
    artboardWidth: 1672,
    scale: 1,
  });

  assert.equal(style.position, "absolute");
  assert.equal(style.left, 1672 - DEFAULT_RUNTIME_DRAWER_RIGHT_INSET - DEFAULT_RUNTIME_DRAWER_WIDTH);
  assert.equal(style.width, DEFAULT_RUNTIME_DRAWER_WIDTH);
});

test("resolveArtboardDrawerStyle converts header offset for scaled artboards", () => {
  const style = resolveArtboardDrawerStyle({
    bounds: { left: 1278, width: 390 },
    artboardWidth: 1672,
    scale: 0.5,
  });

  assert.equal(style.position, "absolute");
  assert.equal(style.top, SHARED_HEADER_HEIGHT / 0.5);
});

test("resolveFallbackFixedDrawerStyle preserves non-artboard drawer fallback placement", () => {
  assert.deepEqual(resolveFallbackFixedDrawerStyle({ left: 40, width: 360 }), {
    position: "fixed",
    left: 40,
    top: SHARED_HEADER_HEIGHT,
    width: 360,
    zIndex: 80,
  });
});
