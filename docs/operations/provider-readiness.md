# Provider Readiness

## Purpose

Provider readiness verifies that The WIZARD can construct provider-facing clients before synchronization or workflow workers depend on them. Phase 0 readiness is intentionally conservative: development uses mock clients, and staging may use only read-only connectivity probes with non-production credentials. The same readiness layer must fail closed when a live-mode provider is missing a required non-secret credential label, has a malformed HTTPS endpoint, or references an invalid certificate file.

Readiness is not a writeback path. It must not create, update, disable, delete, license, unlicense, move, rename, ticket, or otherwise mutate a provider-owned object.

## Development Mock Success Path

Phase 0 scenario `P0-0D-001` is covered by:

```bash
go test ./internal/provider ./internal/config -run 'TestP000D001'
```

That test initializes readiness clients for:

- `zoom`
- `google`
- `aeries`
- `sftp`

Each client runs in `mock` mode, returns `status: ok`, reports `writeback: disabled`, and records that the outbound probe was skipped. The test fails if a mock readiness client calls the injected outbound probe, which proves the DEV success path does not contact real providers or exercise provider writes.

## Configuration

`internal/config.Load` builds sanitized readiness metadata from environment variables. Local development defaults every provider to mock mode:

- `USE_MOCK_ZOOM=true`
- `USE_MOCK_GOOGLE=true`
- `USE_MOCK_AERIES=true`
- `USE_MOCK_SFTP=true`

The configuration contains provider names, mock/read-only flags, endpoint labels, non-secret credential labels, and certificate file paths used only for local shape validation. It must not contain tokens, client secrets, private keys, auth headers, raw service-account JSON, passwords, or certificate material. Health diagnostics must never echo the configured endpoint value, credential label value, certificate path, or file contents.

Live-mode configuration diagnostics are exposed under `/health/ready` and `/health` as `dependencies.provider_<name>` entries. Ready states are:

- `mocked`: the provider is intentionally mock-backed.
- `ok`: the provider is live-mode, read-only, and passed local label, URL, and certificate-file validation.

Blocked states start with `blocked:` and make readiness return `503 Service Unavailable` with `status:"degraded"`. The diagnostic names the environment variable that needs attention, such as `ZOOM_ACCOUNT_ID`, `ZOOM_BASE_URL`, or `AERIES_CERT_FILE`, without copying secret-bearing values.

## Staging Read-Only Probes

Staging may disable a provider mock flag only after that provider has non-production or masked read-only credentials configured outside the repository. The readiness client then requires an injected read-only probe. A missing probe fails closed instead of silently reporting readiness.

Staging evidence should record:

- application revision and configuration revision,
- provider name,
- mock flag value,
- non-secret endpoint or account label,
- probe result,
- timestamp,
- confirmation that writeback remained disabled.

Do not paste credentials, private keys, certificates, auth headers, service-account JSON, or raw provider payloads into evidence, logs, issues, PRs, or generated artifacts.

## Failure Behavior

Readiness fails closed when:

- provider configuration omits the provider name,
- a non-mock client is not explicitly read-only,
- a non-mock client is missing the required non-secret credential label,
- a non-mock client has a missing, relative, or non-HTTPS endpoint,
- a certificate-backed provider has a missing, unreadable, non-PEM, or unparsable certificate file,
- a non-mock read-only client has no probe,
- the read-only probe returns an error.

Failure diagnostics should name the provider and readiness boundary while avoiding secret values.

## Failure Surfacing Verification

Phase 0 scenario `P0-0D-002` is covered by:

```bash
GOCACHE=$PWD/.gocache go test ./internal/provider ./internal/config ./internal/web ./cmd/provisioner -run 'TestP000D002|TestProviderConfigurationDiagnostics|TestHealthReadyFailsProviderReadiness|TestNewServerReportsProviderReadinessFailure'
```

That focused test command simulates:

- `USE_MOCK_ZOOM=false` with the non-secret credential label omitted,
- `USE_MOCK_ZOOM=false` with an `http://` endpoint,
- `USE_MOCK_AERIES=false` with a file that is not a PEM certificate,
- `USE_MOCK_SFTP=false` with `SFTP_USERNAME` set but `SFTP_HOST` omitted.

Expected result: provider configuration diagnostics start with `blocked:`, `/health/ready` returns `503`, `/health/live` remains `200`, and diagnostics name only environment labels.

For staging, perform the same drill with staging-safe mock, sandbox, or masked read-only configuration only. Temporarily disable one `USE_MOCK_*` flag with a known-bad non-production setting, request:

```bash
curl -sS -i https://<staging-host>/health/ready
curl -sS -i https://<staging-host>/health/live
```

Expected result: `/health/ready` returns `503 Service Unavailable` with an actionable `provider_<name>` diagnostic, while `/health/live` returns `200 OK`. Restore the prior staging provider credential/config bundle after the drill, verify readiness returns to the expected state, and record the before/after evidence in the external Phase 0 IncidentIQ testing ticket under `Phase 0 -> 0D -> staging -> P0-0D-002`.
