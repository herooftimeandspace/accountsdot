import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { RuntimeCombobox } from "../components/RuntimeDropdown";
import { RuntimeSortableHeader, RuntimeTableSearch, useRuntimeTableData } from "../components/RuntimeTableControls";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { PenArtboard } from "../lib/PenArtboard";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";

const ROOM_MOVES_ENDPOINT = "/api/v1/dev/pages/room-moves";
const ROOM_MOVES_BULK_ENDPOINT = "/api/v1/dev/pages/room-moves/bulk-draft";
const ROOM_MOVES_DRAFTS_ENDPOINT = "/api/v1/dev/room-moves/drafts";
const ROOM_MOVES_HEADING_ID = "room-moves-heading";
const ROOM_MOVES_TABLE_COLUMNS = [
  { key: "person", label: "Name", value: (row) => row.person },
  { key: "current_room", label: "Current", value: (row) => row.current_room },
  { key: "destination_room", label: "Target", value: (row) => row.destination_room },
  { key: "phone", label: "Phone", value: (row) => row.phone || "No phone" },
  { key: "author", label: "Author", value: (row) => row.author || "DEV mock" },
  { key: "state", label: "State", value: (row) => row.state },
];
const BULK_COLUMNS = [
  { key: "person", label: "Person", value: (row) => [row.person, row.email, row.phone].join(" ") },
  { key: "current_room", label: "Current Room", value: (row) => row.current_room },
  { key: "destination_site", label: "Destination Site", value: (row) => row.destination_site },
  { key: "destination_room", label: "Destination Room", value: (row) => row.destination_room },
  { key: "action", label: "Action", value: (row) => row.action },
];
const HIDDEN_ROOM_MOVES_NODE_SUFFIXES = [
  "f92", "t93", "t94", "t95", "t96", "t97",
  "f100", "t101", "t102", "t103", "t104", "t105", "t106", "t107", "l108",
  "t109", "t110", "t111", "t112", "t113", "f114", "t115", "l116",
  "t117", "t118", "t119", "t120", "t121", "f122", "t123", "l124",
  "t125", "t126", "t127", "t128", "t129", "f130", "t131", "l132",
  "t133", "t134", "t135", "t136", "t137", "f138", "t139", "l140",
  "f142", "t143", "p144", "p145", "t146", "t147",
  "t148", "t149", "t150", "t151", "t152", "t153",
  "t154", "t155", "l156", "f157", "t158",
  "f159", "t160", "f161", "p162", "p163", "t164",
  "t162",
  "f300", "t301", "f302", "t303", "f304", "t305",
];
const HIDDEN_BULK_DRAFT_NODE_SUFFIXES = [];

function nodeIdForSuffix(artboardKey, suffix) {
  return `${artboardKey}__${suffix}`;
}

function hiddenRoomMovesNodeIds(artboardKey, isBulk) {
  const suffixes = isBulk
    ? [...HIDDEN_ROOM_MOVES_NODE_SUFFIXES, ...HIDDEN_BULK_DRAFT_NODE_SUFFIXES]
    : HIDDEN_ROOM_MOVES_NODE_SUFFIXES;
  return suffixes.map((suffix) => nodeIdForSuffix(artboardKey, suffix));
}

async function readJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(payload?.message || `Request failed with ${response.status}`);
    error.status = response.status;
    error.payload = payload;
    throw error;
  }
  return payload;
}

function nodeBox(node, fallback) {
  if (!node) {
    return fallback;
  }
  return {
    left: node.x ?? fallback.left,
    top: node.y ?? fallback.top,
    width: node.width ?? fallback.width,
    height: node.height ?? fallback.height,
  };
}

function statusClass(status) {
  if (["Ready", "Complete", "Allowed"].includes(status)) {
    return "room-moves-runtime__status room-moves-runtime__status--ready";
  }
  if (["Review", "Needs Review", "Manual action"].includes(status)) {
    return "room-moves-runtime__status room-moves-runtime__status--review";
  }
  if (["Scheduled", "Queued", "Waiting"].includes(status)) {
    return "room-moves-runtime__status room-moves-runtime__status--waiting";
  }
  return "room-moves-runtime__status room-moves-runtime__status--neutral";
}

function roomOptionsForSite(rooms, siteId) {
  const options = [];
  const seen = new Set();
  const addOption = (room) => {
    if (!room) {
      return;
    }
    const key = room.id === "none" ? "none" : `${room.site_id}:${room.id}`;
    if (seen.has(key)) {
      return;
    }
    seen.add(key);
    options.push(room);
  };

  addOption(rooms.find((room) => room.id === "none") || {
    id: "none",
    label: "None",
    site_id: siteId,
    site: "",
  });
  rooms
    .filter((room) => room.id !== "none" && room.site_id === siteId)
    .forEach(addOption);

  return options;
}

function personMatchesQuery(person, query) {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return true;
  }
  return [person.name, person.email, person.employee_id, person.role].some((value) =>
    String(value || "").toLowerCase().includes(normalized)
  );
}

function personAutocompleteLabel(person) {
  return `${person.name} · ${person.email} · ${person.employee_id}`;
}

function bulkPersonLabel(person) {
  const extension = person.phone ? `ext ${person.phone}` : "no extension";
  return `${person.name} · ${person.email} · ${extension}`;
}

function findPersonFromAutocompleteValue(people, value) {
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return null;
  }
  return people.find((person) => {
    const exactValues = [
      person.name,
      person.email,
      person.employee_id,
      personAutocompleteLabel(person),
    ];
    return exactValues.some((candidate) => String(candidate || "").toLowerCase() === normalized);
  }) || null;
}

function defaultDestinationRoom(person, destinationSiteId) {
  if (!person) {
    return "none";
  }
  return destinationSiteId === person.site_id ? person.current_room_id || "none" : "none";
}

/**
 * detailLines formats structured API detail fields for the shared drawer detail
 * list. Room Moves uses it for issue #54 primary-room conflict explanations so
 * backend-owned resolution steps and external systems stay readable without
 * creating a page-local drawer variant.
 */
function detailLines(values) {
  if (!Array.isArray(values) || values.length === 0) {
    return "";
  }
  return values.join("\n");
}

function RoomMovesStatusBadge({ status }) {
  return <span className={statusClass(status)}>{status}</span>;
}

function RoomMovesTable({ bounds, rows, selectedRowId, onSelectRow, onCancelRow, cancelingDraftId }) {
  const table = useRuntimeTableData(rows, ROOM_MOVES_TABLE_COLUMNS, {
    defaultSort: { key: "person", direction: "asc" },
  });

  if (!bounds) {
    return null;
  }

  return (
    <section
      className="room-moves-runtime__table"
      style={{ left: bounds.left, top: bounds.top, width: bounds.width, minHeight: bounds.height }}
      aria-labelledby={ROOM_MOVES_HEADING_ID}
    >
      <div className="room-moves-runtime__table-title">Move Set Review</div>
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="room-moves-runtime__table-header">
        {ROOM_MOVES_TABLE_COLUMNS.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
        <div>Cancel</div>
      </div>
      <div className="room-moves-runtime__table-body">
        {table.visibleRows.map((row) => (
          <div
            key={row.id}
            className={`room-moves-runtime__row ${
              selectedRowId === row.id ? "room-moves-runtime__row--selected" : ""
            }`}
          >
            <button
              type="button"
              className="room-moves-runtime__row-open"
              aria-label={`Open room move row for ${row.person}`}
              onClick={() => onSelectRow(row)}
            >
              <div>{row.person}</div>
              <div>{row.current_room}</div>
              <div>{row.destination_room}</div>
              <div>{row.phone}</div>
              <div>{row.author || "DEV mock"}</div>
              <div><RoomMovesStatusBadge status={row.state} /></div>
            </button>
            <button
              type="button"
              className="room-moves-runtime__delete room-moves-runtime__cancel-row"
              onClick={() => onCancelRow(row)}
              disabled={cancelingDraftId === row.draft_id}
            >
              Cancel Move
            </button>
          </div>
        ))}
      </div>
    </section>
  );
}

function RoomMovesActions({ bounds, onMovePerson, onBatchMove, onSiteRollover, busy }) {
  if (!bounds) {
    return null;
  }
  const width = 186;
  const left = Math.min(bounds.left + bounds.width + 36, 1440 - width - 24);
  return (
    <div
      className="room-moves-runtime__actions"
      style={{ left, top: bounds.top + 8, width }}
    >
      <button type="button" onClick={onMovePerson} disabled={busy}>Move Person</button>
      <button type="button" onClick={onBatchMove} disabled={busy}>Batch Move</button>
      <button type="button" onClick={onSiteRollover} disabled={busy}>Site Rollover</button>
    </div>
  );
}

function SingleMoveDrawer({ row, people, rooms, sites, canManageDistrict, onClose, onSaved }) {
  const initialPerson = people.find((person) => person.email === row?.email) || null;
  const [query, setQuery] = useState(initialPerson?.email || "");
  const [selectedPersonId, setSelectedPersonId] = useState(initialPerson?.id || "");
  const selectedPerson = people.find((person) => person.id === selectedPersonId) || null;
  const initialDestinationSiteId = row?.destination_site_id || selectedPerson?.site_id || row?.current_site_id || sites[0]?.id || "";
  const [destinationSiteId, setDestinationSiteId] = useState(initialDestinationSiteId);
  const [destinationRoomId, setDestinationRoomId] = useState(row?.destination_room_id || defaultDestinationRoom(selectedPerson, initialDestinationSiteId));
  const didPreserveInitialRowRoom = useRef(false);
  const [saving, setSaving] = useState(false);
  const [createdDraftId, setCreatedDraftId] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!selectedPerson) {
      return;
    }
    setDestinationSiteId((current) => current || selectedPerson.site_id);
  }, [selectedPerson]);

  useEffect(() => {
    if (row && !didPreserveInitialRowRoom.current) {
      didPreserveInitialRowRoom.current = true;
      return;
    }
    setDestinationRoomId(defaultDestinationRoom(selectedPerson, destinationSiteId));
  }, [destinationSiteId, row, selectedPerson]);

  const autocompleteOptions = people.filter((person) => personMatchesQuery(person, query));
  const availableRooms = roomOptionsForSite(rooms, destinationSiteId);

  function updatePersonQuery(value) {
    setQuery(value);
    const person = findPersonFromAutocompleteValue(people, value);
    setSelectedPersonId(person?.id || "");
  }

  function applyPersonValue(value) {
    setQuery(value);
    const person = findPersonFromAutocompleteValue(people, value);
    if (!person) {
      setSelectedPersonId("");
      return;
    }
    setSelectedPersonId(person.id);
    setQuery(person.email);
    setDestinationSiteId(person.site_id);
  }

  async function saveDraft(action = "save") {
    if (!selectedPerson) {
      setError("Select a person before saving the draft.");
      return;
    }
    setSaving(true);
    setError("");
    try {
      const response = await readJSON(
        await fetch(ROOM_MOVES_DRAFTS_ENDPOINT, {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({
            mode: "mid_year_targeted_move",
            person_id: selectedPerson.id,
            rows: [
              {
                person_id: selectedPerson.id,
                destination_site_id: destinationSiteId,
                destination_room_id: destinationRoomId,
              },
            ],
          }),
        })
      );
      setCreatedDraftId(response.draft.id);
      if (action === "schedule" || action === "apply") {
        const transition = await readJSON(
          await fetch(`${ROOM_MOVES_DRAFTS_ENDPOINT}/${response.draft.id}/${action}`, {
            method: "POST",
            credentials: "same-origin",
            headers: { Accept: "application/json" },
          })
        );
        onSaved(transition.draft);
      } else {
        onSaved(response.draft);
      }
    } catch (saveError) {
      setError(saveError.payload?.errors ? Object.values(saveError.payload.errors).join(" ") : saveError.message);
    } finally {
      setSaving(false);
    }
  }

  async function cancelDraft() {
    if (createdDraftId) {
      try {
        await fetch(`${ROOM_MOVES_DRAFTS_ENDPOINT}/${createdDraftId}`, {
          method: "DELETE",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        });
      } catch {
        // Cancel should still close the draft drawer if the cleanup request fails.
      }
    }
    onClose();
  }

  return (
    <RuntimeDrawer title={row ? row.person : "Move Person"} onClose={onClose}>
      {row?.warning ? (
        <div className="room-moves-runtime__warning-bar">
          <strong>Warning</strong>
          <p>{row.warning}</p>
        </div>
      ) : null}
      <RuntimeDetailList
        items={[
          { label: "State", value: row?.state },
          { label: "Author", value: row?.author },
          { label: "Current room", value: selectedPerson?.current_room || row?.current_room },
          { label: "Current site", value: selectedPerson?.site || row?.current_site },
          { label: "Target room", value: row?.destination_room },
          { label: "Target site", value: row?.destination_site },
          { label: "Current phone", value: selectedPerson?.phone },
          { label: "Phone outcome", value: row?.phone },
          { label: "Reason", value: row?.attention_reason },
          { label: "Automation", value: row?.automation_outcome },
          { label: "Manual owner", value: row?.manual_action_owner },
          { label: "Manual reason", value: row?.manual_action_reason },
          { label: "Resolution steps", value: detailLines(row?.resolution_steps) },
          { label: "External systems", value: detailLines(row?.external_systems) },
        ]}
      />
      <div className="runtime-drawer__section">
        <label className="room-moves-runtime__field" htmlFor="room-move-person-search">
          <span>Employee ID, email, or name</span>
          <RuntimeCombobox
            inputId="room-move-person-search"
            label="Employee ID, email, or name"
            value={query}
            options={autocompleteOptions.map((person) => ({
              value: personAutocompleteLabel(person),
              label: personAutocompleteLabel(person),
            }))}
            onInput={updatePersonQuery}
            onCommit={applyPersonValue}
            placeholder="Search"
          />
        </label>
        {canManageDistrict ? (
          <label className="room-moves-runtime__field" htmlFor="room-move-destination-site">
            <span>Destination site</span>
            <select
              id="room-move-destination-site"
              value={destinationSiteId}
              onChange={(event) => setDestinationSiteId(event.target.value)}
            >
              {sites.map((site) => (
                <option key={site.id} value={site.id}>{site.name}</option>
              ))}
            </select>
          </label>
        ) : null}
        <label className="room-moves-runtime__field" htmlFor="room-move-destination-room">
          <span>Destination room</span>
          <select
            id="room-move-destination-room"
            value={destinationRoomId}
            onChange={(event) => setDestinationRoomId(event.target.value)}
          >
            {availableRooms.map((room) => (
              <option key={`${room.site_id}-${room.id}`} value={room.id}>{room.label}</option>
            ))}
          </select>
        </label>
        {error ? <p className="room-moves-runtime__error">{error}</p> : null}
        <div className="room-moves-runtime__drawer-actions">
          <button type="button" onClick={() => saveDraft("save")} disabled={saving}>Save Draft</button>
          <button type="button" onClick={() => saveDraft("schedule")} disabled={saving}>Schedule</button>
          <button type="button" onClick={() => saveDraft("apply")} disabled={saving}>Apply</button>
          <button type="button" className="room-moves-runtime__delete" onClick={cancelDraft} disabled={saving}>Cancel</button>
        </div>
      </div>
    </RuntimeDrawer>
  );
}

function BulkDraftTable({ bounds, page, onSave, onTransition, onDelete }) {
  const draft = page.draft;
  const [rows, setRows] = useState(draft.rows || []);
  const [effectiveDate, setEffectiveDate] = useState(draft.effective_date || "2026-07-27");
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const table = useRuntimeTableData(rows, BULK_COLUMNS, {
    defaultSort: { key: "person", direction: "asc" },
  });

  useEffect(() => {
    setRows(draft.rows || []);
    setEffectiveDate(draft.effective_date || "2026-07-27");
    setDirty(false);
  }, [draft]);

  async function save(nextRows = rows, nextEffectiveDate = effectiveDate) {
    setSaving(true);
    try {
      await onSave({ ...draft, rows: nextRows, effective_date: nextEffectiveDate });
      setDirty(false);
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    if (!dirty) {
      return undefined;
    }
    const timer = window.setInterval(() => {
      void save(rows, effectiveDate);
    }, 60000);
    return () => window.clearInterval(timer);
  }, [dirty, effectiveDate, rows]);

  /**
   * updateRow owns editable bulk-draft row state before saveBulkDraft sends it to the DEV mock API.
   * The action branches mirror normalizeRoomMoveRows so selecting add/removal gives immediate visual feedback
   * and the saved/reloaded draft keeps the same room-clearing semantics.
   */
  function updateRow(rowId, patch) {
    const nextRows = rows.map((row) => {
      if (row.id !== rowId) {
        return row;
      }
      const next = { ...row, ...patch };
      const action = next.action || "change";
      if (patch.person_id) {
        const person = page.people.find((candidate) => candidate.id === patch.person_id);
        if (person) {
          next.person = person.name;
          next.email = person.email;
          next.employee_id = person.employee_id;
          next.phone = person.phone;
          next.current_site_id = person.site_id;
          next.current_site = person.site;
          next.current_room_id = person.current_room_id;
          next.current_room = person.current_room;
          next.destination_site_id = person.site_id;
          next.destination_site = person.site;
          next.destination_room_id = person.current_room_id || "none";
          next.destination_room = person.current_room || "None";
        }
      }
      if (action === "add") {
        next.current_room_id = "none";
        next.current_room = "";
      }
      if (action === "removal") {
        next.destination_room_id = "none";
      }
      const site = page.sites.find((candidate) => candidate.id === next.destination_site_id);
      const room = roomOptionsForSite(page.rooms, next.destination_site_id).find((candidate) => candidate.id === next.destination_room_id);
      next.destination_site = site?.name || next.destination_site;
      next.destination_room = room?.label || next.destination_room;
      return next;
    });
    setRows(nextRows);
    setDirty(true);
  }

  function addRow() {
    const nextRows = [
      ...rows,
      {
        id: `new-${Date.now()}`,
        person_id: "",
        person: "Select person",
        email: "",
        phone: "",
        employee_id: "",
        current_room: "",
        current_room_id: "none",
        destination_site_id: page.scope_site.id,
        destination_site: page.scope_site.name,
        destination_room_id: "none",
        destination_room: "None",
        action: "change",
      },
    ];
    setRows(nextRows);
    setDirty(true);
  }

  async function cancelRow(rowId) {
    const nextRows = rows.filter((candidate) => candidate.id !== rowId);
    setRows(nextRows);
    setDirty(true);
    await save(nextRows, effectiveDate);
  }

  if (!bounds) {
    return null;
  }

  return (
    <section
      className="room-moves-runtime__bulk"
      style={{ left: bounds.left, top: bounds.top, width: bounds.width, minHeight: bounds.height }}
      aria-labelledby={ROOM_MOVES_HEADING_ID}
    >
      <div className="room-moves-runtime__bulk-titlebar">
        <h2>{page.title}</h2>
        <div className="room-moves-runtime__bulk-actions">
          <label htmlFor="room-move-effective-date">
            <span>Effective date</span>
            <input
              id="room-move-effective-date"
              type="date"
              value={effectiveDate}
              onChange={(event) => {
                setEffectiveDate(event.target.value);
                setDirty(true);
              }}
              onBlur={() => save(rows, effectiveDate)}
            />
          </label>
          <button type="button" onClick={() => save()} disabled={saving}>Save Draft</button>
          <button type="button" onClick={() => onTransition("schedule")}>Schedule</button>
          <button type="button" onClick={() => onTransition("apply")}>Apply</button>
          <button type="button" className="room-moves-runtime__delete" onClick={onDelete}>Discard</button>
        </div>
      </div>
      {draft.warnings?.length ? (
        <div className="room-moves-runtime__warning-bar">
          <strong>Warnings</strong>
          <ul>
            {draft.warnings.map((warning) => <li key={warning}>{warning}</li>)}
          </ul>
        </div>
      ) : null}
      <div className="room-moves-runtime__table-tools">
        <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
        <button type="button" onClick={addRow}>Add</button>
      </div>
      <div className="room-moves-runtime__bulk-header">
        {BULK_COLUMNS.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
        <div>Remove</div>
      </div>
      <div className="room-moves-runtime__bulk-body">
        {table.visibleRows.length === 0 ? (
          <div className="room-moves-runtime__bulk-empty" role="status">
            <strong>No draft rows yet</strong>
            <span>Use Add to start a manual move list. Selecting a person fills in the current room and default destination.</span>
          </div>
        ) : null}
        {table.visibleRows.map((row) => (
          <div key={row.id} className="room-moves-runtime__bulk-row">
            <select value={row.person_id} onChange={(event) => updateRow(row.id, { person_id: event.target.value })}>
              <option value="">Select person...</option>
              {page.people.map((person) => (
                <option key={person.id} value={person.id}>{bulkPersonLabel(person)}</option>
              ))}
            </select>
            <div>{row.current_room || "—"}</div>
            {page.can_manage_district ? (
              <select
                value={row.destination_site_id}
                onChange={(event) => updateRow(row.id, {
                  destination_site_id: event.target.value,
                  destination_room_id: event.target.value === row.current_site_id ? row.current_room_id : "none",
                })}
              >
                {page.sites.map((site) => <option key={site.id} value={site.id}>{site.name}</option>)}
              </select>
            ) : (
              <div>{row.destination_site}</div>
            )}
            <select
              value={row.destination_room_id}
              onChange={(event) => updateRow(row.id, { destination_room_id: event.target.value })}
            >
              {roomOptionsForSite(page.rooms, row.destination_site_id).map((room) => (
                <option key={`${row.id}-${room.site_id}-${room.id}`} value={room.id}>{room.label}</option>
              ))}
            </select>
            <select value={row.action} onChange={(event) => updateRow(row.id, { action: event.target.value })}>
              <option value="add">add</option>
              <option value="change">change</option>
              <option value="removal">removal</option>
            </select>
            <button
              type="button"
              className="room-moves-runtime__delete"
              onClick={() => {
                void cancelRow(row.id);
              }}
            >
              Remove
            </button>
          </div>
        ))}
      </div>
    </section>
  );
}

export function RoomMovesPage({
  session,
  routeKind,
  artboardKey,
  currentSearch = "",
  onNavigate,
  onSearch,
  searchQuery = "",
  onUnauthorized,
  onForbidden,
}) {
  const [pageState, setPageState] = useState("loading");
  const [payload, setPayload] = useState(null);
  const [selectedRow, setSelectedRow] = useState(null);
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const [busy, setBusy] = useState(false);
  const [cancelingDraftId, setCancelingDraftId] = useState("");

  const isBulk = routeKind === "room-moves-bulk-draft";
  const { artboard, status: artboardStatus } = useGeneratedArtboard(artboardKey);
  const meta = generatedArtboardMeta[artboardKey];

  const endpoint = useMemo(() => {
    if (!isBulk) {
      return ROOM_MOVES_ENDPOINT;
    }
    return `${ROOM_MOVES_BULK_ENDPOINT}${currentSearch || ""}`;
  }, [currentSearch, isBulk]);

  useEffect(() => {
    if (!session?.authenticated || !session?.authorized) {
      return undefined;
    }
    const controller = new AbortController();
    async function loadPage() {
      setPageState("loading");
      try {
        const response = await fetch(endpoint, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          signal: controller.signal,
        });
        if (response.status === 401) {
          onUnauthorized?.();
          return;
        }
        if (response.status === 403) {
          onForbidden?.();
          return;
        }
        const nextPayload = await readJSON(response);
        setPayload(nextPayload);
        setPageState("ready");
      } catch (error) {
        if (!controller.signal.aborted) {
          setPageState("error");
          setPayload(null);
        }
      }
    }
    void loadPage();
    return () => controller.abort();
  }, [endpoint, onForbidden, onUnauthorized, reloadKey, session]);

  const textOverrides = useMemo(() => {
    const overrides = buildSharedShellTextOverrides(session);
    return overrides;
  }, [session]);
  const hiddenNodeIds = useMemo(
    () => [
      ...buildSharedShellHiddenNodeIds(session, {
        hideNavHighlight: true,
        hideSearchPlaceholder: true,
        hideAllNavGroups: true,
      }),
      ...hiddenRoomMovesNodeIds(artboardKey, isBulk),
    ],
    [artboardKey, isBulk, session]
  );
  const imageNodeOverrides = useMemo(() => buildSharedShellImageOverrides(session), [session]);

  const refresh = useCallback(() => setReloadKey((value) => value + 1), []);

  async function createDraft(mode) {
    setBusy(true);
    try {
      const response = await readJSON(
        await fetch(ROOM_MOVES_DRAFTS_ENDPOINT, {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ mode, scope_site_id: payload?.page?.scope_site?.id }),
        })
      );
      onNavigate(`/room-moves/bulk-draft?draft_id=${encodeURIComponent(response.draft.id)}`);
    } finally {
      setBusy(false);
    }
  }

  async function saveBulkDraft(draft) {
    const response = await readJSON(
      await fetch(`${ROOM_MOVES_DRAFTS_ENDPOINT}/${draft.id}`, {
        method: "PUT",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({
          mode: draft.mode,
          scope_site_id: draft.scope_site_id,
          effective_date: draft.effective_date,
          rows: draft.rows,
        }),
      })
    );
    setPayload((current) => current ? { ...current, page: { ...current.page, draft: response.draft } } : current);
    return response.draft;
  }

  async function transitionBulkDraft(action) {
    const draftID = payload?.page?.draft?.id;
    if (!draftID) {
      return;
    }
    const response = await readJSON(
      await fetch(`${ROOM_MOVES_DRAFTS_ENDPOINT}/${draftID}/${action}`, {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      })
    );
    setPayload((current) => current ? { ...current, page: { ...current.page, draft: response.draft } } : current);
  }

  async function deleteBulkDraft() {
    const draftID = payload?.page?.draft?.id;
    if (!draftID) {
      return;
    }
    await fetch(`${ROOM_MOVES_DRAFTS_ENDPOINT}/${draftID}`, {
      method: "DELETE",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    onNavigate("/room-moves");
  }

  async function cancelMove(row) {
    if (!row?.draft_id) {
      return;
    }
    setCancelingDraftId(row.draft_id);
    try {
      await readJSON(
        await fetch(`${ROOM_MOVES_DRAFTS_ENDPOINT}/${row.draft_id}/cancel`, {
          method: "POST",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setSelectedRow(null);
      setShowCreateDrawer(false);
      refresh();
    } finally {
      setCancelingDraftId("");
    }
  }

  const renderOverlay = useMemo(
    () =>
      createSharedShellRenderOverlay({
        session,
        onNavigate,
        onSearch,
        searchQuery,
        activeNavKey: "roomMoves",
        activeRoutePath: isBulk ? "/room-moves/bulk-draft" : "/room-moves",
        refreshMetadata: isBulk
          ? null
          : payload?.page?.last_refreshed ?? staticRefreshMetadataForArtboard(meta),
      }),
    [isBulk, meta, onNavigate, onSearch, payload?.page?.last_refreshed, searchQuery, session]
  );

  const fullOverlay = useCallback(
    ({ nodeIndex, textOverrides: overlayTextOverrides }) => {
      const shellOverlay = renderOverlay({ nodeIndex, textOverrides: overlayTextOverrides });
      const routePayloadReady = isBulk ? Boolean(payload?.page?.draft) : Boolean(payload?.page?.rows);
      if (pageState !== "ready" || !payload?.page || !routePayloadReady) {
        return shellOverlay;
      }
      const page = payload.page;
      const tableBounds = isBulk
        ? { left: 288, top: 96, width: 1268, height: 820 }
        : { ...nodeBox(nodeIndex.get("room-moves__f100"), { left: 288, top: 348, width: 1268, height: 480 }), width: 1268 };
      const batchBounds = nodeBox(nodeIndex.get("room-moves__f88"), { left: 996, top: 182, width: 220, height: 148 });

      return (
        <>
          {shellOverlay}
          {!isBulk ? (
            <>
              <RoomMovesActions
                bounds={batchBounds}
                busy={busy}
                onMovePerson={() => setShowCreateDrawer(true)}
                onBatchMove={() => createDraft("manual_move_list")}
                onSiteRollover={() => createDraft("end_of_year_site_move")}
              />
              <RoomMovesTable
                bounds={tableBounds}
                rows={page.rows}
                selectedRowId={selectedRow?.id}
                cancelingDraftId={cancelingDraftId}
                onCancelRow={cancelMove}
                onSelectRow={(row) => {
                  if (row.move_type === "mid_year_targeted_move") {
                    setSelectedRow(row);
                    setShowCreateDrawer(false);
                  } else {
                    onNavigate(`/room-moves/bulk-draft?draft_id=${encodeURIComponent(row.draft_id)}`);
                  }
                }}
              />
            </>
          ) : (
            <BulkDraftTable
              bounds={tableBounds}
              page={page}
              onSave={saveBulkDraft}
              onTransition={transitionBulkDraft}
              onDelete={deleteBulkDraft}
            />
          )}
        </>
      );
    },
    [busy, cancelMove, cancelingDraftId, createDraft, deleteBulkDraft, isBulk, onNavigate, pageState, payload, renderOverlay, saveBulkDraft, selectedRow, transitionBulkDraft]
  );

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Room Moves</h1>
          <p>Preparing the generated Room Moves artboard.</p>
        </section>
      </main>
    );
  }

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Room Moves unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas" aria-busy={pageState === "loading"}>
      <h1 id={ROOM_MOVES_HEADING_ID} className="sr-only">{isBulk ? payload?.page?.title || "Site Rollover" : "Room Moves"}</h1>
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={fullOverlay}
        />
      </div>
      {pageState === "loading" ? (
        <div className="page-loading" role="status" aria-live="polite">
          <h2>Loading Room Moves</h2>
          <p>Loading the DEV room move drafts.</p>
        </div>
      ) : null}
      {pageState === "error" ? (
        <div className="page-loading" role="alert">
          <h2>Unable to load Room Moves</h2>
          <p>The DEV room move mock data could not be loaded.</p>
        </div>
      ) : null}
      {(selectedRow || showCreateDrawer) && payload?.page ? (
        <SingleMoveDrawer
          row={selectedRow}
          people={payload.page.people}
          rooms={payload.page.rooms}
          sites={payload.page.sites}
          canManageDistrict={payload.page.can_manage_district}
          onClose={() => {
            setSelectedRow(null);
            setShowCreateDrawer(false);
          }}
          onSaved={() => {
            setSelectedRow(null);
            setShowCreateDrawer(false);
            refresh();
          }}
        />
      ) : null}
    </main>
  );
}
