# Memory v2 Regression Inventory

## Goal

Protect shipped Memory v2 behavior from regressions across CLI, HTTP, UDS, native tool, extension host, generated CLI/API, and config lifecycle.

## Required Lanes

- Search and write behavior: `TC-SCEN-001`, `TC-INT-001`
- Provider lifecycle and route identity: `TC-SCEN-002`, `TC-INT-003`
- Extractor and dream runtime: `TC-INT-004`, `TC-INT-005`
- Operator-facing UI: `TC-UI-001`, `TC-UI-002`, `TC-UI-003`
- Security and audit: `TC-SEC-001`
- Broad regression sweep: `TC-REG-001`

## Regression Anchors

- controller-backed write stays searchable without reindex.
- workspace_id remains explicit across HTTP and UDS payloads.
- MemoryProvider enable/disable uses the route-selected provider.
- memory_decisions, memory_events, and memory_recall_signals remain queryable.
- frozen snapshot invalidation, _inbox ingestion, _system/extractor/failures recovery, _system/dreaming promotion, and ledger.jsonl inspection stay operator-visible.
