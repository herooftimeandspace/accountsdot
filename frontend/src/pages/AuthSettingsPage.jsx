import { useCallback, useEffect, useMemo, useState } from "react";
import { generatedArtboardMeta } from "../generated/artboards.generated.js";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { PenArtboard } from "../lib/PenArtboard";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const ARTBOARD_KEY = "admin-feature-flags";
const AUTH_SETTINGS_ENDPOINT = "/api/v1/admin/auth-settings";
const EXTERNAL_SOURCES_ENDPOINT = "/api/v1/admin/external-sources";
const AUTH_SETTINGS_HEADING_ID = "auth-settings-heading";
const PANE_LEFT = 306;
const PANE_TOP = 118;
const ARTBOARD_WIDTH = 1672;
const PANE_RIGHT_GUTTER = 48;
const PANE_WIDTH = ARTBOARD_WIDTH - PANE_LEFT - PANE_RIGHT_GUTTER;

const EMPTY_MAPPING = {
  source_type: "group",
  source_value: "",
  attribute_values: "",
  role_keys: "",
  site_codes: "",
  reason: "",
};

const EMPTY_PREVIEW = {
  email: "casey.teacher@staff.wusd.org",
  groups: "",
  ous: "",
  attributes: "wizard_role=Faculty",
};

const DEFAULT_PROVIDER_FIELDS = {
  google: ["client_email", "credential_reference"],
  zoom: ["account_id", "credential_reference"],
  aeries: ["base_url", "certificate_reference"],
  sftp: ["host", "username", "credential_reference"],
};

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

function splitList(value) {
  return value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseAttributes(value) {
  const attributes = {};
  splitList(value).forEach((entry) => {
    const [rawKey, rawValues] = entry.split("=");
    const key = rawKey?.trim();
    if (!key) {
      return;
    }
    attributes[key] = (rawValues || "").split("|").map((item) => item.trim()).filter(Boolean);
  });
  return attributes;
}

function StatusChip({ tone = "neutral", children }) {
  return <span className={`auth-settings-runtime__chip auth-settings-runtime__chip--${tone}`}>{children}</span>;
}

function MappingForm({ title, mode, onSubmit, busy }) {
  const [form, setForm] = useState(EMPTY_MAPPING);
  const isRole = mode === "role";
  const submit = (event) => {
    event.preventDefault();
    onSubmit({
      source_type: form.source_type,
      source_value: form.source_value.trim(),
      attribute_values: splitList(form.attribute_values),
      role_keys: isRole ? splitList(form.role_keys) : [],
      site_codes: isRole ? [] : splitList(form.site_codes),
      reason: form.reason.trim(),
    }).then(() => setForm(EMPTY_MAPPING));
  };
  return (
    <form className="auth-settings-runtime__form" onSubmit={submit}>
      <h3>{title}</h3>
      <label>
        Source
        <select value={form.source_type} onChange={(event) => setForm({ ...form, source_type: event.target.value })}>
          <option value="group">Google group</option>
          <option value="ou">Google OU</option>
          <option value="attribute">SAML/Google attribute</option>
        </select>
      </label>
      <label>
        Source value
        <input value={form.source_value} onChange={(event) => setForm({ ...form, source_value: event.target.value })} placeholder="group@wusd.org or wizard_role" />
      </label>
      {form.source_type === "attribute" ? (
        <label>
          Attribute values
          <input value={form.attribute_values} onChange={(event) => setForm({ ...form, attribute_values: event.target.value })} placeholder="Faculty, IT Admin" />
        </label>
      ) : null}
      <label>
        {isRole ? "Roles" : "Sites"}
        <input
          value={isRole ? form.role_keys : form.site_codes}
          onChange={(event) => setForm({ ...form, [isRole ? "role_keys" : "site_codes"]: event.target.value })}
          placeholder={isRole ? "it_admin, site_secretary" : "bpl, clover-hs"}
        />
      </label>
      <label>
        Change reason
        <input value={form.reason} onChange={(event) => setForm({ ...form, reason: event.target.value })} placeholder="Ticket or operator reason" />
      </label>
      <button type="submit" disabled={busy}>Save</button>
    </form>
  );
}

function MappingTable({ title, rows, kind, onDelete, busy }) {
  return (
    <section className="auth-settings-runtime__section">
      <h3>{title}</h3>
      <div className="auth-settings-runtime__table" role="table" aria-label={title}>
        {rows.map((row) => (
          <div className="auth-settings-runtime__row" role="row" key={`${kind}-${row.id}`}>
            <span>{row.source_type}</span>
            <strong>{row.source_value}</strong>
            <span>{(row.attribute_values || []).join(", ") || "Any value"}</span>
            <span>{kind === "role" ? (row.role_keys || []).join(", ") : (row.site_codes || []).join(", ")}</span>
            <button type="button" disabled={busy} onClick={() => onDelete(row.id, kind)}>Delete</button>
          </div>
        ))}
        {rows.length === 0 ? <p className="auth-settings-runtime__empty">No mappings saved yet.</p> : null}
      </div>
    </section>
  );
}

function PreviewPanel({ preview, setPreview, result, onPreview, busy }) {
  return (
    <section className="auth-settings-runtime__section">
      <header className="auth-settings-runtime__section-header">
        <h2>Validation preview</h2>
        <StatusChip tone="warn">Does not enable live login</StatusChip>
      </header>
      <div className="auth-settings-runtime__preview">
        <label>
          Candidate email
          <input value={preview.email} onChange={(event) => setPreview({ ...preview, email: event.target.value })} />
        </label>
        <label>
          Groups
          <input value={preview.groups} onChange={(event) => setPreview({ ...preview, groups: event.target.value })} placeholder="group@wusd.org, another@wusd.org" />
        </label>
        <label>
          OUs
          <input value={preview.ous} onChange={(event) => setPreview({ ...preview, ous: event.target.value })} placeholder="/Staff/School Site" />
        </label>
        <label>
          Attributes
          <input value={preview.attributes} onChange={(event) => setPreview({ ...preview, attributes: event.target.value })} placeholder="wizard_role=Faculty|IT" />
        </label>
        <button type="button" disabled={busy} onClick={onPreview}>Preview</button>
      </div>
      {result ? (
        <div className="auth-settings-runtime__result" role="status">
          <strong>{result.authorized ? "Authorized by preview" : "Not authorized by preview"}</strong>
          <span>Roles: {(result.roles || []).join(", ") || "None"}</span>
          <span>Sites: {(result.site_scopes || []).join(", ") || "None"}</span>
          <span>Validation: {(result.validation_failures || []).join(", ") || "No failures"}</span>
        </div>
      ) : null}
    </section>
  );
}

function ExternalSourceCard({ source, onToggle, onSaveCredentials, onTest, busy }) {
  const fields = DEFAULT_PROVIDER_FIELDS[source.provider_key] || ["credential_reference"];
  const [reason, setReason] = useState("");
  const [credentialValues, setCredentialValues] = useState({});
  const [labels, setLabels] = useState({});
  return (
    <article className="auth-settings-runtime__source">
      <header>
        <div>
          <h3>{source.provider_label}</h3>
          <p>{source.provider_key}</p>
        </div>
        <label className="auth-settings-runtime__toggle">
          <input
            type="checkbox"
            checked={source.sync_enabled}
            disabled={busy}
            onChange={(event) => onToggle(source.provider_key, event.target.checked, reason)}
          />
          <span>{source.sync_enabled ? "Sync enabled" : "Sync off"}</span>
        </label>
      </header>
      <div className="auth-settings-runtime__credential-grid">
        {fields.map((field) => (
          <label key={field}>
            {field}
            <input
              type="password"
              value={credentialValues[field] || ""}
              onChange={(event) => setCredentialValues({ ...credentialValues, [field]: event.target.value })}
              placeholder="Encrypted after save"
            />
          </label>
        ))}
        <label>
          Secret label
          <input value={labels.credential_reference || ""} onChange={(event) => setLabels({ ...labels, credential_reference: event.target.value })} placeholder="Vault path, certificate label, or owner note" />
        </label>
        <label>
          Change reason
          <input value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Ticket or operator reason" />
        </label>
      </div>
      <div className="auth-settings-runtime__actions">
        <button type="button" disabled={busy} onClick={() => onSaveCredentials(source.provider_key, credentialValues, labels, reason)}>Save credentials</button>
        <button type="button" disabled={busy} onClick={() => onTest(source.provider_key, reason)}>Test read-only</button>
      </div>
      <div className="auth-settings-runtime__meta-list">
        {(source.credentials || []).map((credential) => (
          <span key={credential.field_key}>{credential.field_key}: stored {credential.fingerprint}</span>
        ))}
        <span>Last test: {source.last_test_status || "not run"}</span>
        {source.last_test_summary ? <span>{source.last_test_summary}</span> : null}
      </div>
    </article>
  );
}

function AuthSettingsOverlay({
  payload,
  state,
  message,
  preview,
  setPreview,
  previewResult,
  busy,
  onSaveMapping,
  onDeleteMapping,
  onPreview,
  onToggle,
  onSaveCredentials,
  onTest,
}) {
  return (
    <section
      className="auth-settings-runtime"
      style={{ position: "absolute", left: PANE_LEFT, top: PANE_TOP, width: PANE_WIDTH, zIndex: 2 }}
      aria-labelledby={AUTH_SETTINGS_HEADING_ID}
    >
      <header className="auth-settings-runtime__header">
        <div>
          <h1 id={AUTH_SETTINGS_HEADING_ID}>Auth Settings</h1>
          <p>IT Admin mapping and provider credential controls. Production login and sync execution remain disabled by these settings.</p>
        </div>
        <StatusChip tone="warn">Preview only</StatusChip>
      </header>
      {state === "loading" ? <p className="auth-settings-runtime__status" role="status">Loading auth settings...</p> : null}
      {message ? <p className="auth-settings-runtime__status" role={state === "error" ? "alert" : "status"}>{message}</p> : null}

      <section className="auth-settings-runtime__section auth-settings-runtime__section--forms">
        <MappingForm title="Role mapping" mode="role" busy={busy} onSubmit={(request) => onSaveMapping("role", request)} />
        <MappingForm title="Site-scope mapping" mode="site" busy={busy} onSubmit={(request) => onSaveMapping("site", request)} />
      </section>

      <MappingTable title="Saved role mappings" rows={payload?.role_mappings || []} kind="role" busy={busy} onDelete={onDeleteMapping} />
      <MappingTable title="Saved site-scope mappings" rows={payload?.site_scope_mappings || []} kind="site" busy={busy} onDelete={onDeleteMapping} />

      <PreviewPanel preview={preview} setPreview={setPreview} result={previewResult} busy={busy} onPreview={onPreview} />

      <section className="auth-settings-runtime__section">
        <header className="auth-settings-runtime__section-header">
          <h2>External sources</h2>
          <StatusChip tone="off">Default off</StatusChip>
        </header>
        <div className="auth-settings-runtime__sources">
          {(payload?.external_sources || []).map((source) => (
            <ExternalSourceCard
              key={source.provider_key}
              source={source}
              busy={busy}
              onToggle={onToggle}
              onSaveCredentials={onSaveCredentials}
              onTest={onTest}
            />
          ))}
        </div>
      </section>

      <section className="auth-settings-runtime__section">
        <h2>Audit history</h2>
        <div className="auth-settings-runtime__audit">
          {(payload?.audit_events || []).map((event) => (
            <div key={event.id} className="auth-settings-runtime__audit-row">
              <strong>{event.target_entity}</strong>
              <span>{event.target_id}</span>
              <span>{event.reason}</span>
              <span>{event.actor_id}</span>
            </div>
          ))}
          {(payload?.audit_events || []).length === 0 ? <p className="auth-settings-runtime__empty">No audit events yet.</p> : null}
        </div>
      </section>
    </section>
  );
}

export function AuthSettingsPage({ session, onNavigate, onSearch, searchQuery, onUnauthorized, onForbidden }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const meta = generatedArtboardMeta[ARTBOARD_KEY];
  const [payload, setPayload] = useState(null);
  const [state, setState] = useState("loading");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState(EMPTY_PREVIEW);
  const [previewResult, setPreviewResult] = useState(null);
  const textOverrides = buildSharedShellTextOverrides(session);
  const hiddenNodeIds = buildSharedShellHiddenNodeIds(session, {
    hideNavHighlight: true,
    hideSearchPlaceholder: true,
    hideAllNavGroups: true,
  });
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeNavKey: meta?.activeNav ?? "admin",
    activeRoutePath: "/admin/auth-settings",
    refreshMetadata: null,
  });
  const semanticSummary = artboard
    ? buildArtboardSemanticSummary(artboard, { fallbackTitle: "Auth Settings", textOverrides })
    : { title: "Auth Settings", items: [] };

  const loadAuthSettings = useCallback(async (signal) => {
    setState("loading");
    setMessage("");
    try {
      const nextPayload = await readJSON(
        await fetch(AUTH_SETTINGS_ENDPOINT, { credentials: "same-origin", headers: { Accept: "application/json" }, signal })
      );
      setPayload(nextPayload);
      setState("ready");
    } catch (error) {
      if (signal?.aborted) return;
      if (error.status === 401) return onUnauthorized?.();
      if (error.status === 403) return onForbidden?.();
      setState("error");
      setMessage(error.message);
    }
  }, [onForbidden, onUnauthorized]);

  useEffect(() => {
    const controller = new AbortController();
    void loadAuthSettings(controller.signal);
    return () => controller.abort();
  }, [loadAuthSettings]);

  const reload = useCallback(async () => loadAuthSettings(), [loadAuthSettings]);

  const mutate = useCallback(async (work, successMessage) => {
    setBusy(true);
    setMessage("");
    try {
      await work();
      await reload();
      setMessage(successMessage);
      setState("ready");
    } catch (error) {
      if (error.status === 401) return onUnauthorized?.();
      if (error.status === 403) return onForbidden?.();
      setMessage(error.payload?.message || error.message);
    } finally {
      setBusy(false);
    }
  }, [onForbidden, onUnauthorized, reload]);

  const saveMapping = useCallback((kind, request) => mutate(async () => {
    const path = kind === "role" ? "role-mappings" : "site-scope-mappings";
    await readJSON(await fetch(`${AUTH_SETTINGS_ENDPOINT}/${path}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(request),
    }));
  }, "Mapping saved."), [mutate]);

  const deleteMapping = useCallback((id, kind) => {
    const reason = window.prompt("Change reason for deleting this mapping:");
    if (!reason) return;
    void mutate(async () => {
      const path = kind === "role" ? "role-mappings" : "site-scope-mappings";
      await readJSON(await fetch(`${AUTH_SETTINGS_ENDPOINT}/${path}/${id}`, {
        method: "DELETE",
        credentials: "same-origin",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ reason }),
      }));
    }, "Mapping deleted.");
  }, [mutate]);

  const runPreview = useCallback(async () => {
    setBusy(true);
    setMessage("");
    try {
      const result = await readJSON(await fetch(`${AUTH_SETTINGS_ENDPOINT}/preview`, {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({
          email: preview.email,
          groups: splitList(preview.groups),
          ous: splitList(preview.ous),
          attributes: parseAttributes(preview.attributes),
        }),
      }));
      setPreviewResult(result);
    } catch (error) {
      if (error.status === 401) return onUnauthorized?.();
      if (error.status === 403) return onForbidden?.();
      setMessage(error.payload?.message || error.message);
    } finally {
      setBusy(false);
    }
  }, [onForbidden, onUnauthorized, preview]);

  const toggleSource = useCallback((provider, syncEnabled, reason) => mutate(async () => {
    await readJSON(await fetch(`${EXTERNAL_SOURCES_ENDPOINT}/${encodeURIComponent(provider)}`, {
      method: "PATCH",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ sync_enabled: syncEnabled, reason }),
    }));
  }, "External source toggle updated without starting sync."), [mutate]);

  const saveCredentials = useCallback((provider, fields, labels, reason) => mutate(async () => {
    await readJSON(await fetch(`${EXTERNAL_SOURCES_ENDPOINT}/${encodeURIComponent(provider)}/credentials`, {
      method: "PUT",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ fields, labels, reason }),
    }));
  }, "Credentials saved as encrypted values."), [mutate]);

  const testSource = useCallback((provider, reason) => mutate(async () => {
    await readJSON(await fetch(`${EXTERNAL_SOURCES_ENDPOINT}/${encodeURIComponent(provider)}/test`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ reason }),
    }));
  }, "Read-only credential test completed."), [mutate]);

  const renderOverlay = useCallback(({ nodeIndex, textOverrides: overlayTextOverrides }) => (
    <>
      {sharedShellRenderOverlay?.({ nodeIndex, textOverrides: overlayTextOverrides })}
      <AuthSettingsOverlay
        payload={payload}
        state={state}
        message={message}
        preview={preview}
        setPreview={setPreview}
        previewResult={previewResult}
        busy={busy}
        onSaveMapping={saveMapping}
        onDeleteMapping={deleteMapping}
        onPreview={runPreview}
        onToggle={toggleSource}
        onSaveCredentials={saveCredentials}
        onTest={testSource}
      />
    </>
  ), [busy, deleteMapping, message, payload, preview, previewResult, runPreview, saveCredentials, saveMapping, sharedShellRenderOverlay, state, testSource, toggleSource]);

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Auth Settings</h1>
          <p>Preparing the generated Admin artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Auth Settings unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby={AUTH_SETTINGS_HEADING_ID}>
      <section className="sr-only" aria-labelledby={`${AUTH_SETTINGS_HEADING_ID}-summary`}>
        <h1 id={`${AUTH_SETTINGS_HEADING_ID}-summary`}>{semanticSummary.title}</h1>
        <ul>{semanticSummary.items.map((item) => <li key={item}>{item}</li>)}</ul>
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
