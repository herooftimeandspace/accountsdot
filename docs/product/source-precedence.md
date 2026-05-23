# Source Precedence and Reconciliation Rules

This document defines how The WIZARD resolves fields that can be supplied by an upstream provider and also changed through the dashboard. The goal is to preserve source-system truth while allowing auditable local decisions such as manual contractor records, temporary site or room overrides, preferred/display-name edits, and permission mappings.

The application must not silently overwrite values in either direction. Every reconciled effective value should be explainable from three facts:

- the provider-owned upstream value and when it was observed
- the dashboard-managed manual value and its metadata
- the field-level precedence rule that selected the effective value, raised a conflict, or cleared a temporary override

## Field Inventory

The first reconciliation pass must cover these field families. Later provider integrations may add fields, but they must declare the same ownership and supersession metadata before any write-capable workflow depends on them.

| Field family | Representative fields | Upstream sources | Dashboard-managed source | Effective-value owner |
| --- | --- | --- | --- | --- |
| Employment lifecycle | employment status, hire date, start date, assignment participation, end date for regular employees | Escape / SFTP, InformedK12 where it initiates pending hires | HR/IT workflow notes and review state only | Upstream wins for Escape-backed employees. Dashboard values may explain or defer work but must not replace provider-owned dates or status. |
| Manual contractor lifecycle | contractor start date, contractor end date, employee type, classification, continuation link, contractor active/inactive state | none for active non-Escape records; inactive Escape records may be matching context | HR-managed manual intake and offboarding actions | Dashboard manual record wins while no active Escape replacement exists. Active Escape overlap supersedes the manual contractor record and marks it invalid for audit. |
| Site assignment | primary site, site scope, district-wide vs site-scoped eligibility | Escape, Aeries staff/teacher/scheduling, Google/SAML site scope, InformedK12 form evidence for reviewed site-change signals | HR temporary site override, HR/IT InformedK12-backed primary-site selection, IT Admin permission/site mapping | Provider-owned site wins by default. A documented temporary override or InformedK12-backed dashboard selection may remain effective only until the upstream value matches the intended correction, changes job assignment, is superseded by a newer form-supported decision, is manually reconciled, or conflicts with a different upstream value. |
| Room/location | teaching room, office room, phone location, IncidentIQ location mapping | Aeries staff/teacher/scheduling, IncidentIQ asset location, Zoom phone state | authorized Site Admin/Site Secretary/HR/IT room override, room mapping override | Field-specific. Aeries staff room wins before local overrides for teachers; local room overrides can unblock workflows while upstream correction is pending; IncidentIQ room mapping overrides are app-owned mapping decisions. |
| Legal name | first name, last name, legal display components | Escape for employees, Aeries for students, HR manual intake for non-Escape contractors | HR correction for manual records only | Upstream wins for Escape-backed employees and students. Dashboard legal-name edits are authoritative only for local manual contractor records until an active upstream replacement supersedes them. |
| Preferred/display name | preferred first name, display name, downstream display-name intent | dashboard self-service or HR/IT approved request; Google/Zoom may reflect downstream result after sync | employee/contractor self-service or HR/IT action where authorized | Dashboard request is the app-owned intent for eligible non-student identities. Provider reads verify propagation and may surface drift but do not silently replace the app-owned preferred-name decision. |
| Permission and site mappings | app role, site scope, manual grant/revocation, expiration | SAML claims, Google groups/attributes | IT Admin manual grant, revocation, site-scope mapping | Domain gate, disabled state, breakglass, SAML/Google, manual grants/revocations, and feature rollout resolve in the order defined by `docs/product/permissions-model.md`. Manual revocations intentionally override upstream role signals inside the app while requiring upstream cleanup. |
| Downstream provider identifiers | AD object id, Google id, Zoom user id, IncidentIQ user/asset id, extension ids | owning provider | app linkage/audit notes only | Provider-owned identifiers win. The app stores stable links and must mark ambiguity or missing targets for review rather than inventing replacements. |
| Local workflow state | queue status, approval state, manual notes, carried-forward review notes, workflow snapshots, audit events | none | application workflow engine | Application-owned. Upstream changes may trigger replanning or supersession, but they do not delete audit history or workflow evidence. |

## Manual-Change Metadata

Every dashboard-managed value that can affect a provider-backed workflow must carry enough metadata to decide whether a later upstream change matches, conflicts with, or supersedes it.

Required metadata:

- `field_key`: stable field identifier such as `site_assignment`, `primary_site_selection`, `teaching_room`, `contractor_end_date`, `preferred_name`, or `permission_site_scope`
- `subject_id`: local UUIDv7 subject or workflow id the value applies to
- `manual_value`: the normalized value used by the application, with sensitive subfields masked or encrypted according to the field policy
- `actor`: authenticated user or system actor that created the manual change
- `created_at`: UTC timestamp rendered in the configured district timezone
- `reason`: operator-entered reason, form source, or workflow source that explains why the manual value exists
- `related_reference`: optional IncidentIQ ticket, form id, change request, or workflow id when available
- `owner_role`: role responsible for review, such as HR, IT Admin, Site Admin, or Site Secretary
- `effective_from`: when the manual value begins to apply
- `expires_at` or `review_after`: required for temporary overrides, exception-list entries, and manual revocations unless the specific field is documented as permanent app-owned state
- `supersession_policy`: one of `clear_on_upstream_match`, `preserve_until_expiry`, `preserve_until_manual_clear`, `conflict_on_upstream_difference`, or `supersede_on_authoritative_replacement`
- `source_snapshot`: non-secret fingerprint or summary of the upstream value observed when the manual value was created

Sensitive data, provider credentials, raw SAML assertions, OAuth tokens, service-account JSON, private keys, and raw credential-bearing provider payloads must never be stored in manual-change metadata, audit logs, tickets, fixtures, or generated artifacts.

## Reconciliation Outcomes

Each sync or explicit refresh that sees both upstream and manual values must produce exactly one outcome for each affected field.

| Outcome | When it applies | Required behavior |
| --- | --- | --- |
| `accept_upstream` | No active manual value exists, the manual value expired, or the field is provider-owned and the manual value is only explanatory. | Update the projection from upstream, retain audit history, and show the upstream value as effective. |
| `preserve_manual` | The field is app-owned or has an active manual override whose supersession policy says to preserve it. | Keep the manual value effective, show current upstream value separately, and record that the manual value won by rule. |
| `clear_temporary_override` | The upstream value now matches the intended manual correction or the documented clear condition is met. | End-date the manual value, write an audit event with before/after effective values, and make the upstream value effective without requiring operator action. |
| `mark_conflict` | Upstream changes to a different value while an active manual value still claims precedence, or two upstream authorities disagree without a documented winner. | Preserve the last safe effective value, mark the field as conflict/review required, block write-capable downstream steps that depend on the field, and route the row to the documented owner. |
| `supersede_manual` | A higher-authority source replaces the record, such as an active Escape employee replacing a manual contractor record. | Mark the manual record inactive or invalid according to its field rule, copy only safe identity linkage needed for continuity, preserve audit history, and use the authoritative upstream-backed record for future planning. |
| `blocked_missing_context` | The field cannot be safely resolved because required source data, mapping, or authorization context is missing. | Do not guess. Keep the workflow in review/manual-action state with operator copy that names the missing context and owner. |

Write-capable workflows must use these outcomes as safety gates. A field in `mark_conflict` or `blocked_missing_context` cannot feed a provider mutation until the conflict is resolved, the workflow is replanned against current source truth, or the implementation plan defines a specific safe fallback path.

## Field-Level Rules

- Escape-backed employment status, hire/start/end dates, and active assignment participation are provider-owned. Dashboard manual entries may create review notes, workflow scheduling context, or local exceptions, but they must not overwrite Escape values.
- Manual non-Escape contractor records are app-owned until an active Escape record supersedes them. When superseded, the manual record remains in the database for audit, becomes inactive or invalid, and must not enqueue duplicate onboarding or offboarding work.
- A temporary site or room override must be tied to the owner role that can correct or approve that field. Site/room overrides should clear automatically when upstream later reports the same corrected value. They should conflict, not silently clear, when upstream reports a different value than both the old upstream snapshot and the manual value.
- An InformedK12-backed primary-site selection is a dashboard-managed site-assignment decision for an Escape-backed employee whose Escape site is incomplete, conflicting, stale, or not yet corrected. The selection may use a reviewed active InformedK12 attachment as evidence, but it must keep raw Escape site values, raw InformedK12 form values, parsed site signal, and active dashboard selection as separate facts. It has no fixed automatic expiration; it requires a review-after timestamp for HR visibility and persists until upstream Escape/Aeries data unambiguously matches the selected site, a superseding job assignment or form-supported decision replaces it, or HR/IT manually reconciles it with an audit reason. Later upstream disagreement marks a conflict and blocks dependent write-capable planning instead of silently overwriting the selection or Escape source values.
- Aeries student and teacher data remains upstream-owned. Student exception queues must drive correction in Aeries rather than local app edits. Teacher room context follows the existing rule: Aeries staff wins when Aeries staff, teacher, and scheduling disagree before local overrides apply.
- Preferred/display-name requests for eligible non-student identities are app-owned decisions once submitted through an authorized workflow. Downstream provider values are verification signals. If a provider later disagrees, the app should retry or surface propagation drift rather than treating the provider value as a new source decision.
- Manual permission grants, revocations, and site mappings follow `docs/product/permissions-model.md`. Manual revocations can deny effective in-app access even when Google or SAML still reports the grant, but the UI must identify that upstream cleanup remains required.
- Provider identifiers and account ownership fields are provider-owned. If an upstream identifier changes unexpectedly, reconciliation must verify continuity through known identifiers or mark the row for review rather than attaching the dashboard record to a new provider identity silently.
- Local workflow notes, audit events, approval state, carried-forward manual-review inputs, and workflow snapshots are application-owned history. Upstream correction can supersede future effective values but must not erase the record of the prior manual decision.

## UI Presentation

Operator-facing surfaces that show a reconciled value should use consistent plain-language labels:

- `Effective value`: the value the application is currently using for workflow planning or display
- `Upstream value`: the latest value observed from the named provider or source
- `Manual value`: the active dashboard-managed value, including owner and review/expiration metadata when visible to the persona
- `InformedK12 value`: the retained exact form value plus parsed site signal when a form is used as evidence
- `Dashboard site selection`: the app-level primary-site decision currently used for workflow planning
- `Needs review`: a conflict or missing context blocks safe automation
- `Cleared after upstream correction`: a temporary manual value was automatically ended because upstream now matches the intended correction
- `Superseded by Escape`: a manual contractor record or manual lifecycle value was replaced by an active Escape-backed record
- `Upstream cleanup required`: an in-app permission revocation or manual decision is effective locally but the source system still reports the old value

Rows and drawers should expose why the effective value won. For example: `Manual room override remains active until Aeries reports Room 204 or the review date passes.` Avoid vague copy such as `synced`, `updated`, or `overridden` without naming the source and rule.

Sensitive fields remain masked according to the viewer's role. The UI may show that a manual value exists without revealing the value to unauthorized personas.

## Audit Behavior

Reconciliation must write immutable audit events for material automatic decisions:

- automatic clearing of a temporary override after upstream correction
- automatic clearing or supersession of an InformedK12-backed primary-site selection after upstream correction or newer reviewed evidence
- supersession of a manual contractor record by an active Escape-backed record
- conflict creation, conflict owner changes, and conflict resolution
- manual value preserved over upstream change
- upstream value accepted over an expired or non-authoritative manual value
- permission revocation remaining effective while upstream Google/SAML still reports the grant

Audit events should include field key, subject id, source names, before/after effective values, manual-change id, reconciliation outcome, actor or system actor, timestamp, and non-secret upstream snapshot fingerprints. Audit diffs must redact or omit sensitive fields that the current audit viewer cannot see.

## Required Named Scenarios

These named scenarios are the minimum coverage expected before DB-backed canonical records or write-capable workflows depend on this reconciliation contract:

- `upstream-overwrites-manual`: an Escape-backed field with a non-authoritative manual value accepts the new upstream value and records why the manual value did not win
- `manual-preserved-over-upstream`: an active app-owned or approved override field keeps the manual value effective while displaying the changed upstream value
- `conflict-needs-review`: an upstream change conflicts with an active manual value or another upstream authority, blocks dependent write-capable workflow steps, and routes the row to the documented owner
- `temporary-override-cleared-after-upstream-correction`: a temporary site or room override is end-dated automatically after upstream reports the intended corrected value, with audit evidence
- `informedk12-primary-site-selection-preserved`: a reviewed InformedK12-backed primary-site selection stays effective for planning while Escape remains ambiguous, with Escape values, form values, parsed signal, and dashboard selection shown separately
- `informedk12-primary-site-selection-cleared`: an active InformedK12-backed primary-site selection is cleared or superseded when Escape/Aeries later unambiguously matches the selected site or a newer reviewed form-supported decision replaces it, with audit evidence
