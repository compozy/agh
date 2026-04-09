# Lifecycle Hooks Platform — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Core types and hook taxonomy | pending | medium | — |
| 02 | Declaration normalization, matchers, and ordering | pending | medium | task_01 |
| 03 | Executor contracts and implementations | pending | medium | task_01 |
| 04 | Generic pipeline with sync composition and guards | pending | high | task_02, task_03 |
| 05 | Async worker pool | pending | medium | task_01 |
| 06 | Hooks struct with typed dispatch, registry, and Notifier | pending | high | task_04, task_05 |
| 07 | Migrate skills hook parsing to new declarations | pending | medium | task_01 |
| 08 | Config and agent-definition hook declarations | pending | medium | task_01 |
| 09 | Wire Hooks in daemon — replace notifierFanout | pending | critical | task_06, task_07, task_08 |
| 10 | Integrate session, input, prompt, event, and agent dispatch | pending | high | task_06, task_09 |
| 11 | Integrate turn, message, and context dispatch | pending | medium | task_10 |
| 12 | Hook observability storage and HTTP introspection | pending | medium | task_09 |
