#!/usr/bin/env python3
"""Validate the checked-in CI/CD promotion contract.

This static check does not try to emulate GitHub branch protection. It verifies
the repository-owned pieces that can drift during normal edits: workflow files,
local target names, promotion-token usage, release-readiness placeholders, and
the durable docs that explain the manual settings maintainers must configure in
GitHub.
"""

from __future__ import annotations

import argparse
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
EXPECTED_GO_VERSION = "1.26.3"


def _read(relative_path: str) -> str:
    path = ROOT / relative_path
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        raise AssertionError(f"Missing required CI/CD file: {relative_path}") from None


def _require_contains(relative_path: str, *needles: str) -> None:
    text = _read(relative_path)
    missing = [needle for needle in needles if needle not in text]
    if missing:
        joined = ", ".join(repr(needle) for needle in missing)
        raise AssertionError(f"{relative_path} is missing required text: {joined}")


def _validate_workflows() -> None:
    _require_contains(
        ".github/workflows/quality.yml",
        "name: quality",
        "dev-gate:",
        "staging-gate:",
        "main-gate:",
        "python3 scripts/run_local_ci.py --target dev",
        "python3 scripts/run_local_ci.py --target staging",
        "python3 scripts/run_local_ci.py --target main",
        "pull_request:",
        "branches:",
        "- dev",
        "- staging",
        "- main",
    )
    _require_contains(
        ".github/workflows/promotion.yml",
        "name: promotion",
        "PROMOTION_PR_TOKEN",
        "github.token",
        "--base staging",
        "--head dev",
        "promote/staging-to-main",
        "--base main",
    )
    _require_contains(
        ".github/workflows/release-prep-check.yml",
        "name: release-prep-check",
        "promote/staging-to-main",
        "External promotion runbook:",
        "IncidentIQ testing ticket:",
        "Release/deployment metadata:",
        "frontend/dist",
    )


def _validate_docs() -> None:
    required_doc_text = (
        "Branch Gate Semantics",
        "PROMOTION_PR_TOKEN",
        "dev gate",
        "staging gate",
        "main gate",
        "external IncidentIQ testing ticket",
        "External promotion runbook",
        "semver",
    )
    _require_contains("docs/operations/promotion-pipeline.md", *required_doc_text)
    _require_contains("README.md", "scripts/run_local_ci.py --target dev", "docs/operations/promotion-pipeline.md")
    _require_contains("docs/planning/implementation-plan.md", "Branch Gate Semantics", "docs/operations/promotion-pipeline.md")
    _require_contains("docs/testing/test-matrix.md", "promotion-pipeline", "P0-0E-003")
    _require_contains("docs/operations/environment-data-playbook.md", "GitHub Branch Gates", "PROMOTION_PR_TOKEN")


def _validate_local_runner() -> None:
    _require_contains(
        "scripts/run_local_ci.py",
        'TARGETS = ("dev", "staging", "main")',
        "pipeline-static-validation",
        "go-repo-tests",
        "frontend-accessibility",
        "release-prep-static-validation",
    )


def _validate_go_toolchain() -> None:
    """Keep branch-gate and container Go versions pinned to the patched runtime.

    The staging gate runs govulncheck through `make security`. GO-2026-4971
    made Go 1.26.2 an unsafe promotion runtime, so the repository-owned
    promotion validator checks every Go entrypoint used by local gates,
    container fallbacks, Compose, and the deployment image before a branch can
    advance through the documented pipeline.
    """

    _require_contains("go.mod", f"go {EXPECTED_GO_VERSION}")
    _require_contains("Makefile", f"GO_IMAGE ?= golang:{EXPECTED_GO_VERSION}")
    _require_contains("compose.yaml", f"image: golang:{EXPECTED_GO_VERSION}")
    _require_contains("deploy/Dockerfile", f"FROM golang:{EXPECTED_GO_VERSION} AS build")


def _validate_release_prep() -> None:
    release_prep = _read(".github/workflows/release-prep-check.yml")
    promotion = _read(".github/workflows/promotion.yml")
    if "REQUIRED BEFORE MERGE" not in promotion:
        raise AssertionError("promotion.yml must create main PRs with explicit release-readiness placeholders")
    if "pull_request:" not in release_prep or "branches:" not in release_prep or "- main" not in release_prep:
        raise AssertionError("release-prep-check.yml must run on pull requests targeting main")
    if "workflow_run:" in release_prep or "push:" in release_prep:
        raise AssertionError("release-prep-check.yml must not run from push or workflow_run events")
    if "github.event.pull_request.head.repo.full_name == github.repository" in release_prep:
        raise AssertionError("release-prep-check.yml must fail cross-repository PRs inside a step, not skip the job")
    if '"${GH_HEAD_REPO}" != "${GH_REPO}"' not in release_prep:
        raise AssertionError("release-prep-check.yml must explicitly reject cross-repository promotion PRs")
    if '"${GH_HEAD_REF}" != "promote/staging-to-main"' not in release_prep:
        raise AssertionError("release-prep-check.yml must reject main PRs outside promote/staging-to-main")
    if "git fetch origin staging" not in release_prep or "git merge-base --is-ancestor origin/staging HEAD" not in release_prep:
        raise AssertionError("release-prep-check.yml must require the main candidate to contain the latest staging tip")
    if "git ls-files frontend/dist" not in release_prep:
        raise AssertionError("release-prep-check.yml must reject committed frontend/dist release evidence")
    if "grep -Eiq 'REQUIRED BEFORE MERGE|TBD|TODO|missing'" not in release_prep:
        raise AssertionError("release-prep-check.yml must fail unresolved release-readiness placeholders")


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate CI/CD promotion files and docs.")
    parser.add_argument(
        "--release-prep",
        action="store_true",
        help="Also validate main-promotion release-prep checks.",
    )
    args = parser.parse_args()

    try:
        _validate_workflows()
        _validate_docs()
        _validate_local_runner()
        _validate_go_toolchain()
        if args.release_prep:
            _validate_release_prep()
    except AssertionError as exc:
        print(f"CI/CD promotion validation failed: {exc}", file=sys.stderr)
        return 1
    print("CI/CD promotion validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
