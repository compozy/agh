# Tasks: Cross-Session Memory System

## Task List

| #   | Title                               | Status  | Complexity | Dependencies              |
| --- | ----------------------------------- | ------- | ---------- | ------------------------- |
| 01  | Memdir Core Package                 | completed | medium   | —                         |
| 02  | Dream Consolidation Package         | completed | high     | task_01                   |
| 03  | Prompt Assembler Memory Integration | completed | medium   | task_01                   |
| 04  | Kernel & Session Integration        | completed | high       | task_01, task_02, task_03 |
| 05  | CLI Memory Commands                 | completed | medium   | task_04                   |

## Dependency Graph

```
task_01 (memdir core)
  ├── task_02 (dream) ──────┐
  └── task_03 (prompt) ─────┤
                            ▼
                     task_04 (kernel integration)
                            │
                            ▼
                     task_05 (CLI)
```

Tasks 02 and 03 can be implemented in parallel.
