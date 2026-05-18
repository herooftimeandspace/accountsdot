# Permissions Matrix

This document records the current production authorization boundary for The WIZARD. It does not replace the editable permissions model or breakglass runtime work; it defines the Google SAML and Google group/attribute contract those follow-up surfaces must preserve.

## Source Order

- `PRODUCT_REQUIREMENTS.md` defines the staff-only product boundary, allowed domains, explicit student denial, and same-URL access-denied requirement.
- `IMPLEMENTATION_PLAN.md` defines the implementation contract, staged rollout constraints, and follow-up work still required before production SAML is live.
- `internal/auth/production.go` contains the current checked-in evaluator for verified Google identity data.
- `TEST_MATRIX.md` defines the scenarios that must be evidenced in dev and staging.

## Production Auth Flow

1. Google Workspace authenticates the user through SAML.
2. The application receives verified identity data from the future SAML middleware: email address, group memberships, and configured SAML attributes.
3. The application canonicalizes the email address and applies the domain gate before any normal role authorization.
4. The application denies `@stu.wusd.org` even if Google groups or attributes would otherwise map to an application role.
5. The application allows only `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org` for normal SAML users.
6. The future breakglass runtime may bypass the domain gate only for named local emergency accounts after its own local-auth and network-source checks pass.
7. The application maps Google groups and SAML attributes to stable role ids.
8. A user with no mapped role is authenticated but not authorized and must receive access denied.
9. The application maps current Google groups and SAML attributes to site scopes on each authorization evaluation so changed assignments do not leave stale cross-site access.
10. Route and API handlers must enforce the resulting role and scope server-side. Frontend hiding is not a production authorization control.

## Stable Role IDs

| Role ID | Product Role | Scope Expectation |
| --- | --- | --- |
| `it_admin` | IT Admin | District-wide by default. |
| `human_resources` | Human Resources | District-wide lifecycle visibility for HR-owned workflows. |
| `site_admin` | Administrative Staff / Site Admin Staff | Site-scoped by default, with multi-site expansion only from approved Google group or attribute mappings. |
| `site_secretary` | Site Secretary | Site-scoped student cleanup and room-move participation. |
| `device_wrangler` | Device Wrangler | Site-scoped student device-accountability reporting. |
| `faculty_staff` | Faculty and Staff | Limited self-service only. |

## Configuration Contract

The checked-in environment contract is:

- `AUTH_ALLOWED_EMAIL_DOMAINS`: comma-separated normal SAML domains. Default: `wusd.org,it.wusd.org,staff.wusd.org`.
- `AUTH_DENIED_EMAIL_DOMAINS`: comma-separated explicit denied domains. Default: `stu.wusd.org`.
- `GOOGLE_SAML_ENTITY_ID`: service-provider entity id configured in Google Workspace.
- `GOOGLE_SAML_ACS_URL`: assertion consumer service URL configured in Google Workspace.
- `GOOGLE_SAML_IDP_METADATA_URL`: Google-hosted metadata URL when the deployment uses metadata discovery.
- `GOOGLE_SAML_IDP_SSO_URL`: Google IdP sign-in URL when metadata discovery is not used.
- `GOOGLE_SAML_IDP_CERT_FILE`: path to the deployment-managed IdP certificate file. Do not commit certificate material.
- `GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON`: JSON array of group-to-role mappings.
- `GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON`: JSON array of SAML-attribute-to-role mappings.
- `GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON`: JSON array of group or SAML-attribute-to-site-scope mappings.

Example mapping shape:

```json
[
  {
    "group": "wizard-it-admins@wusd.org",
    "roles": ["it_admin"]
  }
]
```

```json
[
  {
    "attribute": "wizard_role",
    "values": ["Site Secretary"],
    "roles": ["site_secretary"]
  }
]
```

```json
[
  {
    "source_type": "group",
    "source": "wizard-bpl-scope@wusd.org",
    "sites": ["bpl"]
  },
  {
    "source_type": "attribute",
    "source": "wizard_site",
    "values": ["Clover HS"],
    "sites": ["clover-hs"]
  }
]
```

## Current Gaps

- The current branch adds the evaluator, config contract, tests, and docs. It does not yet validate SAML assertions or issue a production session.
- Google Workspace admin decisions are still needed for the exact group names, SAML attribute names, ACS URL, metadata source, and certificate delivery method.
- Persistent manual site-scope administration is still follow-up work. Until it exists, production site scopes should come from deployment-managed mapping JSON or Google group/attribute inputs.
- Breakglass runtime behavior is intentionally owned outside this production Google SAML slice.
