# Memory v2 Traceability

## Public Surface Map

- CLI: write, search, provider, extractor, reset, reload, recall trace
- HTTP: controller-backed write, search, Memory Settings, Session Inspector endpoints
- UDS: daemon-backed parity for CLI transport
- native tool: agent-managed read/search hooks
- extension host: MemoryProvider and hosted integration surfaces
- generated CLI/API: published docs and OpenAPI projections
- config lifecycle: defaults, bootstrap migration, enum normalization, workspace metadata
- Knowledge: operator-facing memory overview and usage model

## Storage and Artifact Map

- memory_decisions
- memory_events
- memory_recall_signals
- frozen snapshot
- _inbox
- _system/extractor/failures
- _system/dreaming
- ledger.jsonl

## Completed Task Mapping

- task_01 -> controller-backed write bootstrap and validation
- task_02 -> controller decision persistence
- task_03 -> event journal parity
- task_04 -> recall signal capture
- task_05 -> workspace_id scope binding
- task_06 -> CLI memory verbs
- task_07 -> HTTP memory verbs
- task_08 -> UDS memory verbs
- task_09 -> native tool access
- task_10 -> extension host exposure
- task_11 -> MemoryProvider lifecycle
- task_12 -> extractor queue and failures
- task_13 -> dream orchestration
- task_14 -> frozen snapshot reload
- task_15 -> session ledger.jsonl
- task_16 -> _inbox and daily flows
- task_17 -> Memory Settings config lifecycle
- task_18 -> Knowledge and operator docs
- task_19 -> Session Inspector workflow
- task_20 -> generated CLI/API publication
- task_21 -> search observability
- task_22 -> reset, reload, and promotion coverage
- task_23 -> retry and lifecycle hardening
- task_24 -> release regression evidence
