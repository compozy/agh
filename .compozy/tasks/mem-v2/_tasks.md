# Memory v2 (Slice 1) — Task List

**GREENFIELD (alpha):** do not preserve legacy behavior for old local state. These tasks implement the approved Memory v2 Slice 1 TechSpec by hard-cutting the existing memory subsystem into the new contract, storage, controller, recall, provider, manageability, web, docs, and QA surfaces in one atomic program.

The normative inputs are `_techspec.md` and ADR-001 through ADR-012 in this directory. The repaired TechSpec remains the source of truth for behavior, sequencing intent, and delete targets; these tasks translate it into independently implementable execution slices with explicit dependencies.

## MVP Boundary

MVP includes all Slice 1 behavior from the approved TechSpec: stable workspace identity, per-workspace memory databases, controller-gated writes with `memory_decisions`, deterministic recall with live recall signals, the bundled `MemoryProvider`, daemon-owned extractor inbox, dreaming v2 promotion gates, session lineage plus forensic ledgers, CLI/HTTP/UDS/native-tool/extension manageability, minimum web surfaces, and runtime/docs co-ship. MVP is implemented by tasks `01-24`. Tasks `25-26` plan and execute release-grade QA. Post-MVP remains out of scope: Slice 2 compaction, Slice 3 vector retrieval, Slice 4 external providers, Slice 5 network federation, Slice 6 KG/bi-temporal memory.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Memory Contract Extraction and Hard Cut | completed | critical | - |
| 02 | Memory Schema and Workspace DB Identity | completed | critical | - |
| 03 | Atomic Store, Workspacedb, and Replay Core | completed | critical | task_01, task_02 |
| 04 | Scan Policy and Memory Prompt Assets | completed | high | task_01 |
| 05 | Write Controller and Decisions WAL | completed | critical | task_01, task_03, task_04 |
| 06 | Deterministic Recall, Signals, and Shadow Rules | completed | critical | task_01, task_03 |
| 07 | Local Provider and Registry Surface | completed | high | task_01, task_03, task_05, task_06 |
| 08 | Frozen Snapshot and Prompt Assembly | completed | high | task_06, task_07 |
| 09 | Memory Observability and SSE Hygiene | completed | critical | task_02, task_05, task_06 |
| 10 | Extractor Hook, Inbox, and Runtime Queue | completed | critical | task_05, task_09 |
| 11 | Dreaming Runtime and Promotion Gates | completed | critical | task_05, task_06, task_07 |
| 12 | Session Lineage and Ledger Materialization | completed | high | task_02, task_03 |
| 13 | Config and Settings Backend | completed | high | task_07, task_11 |
| 14 | Public Memory Contract Surface | completed | critical | task_05, task_06, task_07, task_10, task_11, task_12, task_13 |
| 15 | Codegen and Generated Consumer Refresh | completed | high | task_14 |
| 16 | HTTP and UDS Route Parity | completed | high | task_14 |
| 17 | CLI Memory Hard Cut | completed | high | task_14, task_16 |
| 18 | Native Tools and Extension Host Memory Surfaces | completed | high | task_07, task_14, task_16 |
| 19 | Daemon Wiring and Boundary Registration | completed | critical | task_08, task_09, task_10, task_11, task_12, task_13, task_17, task_18 |
| 20 | Web Knowledge Surface | completed | high | task_15, task_19 |
| 21 | Web Memory Settings Surface | completed | high | task_15, task_19 |
| 22 | Web Session Inspector Memory Surface | completed | medium | task_15, task_19 |
| 23 | Runtime Memory and Configuration Docs | completed | high | task_13, task_17, task_18, task_20, task_21, task_22 |
| 24 | CLI/API Reference and Discoverability Co-Ship | completed | high | task_15, task_16, task_17, task_18, task_23 |
| 25 | QA Plan and Test Coverage | completed | high | task_24 |
| 26 | Real-Scenario QA Execution | completed | critical | task_25 |
