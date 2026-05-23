export const DEFAULT_MERAKI_LAST_SEEN_FILTER = "all";

export const MERAKI_LAST_SEEN_FILTERS = [
  { value: "all", label: "All devices" },
  { value: "assigned_student", label: "Assigned student devices" },
  { value: "classroom_spare", label: "Classroom spares" },
];

/**
 * merakiLastSeenRowsForAssignmentFilter applies the dashboard's assignment-type
 * filter. Ambiguous rows stay visible in All devices only so operators do not
 * accidentally treat them as assigned-student or spare-pool inventory.
 */
export function merakiLastSeenRowsForAssignmentFilter(rows, assignmentFilter) {
  const sourceRows = Array.isArray(rows) ? rows : [];
  if (assignmentFilter === "assigned_student") {
    return sourceRows.filter((row) => row.assignment_type === "assigned_student");
  }
  if (assignmentFilter === "classroom_spare") {
    return sourceRows.filter((row) => row.assignment_type === "classroom_spare");
  }
  return sourceRows;
}

/**
 * merakiLastSeenLastSeenSortValue returns the machine-sortable timestamp for
 * last-seen ordering while leaving the display-formatted timestamp untouched.
 */
export function merakiLastSeenLastSeenSortValue(row) {
  return row?.last_seen_at || row?.last_seen_iso || row?.last_seen || "";
}

/**
 * merakiLastSeenAssignmentLabel returns stable visible copy for the assignment
 * badge even when a future provider-backed payload omits the preformatted label.
 */
export function merakiLastSeenAssignmentLabel(row) {
  if (row?.assignment_type_label) {
    return row.assignment_type_label;
  }
  if (row?.assignment_type === "assigned_student") {
    return "Assigned student device";
  }
  if (row?.assignment_type === "classroom_spare") {
    return "Classroom spare / spare pool";
  }
  return "Ambiguous assignment";
}

/**
 * merakiLastSeenStudentLabel keeps classroom spares from being forced into an
 * inaccurate student match while still making missing assigned-student matches
 * visually reviewable.
 */
export function merakiLastSeenStudentLabel(row) {
  if (row?.student) {
    return row.student;
  }
  if (row?.assignment_type === "classroom_spare") {
    return "No student owner";
  }
  return "Needs review";
}

export function merakiLastSeenStatusClass(row) {
  if (row?.match_state === "ambiguous" || row?.assignment_type === "ambiguous") {
    return "reports-runtime__status reports-runtime__status--review";
  }
  if (row?.match_state === "unmatched" || row?.match_state === "stale") {
    return "reports-runtime__status reports-runtime__status--critical";
  }
  return "reports-runtime__status reports-runtime__status--ready";
}
