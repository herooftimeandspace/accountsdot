import assert from "node:assert/strict";
import test from "node:test";

import {
  clampSidebarOffset,
  sidebarOffsetForFocusedRect,
  sidebarHasOverflow,
  sidebarOffsetForWheel,
  sidebarWheelDeltaPixels,
} from "./sharedShellSidebarScroll.mjs";

test("clampSidebarOffset leaves the sidebar unshifted when content fits the viewport", () => {
  assert.equal(clampSidebarOffset(-80, { viewportHeight: 900, contentBottom: 760 }), 0);
});

test("sidebarOffsetForWheel clamps independent sidebar scroll to the reachable lower bound", () => {
  const geometry = { viewportHeight: 600, contentBottom: 820 };

  assert.equal(sidebarOffsetForWheel(0, 80, geometry), -80);
  assert.equal(sidebarOffsetForWheel(-80, 500, geometry), -232);
  assert.equal(sidebarOffsetForWheel(-80, -500, geometry), 0);
});

test("sidebarHasOverflow distinguishes fitted sidebars from bounded scroll ranges", () => {
  assert.equal(sidebarHasOverflow({ viewportHeight: 900, contentBottom: 760 }), false);
  assert.equal(sidebarHasOverflow({ viewportHeight: 600, contentBottom: 820 }), true);
  assert.equal(sidebarHasOverflow({ viewportHeight: 600, contentBottom: 588 }), false);
  assert.equal(sidebarHasOverflow({ viewportHeight: 600, contentBottom: 589 }), true);
});

test("sidebarWheelDeltaPixels preserves pixel-mode wheel deltas", () => {
  assert.equal(sidebarWheelDeltaPixels(80, 0, { viewportHeight: 600 }), 80);
});

test("sidebarWheelDeltaPixels converts line-mode wheel deltas to pixel equivalents", () => {
  assert.equal(sidebarWheelDeltaPixels(3, 1, { viewportHeight: 600 }), 48);
  assert.equal(sidebarOffsetForWheel(0, sidebarWheelDeltaPixels(3, 1, { viewportHeight: 600 }), {
    viewportHeight: 600,
    contentBottom: 820,
  }), -48);
});

test("sidebarWheelDeltaPixels converts page-mode wheel deltas using viewport height", () => {
  assert.equal(sidebarWheelDeltaPixels(1, 2, { viewportHeight: 600 }), 600);
  assert.equal(sidebarWheelDeltaPixels(-1, 2, { viewportHeight: 600 }), -600);
});

test("sidebarOffsetForFocusedRect reveals controls hidden below a short viewport", () => {
  const geometry = { viewportHeight: 600, contentBottom: 820 };

  assert.equal(sidebarOffsetForFocusedRect(0, { top: 758, bottom: 802 }, geometry), -214);
});

test("sidebarOffsetForFocusedRect scrolls back toward the top for focused upper controls", () => {
  const geometry = { viewportHeight: 600, contentBottom: 820 };

  assert.equal(sidebarOffsetForFocusedRect(-180, { top: -24, bottom: 18 }, geometry), -144);
});
