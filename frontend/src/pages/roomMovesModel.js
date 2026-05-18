// roomMoveMatchesCurrentRoom is the side-drawer guard before RoomMovesPage sends
// a DEV mock draft create/update request. It compares stable room ids so labels
// that happen to match across sites do not block legitimate inter-site moves.
export function roomMoveMatchesCurrentRoom(person, destinationSiteId, destinationRoomId) {
  if (!person || !destinationSiteId || !destinationRoomId || destinationRoomId === "none") {
    return false;
  }
  return person.site_id === destinationSiteId && person.current_room_id === destinationRoomId;
}

// roomMoveSameRoomMessage keeps the drawer copy aligned with the DEV API
// validation text for issues #174 and #175.
export function roomMoveSameRoomMessage(person) {
  const room = person?.current_room || "that room";
  return `${person?.name || "This person"} is already in ${room}. Choose a different destination room.`;
}

// defaultDestinationRoom mirrors the Room Moves PRD: same-site drawers show the
// current room as context, while inter-site moves start with None.
export function defaultDestinationRoom(person, destinationSiteId) {
  if (!person) {
    return "none";
  }
  return destinationSiteId === person.site_id ? person.current_room_id || "none" : "none";
}

// roomMoveDrawerActionLabels documents the single-move drawer action contract
// covered by the model test; the rendered drawer intentionally omits Schedule.
export function roomMoveDrawerActionLabels() {
  return ["Save Draft", "Save and Apply", "Cancel"];
}

// roomMoveDrawerClosedState centralizes the successful-save and cancel outcome
// so dirty drawer edits are discarded instead of leaking into the next open row.
export function roomMoveDrawerClosedState() {
  return { selectedRow: null, showCreateDrawer: false };
}
