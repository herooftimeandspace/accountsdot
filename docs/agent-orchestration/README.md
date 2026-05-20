# Agent Orchestration

This directory defines the workspace-specific version of OpenAI's Symphony orchestration pattern for The WIZARD repository.

Symphony's useful idea is not a large control-plane product. It is a small, durable contract: the issue tracker is the work queue, each eligible issue receives an isolated agent workspace, the repository owns the workflow prompt, and humans review the resulting branches and pull requests. The OpenAI reference describes this as a written specification that supervises agentic work, with the tracker acting as the control plane.

For this repository, the tracker is GitHub Issues rather than Linear. The safety posture is also stricter because this project handles school district identity, personnel, room, phone, and provider-sync workflows. Agents may prepare code, documentation, tests, UI artifacts, and review packets, but they must not perform production provider writes, bypass staging gates, commit secrets, or weaken the documented dev/staging/main promotion model.

## Files

- `SPEC.md` defines the local service contract, domain model, state policy, workspace layout, retry behavior, observability, and safety boundaries for a future long-running runner.
- `../../.agents/WORKFLOW.md` is the repo-owned workflow prompt and front matter a runner should load before dispatching an issue to Codex.

## Current Status

This is a spec-first slice. It creates the durable contract needed by future automation, but it does not start a background daemon or grant any new credentials. A future implementation should treat this directory and `.agents/WORKFLOW.md` as the source of truth for runner behavior.

