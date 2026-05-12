# Implemented-Page Annotation Ledgers

This directory holds durable annotation ledgers for implemented `.pen`-backed pages. Before an annotation-driven UI pass starts, active Codex annotate feedback must be copied into the relevant page section below or into a more specific page ledger.

Each row must keep the same columns:

| ID | Page | Source | Layer | Expected fix location | Status | Durable guard |
| --- | --- | --- | --- | --- | --- | --- |

Valid layers: `pipeline`, `.pen layout`, `docs/new behavior`, `runtime behavior`, `review artifact`.

Valid statuses: `open`, `closed`, `reclassified as behavior`, `accepted exception`, `still failing`.

Closed annotations remain documented until the fix is protected by a lint rule, shared primitive rule, docs update, or explicit one-time-fix note.
