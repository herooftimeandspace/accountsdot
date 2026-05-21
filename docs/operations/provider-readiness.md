# Provider Readiness

## Purpose

Provider readiness verifies that The WIZARD can construct provider-facing clients before synchronization or workflow workers depend on them. Phase 0 readiness is intentionally conservative: development uses mock clients, and staging may use only read-only connectivity probes with non-production credentials.

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

The configuration contains provider names, mock/read-only flags, endpoint labels, and non-secret credential labels. It must not contain tokens, client secrets, private keys, auth headers, raw service-account JSON, passwords, or certificate material.

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
- a non-mock read-only client has no probe,
- the read-only probe returns an error.

Failure diagnostics should name the provider and readiness boundary while avoiding secret values. Scenario `P0-0D-002` owns the detailed missing or bad credential failure-surfacing evidence.
