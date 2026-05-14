import { useCallback, useMemo, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { PenArtboard } from "../lib/PenArtboard";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";

const ARTBOARD_KEY = "student-data-cleanup";
const STUDENT_DATA_HEADING_ID = "student-data-cleanup-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const PANE_WIDTH = 1348;
const DRAWER_BOUNDS = { left: 1278, top: 92, width: 390, height: 802 };
const AERIES_LINK_BASE = "https://mock.wusd.local/aeries";

const STUDENT_DATA_HELP_CONTENT = {
  title: "Student Data Cleanup help",
  sections: [
    {
      heading: "What this page shows",
      body:
        "This page lists active, unresolved student name issues detected during sync. These records can affect account creation or downstream matching if they are not corrected in Aeries.",
    },
    {
      heading: "How to use it",
      body:
        "Use table search, issue type, grade filters, and column sorting to find a record. Select a row to compare the current Aeries name values with the suggested values to paste back into Aeries.",
    },
    {
      heading: "Where corrections happen",
      body:
        "This dashboard is informational only. Student records cannot be edited here. All corrections must be made in Aeries, then the sync button can be used to recheck the queue.",
    },
  ],
};

// Mirrors the legacy Aeries AD sync invalid-name gate:
// after removing ordinary spaces, first + last name must match /^[a-zA-Z]+$/,
// and neither first nor last name may begin or end with a space.
const STUDENT_ROWS = [
  {
    id: "carlos-nuno",
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

const STUDENT_COLUMNS = [
  { key: "studentId", label: "Student ID", value: (row) => row.studentId },
  { key: "studentName", label: "Student Name", value: (row) => row.studentName },
  {
    key: "firstNameRaw",
    label: "Current first name",
    value: (row) => row.firstNameRaw,
  },
  {
    key: "firstNameClean",
    label: "Suggested first name",
    value: (row) => row.firstNameClean,
  },
  {
    key: "lastNameRaw",
    label: "Current last name",
    value: (row) => row.lastNameRaw,
  },
  {
    key: "lastNameClean",
    label: "Suggested last name",
    value: (row) => row.lastNameClean,
  },
  { key: "issueType", label: "Issue Type", value: (row) => row.issueType },
  { key: "grade", label: "Grade", value: (row) => row.grade, sortValue: (row) => Number(row.grade) },
  { key: "submitted", label: "Submitted", value: (row) => row.submitted },
];

/**
 * collectAllNodeIds appends a generated artboard subtree to the hidden-node list used by StudentDataCleanupPage. It receives one .pen node plus the accumulator owned by collectPaneNodeIds, and it returns through that accumulator so the runtime table, filters, and drawer can replace the static page-pane artwork without hiding shared shell nodes.
 */
function collectAllNodeIds(node, ids) {
  ids.push(node.id);
  for (const child of node.children || []) {
    collectAllNodeIds(child, ids);
  }
}

/**
 * collectPaneNodeIds finds the generated Student Data Cleanup pane descendants that should be hidden under the runtime overlay. StudentDataCleanupPage calls it after loading the generated artboard; the returned ids preserve the shared shell while preventing duplicate static filters, rows, helper copy, or stale labels from rendering behind the live React controls.
 */
function collectPaneNodeIds(node, ids = []) {
  const isPaneNode = (node.x ?? 0) >= 280 && (node.y ?? 0) >= 88;
  if (isPaneNode) {
    collectAllNodeIds(node, ids);
    return ids;
  }
  for (const child of node.children || []) {
    collectPaneNodeIds(child, ids);
  }
  return ids;
}

/**
 * aeriesLink returns the configured Aeries base site used by StudentDataDrawer. The Student Data Cleanup route links to the general Aeries site because the dashboard cannot safely deep-link to a specific student record; operators use the displayed Student ID after the new tab opens.
 */
function aeriesLink() {
  return AERIES_LINK_BASE;
}

/**
 * uniqueValues derives the issue-type and grade filter options from the current DEV mock rows. StudentDataOverlay calls it during render so the select controls reflect the available queue data while preserving the source values displayed in each row.
 */
function uniqueValues(rows, key) {
  return [...new Set(rows.map((row) => row[key]))].sort((left, right) =>
    String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: "base" })
  );
}

/**
 * StudentDataDrawer shows the selected Student Data Cleanup row in the shared right-hand drawer. StudentDataOverlay passes the selected row here; the drawer returns read-only current and suggested Aeries name values, explains that edits happen in Aeries, and links only to the base Aeries site so the page does not imply a record-level deep link or local student-data write path.
 */
function StudentDataDrawer({ row, onClose }) {
  if (!row) {
    return null;
  }
  return (
    <RuntimeDrawer title={row.studentName} bounds={DRAWER_BOUNDS} onClose={onClose}>
      <RuntimeDetailList
        items={[
          { label: "Student ID", value: row.studentId },
          { label: "Grade", value: row.grade },
          { label: "Issue Type", value: row.issueType },
          { label: "Submitted", value: row.submitted },
          { label: "Current first name", value: row.firstNameRaw },
          { label: "Suggested first name", value: row.firstNameClean },
          { label: "Current last name", value: row.lastNameRaw },
          { label: "Suggested last name", value: row.lastNameClean },
        ]}
      />
      <div className="runtime-drawer__section">
        <p>
          <strong>Source system</strong>
          <span>
            Corrections must be made in Aeries. Open Aeries in a new tab, search for this Student ID, and update the
            source name fields there. This dashboard cannot edit student data.
          </span>
        </p>
        <a className="student-data-runtime__aeries-link" href={aeriesLink()} target="_blank" rel="noreferrer">
          Open Aeries
        </a>
      </div>
    </RuntimeDrawer>
  );
}

/**
 * StudentDataOverlay owns the live Student Data Cleanup table over the generated .pen shell. StudentDataCleanupPage supplies DEV mock rows, filter state, row-selection handlers, and sync button state; this component returns the searchable/sortable runtime table and keeps all displayed current-name values faithful to the Aeries source strings.
 */
function StudentDataOverlay({
  rows,
  selectedRowId,
  filters,
  onFilterChange,
  onClearFilters,
  onSelectRow,
  onSync,
  syncStatus,
}) {
  const columns = useMemo(() => STUDENT_COLUMNS, []);
  const filteredRows = useMemo(() => {
    return rows.filter((row) => {
      const issueMatches = filters.issueType === "all" || row.issueType === filters.issueType;
      const gradeMatches = filters.grade === "all" || row.grade === filters.grade;
      return issueMatches && gradeMatches;
    });
  }, [filters, rows]);
  const table = useRuntimeTableData(filteredRows, columns, {
    defaultSort: { key: "studentId", direction: "asc" },
  });
  const visibleCount = table.visibleRows.length;
  const totalCount = rows.length;
  return (
    <section
      className="student-data-runtime"
      style={{
        position: "absolute",
        left: PANE_LEFT,
        top: PANE_TOP,
        width: PANE_WIDTH,
        zIndex: 2,
      }}
      aria-labelledby={STUDENT_DATA_HEADING_ID}
    >
      <header className="student-data-runtime__header">
        <div>
          <h1 id={STUDENT_DATA_HEADING_ID}>Student Data Cleanup</h1>
          <p>Review active student-name field issues that must be corrected in Aeries.</p>
        </div>
        <button type="button" className="student-data-runtime__sync" onClick={onSync}>
          {syncStatus}
        </button>
      </header>
      <section className="student-data-runtime__summary" aria-label="Student data cleanup summary">
        <div>
          <strong>{totalCount} active issues</strong>
          <span>All must be corrected in Aeries.</span>
        </div>
        <div>
          <strong>Last sync</strong>
          <span>May 2, 2025 9:05 AM PT</span>
        </div>
        <div>
          <strong>Next sync</strong>
          <span>in 55 minutes</span>
        </div>
      </section>
      <div className="student-data-runtime__table-card">
        <div className="student-data-runtime__toolbar">
          <RuntimeTableSearch
            value={table.searchQuery}
            onChange={table.setSearchQuery}
            placeholder="Search by student name, student ID, issue type, or grade..."
          />
          <label>
            <span>Issue type</span>
            <select value={filters.issueType} onChange={(event) => onFilterChange({ issueType: event.target.value })}>
              <option value="all">All issues</option>
              {uniqueValues(rows, "issueType").map((issueType) => (
                <option key={issueType} value={issueType}>
                  {issueType}
                </option>
              ))}
            </select>
          </label>
          <label>
            <span>Grade</span>
            <select value={filters.grade} onChange={(event) => onFilterChange({ grade: event.target.value })}>
              <option value="all">All grades</option>
              {uniqueValues(rows, "grade").map((grade) => (
                <option key={grade} value={grade}>
                  {grade}
                </option>
              ))}
            </select>
          </label>
          <button type="button" onClick={onClearFilters}>
            Clear filters
          </button>
        </div>
        <div className="student-data-runtime__table-header">
          {columns.map((column) => (
            <div key={column.key}>
              <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
            </div>
          ))}
        </div>
        <div className="student-data-runtime__table-body">
          {table.visibleRows.map((row) => (
            <button
              key={row.id}
              type="button"
              className={`student-data-runtime__row ${
                selectedRowId === row.id ? "student-data-runtime__row--selected" : ""
              }`}
              aria-label={`Open student data cleanup row for ${row.studentName}`}
              aria-pressed={selectedRowId === row.id}
              onClick={() => onSelectRow(row)}
            >
              <div>{row.studentId}</div>
              <div>{row.studentName}</div>
              <div>{row.firstNameRaw}</div>
              <div>{row.firstNameClean}</div>
              <div>{row.lastNameRaw}</div>
              <div>{row.lastNameClean}</div>
              <div>{row.issueType}</div>
              <div>{row.grade}</div>
              <div>{row.submitted}</div>
            </button>
          ))}
          {!visibleCount ? (
            <div className="student-data-runtime__empty">No active student data cleanup rows match the current filters.</div>
          ) : null}
        </div>
        <div className="student-data-runtime__footer">
          Showing {visibleCount ? 1 : 0} to {visibleCount} of {totalCount} issues
        </div>
      </div>
    </section>
  );
}

/**
 * StudentDataCleanupPage is the /student-data-cleanup route rendered by frontend/src/app.jsx after route authorization. It loads the generated artboard shell, hides the obsolete static pane, renders runtime-owned filters/table/drawer behavior, and keeps the page informational only: the local sync button changes temporary UI state, but no student record or provider data is written.
 */
export function StudentDataCleanupPage({ session, onNavigate, onSearch, searchQuery }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [filters, setFilters] = useState({ issueType: "all", grade: "all" });
  const [selectedRow, setSelectedRow] = useState(null);
  const [syncStatus, setSyncStatus] = useState("Sync now");
  const textOverrides = buildSharedShellTextOverrides(session);
  const paneNodeIds = useMemo(() => artboard ? collectPaneNodeIds(artboard) : [], [artboard]);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...paneNodeIds);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? "studentDataCleanup",
    refreshMetadata: staticRefreshMetadataForArtboard(ARTBOARD_KEY),
    helpContent: STUDENT_DATA_HELP_CONTENT,
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Student Data Cleanup",
        textOverrides,
      })
    : { title: "Student Data Cleanup", items: [] };
  const selectedPayloadRow = selectedRow ? STUDENT_ROWS.find((row) => row.id === selectedRow.id) || selectedRow : null;

  const handleFilterChange = useCallback((change) => {
    setFilters((current) => ({ ...current, ...change }));
    setSelectedRow(null);
  }, []);
  const handleClearFilters = useCallback(() => {
    setFilters({ issueType: "all", grade: "all" });
    setSelectedRow(null);
  }, []);
  const handleSync = useCallback(() => {
    setSyncStatus("Synced");
    window.setTimeout(() => setSyncStatus("Sync now"), 1400);
  }, []);
  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <StudentDataOverlay
        rows={STUDENT_ROWS}
        selectedRowId={selectedPayloadRow?.id}
        filters={filters}
        onFilterChange={handleFilterChange}
        onClearFilters={handleClearFilters}
        onSelectRow={setSelectedRow}
        onSync={handleSync}
        syncStatus={syncStatus}
      />
      <StudentDataDrawer row={selectedPayloadRow} onClose={() => setSelectedRow(null)} />
    </>
  ), [filters, handleClearFilters, handleFilterChange, handleSync, selectedPayloadRow, sharedShellRenderOverlay, syncStatus]);

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Student Data Cleanup</h1>
          <p>Preparing the generated Student Data Cleanup artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Student Data Cleanup unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={STUDENT_DATA_HEADING_ID}>
      <section className="sr-only" aria-labelledby={`${STUDENT_DATA_HEADING_ID}-summary`}>
        <h1 id={`${STUDENT_DATA_HEADING_ID}-summary`}>{semanticSummary.title}</h1>
        <ul>
          {semanticSummary.items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      </section>
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
      </div>
    </main>
  );
}
