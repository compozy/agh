# Memory v2 Test Plan

## Scope

Memory v2 validation covers controller-backed write, search, reindex, recall, provider lifecycle, extractor recovery, dream execution, daily/session artifacts, MemoryProvider behavior, and every public control surface: CLI, HTTP, UDS, native tool, extension host, Knowledge, Memory Settings, Session Inspector, generated CLI/API, and config lifecycle.

## Execution Matrix

- Scenario lane: `TC-SCEN-001`, `TC-SCEN-002`
- Integration lane: `TC-INT-001`, `TC-INT-002`, `TC-INT-003`, `TC-INT-004`, `TC-INT-005`
- UI lane: `TC-UI-001`, `TC-UI-002`, `TC-UI-003`
- Security lane: `TC-SEC-001`
- Regression lane: `TC-REG-001`

## Traceability Coverage

- task_01: controller-backed write contract and storage bootstrap
- task_02: memory_decisions persistence and audit view
- task_03: memory_events timeline and replay-safe capture
- task_04: memory_recall_signals and weighted recall scoring
- task_05: workspace_id routing and scope validation
- task_06: CLI surface for list/show/write/edit/delete/search
- task_07: HTTP control plane and transport parity
- task_08: UDS control plane and daemon parity
- task_09: native tool reachability and agent-manageable paths
- task_10: extension host read/write surface
- task_11: MemoryProvider registry and provider selection
- task_12: extractor queue and _system/extractor/failures handling
- task_13: dream runtime and _system/dreaming artifacts
- task_14: frozen snapshot invalidation and reload behavior
- task_15: session ledger and ledger.jsonl forensic output
- task_16: daily log rotation and _inbox coverage
- task_17: config lifecycle and Memory Settings updates
- task_18: Knowledge page and help-path updates
- task_19: Session Inspector visibility and session replay
- task_20: generated CLI/API documentation parity
- task_21: search and controller-backed write observability
- task_22: workspace/global/agent scope promotion and reset
- task_23: dream retry and provider disable/enable behavior
- task_24: release regression checklist and operator evidence

## Exit Criteria

- Every case remains **Status: Not Run** until executed by QA.
- Searchable writes are proven without reindex on CLI, HTTP, and UDS.
- Provider lifecycle uses path-selected identifiers only.
- Evidence captures memory_decisions, memory_events, memory_recall_signals, frozen snapshot behavior, _inbox, _system/extractor/failures, _system/dreaming, and ledger.jsonl artifacts.
