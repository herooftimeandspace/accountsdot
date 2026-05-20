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
  {
    id: "noah-kim",
    student: "Noah Kim",
    studentId: "3506044",
    grade: "7",
    site: "Clover High School",
    deviceAssignments: 6,
    linkedTickets: 2,
    assignmentCountsByRange: { 30: 3, 60: 5, 90: 6, 180: 8, 365: 9 },
    ticketCountsByRange: { 30: 1, 60: 1, 90: 2, 180: 2, 365: 3 },
    daysSinceLastTicket: 33,
    trend: [2, 3, 5, 6, 8, 9],
    note: "High assignment velocity across the year. Confirm whether repeated swaps point to storage, case, or checkout-process issues.",
    devices: [
      { serial: "CHS-24-30018", type: "Chromebook", status: "Active" },
      { serial: "CHS-24-29814", type: "Chromebook", status: "Returned" },
      { serial: "CHS-24-28755", type: "Chromebook", status: "Returned" },
      { serial: "CHS-24-27240", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1785128", summary: "Repeated Checkout", status: "Open" },
      { id: "INC-1778044", summary: "Damaged Case", status: "Closed" },
    ],
  },
  {
    id: "aaliyah-brooks",
    student: "Aaliyah Brooks",
    studentId: "3507112",
    grade: "6",
    site: "Desert View",
    deviceAssignments: 1,
    linkedTickets: 5,
    assignmentCountsByRange: { 30: 0, 60: 1, 90: 1, 180: 1, 365: 2 },
    ticketCountsByRange: { 30: 3, 60: 4, 90: 5, 180: 6, 365: 7 },
    daysSinceLastTicket: 5,
    trend: [1, 3, 4, 5, 6, 7],
    note: "Ticket-heavy pattern without many assignment swaps. Check whether recurring charger and case tickets need a different intervention.",
    devices: [
      { serial: "DV-24-22770", type: "Chromebook", status: "Active" },
      { serial: "DV-24-22164", type: "Chromebook", status: "Returned" },
    ],
    tickets: [
      { id: "INC-1786901", summary: "Charger Missing", status: "Open" },
      { id: "INC-1785604", summary: "Case Damage", status: "Closed" },
      { id: "INC-1781229", summary: "Won't Charge", status: "Closed" },
      { id: "INC-1779055", summary: "Keyboard Cover Missing", status: "Closed" },
      { id: "INC-1775120", summary: "Loaner Support", status: "Closed" },
    ],
  },
  {
    id: "omar-castillo",
    student: "Omar Castillo",
    studentId: "3508449",
    grade: "10",
    site: "Franklin Middle School",
    deviceAssignments: 3,
    linkedTickets: 1,
    assignmentCountsByRange: { 30: 2, 60: 2, 90: 3, 180: 5, 365: 6 },
    ticketCountsByRange: { 30: 0, 60: 0, 90: 1, 180: 3, 365: 4 },
    daysSinceLastTicket: 74,
    trend: [0, 1, 2, 3, 5, 6],
    note: "Longer lookbacks expose repeated assignments that are not obvious in the shorter ticket view.",
    devices: [
      { serial: "FMS-24-32108", type: "Chromebook", status: "Active" },
      { serial: "FMS-24-31944", type: "Chromebook", status: "Returned" },
      { serial: "FMS-24-30052", type: "Chromebook", status: "Returned" },
    ],
    tickets: [{ id: "INC-1769442", summary: "Cracked Shell", status: "Closed" }],
  },
];

export const FREQUENT_FLIERS_REPRESENTATIVE_COMBINATIONS = [
  { threshold: 2, metric: "devices", range: "90", reason: "default device-assignment queue" },
  { threshold: 3, metric: "devices", range: "30", reason: "strict short-window device review" },
  { threshold: 4, metric: "devices", range: "365", reason: "strict full-year device review" },
  { threshold: 2, metric: "tickets", range: "30", reason: "short-window ticket queue" },
  { threshold: 4, metric: "tickets", range: "365", reason: "full-year ticket escalation queue" },
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
 * equal comparison to the active threshold, metric, and lookback values. The
 * Frequent Fliers route calls this helper on every dropdown change, so the DEV
 * mock table refreshes without a separate Apply step.
 */
export function frequentFliersRowsForFilters(rows, filters) {
  return rows.filter((row) => metricCountForRange(row, filters.metric, filters.range) >= filters.threshold);
}

/**
 * frequentFliersCombinationSignature gives tests and Browser notes a compact
 * way to compare visible DEV mock rows for representative filter combinations
 * without depending on table rendering internals.
 */
export function frequentFliersCombinationSignature(rows, filters) {
  return frequentFliersRowsForFilters(rows, filters).map((row) => row.id).join("|");
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
