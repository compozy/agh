# Tasks UI — Task List

**GREENFIELD (alpha):** nao sacrificar qualidade por retrocompatibilidade. Mudancas breaking sao aceitaveis quando alinhadas ao TechSpec; evitar shims, compat layers, ou read models improvisados so para preservar limites antigos.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | First-class task semantics and validation | completed | high | — |
| 02 | Persistent task fields and lifecycle reconciliation | pending | critical | task_01 |
| 03 | Enriched task reads and draft publication | pending | high | task_02 |
| 04 | Task live timelines, streams, and run detail views | pending | high | task_02 |
| 05 | Observer-backed dashboard read models | pending | medium | task_02 |
| 06 | Inbox triage and approval read models | pending | high | task_02 |
| 07 | Task API contracts and OpenAPI codegen | pending | critical | task_03, task_04, task_05, task_06 |
| 08 | Shared task handlers in api/core | pending | high | task_07 |
| 09 | HTTP task transport and route wiring | pending | medium | task_08 |
| 10 | UDS task transport and parity coverage | pending | medium | task_08 |
| 11 | Host API parity for task read and aggregate surfaces | pending | medium | task_07 |
| 12 | Tasks entrypoint and route shell | pending | medium | task_09, task_10 |
| 13 | web/src/systems/tasks scaffold | pending | high | task_12 |
| 14 | List, kanban, empty-state, and create modal | pending | high | task_13 |
| 15 | Detail timeline and run detail routes | pending | high | task_13 |
| 16 | Dashboard and inbox routes | pending | high | task_13 |
| 17 | Multi-agent live route and live-state polish | pending | high | task_15 |
| 18 | Tasks QA plan and regression artifacts | pending | high | task_11, task_14, task_15, task_16, task_17 |
| 19 | Tasks QA execution and settings-aligned browser E2E | pending | critical | task_18 |
