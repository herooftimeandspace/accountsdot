import test from "node:test";
import assert from "node:assert/strict";

import { resolveDevToolbarAnchorStyle } from "./devPersonaToolbarGeometry.mjs";

test("DEV persona toolbar pill width is inset inside the shared sidebar bounds", () => {
  const style = resolveDevToolbarAnchorStyle({
    sidebarLeft: 0,
    sidebarRight: 260,
    platformStatusBottom: 742,
  });

  assert.deepEqual(style, {
    left: "16px",
    top: "750px",
    width: "228px",
  });
});

test("DEV persona toolbar keeps a usable inset on narrow sidebars", () => {
  const style = resolveDevToolbarAnchorStyle({
    sidebarLeft: 12,
    sidebarRight: 172,
    platformStatusBottom: 100,
  });

  assert.deepEqual(style, {
    left: "20px",
    top: "108px",
    width: "144px",
  });
});

test("DEV persona toolbar does not anchor against missing sidebar measurements", () => {
  assert.equal(
    resolveDevToolbarAnchorStyle({
      sidebarLeft: Number.POSITIVE_INFINITY,
      sidebarRight: Number.NEGATIVE_INFINITY,
      platformStatusBottom: 742,
    }),
    null,
  );
});

test("DEV persona toolbar does not anchor against collapsed sidebar measurements", () => {
  assert.equal(
    resolveDevToolbarAnchorStyle({
      sidebarLeft: 144,
      sidebarRight: 144,
      platformStatusBottom: 742,
    }),
    null,
  );
});

test("DEV persona toolbar does not create a zero-width pill for tiny sidebar measurements", () => {
  assert.equal(
    resolveDevToolbarAnchorStyle({
      sidebarLeft: 10,
      sidebarRight: 22,
      platformStatusBottom: 200,
    }),
    null,
  );
});
