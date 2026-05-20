# Reference Input Snapshot Integrity

This runbook explains the repository-local reference input guard used by Phase 0 scenario `P0-0A-001`. The guard exists so development and staging runs fail early when required reference snapshots are missing instead of depending on a developer workstation path, cloud-drive mount, browser download, or undocumented local file.

## Startup Guard

The Go service validates the current mandatory reference input baseline during `config.Load()` before the HTTP server is created. The guard checks these repo-relative files:

- `docs/reference-inputs/README.md`
- `docs/reference-inputs/VENDORED_INVENTORY.md`
- `docs/reference-inputs/branding/Firefly.png`
- `docs/reference-inputs/branding/google-g.png`
- `docs/reference-inputs/branding/Wordmarks/Gold W black outline.png`

The guard also checks local Markdown links in `docs/reference-inputs/README.md` and `docs/reference-inputs/VENDORED_INVENTORY.md`. Local links must resolve inside the repository. Absolute workstation paths such as `/Users/...`, parent-directory escapes, and missing local targets fail validation.

When validation fails, startup returns an error beginning with `required repo-local reference input snapshots are missing` or `repo-local reference input documentation links are invalid`. The error names the exact repo-relative file or invalid link so the operator can restore the snapshot or fix the documentation reference before retrying the deployment.

## Required Versus Deferred Inputs

The startup-required baseline is intentionally narrow. It covers the reference files already used by the current DEV shell, design sync, and documentation inventory.

Additional provider SDK snapshots, legacy scripts, pipeline references, historical exports, and Zoom reference files may still be listed in planning docs as future implementation inputs. Those files must be vendored under `docs/reference-inputs/` and recorded in `docs/reference-inputs/VENDORED_INVENTORY.md` before implementation relies on them. Until that happens, code and staging deployments must not read those future inputs from workstation-only paths or treat them as present.

## Dev Verification

For `P0-0A-001` dev evidence, run:

```bash
go test ./internal/referenceinputs ./internal/config ./cmd/provisioner
```

This proves:

- the checked-in reference input baseline is present,
- missing required snapshots fail clearly with the missing repo-relative path,
- local reference-input Markdown links resolve within the repository,
- config loading fails before server startup when the reference input guard fails.

## Staging Verification

For staging promotion evidence, run the same guard from the staging deployment checkout before starting or promoting the service:

```bash
go test ./internal/referenceinputs ./internal/config ./cmd/provisioner
```

The staging evidence should record:

- repository revision or branch,
- command output showing the reference-input tests passed,
- confirmation that the deployment uses the checked-out `docs/reference-inputs/` tree,
- confirmation that no reference input path points to a workstation directory, cloud-drive mount, or missing artifact.

If staging startup fails with the reference input guard, restore the missing repo-local snapshot from the last known-good revision or add the intentionally required sanitized snapshot to `docs/reference-inputs/` with a matching `VENDORED_INVENTORY.md` entry before retrying promotion.
