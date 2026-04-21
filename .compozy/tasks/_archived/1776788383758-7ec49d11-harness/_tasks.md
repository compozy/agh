# Harness - Task List

**GREENFIELD (alpha):** nao sacrificar qualidade por retrocompatibilidade. Mudancas breaking sao aceitaveis quando alinhadas ao TechSpec; evitar compat layers, eventos ambiguos, ou runtime paralelo so para preservar seams antigos.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Harness context resolver and turn-origin foundations | completed | high | - |
| 02 | Startup prompt section registry and network startup overlay | completed | high | task_01 |
| 03 | Ordered prompt augmentation composite over the current manager seam | completed | high | task_01 |
| 04 | Synthetic prompt submission and session-event persistence model | completed | high | task_01 |
| 05 | Transcript, hooks, and extension host support for synthetic turns | completed | high | task_04 |
| 06 | Detached harness work on task runtime metadata and submission paths | completed | high | task_01 |
| 07 | Task-run completion to synthetic reentry bridge | completed | critical | task_04, task_06 |
| 08 | Harness observability, event summaries, and integration hardening | completed | high | task_02, task_03, task_05, task_07 |
| 09 | Harness QA plan and regression artifacts | completed | high | task_08 |
| 10 | Harness QA execution and daemon/runtime E2E | completed | critical | task_09 |
