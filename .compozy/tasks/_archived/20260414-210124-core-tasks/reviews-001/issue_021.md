---
status: resolved
file: internal/extension/host_api_tasks.go
line: 162
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:8599893dc8d5
review_hash: 8599893dc8d5
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 021: Consider adding server-side filtering for task runs.
## Review Comment

`handleTasksRuns` fetches the entire `TaskView` (including all runs, events, dependencies, children) and then filters runs in-memory. For tasks with many runs, this could be inefficient.

Consider adding a dedicated `ListTaskRuns(ctx, taskID, query, actor)` method to the manager that applies filters at the database level, rather than fetching everything and filtering client-side.

## Triage

- Decision: `INVALID`
- Notes:
  The current implementation is functionally correct and already enforces read authority through the existing `task.Manager` surface. This comment is proposing a new manager capability (`ListTaskRuns` scoped by task) and corresponding Host API contract expansion, not pointing to an incorrect result or a failing behavior in the current code path.
  `handleTasksRuns` is scoped to a single task and uses the existing canonical `GetTask` view, which keeps the endpoint aligned with the current manager contract. Adding a database-level run listing path would require widening `internal/task` and `internal/extension` interfaces outside this batch’s scoped files for a speculative optimization.
  No production defect or regression was confirmed here, so this batch will not expand the task manager API solely for this suggestion.
  Resolution: Closed as a design enhancement request rather than a defect. No code change was made.
