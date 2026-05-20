export const SITE_SCOPED_STUDENT_DATA_PERSONAS = new Set(["site_admin", "site_secretary"]);

// Mirrors the legacy Aeries AD sync invalid-name gate:
// after removing ordinary spaces, first + last name must match /^[a-zA-Z]+$/,
// and neither first nor last name may begin or end with a space. siteId is
// used only for DEV visibility scoping and is intentionally not a table column.
export const STUDENT_DATA_CLEANUP_ROWS = [
  {
    id: "carlos-nuno",
    siteId: "clover-hs",
    siteName: "Clover High School",
    studentId: "0001021",
    studentName: "Carlos Nuno",
    firstNameRaw: "Carlos",
    lastNameRaw: "Nuño",
    firstNameClean: "Carlos",
    lastNameClean: "Nuno",
    issueType: "Invalid character",
    grade: "11",
    submitted: "May 2, 2025 8:58 AM PT",
  },
  {
    id: "alex-oneil",
    siteId: "desert-view",
    siteName: "Desert View",
    studentId: "0001087",
    studentName: "Alex O'Neil",
    firstNameRaw: "Alex",
    lastNameRaw: "O'Neil",
    firstNameClean: "Alex",
    lastNameClean: "ONeil",
    issueType: "Invalid character",
    grade: "10",
    submitted: "May 2, 2025 8:56 AM PT",
  },
  {
    id: "jose-martinez",
    siteId: "clover-hs",
    siteName: "Clover High School",
    studentId: "0001142",
    studentName: "Jose Martinez",
    firstNameRaw: "Jose",
    lastNameRaw: "Martínez",
    firstNameClean: "Jose",
    lastNameClean: "Martinez",
    issueType: "Invalid character",
    grade: "12",
    submitted: "May 2, 2025 8:54 AM PT",
  },
  {
    id: "taylor-smith-jones",
    siteId: "franklin-ms",
    siteName: "Franklin Middle School",
    studentId: "0001233",
    studentName: "Taylor Smith-Jones",
    firstNameRaw: "Taylor",
    lastNameRaw: "Smith-Jones",
    firstNameClean: "Taylor",
    lastNameClean: "SmithJones",
    issueType: "Invalid character",
    grade: "9",
    submitted: "May 2, 2025 8:52 AM PT",
  },
  {
    id: "ava-oneill",
    siteId: "clover-hs",
    siteName: "Clover High School",
    studentId: "0001294",
    studentName: "Ava ONeill",
    firstNameRaw: "Ava",
    lastNameRaw: "O’Neill",
    firstNameClean: "Ava",
    lastNameClean: "ONeill",
    issueType: "Smart punctuation",
    grade: "8",
    submitted: "May 2, 2025 8:49 AM PT",
  },
  {
    id: "noah-chenlee",
    siteId: "desert-view",
    siteName: "Desert View",
    studentId: "0001358",
    studentName: "Noah ChenLee",
    firstNameRaw: "Noah",
    lastNameRaw: "Chen-Lee",
    firstNameClean: "Noah",
    lastNameClean: "ChenLee",
    issueType: "Hyphen",
    grade: "7",
    submitted: "May 2, 2025 8:47 AM PT",
  },
  {
    id: "liam-carter",
    siteId: "clover-hs",
    siteName: "Clover High School",
    studentId: "0001399",
    studentName: "Liam Carter",
    firstNameRaw: "Liam2",
    lastNameRaw: "Carter",
    firstNameClean: "Liam",
    lastNameClean: "Carter",
    issueType: "Digit",
    grade: "6",
    submitted: "May 2, 2025 8:45 AM PT",
  },
  {
    id: "mila-obrien",
    siteId: "desert-view",
    siteName: "Desert View",
    studentId: "0001442",
    studentName: "Mila OBrien",
    firstNameRaw: "Mila",
    lastNameRaw: "OBrien#",
    firstNameClean: "Mila",
    lastNameClean: "OBrien",
    issueType: "Symbol",
    grade: "5",
    submitted: "May 2, 2025 8:43 AM PT",
  },
  {
    id: "erin-park",
    siteId: "clover-hs",
    siteName: "Clover High School",
    studentId: "0001488",
    studentName: "Erin Park",
    firstNameRaw: " Erin",
    lastNameRaw: "Park",
    firstNameClean: "Erin",
    lastNameClean: "Park",
    issueType: "Leading whitespace",
    grade: "4",
    submitted: "May 2, 2025 8:41 AM PT",
  },
  {
    id: "owen-reed",
    siteId: "franklin-ms",
    siteName: "Franklin Middle School",
    studentId: "0001527",
    studentName: "Owen Reed",
    firstNameRaw: "Owen ",
    lastNameRaw: "Reed",
    firstNameClean: "Owen",
    lastNameClean: "Reed",
    issueType: "Trailing whitespace",
    grade: "3",
    submitted: "May 2, 2025 8:39 AM PT",
  },
  {
    id: "zoe-kim",
    siteId: "clover-hs",
    siteName: "Clover High School",
    studentId: "0001583",
    studentName: "Zoe Kim",
    firstNameRaw: "Zoe",
    lastNameRaw: " Kim",
    firstNameClean: "Zoe",
    lastNameClean: "Kim",
    issueType: "Leading whitespace",
    grade: "2",
    submitted: "May 2, 2025 8:37 AM PT",
  },
  {
    id: "ivy-stone",
    siteId: "desert-view",
    siteName: "Desert View",
    studentId: "0001614",
    studentName: "Ivy Stone",
    firstNameRaw: "Ivy",
    lastNameRaw: "Stone ",
    firstNameClean: "Ivy",
    lastNameClean: "Stone",
    issueType: "Trailing whitespace",
    grade: "1",
    submitted: "May 2, 2025 8:35 AM PT",
  },
];

function urlSiteId() {
  if (typeof window === "undefined") {
    return "";
  }
  return new URLSearchParams(window.location.search).get("site_id")?.trim() ?? "";
}

/**
 * effectiveStudentDataCleanupSiteId resolves the site boundary for the frontend-only Student Data Cleanup mock row set.
 * StudentDataCleanupPage calls it whenever the DEV session or header scope changes; district-wide personas intentionally
 * return an empty site so they keep the full district queue, while Site Admin and Site Secretary resolve to the active
 * visible site and fail closed when no assigned/current site is available.
 */
export function effectiveStudentDataCleanupSiteId(session) {
  const personaId = session?.current_persona?.id ?? "";
  if (!SITE_SCOPED_STUDENT_DATA_PERSONAS.has(personaId)) {
    return "";
  }

  const visibleSiteIds = new Set(
    (Array.isArray(session?.visible_sites) ? session.visible_sites : []).map((site) => site.id)
  );
  const requestedSiteId = urlSiteId();
  if (requestedSiteId && visibleSiteIds.has(requestedSiteId)) {
    return requestedSiteId;
  }
  if (session?.current_site_id && (!visibleSiteIds.size || visibleSiteIds.has(session.current_site_id))) {
    return session.current_site_id;
  }
  if (session?.default_site_id && (!visibleSiteIds.size || visibleSiteIds.has(session.default_site_id))) {
    return session.default_site_id;
  }
  return "";
}

/**
 * studentDataCleanupRowsForSession applies the documented Site Admin and Site Secretary site scope before page-local search,
 * filters, counts, footer totals, or drawer selection can see the mock rows. The route has no DEV API in this slice, so this
 * helper is the frontend equivalent of a future protected payload query.
 */
export function studentDataCleanupRowsForSession(rows, session) {
  const effectiveSiteId = effectiveStudentDataCleanupSiteId(session);
  const personaId = session?.current_persona?.id ?? "";
  if (!SITE_SCOPED_STUDENT_DATA_PERSONAS.has(personaId)) {
    return rows;
  }
  if (!effectiveSiteId) {
    return [];
  }
  return rows.filter((row) => row.siteId === effectiveSiteId);
}
