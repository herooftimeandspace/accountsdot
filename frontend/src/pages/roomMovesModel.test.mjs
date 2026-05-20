import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";

const modelUrl = new URL("./roomMovesModel.js", import.meta.url);
const modelSource = fs.readFileSync(modelUrl, "utf8");
const modelModule = await import(
  `data:text/javascript;base64,${Buffer.from(modelSource).toString("base64")}`
);

const {
  defaultDestinationRoom,
  roomMoveSingleDraftRequest,
  roomMoveDrawerActionLabels,
  roomMoveDrawerClosedState,
  roomMoveMatchesCurrentRoom,
  roomMoveSameRoomMessage,
} = modelModule;

const jamieReed = {
  id: "jamie-reed",
  name: "Jamie Reed",
  site_id: "desert-view",
  current_room_id: "dve-c118",
  current_room: "C-118",
};

test("room move drawer blocks same stable room ids before submitting", () => {
  assert.equal(roomMoveMatchesCurrentRoom(jamieReed, "desert-view", "dve-c118"), true);
  assert.equal(roomMoveMatchesCurrentRoom(jamieReed, "desert-view", "dve-c122"), false);
  assert.equal(roomMoveMatchesCurrentRoom(jamieReed, "clover-hs", "dve-c118"), false);
  assert.equal(roomMoveMatchesCurrentRoom(jamieReed, "desert-view", "none"), false);
  assert.equal(
    roomMoveSameRoomMessage(jamieReed),
    "Jamie Reed is already in C-118. Choose a different destination room."
  );
});

test("room move drawer exposes Save and Apply without Schedule and closes on success or cancel", () => {
  assert.deepEqual(roomMoveDrawerActionLabels(), ["Save Draft", "Save and Apply", "Cancel"]);
  assert.equal(roomMoveDrawerActionLabels().includes("Schedule"), false);
  assert.equal(defaultDestinationRoom(jamieReed, "desert-view"), "dve-c118");
  assert.equal(defaultDestinationRoom(jamieReed, "clover-hs"), "none");
  assert.deepEqual(roomMoveDrawerClosedState(), { selectedRow: null, showCreateDrawer: false });
});

test("room move drawer preserves existing seeded-row scope on draft update", () => {
  const updatePayload = roomMoveSingleDraftRequest(
    { draft_id: "single-jamie-reed", current_site_id: "desert-view" },
    jamieReed,
    "desert-view",
    "dve-c122"
  );
  assert.equal(updatePayload.scope_site_id, "desert-view");
  assert.deepEqual(updatePayload.rows, [
    {
      person_id: "jamie-reed",
      destination_site_id: "desert-view",
      destination_room_id: "dve-c122",
    },
  ]);

  const createPayload = roomMoveSingleDraftRequest(null, jamieReed, "desert-view", "dve-c122");
  assert.equal(Object.hasOwn(createPayload, "scope_site_id"), false);
});
