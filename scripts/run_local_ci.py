#!/usr/bin/env python3
"""Run the local command set that mirrors the GitHub branch gates.

The script is intentionally small and dependency-free so a new contributor can
run it before opening a promotion PR. GitHub Actions calls the same script from
`.github/workflows/quality.yml`; that keeps the branch policy in one place
instead of letting local docs and hosted CI drift apart.
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from collections.abc import Sequence

TARGETS = ("dev", "staging", "main")


def _run(name: str, cmd: Sequence[str], *, dry_run: bool) -> int:
    printable = " ".join(cmd)
    print(f"==> {name}: {printable}", flush=True)
    if dry_run:
        return 0
    return subprocess.run(list(cmd), check=False).returncode


def _commands_for_target(target: str) -> list[tuple[str, list[str]]]:
    commands: list[tuple[str, list[str]]] = [
        ("npm-install", ["npm", "ci"]),
        ("pipeline-static-validation", [sys.executable, "scripts/check_ci_promotion.py"]),
        ("environment-role-static-validation", ["npm", "run", "environment-roles:check"]),
        ("go-repo-tests", ["make", "test"]),
        ("pen-drift-check", ["npm", "run", "pen:check"]),
        ("pen-lint", ["npm", "run", "pen:lint"]),
        ("frontend-build", ["npm", "run", "build:web"]),
    ]
    if target in {"staging", "main"}:
        commands.extend(
            [
                ("security", ["make", "security"]),
                ("frontend-accessibility", ["npm", "run", "a11y:check"]),
            ],
        )
    if target == "main":
        commands.append(("release-prep-static-validation", [sys.executable, "scripts/check_ci_promotion.py", "--release-prep"]))
    return commands


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Run the same branch-specific checks that GitHub Actions enforces.",
    )
    parser.add_argument(
        "--target",
        choices=TARGETS,
        default="dev",
        help="Branch policy to mirror locally. Defaults to dev.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print the commands for the selected target without executing them.",
    )
    args = parser.parse_args()

    for name, cmd in _commands_for_target(args.target):
        if _run(name, cmd, dry_run=args.dry_run) != 0:
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
