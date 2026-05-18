import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";

import { artboardHasSharedShell } from "./sharedShellArtboard.mjs";

function readArtboard(name) {
  const artboardUrl = new URL(`../generated/${name}.artboard.json`, import.meta.url);
  return JSON.parse(fs.readFileSync(artboardUrl, "utf8"));
}

test("artboardHasSharedShell only enables sticky shell behavior for logged-in shell artboards", () => {
  assert.equal(artboardHasSharedShell(readArtboard("login")), false);
  assert.equal(artboardHasSharedShell(readArtboard("error-logged-out")), false);
  assert.equal(artboardHasSharedShell(readArtboard("error-logged-in")), true);
});
