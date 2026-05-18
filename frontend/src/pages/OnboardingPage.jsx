import { useCallback, useEffect, useRef, useState } from "react";
import * as lucideIcons from "lucide-static";
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

const ONBOARDING_ENDPOINT = "/api/v1/dev/pages/onboarding";
const MANUAL_DRAFTS_ENDPOINT = "/api/v1/dev/onboarding/manual-drafts";
const ONBOARDING_ROWS_ENDPOINT = "/api/v1/dev/onboarding/rows";
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
  personal_phone: "",
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
  "t500",
  "t501",
  "t502",
  "t503",
  "t504",
].map((id) => `onboarding__${id}`);
const ADD_MANUAL_NODE_ID = "onboarding__f109";
const ADD_MANUAL_LABEL_NODE_ID = "onboarding__t110";
const ONBOARDING_TABLE_FRAME_NODE_ID = "onboarding__f116";
const ONBOARDING_TABLE_COLUMNS = [
  { key: "date_added", label: "Date Added", value: (row) => row.date_added || "Unknown" },
  { key: "start_date", label: "Start", value: (row) => formatOnboardingDate(row.start_date) || "Unknown", sortValue: (row) => row.start_date || "" },
  { key: "person", label: "Person", value: (row) => row.person },
  { key: "site", label: "Site", value: (row) => row.site },
  { key: "current_step", label: "Current Step", value: (row) => row.current_step },
  { key: "issue_action", label: "Issue / Action", value: (row) => row.issue_action },
  { key: "workflow_status", label: "Workflow Status", value: (row) => row.workflow_status },
];

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
    personal_phone: formatPersonalPhoneInput(draft?.personal_phone ?? ""),
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

function formatOnboardingDate(value) {
  if (!value) {
    return "";
  }
  const parsed = new Date(`${value}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

function LeadTimeWarning({ id, placement = "center" }) {
  return (
    <span
      className={`onboarding-runtime__warning onboarding-runtime__warning--${placement}`}
      tabIndex={0}
      aria-describedby={id}
    >
      <span
        aria-hidden="true"
        dangerouslySetInnerHTML={{
          __html: lucideIcons.AlertTriangle.replace(/width="[^"]+"/, 'width="16"').replace(
            /height="[^"]+"/,
            'height="16"'
          ),
        }}
      />
      <span id={id} role="tooltip" className="onboarding-runtime__warning-tooltip">
        {LEAD_TIME_WARNING}
      </span>
    </span>
  );
}

function changeReasonLabel(reason) {
  const labels = {
    assignment_add: "Secondary / tertiary assignment",
    role_change: "Role change",
    same_site_transfer: "Same-site transfer",
    site_transfer: "Site transfer",
    reactivate_same_role: "Reactivation into same role",
    reactivate_role_change: "Reactivation into different role",
    reactivate_non_escape: "Reactivated as manual Non-Escape contractor",
    active_escape_contractor_collision: "Active Escape contractor collision",
  };
  return labels[reason] ?? reason ?? "";
}

function statusClass(status) {
  if (["Ready", "Ready to Provision", "Healthy", "Complete", "Allowed"].includes(status)) {
    return "onboarding-runtime__status onboarding-runtime__status--ready";
  }
  if (["Blocked", "Invalid", "Failed", "Error", "Incomplete Data", "Warning"].includes(status)) {
    return "onboarding-runtime__status onboarding-runtime__status--critical";
  }
  if (["Needs Review", "Review", "Manual action", "External action"].includes(status)) {
    return "onboarding-runtime__status onboarding-runtime__status--review";
  }
  if (["Queued", "Scheduled", "Waiting"].includes(status)) {
    return "onboarding-runtime__status onboarding-runtime__status--waiting";
  }
  if (["In Progress", "Running"].includes(status)) {
    return "onboarding-runtime__status onboarding-runtime__status--active";
  }
  return "onboarding-runtime__status onboarding-runtime__status--neutral";
}

function missingFieldLabel(field) {
  const labels = {
    start_date: "Start date",
    ssn_last4: "Last 4 SSN",
    employee_type: "Employee type",
    classification: "Classification",
    first_name: "First name",
    last_name: "Last name",
    job_title: "Job title",
    site_id: "Site",
    personal_email: "Personal email",
    personal_phone: "Personal phone",
    preferred_device: "Preferred device",
    requested_aeries_access: "Requested Aeries access",
  };
  return labels[field] ?? field;
}

function fieldHasProblem(field, draft, errors, showValidationFeedback) {
  return Boolean(showValidationFeedback && (errors?.[field] || draft?.missing_fields?.includes(field)));
}

function fieldClassName(field, draft, errors, showValidationFeedback, extraClass = "") {
  return [
    "onboarding-runtime__field",
    fieldHasProblem(field, draft, errors, showValidationFeedback) ? "onboarding-runtime__field--problem" : "",
    extraClass,
  ]
    .filter(Boolean)
    .join(" ");
}

function OnboardingTableOverlay({ bounds, rows, selectedRowId, onSelectRow }) {
  const table = useRuntimeTableData(rows, ONBOARDING_TABLE_COLUMNS, {
    defaultSort: { key: "date_added", direction: "asc" },
  });

  if (!bounds) {
    return null;
  }
  const visibleRows = table.visibleRows;
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
      <RuntimeTableSearch value={table.searchQuery} onChange={table.setSearchQuery} />
      <div className="onboarding-runtime__table-header">
        {ONBOARDING_TABLE_COLUMNS.map((column) => (
          <div key={column.key}>
            <RuntimeSortableHeader column={column} sortState={table.sortState} onSort={table.toggleSort} />
          </div>
        ))}
      </div>
      <div className="onboarding-runtime__table-body">
        {visibleRows.map((row) => (
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
            <div title={row.date_added_reason}>{row.date_added || "Unknown"}</div>
            <div className="onboarding-runtime__start-cell">
              <span>{formatOnboardingDate(row.start_date) || "Unknown"}</span>
              {row.lead_time_warning ? <LeadTimeWarning id={`lead-time-warning-${row.id}`} /> : null}
            </div>
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
        Showing {visibleRows.length ? 1 : 0} to {visibleRows.length} of {totalRows} upcoming people
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
    >
      Add Non-Escape Record
    </button>
  );
}

function workflowMissingFields(row, step) {
  if (row?.id === "evan-ruiz" && step?.name === "HR intake") {
    return ["Employment type"];
  }
  if (row?.kind === "manual" && row?.missing_fields?.length) {
    return row.missing_fields.map(missingFieldLabel);
  }
  return [];
}

function linkedTicketStatus(ticketText) {
  const trimmed = String(ticketText || "").trim();
  if (!trimmed) {
    return { number: "", status: "" };
  }
  const [number, ...statusParts] = trimmed.split(/\s+/);
  return {
    number,
    status: statusParts.join(" "),
  };
}

function TicketStatusLine({ label, ticketText }) {
  const ticket = linkedTicketStatus(ticketText);
  return (
    <p>
      <strong>{label}</strong>
      {ticket.number ? (
        <span>
          <a href={`https://mock.wusd.local/incidentiq/tickets/${ticket.number}`} target="_blank" rel="noreferrer">
            {ticket.number}
          </a>
          {ticket.status ? ` ${ticket.status}` : ""}
        </span>
      ) : (
        <span>Not linked</span>
      )}
    </p>
  );
}

function RoomOverrideForm({ row, formOptions, onSaved }) {
  const [room, setRoom] = useState(row?.room_id ?? "");
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);
  const roomOptions = (formOptions?.rooms ?? []).filter((option) => !row?.site_id || option.site_id === row.site_id);

  async function submitRoomOverride(event) {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    try {
      const saved = await readJSON(
        await fetch(`${ONBOARDING_ROWS_ENDPOINT}/${row.id}/room`, {
          method: "PUT",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify({ room_id: room }),
        })
      );
      setMessage(`Saved DEV room override for ${saved.row?.room_name || "the selected room"}.`);
      onSaved?.(saved);
    } catch (error) {
      setMessage(error.payload?.errors?.room_id || error.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <form className="onboarding-runtime__room-override" onSubmit={submitRoomOverride}>
      <h4>Override room from IncidentIQ</h4>
      <label htmlFor="room-override-room">
        <span>Room</span>
        <select
          id="room-override-room"
          value={room}
          onChange={(event) => setRoom(event.target.value)}
        >
          <option value="">None</option>
          {roomOptions.map((option) => (
            <option key={option.id} value={option.id}>
              {option.name}
            </option>
          ))}
        </select>
      </label>
      <button type="submit" disabled={saving}>{saving ? "Saving..." : "Save Room"}</button>
      {message ? <p role="status">{message}</p> : null}
    </form>
  );
}

function WorkflowDrawer({ row, formOptions, onClose, onRoomSaved }) {
  if (!row) {
    return null;
  }
  return (
    <RuntimeDrawer title={row.person} onClose={onClose}>
      {row.late_start ? (
        <div className="onboarding-runtime__late-start">
          <strong>Late-start warning</strong>
          <p>{LEAD_TIME_WARNING}</p>
        </div>
      ) : null}
      <RuntimeDetailList
        items={[
          { label: "Start", value: row.start_date },
          { label: "Effective date", value: row.effective_date },
          { label: "Date Added", value: row.date_added },
          { label: "Added Because", value: row.date_added_reason },
          { label: "Change reason", value: changeReasonLabel(row.change_reason) },
          { label: "Scheduled for", value: row.scheduled_for },
          { label: "Site", value: row.site },
          { label: "Current Step", value: row.current_step },
          { label: "Issue / Action", value: row.issue_action },
          { label: "Status", value: row.workflow_status },
          { label: "Assigned Email", value: row.assigned_email },
          { label: "Employee ID", value: row.employee_number },
          { label: "IncidentIQ", value: row.incident_iq },
        ]}
      />
      {row.linked_escape_record ? (
        <div className="runtime-drawer__section">
          <h3>Linked Escape Record</h3>
          <RuntimeDetailList
            items={[
              { label: "Person", value: row.linked_escape_record.person },
              { label: "Site", value: row.linked_escape_record.site },
              { label: "Assigned Email", value: row.linked_escape_record.assigned_email },
              { label: "Employee ID", value: row.linked_escape_record.employee_number },
              { label: "Start Date", value: row.linked_escape_record.start_date },
              { label: "Current Step", value: row.linked_escape_record.current_step },
              { label: "Workflow Status", value: row.linked_escape_record.workflow_status },
            ]}
          />
        </div>
      ) : null}
      <div className="runtime-drawer__section">
        <TicketStatusLine label="Earliest matching Aeries ticket:" ticketText={row.aeries_ticket} />
        <TicketStatusLine label="Earliest matching Verkada ticket:" ticketText={row.verkada_ticket} />
      </div>
      {row.workflow_steps?.length ? (
        <div className="onboarding-runtime__workflow-steps">
          <h3>Workflow Steps</h3>
          {row.workflow_steps.map((step) => (
            <section key={step.name} className="onboarding-runtime__workflow-step">
              <div>
                <strong>{step.name}</strong>
                <span className={statusClass(step.status)}>{step.status}</span>
              </div>
              <p>{step.detail}</p>
              {workflowMissingFields(row, step).length ? (
                <div className="onboarding-runtime__missing-fields">
                  <strong>Missing field(s)</strong>
                  <ul>
                    {workflowMissingFields(row, step).map((field) => (
                      <li key={field}>{field}</li>
                    ))}
                  </ul>
                </div>
              ) : null}
              {step.actions?.length ? (
                <ul>
                  {step.actions.map((action) => (
                    <li key={`${step.name}-${action.label}`}>
                      <a href={action.href} target="_blank" rel="noreferrer">
                        {action.label}
                      </a>
                      <span>{action.system}</span>
                      <p>{action.resolution}</p>
                    </li>
                  ))}
                </ul>
              ) : null}
            </section>
          ))}
          {row.can_update_room ? <RoomOverrideForm row={row} formOptions={formOptions} onSaved={onRoomSaved} /> : null}
        </div>
      ) : null}
      {!row.workflow_steps?.length && row.can_update_room ? (
        <RoomOverrideForm row={row} formOptions={formOptions} onSaved={onRoomSaved} />
      ) : null}
    </RuntimeDrawer>
  );
}

function AddManualErrorDrawer({ message, onClose }) {
  if (!message) {
    return null;
  }
  return (
    <RuntimeDrawer title="Add Non-Escape Record" onClose={onClose}>
      <div className="onboarding-runtime__generated">
        <span>Status</span>
        <strong className="onboarding-runtime__generated-warning">Unable to open intake drawer</strong>
        <span>Reason</span>
        <strong className="onboarding-runtime__generated-warning">{message}</strong>
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

function SelectField({ id, label, value, options, onChange, required = false, className = "" }) {
  return (
    <label className={className || "onboarding-runtime__field"} htmlFor={id}>
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
  showValidationFeedback,
  saving,
  onChange,
  onClose,
  onSave,
  onDelete,
}) {
  const leadTimeDays = daysBetween(form.start_date, currentDate);
  const showLeadTimeWarning = leadTimeDays !== null && leadTimeDays <= 3;
  const replacingEmployee = formOptions.replacing_employees.find((employee) => employee.id === form.replacing_employee_id);
  const missingSummary = showValidationFeedback && draft.missing_fields?.length
    ? `Missing required fields: ${draft.missing_fields.map(missingFieldLabel).join(", ")}`
    : "";
  const isCollision =
    draft.validity_state === "invalid" &&
    draft.invalid_reason === "active_escape_contractor_collision";

  if (isCollision) {
    return (
      <RuntimeDrawer title="Invalid Manual Entry" onClose={onClose}>
        <div className="onboarding-runtime__collision">
          <span className="onboarding-runtime__collision-badge">Invalid contractor collision</span>
          <p className="onboarding-runtime__collision-copy">
            Invalid contractor entry. This person is already an active Escape employee. Escape always
            takes precedence. We cannot hire an active employee as a contractor. Delete the manual
            entry to resolve this collision.
          </p>
          {draft.linked_escape_record ? (
            <div className="runtime-drawer__section">
              <h3>Linked Escape Record</h3>
              <RuntimeDetailList
                items={[
                  { label: "Person", value: draft.linked_escape_record.person },
                  { label: "Site", value: draft.linked_escape_record.site },
                  { label: "Assigned Email", value: draft.linked_escape_record.assigned_email },
                  { label: "Employee ID", value: draft.linked_escape_record.employee_number },
                  { label: "Start Date", value: draft.linked_escape_record.start_date },
                  { label: "Current Step", value: draft.linked_escape_record.current_step },
                  { label: "Workflow Status", value: draft.linked_escape_record.workflow_status },
                ]}
              />
            </div>
          ) : null}
          {errors.form ? <p className="onboarding-runtime__field-error">{errors.form}</p> : null}
          <button
            type="button"
            className="onboarding-runtime__delete"
            onClick={onDelete}
            disabled={saving || !draft.can_delete_manual_entry}
          >
            {saving ? "Deleting..." : "Delete Manual Entry"}
          </button>
        </div>
      </RuntimeDrawer>
    );
  }

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
          {missingSummary ? (
            <>
              <span>Required Fields</span>
              <strong className="onboarding-runtime__generated-warning">{missingSummary}</strong>
            </>
          ) : null}
          {errors.form ? (
            <>
              <span>Save Error</span>
              <strong className="onboarding-runtime__generated-warning">{errors.form}</strong>
            </>
          ) : null}
        </div>

        <label className={fieldClassName("start_date", draft, errors, showValidationFeedback)} htmlFor="manual-start-date">
          <span>
            Start date
            {showLeadTimeWarning ? (
              <LeadTimeWarning id="manual-start-date-lead-time-warning" placement="drawer" />
            ) : null}
          </span>
          <input
            id="manual-start-date"
            type="date"
            required
            value={form.start_date}
            onChange={(event) => onChange("start_date", event.target.value)}
          />
          <FieldError value={showValidationFeedback ? errors.start_date : ""} />
        </label>

        <label className={fieldClassName("ssn_last4", draft, errors, showValidationFeedback)} htmlFor="manual-ssn-last4">
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
          <FieldError value={showValidationFeedback ? errors.ssn_last4 : ""} />
        </label>

        <SelectField id="manual-employee-type" label="Employee type" value={form.employee_type} options={formOptions.employee_types} required className={fieldClassName("employee_type", draft, errors, showValidationFeedback)} onChange={(value) => onChange("employee_type", value)} />
        <SelectField id="manual-classification" label="Classification" value={form.classification} options={formOptions.classifications} required className={fieldClassName("classification", draft, errors, showValidationFeedback)} onChange={(value) => onChange("classification", value)} />

        <label className={fieldClassName("first_name", draft, errors, showValidationFeedback)} htmlFor="manual-first-name">
          <span>First name</span>
          <input id="manual-first-name" type="text" required value={form.first_name} onChange={(event) => onChange("first_name", event.target.value)} />
        </label>

        <label className={fieldClassName("last_name", draft, errors, showValidationFeedback)} htmlFor="manual-last-name">
          <span>Last name</span>
          <input id="manual-last-name" type="text" required value={form.last_name} onChange={(event) => onChange("last_name", event.target.value)} />
        </label>

        <SelectField id="manual-job-title" label="Job title" value={form.job_title} options={formOptions.job_titles} required className={fieldClassName("job_title", draft, errors, showValidationFeedback)} onChange={(value) => onChange("job_title", value)} />
        <SelectField id="manual-site" label="Site" value={form.site_id} options={formOptions.sites} required className={fieldClassName("site_id", draft, errors, showValidationFeedback)} onChange={(value) => onChange("site_id", value)} />

        <SelectField id="manual-replacing" label="Replacing" value={form.replacing_employee_id} options={formOptions.replacing_employees} onChange={(value) => onChange("replacing_employee_id", value)} />
        <SelectField id="manual-room" label="Room / classroom" value={form.room_id} options={formOptions.rooms} onChange={(value) => onChange("room_id", value)} />
        {replacingEmployee ? (
          <p className="onboarding-runtime__hint">{replacingEmployee.email}</p>
        ) : null}

        <label className={fieldClassName("personal_email", draft, errors, showValidationFeedback, "onboarding-runtime__field--full")} htmlFor="manual-personal-email">
          <span>Personal email</span>
          <input id="manual-personal-email" type="email" required value={form.personal_email} onChange={(event) => onChange("personal_email", event.target.value)} />
          <FieldError value={showValidationFeedback ? errors.personal_email : ""} />
        </label>

        <label className={fieldClassName("personal_phone", draft, errors, showValidationFeedback, "onboarding-runtime__field--full")} htmlFor="manual-personal-phone">
          <span>Personal phone number</span>
          <input
            id="manual-personal-phone"
            type="tel"
            inputMode="tel"
            autoComplete="tel"
            pattern="\([0-9]{3}\) [0-9]{3}-[0-9]{4}"
            placeholder="(707) 555-0134"
            required
            value={form.personal_phone}
            onChange={(event) => onChange("personal_phone", formatPersonalPhoneInput(event.target.value))}
          />
          <FieldError value={showValidationFeedback ? errors.personal_phone : ""} />
        </label>

        <SelectField id="manual-device" label="Preferred device" value={form.preferred_device} options={formOptions.preferred_devices} required className={fieldClassName("preferred_device", draft, errors, showValidationFeedback)} onChange={(value) => onChange("preferred_device", value)} />
        <SelectField id="manual-aeries" label="Requested Aeries access" value={form.requested_aeries_access} options={formOptions.requested_aeries_access} required className={fieldClassName("requested_aeries_access", draft, errors, showValidationFeedback)} onChange={(value) => onChange("requested_aeries_access", value)} />

        <label className="onboarding-runtime__field onboarding-runtime__field--full" htmlFor="manual-notes">
          <span>Notes</span>
          <textarea id="manual-notes" value={form.notes} onChange={(event) => onChange("notes", event.target.value)} />
        </label>

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
  const [addManualError, setAddManualError] = useState("");
  const [draftForm, setDraftForm] = useState(EMPTY_DRAFT_FORM);
  const [draftErrors, setDraftErrors] = useState({});
  const [manualSaveAttempted, setManualSaveAttempted] = useState(false);
  const [saving, setSaving] = useState(false);
  const dirtyRef = useRef(false);
  const activeDraftRef = useRef(null);
  const draftFormRef = useRef(EMPTY_DRAFT_FORM);

  const { artboard, status: artboardStatus } = useGeneratedArtboard("onboarding");
  const meta = generatedArtboardMeta.onboarding;
  const personaId = session?.current_persona?.id ?? "";

  const loadPage = useCallback(async () => {
    setPageState("loading");
    try {
      const endpoint = `${ONBOARDING_ENDPOINT}${window.location.search || ""}`;
      const nextPayload = await readJSON(
        await fetch(endpoint, {
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
  }, [onForbidden, onUnauthorized, personaId]);

  useEffect(() => {
    setSelectedRow(null);
    setActiveDraft(null);
    setAddManualError("");
    setDraftErrors({});
    setManualSaveAttempted(false);
    dirtyRef.current = false;
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
    setActiveDraft(null);
    setDraftErrors({});
    setManualSaveAttempted(false);
    setAddManualError("");
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
      setAddManualError(error.message);
    }
  }, [loadPage]);

  const handleSelectRow = useCallback((row) => {
    setSelectedRow(row);
    setAddManualError("");
    if (row.kind === "manual") {
      const draft = payload?.page?.drafts?.find((candidate) => candidate.id === row.manual_draft_id);
      if (draft) {
        setActiveDraft(draft);
        setDraftForm(draftToForm(draft));
        setDraftErrors({});
        setManualSaveAttempted(false);
        dirtyRef.current = false;
      }
      return;
    }
    setActiveDraft(null);
    setManualSaveAttempted(false);
  }, [payload]);

  const handleDraftChange = useCallback((field, value) => {
    setDraftForm((current) => ({ ...current, [field]: value }));
    dirtyRef.current = true;
  }, []);

  const handleCloseDraft = useCallback(() => {
    if (
      dirtyRef.current &&
      activeDraftRef.current &&
      activeDraftRef.current.validity_state !== "invalid"
    ) {
      void saveDraft(activeDraftRef.current, draftFormRef.current);
    }
    setActiveDraft(null);
    setSelectedRow(null);
  }, [saveDraft]);

  const handleSaveDraft = useCallback(async () => {
    setManualSaveAttempted(true);
    const saved = await saveDraft();
    if (!saved) {
      return;
    }
    if (saved.missing_fields?.length || saved.validity_state === "invalid") {
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

  const handleDeleteDraft = useCallback(async () => {
    if (!activeDraft?.id) {
      return;
    }
    setSaving(true);
    setDraftErrors({});
    try {
      await readJSON(
        await fetch(`${MANUAL_DRAFTS_ENDPOINT}/${activeDraft.id}`, {
          method: "DELETE",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      dirtyRef.current = false;
      setActiveDraft(null);
      setSelectedRow(null);
      await loadPage();
    } catch (error) {
      setDraftErrors({ form: error.message });
    } finally {
      setSaving(false);
    }
  }, [activeDraft?.id, loadPage]);

  const handleRoomSaved = useCallback((saved) => {
    setPayload((current) => current ? { ...current, page: { ...current.page, rows: saved.rows ?? current.page.rows } } : current);
    if (saved.row) {
      setSelectedRow(saved.row);
    }
  }, []);

  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  hiddenNodeIds.push(...STATIC_ONBOARDING_TABLE_NODE_IDS);
  hiddenNodeIds.push(ADD_MANUAL_NODE_ID, ADD_MANUAL_LABEL_NODE_ID);
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? null,
    activeRoutePath: "/onboarding",
    refreshMetadata: payload?.page?.last_refreshed ?? staticRefreshMetadataForArtboard("onboarding"),
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, {
        fallbackTitle: "Onboarding Dashboard",
        textOverrides,
      })
    : { title: "Onboarding Dashboard", items: [] };

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
            showValidationFeedback={manualSaveAttempted}
            saving={saving}
            onChange={handleDraftChange}
            onClose={handleCloseDraft}
            onSave={handleSaveDraft}
            onDelete={handleDeleteDraft}
          />
        ) : addManualError ? (
          <AddManualErrorDrawer message={addManualError} onClose={() => setAddManualError("")} />
        ) : (
          <WorkflowDrawer
            row={selectedRow}
            formOptions={formOptions}
            onClose={() => setSelectedRow(null)}
            onRoomSaved={handleRoomSaved}
          />
        )}
      </>
    );
  }, [
    activeDraft,
    addManualError,
    draftErrors,
    draftForm,
    formOptions,
    handleAddManual,
    handleCloseDraft,
    handleDraftChange,
    handleSaveDraft,
    handleSelectRow,
    handleRoomSaved,
    payload,
    manualSaveAttempted,
    rows,
    saving,
    selectedRow,
    sharedShellRenderOverlay,
  ]);

  if (artboardStatus === "loading" || pageState === "loading") {
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

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Onboarding unavailable</h1></main>;
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
