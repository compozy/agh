# Core Tasks and Subtasks — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Bootstrap the `internal/task` domain | completed | high | — |
| 02 | Persist core task and run records in `globaldb` | completed | high | task_01 |
| 03 | Persist task dependencies, audit trail, and idempotency | completed | high | task_01, task_02 |
| 04 | Implement `TaskManager` creation, mutation, and identity rules | completed | critical | task_01, task_02, task_03 |
| 05 | Implement `TaskRun` lifecycle and propagated cancellation | completed | critical | task_04 |
| 06 | Wire the session bridge, dedicated subtask sessions, and boot recovery | completed | critical | task_01, task_05 |
| 07 | Add task and run API contracts plus core handlers | completed | high | task_04, task_05 |
| 08 | Expose task and run routes through HTTP and UDS | completed | medium | task_07 |
| 09 | Add the `agh task` CLI command group | completed | medium | task_08 |
| 10 | Integrate automation with task-backed work items | completed | high | task_05, task_06 |
| 11 | Integrate extension host APIs with the task domain | completed | high | task_05, task_06 |
| 12 | Integrate network ingress and channel binding for tasks | completed | high | task_05, task_06 |
| 13 | Add observe projections, health queries, and task metrics | completed | high | task_05, task_06, task_12 |
