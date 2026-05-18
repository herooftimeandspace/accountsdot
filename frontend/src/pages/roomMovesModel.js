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

// roomMoveSingleDraftRequest builds the create/update payload for
// RoomMovesPage's single-move drawer. Existing seeded rows carry the original
// review-row site as scope_site_id so IT Admin edits do not rebuild a
// non-default-site draft under the persona's default scope.
export function roomMoveSingleDraftRequest(row, selectedPerson, destinationSiteId, destinationRoomId) {
  const request = {
    mode: "mid_year_targeted_move",
    person_id: selectedPerson.id,
    rows: [
      {
        person_id: selectedPerson.id,
        destination_site_id: destinationSiteId,
        destination_room_id: destinationRoomId,
      },
    ],
  };
  if (row?.draft_id) {
    request.scope_site_id = row.current_site_id || selectedPerson?.site_id || destinationSiteId;
  }
  return request;
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
