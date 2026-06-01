# CI/CD Promotion Pipeline

This document defines the checked-in CI/CD contract for moving The WIZARD through `dev`, `staging`, and `main`. It is written for a new contributor who needs to understand which commands run, which branch owns each gate, and which GitHub settings must be configured manually because workflows cannot enforce them by themselves.

## Branch Gate Semantics

The repository uses three long-lived branches:

| Branch | Role | Gate | Promotion Result |
| --- | --- | --- | --- |
| `dev` | Developer integration branch backed by mocks by default. | `dev gate` in `.github/workflows/quality.yml`. | A successful authoritative `dev` push refreshes a promotion PR from `dev` to `staging`. |
| `staging` | Realistic proving ground using sandbox or masked production-derived data. | `staging gate`, which includes the `dev gate` command set plus security and frontend accessibility checks. | A successful authoritative `staging` push refreshes a promotion PR from a disposable `promote/staging-to-main` branch to `main`. |
| `main` | Production branch. | `main gate`, which includes the `staging gate` command set plus release-prep static validation, and `release-prep-check` on main promotion PRs. | Merge only after external promotion runbook, IncidentIQ evidence, and deployment metadata are complete. |

`dev` must prove ordinary developer quality before work can be proposed for `staging`. `staging` must prove security, frontend, and environment-safe readiness before work can be proposed for `main`. `main` must prove that the deployment candidate is based on the latest `staging` revision and that release evidence has been recorded outside the repository before production merge.

## Local Commands

Run the same target names locally before opening or merging promotion work:

```bash
python3 scripts/run_local_ci.py --target dev
python3 scripts/run_local_ci.py --target staging
python3 scripts/run_local_ci.py --target main
```

Use `--dry-run` to print the command list without executing it:

```bash
python3 scripts/run_local_ci.py --target main --dry-run
```

The target command sets intentionally build on one another:

- `dev`: `npm ci`, CI/CD static validation, `make test`, `.pen` drift check, `.pen` lint, and frontend production build.
- `staging`: everything in `dev`, plus `make security` and `npm run a11y:check`.
- `main`: everything in `staging`, plus release-prep static validation.

The static validator is:

```bash
python3 scripts/check_ci_promotion.py
python3 scripts/check_ci_promotion.py --release-prep
```

It checks only repository-owned workflow and documentation contracts. The check
also verifies that every Go entrypoint used by branch gates, local/container
fallbacks, Compose, and the deployment build image is pinned to the patched Go
toolchain version documented in the repository. That prevents a vulnerable
toolchain rollback from reaching `make security` and reopening a staging
govulncheck failure. It does not replace GitHub branch protection, environment
protection, external IncidentIQ evidence, or the external promotion runbook.

### Smallest Safe Main-Promotion Proof

When the application is not stable enough to promote to production, prove the
main-promotion pipeline without merging anything to `main` and without starting
production deployment work. The safe proof is repository-local and PR-shape
focused:

```bash
python3 scripts/check_ci_promotion.py --release-prep
python3 scripts/run_local_ci.py --target staging --dry-run
python3 scripts/run_local_ci.py --target main --dry-run
```

This proof confirms the checked-in workflows and local branch-gate mapping,
including the `main` gate command set, without running a hosted `main` workflow
or merging a promotion PR. The release-prep validator also checks that the
main-promotion PR guard is a `pull_request` check for `main`, rejects branches
other than `promote/staging-to-main`, requires the candidate to contain the
latest `staging` tip, rejects committed `frontend/dist` output, and blocks while
the external promotion runbook, IncidentIQ testing ticket, or release/deployment
metadata placeholders remain unresolved.

Do not merge the generated `promote/staging-to-main` PR until maintainers are
ready for production promotion and the required external evidence exists. A
blocked or placeholder-bearing main promotion PR is useful evidence that the
gate is wired correctly; it is not approval to deploy.

## GitHub Workflows

`.github/workflows/quality.yml` runs the branch gates on pull requests, pushes to `dev`, `staging`, and `main`, manual dispatches, and the weekly schedule. Pull requests created by promotion automation still run normal `pull_request` checks because promotion PRs are authored with `PROMOTION_PR_TOKEN`, not `github.token`.

`.github/workflows/promotion.yml` listens for successful `quality` workflow runs:

- A successful push validation on `dev` creates or refreshes the open `dev` to `staging` promotion PR.
- A successful push validation on `staging` force-refreshes `promote/staging-to-main` from the validated staging commit, then creates or refreshes the open PR from that disposable branch to `main`.

Bootstrap note: if `main` is still missing `quality.yml`, `promotion.yml`, or `release-prep-check.yml`, treat that as an incomplete pipeline bootstrap rather than a reason to copy workflow files by hand. First land the workflow set on `dev`, allow the `dev` gate to open or refresh the `dev` to `staging` PR, merge only after the `staging` gate is clean, and then let the successful `staging` push open or refresh the `promote/staging-to-main` PR. The default branch workflow inventory is complete only after that documented promotion PR merges to `main` and `main` contains the same three workflow files.

Issue #197 remains open for the post-bootstrap promotion verification after the workflow set lands. Closing it requires confirming the `dev -> staging -> main` promotion path is present on the default branch through the documented PR flow, that the expected gate checks are registered on the relevant branch heads, that `PROMOTION_PR_TOKEN` is configured with the minimum required permissions, and that any required staging or production GitHub environments and branch-protection rules are manually configured by maintainers.

`.github/workflows/release-prep-check.yml` runs on PRs targeting `main`. It enforces the current release-prep policy:

- the PR must come from `promote/staging-to-main`;
- the PR head must contain the current `origin/staging` tip;
- committed `frontend/dist` output is rejected because release builds must be generated from the approved revision during deployment;
- the PR body must replace the placeholders for `External promotion runbook:`, `IncidentIQ testing ticket:`, and `Release/deployment metadata:` before merge.

## Promotion PR Metadata

This repository does not currently define package-version files, semver labels, deployment manifests, or release-note generation rules for The WIZARD. Until those product decisions exist in `docs/planning/implementation-plan.md` and this document, workflows must not invent a version bump, deployment target, or release-note format.

Promotion PRs therefore carry release metadata in their body:

- `dev` to `staging` PRs identify the validated source commit and workflow run, then remind reviewers to verify dev evidence and write-path rollback documentation before staging validation starts.
- `staging` to `main` PRs identify the validated source commit and workflow run, then include required placeholders for the external promotion runbook, the external IncidentIQ testing ticket, and release/deployment metadata. The `release-prep-check` workflow blocks merge while those placeholders remain unresolved.

If the project later adopts semver labels, release notes, deployment manifests, or package version files, the decision must be documented before the workflow mutates those files or labels.

## Manual GitHub Settings Required

The following settings are manual repository-administration prerequisites and cannot be guaranteed by checked-in workflows:

- Confirm long-lived `dev`, `staging`, and `main` branches exist.
- Set the default integration branch to the branch maintainers choose for feature work, currently expected to be `dev` unless project policy changes.
- Protect `dev`, `staging`, and `main` with required pull requests and required status checks.
- Require `dev gate` before merging feature work into `dev`.
- Require `dev gate` and `staging gate` before merging promotion PRs into `staging`.
- Require `dev gate`, `staging gate`, `main gate`, and `release-prep-check` before merging promotion PRs into `main`.
- Block direct pushes to protected branches except for explicitly approved repository automation. Prefer promotion PRs over branch-protection bypasses.
- Configure required reviews, stale-review dismissal, conversation resolution, and administrator enforcement according to maintainer policy.
- Add a repository or organization secret named `PROMOTION_PR_TOKEN`. The token must be able to create and edit pull requests and push the disposable `promote/staging-to-main` branch. Use the minimum permissions practical for the chosen GitHub account or app.
- Confirm Actions permissions allow workflow read access, check read access, pull-request write access, and content write access only where the workflow needs them.
- Create and protect any required GitHub environments such as `staging`, `production`, or `integration-tests` before live deployment or integration jobs depend on them.
- Keep staging and production deployment secrets separate. Do not reuse production credentials in staging.
- Configure badge, Pages, artifact, or deployment publishing settings only after the repository documents those destinations.
- Confirm the dedicated promotion token triggers downstream `pull_request` workflows; promotion PRs created with `github.token` are not acceptable because required checks can be left missing.

## Safety Model

The pipeline follows the environment safety rules in `docs/operations/environment-data-playbook.md`:

- `dev` uses mocks by default and may be freely breakable.
- `staging` is the proving ground for representative data, sandbox providers, masked production-derived data, and write-path safety.
- `main` is production and must never be the first place a new write path is exercised.

Workflow output, badge payloads, promotion PR bodies, and release metadata must never contain provider credentials, raw service-account JSON, bearer tokens, private keys, passwords, or production data snapshots. Evidence links belong in the external IncidentIQ testing ticket and external promotion runbook; the repository defines the rules but is not the live release ledger.
