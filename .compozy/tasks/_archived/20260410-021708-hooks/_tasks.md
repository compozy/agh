# Lifecycle Hooks Platform — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Core types and hook taxonomy | completed | medium | — |
| 02 | Declaration normalization, matchers, and ordering | completed | medium | task_01 |
| 03 | Executor contracts and implementations | completed | medium | task_01 |
| 04 | Generic pipeline with sync composition and guards | completed | high | task_02, task_03 |
| 05 | Async worker pool | completed | medium | task_01 |
| 06 | Hooks struct with typed dispatch, registry, and Notifier | completed | high | task_04, task_05 |
| 07 | Migrate skills hook parsing to new declarations | completed | medium | task_01 |
| 08 | Config and agent-definition hook declarations | completed | medium | task_01 |
| 09 | Wire Hooks in daemon — replace notifierFanout | completed | critical | task_06, task_07, task_08 |
| 10 | Integrate session, input, prompt, event, and agent dispatch | completed | high | task_06, task_09 |
| 11 | Integrate turn, message, and context dispatch | completed | medium | task_10 |
| 12 | Hook observability storage and HTTP introspection | completed | medium | task_09 |
