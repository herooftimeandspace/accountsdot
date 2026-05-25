# Breakglass Access

Breakglass access is the local emergency sign-in path for non-production recovery when Google SAML or Google Workspace role mapping is unavailable. It is separate from the DEV persona switcher and grants the local IT Admin persona only after the named local account, token hash, source address, and audit write all pass.

## Scope

- `POST /api/v1/breakglass/login` is enabled only when `APP_ENV` is `development` or `staging`.
- Production must continue to use the documented Google SAML authorization model.
- Normal DEV persona switching remains available only in `APP_ENV=development` through `/api/v1/dev/login`.
- In staging, DEV session and page routes can be consumed only by a valid breakglass cookie. Staging must not enable normal DEV persona switching.

## Account Configuration

Set `BREAKGLASS_ACCOUNTS` to a comma-separated list of named local emergency account ids. Use one named account per approved IT admin or emergency operator. Do not use a shared account.

Each listed account requires a matching SHA-256 token hash environment variable:

```text
BREAKGLASS_ACCOUNTS=emergency-alex,emergency-casey
BREAKGLASS_TOKEN_SHA256_EMERGENCY_ALEX=<sha256 hex digest supplied outside the repo>
BREAKGLASS_TOKEN_SHA256_EMERGENCY_CASEY=<sha256 hex digest supplied outside the repo>
```

The environment variable suffix is built by replacing non-alphanumeric characters in the account id with `_`, trimming leading or trailing `_`, and uppercasing the result. For example, `emergency-alex` and `emergency.alex` both map to `BREAKGLASS_TOKEN_SHA256_EMERGENCY_ALEX`; that collision makes the whole breakglass configuration invalid so two named accounts cannot accidentally share one credential.

Missing or malformed token hashes also make the whole breakglass configuration invalid. The login route returns `breakglass_configuration_invalid` and does not issue a session cookie.

Raw breakglass tokens are secrets. Do not commit them, paste them into issues or pull requests, write them to logs, include them in fixtures, or store them in generated artifacts. Staging token hashes must be supplied by environment configuration or a managed secret system outside this repository.

## Source Address Restrictions

By default, breakglass login is restricted to these source networks:

- `10.23.0.0/16`
- `10.19.100.0/24`

Use `BREAKGLASS_ALLOWED_CIDRS` to replace the default list when staging needs a different approved source-address policy:

```text
BREAKGLASS_ALLOWED_CIDRS=10.23.0.0/16,10.19.100.0/24
```

Direct requests are checked against `RemoteAddr`. `X-Forwarded-For` is ignored unless the immediate peer address is listed in `BREAKGLASS_TRUSTED_PROXY_CIDRS`:

```text
BREAKGLASS_TRUSTED_PROXY_CIDRS=192.0.2.0/24
```

Only configure trusted proxy CIDRs when staging traffic actually arrives through a known reverse proxy that sets forwarded client addresses. Without that explicit trusted-proxy setting, an untrusted client cannot bypass the source-address check by sending its own `X-Forwarded-For` header.

## Login And Session Behavior

A successful login returns the same local session payload shape used by the DEV frontend session endpoint, with these breakglass-specific fields:

- `authentication_mode: "breakglass"`
- `breakglass_account_id: "<named account id>"`
- `current_persona.id: "it_admin"`

The session cookie value is scoped with the internal `breakglass:` prefix and is `HttpOnly`, `SameSite=Lax`, and `Secure` in staging or on HTTPS requests. The cookie expiration is twelve hours.

Cookie expiration is not actively audited in this cookie-only Phase 0 slice because the server does not yet have a durable session table or expiration worker. Before promotion claims expiration audit evidence, the release owner must document whether cookie expiration requires a durable session table, an expiration audit worker, or an approved statement that explicit sign-out evidence is sufficient for this phase.

## Audit Evidence

Breakglass login and sign-out fail closed if audit storage cannot initialize or record the required event.

When `DATABASE_URL` is unset, database-free DEV tests use the process-local memory audit store. When `DATABASE_URL` is set, events are written to `audit_log` with actor type `breakglass_local_account` and sanitized metadata only. Audit rows must not include raw tokens.

Expected sanitized events include:

- `login_attempt` with outcome `allowed`
- `access_granted` with outcome `allowed`
- denied `login_attempt` for `unknown_account`
- denied `login_attempt` for `source_address_denied`
- denied `login_attempt` for `token_denied`
- `sign_out` with outcome `allowed` when the local session layer observes explicit breakglass logout

Useful verification commands for the implemented handler tests:

```bash
go test ./internal/web -run 'TestBreakglass|TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment/.*breakglass'
npm run docs:comments:check
git diff --check
```
