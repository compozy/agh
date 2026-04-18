# Tasks UI — Task List

**GREENFIELD (alpha):** nao sacrificar qualidade por retrocompatibilidade. Mudancas breaking sao aceitaveis quando alinhadas ao TechSpec; evitar shims, compat layers, ou read models improvisados so para preservar limites antigos.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | First-class task semantics and validation | completed | high | — |
| 02 | Persistent task fields and lifecycle reconciliation | completed | critical | task_01 |
| 03 | Enriched task reads and draft publication | completed | high | task_02 |
| 04 | Task live timelines, streams, and run detail views | completed | high | task_02 |
| 05 | Observer-backed dashboard read models | completed | medium | task_02 |
| 06 | Inbox triage and approval read models | completed | high | task_02 |
| 07 | Task API contracts and OpenAPI codegen | completed | critical | task_03, task_04, task_05, task_06 |
| 08 | Shared task handlers in api/core | completed | high | task_07 |
| 09 | HTTP task transport and route wiring | completed | medium | task_08 |
| 10 | UDS task transport and parity coverage | completed | medium | task_08 |
| 11 | Host API parity for task read and aggregate surfaces | completed | medium | task_07 |
| 12 | Tasks entrypoint and route shell | completed | medium | task_09, task_10 |
| 13 | web/src/systems/tasks scaffold | completed | high | task_12 |
| 14 | List, kanban, empty-state, and create modal | completed | high | task_13 |
| 15 | Detail timeline and run detail routes | completed | high | task_13 |
| 16 | Dashboard and inbox routes | completed | high | task_13 |
| 17 | Multi-agent live route and live-state polish | completed | high | task_15 |
| 18 | Tasks QA plan and regression artifacts | pending | high | task_11, task_14, task_15, task_16, task_17 |
| 19 | Tasks QA execution and settings-aligned browser E2E | pending | critical | task_18 |
