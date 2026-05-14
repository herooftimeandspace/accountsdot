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
    _require_contains("docs/promotion-pipeline.md", *required_doc_text)
    _require_contains("README.md", "scripts/run_local_ci.py --target dev", "docs/promotion-pipeline.md")
    _require_contains("IMPLEMENTATION_PLAN.md", "Branch Gate Semantics", "docs/promotion-pipeline.md")
    _require_contains("TEST_MATRIX.md", "promotion-pipeline", "P0-0E-003")
    _require_contains("ENVIRONMENT_DATA_PLAYBOOK.md", "GitHub Branch Gates", "PROMOTION_PR_TOKEN")


def _validate_local_runner() -> None:
    _require_contains(
        "scripts/run_local_ci.py",
        'TARGETS = ("dev", "staging", "main")',
        "pipeline-static-validation",
        "go-repo-tests",
        "frontend-accessibility",
        "release-prep-static-validation",
    )


def _validate_release_prep() -> None:
    release_prep = _read(".github/workflows/release-prep-check.yml")
    if "REQUIRED BEFORE MERGE" not in _read(".github/workflows/promotion.yml"):
        raise AssertionError("promotion.yml must create main PRs with explicit release-readiness placeholders")
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
        if args.release_prep:
            _validate_release_prep()
    except AssertionError as exc:
        print(f"CI/CD promotion validation failed: {exc}", file=sys.stderr)
        return 1
    print("CI/CD promotion validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
