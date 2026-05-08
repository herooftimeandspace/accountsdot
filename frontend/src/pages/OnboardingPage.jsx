import { useCallback, useEffect, useRef, useState } from "react";
import * as lucideIcons from "lucide-static";
import { RuntimeDetailList, RuntimeDrawer } from "../components/RuntimeDrawer";
import { generatedArtboards, generatedArtboardMeta } from "../generated/artboards.generated.js";
import { PenArtboard } from "../lib/PenArtboard";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
  staticRefreshMetadataForArtboard,
} from "../lib/sharedShellPresentation";

const ONBOARDING_ENDPOINT = "/api/v1/dev/pages/onboarding";
const MANUAL_DRAFTS_ENDPOINT = "/api/v1/dev/onboarding/manual-drafts";
const ONBOARDING_HEADING_ID = "onboarding-heading";
const LEAD_TIME_WARNING =
  "The start date is ≤ 3 days from the current date. Access to some systems may be delayed beyond the start date.";
const EMPTY_DRAFT_FORM = {
  start_date: "",
  ssn_last4: "",
  employee_type: "",
  classification: "",
  first_name: "",
  last_name: "",
  job_title: "",
  site_id: "",
  personal_email: "",
  preferred_device: "",
  requested_aeries_access: "",
  replacing_employee_id: "",
  room_id: "",
  notes: "",
};
const STATIC_ONBOARDING_TABLE_NODE_IDS = [
  "t117",
  "t118",
  "t119",
  "t120",
  "t121",
  "t122",
  "t123",
  "l124",
  "t125",
  "t126",
  "t127",
  "t128",
  "t129",
  "t130",
  "f131",
  "t132",
  "l133",
  "t134",
  "t135",
  "t136",
  "t137",
  "t138",
  "t139",
  "f140",
  "t141",
  "l142",
  "t143",
  "t144",
  "t145",
  "t146",
  "t147",
  "t148",
  "f149",
  "t150",
  "l151",
  "t152",
  "t153",
  "t154",
  "t155",
  "t156",
  "t157",
  "f158",
  "t159",
  "l160",
  "t161",
].map((id) => `onboarding__${id}`);
const ADD_MANUAL_NODE_ID = "onboarding__f109";
const ADD_MANUAL_LABEL_NODE_ID = "onboarding__t110";
const ONBOARDING_TABLE_FRAME_NODE_ID = "onboarding__f116";

function nodeBox(node) {
  if (!node) {
    return null;
  }
  return {
    left: node.x ?? 0,
    top: node.y ?? 0,
    width: node.width ?? 0,
    height: node.height ?? 0,
  };
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

function draftToForm(draft) {
  return {
    start_date: draft?.start_date ?? "",
    ssn_last4: draft?.ssn_last4 ?? "",
    employee_type: draft?.employee_type ?? "",
    classification: draft?.classification ?? "",
    first_name: draft?.first_name ?? "",
    last_name: draft?.last_name ?? "",
    job_title: draft?.job_title ?? "",
    site_id: draft?.site_id ?? "",
    personal_email: draft?.personal_email ?? "",
    preferred_device: draft?.preferred_device ?? "",
    requested_aeries_access: draft?.requested_aeries_access ?? "",
    replacing_employee_id: draft?.replacing_employee_id ?? "",
    room_id: draft?.room_id ?? "",
    notes: draft?.notes ?? "",
  };
}

function daysBetween(startDate, currentDate) {
  if (!startDate || !currentDate) {
    return null;
  }
  const start = new Date(`${startDate}T00:00:00`);
  const current = new Date(`${currentDate}T00:00:00`);
  if (Number.isNaN(start.getTime()) || Number.isNaN(current.getTime())) {
    return null;
  }
  return Math.ceil((start.getTime() - current.getTime()) / 86400000);
}

function statusClass(status) {
  if (status === "Ready" || status === "Ready to Provision") {
    return "onboarding-runtime__status onboarding-runtime__status--ready";
  }
  if (status === "Blocked" || status === "Needs Review" || status === "Incomplete Data") {
    return "onboarding-runtime__status onboarding-runtime__status--warning";
  }
  return "onboarding-runtime__status";
}

function OnboardingTableOverlay({ bounds, rows, selectedRowId, onSelectRow }) {
  if (!bounds) {
    return null;
  }
  const manualRows = rows.filter((row) => row.kind === "manual").length;
  const totalRows = Math.max(rows.length, 42 + manualRows);
  return (
    <section
      className="onboarding-runtime__table"
      style={{
        position: "absolute",
        left: bounds.left + 18,
        top: bounds.top + 14,
        width: Math.max(0, bounds.width - 36),
        height: Math.max(0, bounds.height - 28),
        zIndex: 2,
      }}
      aria-labelledby={ONBOARDING_HEADING_ID}
    >
      <div className="onboarding-runtime__table-title">Upcoming Staff Onboarding</div>
      <div className="onboarding-runtime__table-header">
        <div>Start</div>
        <div>Person</div>
        <div>Site</div>
        <div>Current Step</div>
        <div>Issue / Action</div>
        <div>Workflow Status</div>
      </div>
      <div className="onboarding-runtime__table-body">
        {rows.map((row) => (
          <button
            key={row.id}
            type="button"
            className={`onboarding-runtime__row ${
              selectedRowId === row.id ? "onboarding-runtime__row--selected" : ""
            }`}
            aria-label={`Open onboarding row for ${row.person}`}
            aria-pressed={selectedRowId === row.id}
            onClick={() => onSelectRow(row)}
          >
            <div>{row.start_date || "Unknown"}</div>
            <div>{row.person}</div>
            <div>{row.site}</div>
            <div>{row.current_step}</div>
            <div>{row.issue_action}</div>
            <div>
              <span className={statusClass(row.workflow_status)}>{row.workflow_status}</span>
            </div>
          </button>
        ))}
      </div>
      <div className="onboarding-runtime__table-footer">
        Showing 1 to {Math.min(rows.length, 4)} of {totalRows} upcoming people
      </div>
    </section>
  );
}

function AddManualOverlay({ bounds, canManageManual, onAdd }) {
  if (!bounds || !canManageManual) {
    return null;
  }
  return (
    <button
      type="button"
      className="onboarding-runtime__add-hotspot"
      aria-label="Add Non-Escape Record"
      title="Add Non-Escape Record"
      style={{
        position: "absolute",
        left: bounds.left,
        top: bounds.top,
        width: bounds.width,
        height: bounds.height,
        zIndex: 4,
      }}
      onClick={onAdd}
    />
  );
}

function WorkflowDrawer({ row, onClose }) {
  if (!row) {
    return null;
  }
  return (
    <RuntimeDrawer title={row.person} onClose={onClose}>
      <RuntimeDetailList
        items={[
          { label: "Start", value: row.start_date },
          { label: "Site", value: row.site },
          { label: "Current Step", value: row.current_step },
          { label: "Issue / Action", value: row.issue_action },
          { label: "Status", value: row.workflow_status },
          { label: "Assigned Email", value: row.assigned_email },
          { label: "Employee ID", value: row.employee_number },
          { label: "IncidentIQ", value: row.incident_iq },
        ]}
      />
      <div className="runtime-drawer__section">
        <p>
          <strong>Earliest matching Aeries ticket:</strong>
          <span>{row.aeries_ticket || "Not linked"}</span>
        </p>
        <p>
          <strong>Earliest matching Verkada ticket:</strong>
          <span>{row.verkada_ticket || "Not linked"}</span>
        </p>
      </div>
    </RuntimeDrawer>
  );
}

function FieldError({ value }) {
  if (!value) {
    return null;
  }
  return <span className="onboarding-runtime__field-error">{value}</span>;
}

function SelectField({ id, label, value, options, onChange, required = false }) {
  return (
    <label className="onboarding-runtime__field" htmlFor={id}>
      <span>{label}</span>
      <select id={id} value={value} required={required} onChange={(event) => onChange(event.target.value)}>
        <option value="">Select...</option>
        {options.map((option) => {
          const optionValue = typeof option === "string" ? option : option.id;
          const optionLabel = typeof option === "string" ? option : option.name;
          return (
            <option key={optionValue} value={optionValue}>
              {optionLabel}
            </option>
          );
        })}
      </select>
    </label>
  );
}

function ManualDraftDrawer({
  draft,
  form,
  formOptions,
  currentDate,
  errors,
  saving,
  onChange,
  onClose,
  onSave,
}) {
  const leadTimeDays = daysBetween(form.start_date, currentDate);
  const showLeadTimeWarning = leadTimeDays !== null && leadTimeDays <= 3;
  const replacingEmployee = formOptions.replacing_employees.find((employee) => employee.id === form.replacing_employee_id);

  return (
    <RuntimeDrawer title="Add Non-Escape Record" onClose={onClose}>
      <form className="onboarding-runtime__form" onSubmit={(event) => event.preventDefault()}>
        <div className="onboarding-runtime__generated">
          <span>Status</span>
          <strong>{draft.status}</strong>
          {draft.generated_email ? (
            <>
              <span>Generated Email</span>
              <strong>{draft.generated_email}</strong>
            </>
          ) : null}
          {draft.generated_employee_id ? (
            <>
              <span>Generated ID</span>
              <strong>{draft.generated_employee_id}</strong>
            </>
          ) : null}
        </div>

        <label className="onboarding-runtime__field" htmlFor="manual-start-date">
          <span>
            Start date
            {showLeadTimeWarning ? (
              <span className="onboarding-runtime__warning" title={LEAD_TIME_WARNING} aria-label={LEAD_TIME_WARNING}>
                <span
                  aria-hidden="true"
                  dangerouslySetInnerHTML={{
                    __html: lucideIcons.AlertTriangle.replace(/width="[^"]+"/, 'width="16"').replace(
                      /height="[^"]+"/,
                      'height="16"'
                    ),
                  }}
                />
              </span>
            ) : null}
          </span>
          <input
            id="manual-start-date"
            type="date"
            required
            value={form.start_date}
            onChange={(event) => onChange("start_date", event.target.value)}
          />
          <FieldError value={errors.start_date} />
        </label>

        <label className="onboarding-runtime__field" htmlFor="manual-ssn-last4">
          <span>Last 4 SSN</span>
          <input
            id="manual-ssn-last4"
            inputMode="numeric"
            pattern="[0-9]{4}"
            maxLength={4}
            required
            value={form.ssn_last4}
            onChange={(event) => onChange("ssn_last4", event.target.value.replace(/\D/g, "").slice(0, 4))}
          />
          <FieldError value={errors.ssn_last4} />
        </label>

        <SelectField id="manual-employee-type" label="Employee type" value={form.employee_type} options={formOptions.employee_types} required onChange={(value) => onChange("employee_type", value)} />
        <SelectField id="manual-classification" label="Classification" value={form.classification} options={formOptions.classifications} required onChange={(value) => onChange("classification", value)} />

        <label className="onboarding-runtime__field" htmlFor="manual-first-name">
          <span>First name</span>
          <input id="manual-first-name" type="text" required value={form.first_name} onChange={(event) => onChange("first_name", event.target.value)} />
        </label>

        <label className="onboarding-runtime__field" htmlFor="manual-last-name">
          <span>Last name</span>
          <input id="manual-last-name" type="text" required value={form.last_name} onChange={(event) => onChange("last_name", event.target.value)} />
        </label>

        <SelectField id="manual-job-title" label="Job title" value={form.job_title} options={formOptions.job_titles} required onChange={(value) => onChange("job_title", value)} />
        <SelectField id="manual-site" label="Site" value={form.site_id} options={formOptions.sites} required onChange={(value) => onChange("site_id", value)} />

        <label className="onboarding-runtime__field" htmlFor="manual-personal-email">
          <span>Personal email</span>
          <input id="manual-personal-email" type="email" required value={form.personal_email} onChange={(event) => onChange("personal_email", event.target.value)} />
          <FieldError value={errors.personal_email} />
        </label>

        <SelectField id="manual-device" label="Preferred device" value={form.preferred_device} options={formOptions.preferred_devices} required onChange={(value) => onChange("preferred_device", value)} />
        <SelectField id="manual-aeries" label="Requested Aeries access" value={form.requested_aeries_access} options={formOptions.requested_aeries_access} required onChange={(value) => onChange("requested_aeries_access", value)} />
        <SelectField id="manual-replacing" label="Replacing" value={form.replacing_employee_id} options={formOptions.replacing_employees} onChange={(value) => onChange("replacing_employee_id", value)} />
        {replacingEmployee ? (
          <p className="onboarding-runtime__hint">{replacingEmployee.email}</p>
        ) : null}
        <SelectField id="manual-room" label="Room / classroom" value={form.room_id} options={formOptions.rooms} onChange={(value) => onChange("room_id", value)} />

        <label className="onboarding-runtime__field" htmlFor="manual-notes">
          <span>Notes</span>
          <textarea id="manual-notes" value={form.notes} onChange={(event) => onChange("notes", event.target.value)} />
        </label>

        {draft.missing_fields?.length ? (
          <p className="onboarding-runtime__missing">Missing required fields: {draft.missing_fields.join(", ")}</p>
        ) : null}
        {errors.form ? <p className="onboarding-runtime__missing">{errors.form}</p> : null}
        <button type="button" className="onboarding-runtime__save" onClick={onSave} disabled={saving}>
          {saving ? "Saving..." : "Save"}
        </button>
      </form>
    </RuntimeDrawer>
  );
}

export function OnboardingPage({ session, onNavigate, onSearch, searchQuery = "", onUnauthorized, onForbidden }) {
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [selectedRow, setSelectedRow] = useState(null);
  const [activeDraft, setActiveDraft] = useState(null);
  const [draftForm, setDraftForm] = useState(EMPTY_DRAFT_FORM);
  const [draftErrors, setDraftErrors] = useState({});
  const [saving, setSaving] = useState(false);
  const dirtyRef = useRef(false);
  const activeDraftRef = useRef(null);
  const draftFormRef = useRef(EMPTY_DRAFT_FORM);

  const artboard = generatedArtboards.onboarding;
  const meta = generatedArtboardMeta.onboarding;

  const loadPage = useCallback(async () => {
    setPageState("loading");
    try {
      const nextPayload = await readJSON(
        await fetch(ONBOARDING_ENDPOINT, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setPayload(nextPayload);
      setPageState("ready");
    } catch (error) {
      if (error.status === 401 && onUnauthorized) {
        onUnauthorized();
        return;
      }
      if (error.status === 403 && onForbidden) {
        onForbidden();
        return;
      }
      setPageState("error");
    }
  }, [onForbidden, onUnauthorized]);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

  useEffect(() => {
    activeDraftRef.current = activeDraft;
    draftFormRef.current = draftForm;
  }, [activeDraft, draftForm]);

  const saveDraft = useCallback(async (draft = activeDraftRef.current, form = draftFormRef.current) => {
    if (!draft) {
      return null;
    }
    setSaving(true);
    setDraftErrors({});
    try {
      const saved = await readJSON(
        await fetch(`${MANUAL_DRAFTS_ENDPOINT}/${draft.id}`, {
          method: "PUT",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify(form),
        })
      );
      setActiveDraft(saved.draft);
      setDraftForm(draftToForm(saved.draft));
      dirtyRef.current = false;
      await loadPage();
      return saved.draft;
    } catch (error) {
      setDraftErrors(error.payload?.errors ?? { form: error.message });
      return null;
    } finally {
      setSaving(false);
    }
  }, [loadPage]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      if (dirtyRef.current && activeDraftRef.current) {
        void saveDraft(activeDraftRef.current, draftFormRef.current);
      }
    }, 60000);
    return () => window.clearInterval(interval);
  }, [saveDraft]);

  const handleAddManual = useCallback(async () => {
    setSelectedRow(null);
    setDraftErrors({});
    try {
      const created = await readJSON(
        await fetch(MANUAL_DRAFTS_ENDPOINT, {
          method: "POST",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setActiveDraft(created.draft);
      setDraftForm(draftToForm(created.draft));
      dirtyRef.current = false;
      await loadPage();
    } catch (error) {
      setDraftErrors({ form: error.message });
    }
  }, [loadPage]);

  const handleSelectRow = useCallback((row) => {
    setSelectedRow(row);
    if (row.kind === "manual") {
      const draft = payload?.page?.drafts?.find((candidate) => candidate.id === row.manual_draft_id);
      if (draft) {
        setActiveDraft(draft);
        setDraftForm(draftToForm(draft));
        setDraftErrors({});
        dirtyRef.current = false;
      }
      return;
    }
    setActiveDraft(null);
  }, [payload]);

  const handleDraftChange = useCallback((field, value) => {
    setDraftForm((current) => ({ ...current, [field]: value }));
    dirtyRef.current = true;
  }, []);

  const handleCloseDraft = useCallback(() => {
    if (dirtyRef.current && activeDraftRef.current) {
      void saveDraft(activeDraftRef.current, draftFormRef.current);
    }
    setActiveDraft(null);
    setSelectedRow(null);
  }, [saveDraft]);

  const handleSaveDraft = useCallback(async () => {
    const saved = await saveDraft();
    if (!saved) {
      return;
    }
    if (saved.missing_fields?.length) {
      return;
    }
    setSaving(true);
    try {
      const finalized = await readJSON(
        await fetch(`${MANUAL_DRAFTS_ENDPOINT}/${saved.id}/finalize`, {
          method: "POST",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setActiveDraft(finalized.draft);
      setDraftForm(draftToForm(finalized.draft));
      await loadPage();
    } catch (error) {
      setDraftErrors({ form: error.message });
    } finally {
      setSaving(false);
    }
  }, [loadPage, saveDraft]);

  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...STATIC_ONBOARDING_TABLE_NODE_IDS);
  if (!payload?.page?.can_manage_manual) {
    hiddenNodeIds.push(ADD_MANUAL_NODE_ID, ADD_MANUAL_LABEL_NODE_ID);
  }
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? null,
    refreshMetadata: payload?.page?.last_refreshed ?? staticRefreshMetadataForArtboard("onboarding"),
  });
  const semanticSummary = buildArtboardSemanticSummary(artboard, {
    fallbackTitle: "Onboarding Dashboard",
    textOverrides,
  });

  const rows = payload?.page?.rows ?? [];
  const formOptions = payload?.form ?? {
    employee_types: [],
    classifications: [],
    job_titles: [],
    sites: [],
    preferred_devices: [],
    requested_aeries_access: [],
    replacing_employees: [],
    rooms: [],
  };

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => {
    const tableBounds = nodeBox(nodeIndex.get(ONBOARDING_TABLE_FRAME_NODE_ID));
    const addBounds = nodeBox(nodeIndex.get(ADD_MANUAL_NODE_ID));
    return (
      <>
        {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
        <OnboardingTableOverlay
          bounds={tableBounds}
          rows={rows}
          selectedRowId={selectedRow?.id}
          onSelectRow={handleSelectRow}
        />
        <AddManualOverlay
          bounds={addBounds}
          canManageManual={Boolean(payload?.page?.can_manage_manual)}
          onAdd={handleAddManual}
        />
        {activeDraft ? (
          <ManualDraftDrawer
            draft={activeDraft}
            form={draftForm}
            formOptions={formOptions}
            currentDate={payload?.page?.current_date}
            errors={draftErrors}
            saving={saving}
            onChange={handleDraftChange}
            onClose={handleCloseDraft}
            onSave={handleSaveDraft}
          />
        ) : (
          <WorkflowDrawer row={selectedRow} onClose={() => setSelectedRow(null)} />
        )}
      </>
    );
  }, [
    activeDraft,
    draftErrors,
    draftForm,
    formOptions,
    handleAddManual,
    handleCloseDraft,
    handleDraftChange,
    handleSaveDraft,
    handleSelectRow,
    payload,
    rows,
    saving,
    selectedRow,
    sharedShellRenderOverlay,
  ]);

  if (pageState === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Onboarding</h1>
          <p>Loading the DEV onboarding dashboard.</p>
        </section>
      </main>
    );
  }

  if (pageState === "error") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Onboarding unavailable</h1>
          <p>The DEV onboarding dashboard could not be loaded.</p>
        </section>
      </main>
    );
  }

  return (
    <main
      id="main-content"
      className="page-canvas page-canvas--static"
      aria-labelledby={ONBOARDING_HEADING_ID}
    >
      <section className="sr-only" aria-labelledby={ONBOARDING_HEADING_ID}>
        <h1 id={ONBOARDING_HEADING_ID}>{payload?.page?.title || semanticSummary.title}</h1>
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
