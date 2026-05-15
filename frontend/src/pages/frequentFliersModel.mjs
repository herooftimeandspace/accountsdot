export const DEFAULT_FREQUENT_FLIERS_FILTERS = { threshold: 2, metric: "devices", range: "90" };
export const FREQUENT_FLIERS_RANGE_OPTIONS = [
  { value: "30", label: "30 days" },
  { value: "60", label: "60 days" },
  { value: "90", label: "90 days" },
  { value: "180", label: "6 months" },
  { value: "365", label: "1 year" },
];

export const FREQUENT_FLIER_ROWS = [
  {
    id: "jason-rodriguez",
    student: "Jason Rodriguez",
    studentId: "3504011",
    grade: "10",
    site: "Desert View",
    deviceAssignments: 4,
    linkedTickets: 3,
    assignmentCountsByRange: { 30: 1, 60: 3, 90: 4, 180: 4, 365: 5 },
    ticketCountsByRange: { 30: 0, 60: 2, 90: 3, 180: 3, 365: 4 },
    daysSinceLastTicket: 42,
    trend: [1, 2, 3, 4, 4, 3],
    note: "Multiple physical damage incidents in the selected range. Tech Support follow-up scheduled for May 2, 2025.",
    devices: [
      { serial: "CLA-24-27891", type: "Chromebook", status: "Active" },
      { serial: "CLA-24-26412", type: "Chromebook", status: "Returned" },
      { serial: "CLA-24-25103", type: "Chromebook", status: "Returned" },
      { serial: "CLA-24-23987", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1782345", summary: "Broken Screen", status: "Closed" },
      { id: "INC-1758991", summary: "Broken Hinge", status: "Closed" },
      { id: "INC-1741123", summary: "Keyboard Not Working", status: "Closed" },
    ],
  },
  {
    id: "maria-nguyen",
    student: "Maria Nguyen",
    studentId: "3501187",
    grade: "9",
    site: "Clover High School",
    deviceAssignments: 3,
    linkedTickets: 2,
    assignmentCountsByRange: { 30: 2, 60: 2, 90: 3, 180: 3, 365: 4 },
    ticketCountsByRange: { 30: 1, 60: 2, 90: 2, 180: 3, 365: 3 },
    daysSinceLastTicket: 18,
    trend: [0, 1, 1, 2, 3, 3],
    note: "Two recent device exchanges are tied to keyboard damage. Library staff should confirm storage expectations.",
    devices: [
      { serial: "CLA-24-19912", type: "Chromebook", status: "Active" },
      { serial: "CLA-24-18801", type: "Chromebook", status: "Returned" },
      { serial: "CLA-24-17144", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1780042", summary: "Keyboard Damage", status: "Closed" },
      { id: "INC-1776620", summary: "Loaner Exchange", status: "Closed" },
    ],
  },
  {
    id: "devon-price",
    student: "Devon Price",
    studentId: "3502235",
    grade: "11",
    site: "Franklin Middle School",
    deviceAssignments: 2,
    linkedTickets: 4,
    assignmentCountsByRange: { 30: 1, 60: 2, 90: 2, 180: 3, 365: 3 },
    ticketCountsByRange: { 30: 2, 60: 3, 90: 4, 180: 4, 365: 5 },
    daysSinceLastTicket: 9,
    trend: [1, 1, 2, 2, 3, 4],
    note: "Ticket volume is higher than assignment count. Review charger and case notes before assigning a replacement.",
    devices: [
      { serial: "FMS-24-11098", type: "Chromebook", status: "Active" },
      { serial: "FMS-24-10870", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1784210", summary: "Screen Flicker", status: "Open" },
      { id: "INC-1783022", summary: "Missing Charger", status: "Closed" },
      { id: "INC-1779234", summary: "Case Damage", status: "Closed" },
      { id: "INC-1774122", summary: "Replacement Request", status: "Closed" },
    ],
  },
  {
    id: "sophia-patel",
    student: "Sophia Patel",
    studentId: "3503407",
    grade: "8",
    site: "Canyon Ridge",
    deviceAssignments: 2,
    linkedTickets: 1,
    assignmentCountsByRange: { 30: 0, 60: 0, 90: 2, 180: 2, 365: 2 },
    ticketCountsByRange: { 30: 0, 60: 0, 90: 1, 180: 1, 365: 2 },
    daysSinceLastTicket: 61,
    trend: [0, 0, 1, 1, 2, 2],
    note: "Meets the assignment threshold only. No recent ticket pattern is visible.",
    devices: [
      { serial: "CRM-24-09418", type: "Chromebook", status: "Active" },
      { serial: "CRM-24-08064", type: "Chromebook", status: "Returned" },
    ],
    tickets: [{ id: "INC-1768019", summary: "Cracked Bezel", status: "Closed" }],
  },
  {
    id: "eli-washington",
    student: "Eli Washington",
    studentId: "3505128",
    grade: "12",
    site: "District Office",
    deviceAssignments: 1,
    linkedTickets: 2,
    assignmentCountsByRange: { 30: 1, 60: 1, 90: 1, 180: 2, 365: 2 },
    ticketCountsByRange: { 30: 1, 60: 2, 90: 2, 180: 2, 365: 3 },
    daysSinceLastTicket: 27,
    trend: [0, 0, 1, 1, 1, 2],
    note: "Ticket threshold match without repeated assignments. Review before escalating.",
    devices: [{ serial: "DO-24-03117", type: "Chromebook", status: "Active" }],
    tickets: [
      { id: "INC-1779901", summary: "Power Issue", status: "Closed" },
      { id: "INC-1774488", summary: "Trackpad Issue", status: "Closed" },
    ],
  },
];

const DEVICE_LINK_BASE = "https://mock.wusd.local/incidentiq/assets";
const TICKET_LINK_BASE = "https://mock.wusd.local/incidentiq/tickets";

/**
 * rangeLabelForValue keeps the user-facing lookback labels in one place so the
 * filter control, drawer rule summary, and fallback behavior stay aligned when
 * the documented Frequent Fliers range set changes.
 */
export function rangeLabelForValue(range) {
  return FREQUENT_FLIERS_RANGE_OPTIONS.find((option) => option.value === range)?.label || "90 days";
}

/**
 * metricCountForRange reads the DEV mock count for the selected Frequent
 * Fliers lookback. The route is frontend/static-only in this slice, so this
 * helper is the local equivalent of the future API range parameter.
 */
export function metricCountForRange(row, metric, range) {
  const counts = metric === "tickets" ? row.ticketCountsByRange : row.assignmentCountsByRange;
  const fallback = metric === "tickets" ? row.linkedTickets : row.deviceAssignments;
  return counts?.[range] ?? fallback;
}

/**
 * frequentFliersRowsForFilters applies the documented fixed greater-than-or-
 * equal comparison after a user commits threshold, metric, and lookback values.
 */
export function frequentFliersRowsForFilters(rows, filters) {
  return rows.filter((row) => metricCountForRange(row, filters.metric, filters.range) >= filters.threshold);
}

/**
 * linkForDevice builds deterministic DEV IncidentIQ asset URLs for drawer rows;
 * it never contacts IncidentIQ and keeps demo navigation free of live provider
 * credentials or production identifiers.
 */
export function linkForDevice(serial) {
  return `${DEVICE_LINK_BASE}/${encodeURIComponent(serial)}`;
}

/**
 * linkForTicket builds deterministic DEV IncidentIQ ticket URLs for the row
 * drawer, matching the documented mock-link behavior without a live ticketing
 * read.
 */
export function linkForTicket(ticketId) {
  return `${TICKET_LINK_BASE}/${encodeURIComponent(ticketId)}`;
}

/**
 * trendClass maps a trend bucket to the shared severity palette relative to the
 * committed threshold, letting the table show below-threshold, review, and
 * critical patterns without adding another status field to the mock rows.
 */
export function trendClass(value, threshold) {
  if (value >= threshold + 2) {
    return "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--critical";
  }
  if (value >= threshold) {
    return "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--review";
  }
  return "frequent-fliers-runtime__trend-bar frequent-fliers-runtime__trend-bar--ready";
}
