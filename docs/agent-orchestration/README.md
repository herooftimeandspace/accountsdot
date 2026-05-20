# Agent Orchestration

This directory documents the repo-local Symphony-style orchestration contract for Codex work in The WIZARD.

- [SPEC.md](SPEC.md) is the authoritative specification for issue eligibility, branch/worktree ownership, safety gates, observability, and handoff states.
- [../../WORKFLOW.md](../../WORKFLOW.md) is the concise workflow prompt/config contract that agents can load before claiming or reconciling issue work.

The contract is intentionally spec-first. It does not introduce a daemon, scheduler, credentials, provider writes, or a local runner. GitHub issues and pull requests remain the durable control plane until the repository explicitly documents and approves executable orchestration.
