import assert from "node:assert/strict";
import test from "node:test";

import {
  clampSidebarOffset,
  sidebarOffsetForFocusedRect,
  sidebarOffsetForWheel,
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

test("sidebarOffsetForFocusedRect reveals controls hidden below a short viewport", () => {
  const geometry = { viewportHeight: 600, contentBottom: 820 };

  assert.equal(sidebarOffsetForFocusedRect(0, { top: 758, bottom: 802 }, geometry), -214);
});

test("sidebarOffsetForFocusedRect scrolls back toward the top for focused upper controls", () => {
  const geometry = { viewportHeight: 600, contentBottom: 820 };

  assert.equal(sidebarOffsetForFocusedRect(-180, { top: -24, bottom: 18 }, geometry), -144);
});
