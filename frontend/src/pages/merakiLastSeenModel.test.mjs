import assert from "node:assert/strict";
import test from "node:test";

import {
  DEFAULT_MERAKI_LAST_SEEN_FILTER,
  MERAKI_LAST_SEEN_FILTERS,
  merakiLastSeenAssignmentLabel,
  merakiLastSeenRowsForAssignmentFilter,
  merakiLastSeenStatusClass,
  merakiLastSeenStudentLabel,
} from "./merakiLastSeenModel.mjs";

const rows = [
  { id: "assigned", assignment_type: "assigned_student", student: "Maria Nguyen", match_state: "matched" },
  { id: "spare", assignment_type: "classroom_spare", student: "", match_state: "matched" },
  { id: "ambiguous", assignment_type: "ambiguous", student: "", match_state: "ambiguous" },
];

test("Meraki Last Seen filter defaults expose all devices", () => {
  assert.equal(DEFAULT_MERAKI_LAST_SEEN_FILTER, "all");
  assert.deepEqual(MERAKI_LAST_SEEN_FILTERS.map((option) => option.value), [
    "all",
    "assigned_student",
    "classroom_spare",
  ]);
});

test("Meraki Last Seen assignment filter separates student devices and spares", () => {
  assert.deepEqual(merakiLastSeenRowsForAssignmentFilter(rows, "all").map((row) => row.id), [
    "assigned",
    "spare",
    "ambiguous",
  ]);
  assert.deepEqual(merakiLastSeenRowsForAssignmentFilter(rows, "assigned_student").map((row) => row.id), [
    "assigned",
  ]);
  assert.deepEqual(merakiLastSeenRowsForAssignmentFilter(rows, "classroom_spare").map((row) => row.id), [
    "spare",
  ]);
});

test("Meraki Last Seen spares do not require inaccurate student labels", () => {
  assert.equal(merakiLastSeenStudentLabel(rows[0]), "Maria Nguyen");
  assert.equal(merakiLastSeenStudentLabel(rows[1]), "No student owner");
  assert.equal(merakiLastSeenStudentLabel(rows[2]), "Needs review");
  assert.equal(merakiLastSeenAssignmentLabel(rows[0]), "Assigned student device");
  assert.equal(merakiLastSeenAssignmentLabel(rows[1]), "Classroom spare / spare pool");
  assert.equal(merakiLastSeenAssignmentLabel(rows[2]), "Ambiguous assignment");
});

test("Meraki Last Seen ambiguous matches stay visually reviewable", () => {
  assert.equal(merakiLastSeenStatusClass(rows[0]), "reports-runtime__status reports-runtime__status--ready");
  assert.equal(merakiLastSeenStatusClass(rows[2]), "reports-runtime__status reports-runtime__status--review");
});
