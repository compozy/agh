---
status: resolved
file: internal/api/core/tasks.go
line: 395
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107556463,nitpick_hash:d9fff2a61f74
review_hash: d9fff2a61f74
source_review_id: "4107556463"
source_review_submitted_at: "2026-04-14T16:23:06Z"
---

# Issue 001: Push run filtering and limiting into the task service.
## Review Comment

`ListTaskRuns` currently loads the full task view and then filters runs in memory. For tasks with many runs, that turns a paged run listing into a full detail read and bypasses storage-side filtering/limit enforcement. A dedicated `ListTaskRuns`/`GetTaskRuns` service call would scale much better here.

Also applies to: 1078-1092

## Triage

- Decision: `valid`
- Notes:
  The handler currently calls `manager.GetTask(...)`, which loads the full task view including children, dependencies, events, and every run, then applies `filterTaskRuns(...)` in memory. That defeats store-side run filtering and limit enforcement for `/tasks/:id/runs`.
  Root cause: the API task service does not expose a dedicated run-list method, so the handler reuses the full-detail read path.
  Planned fix: add a task-service `ListTaskRuns` path that preserves read authorization and task-not-found semantics, switch the handler to use it, and add regression coverage for the handler call path.

## Resolution

- Added a dedicated `ListTaskRuns` manager/service path in `internal/task/manager.go`, surfaced it through the API task-service interface/stub, and switched `internal/api/core/tasks.go` to use it instead of loading the full task graph.
- Updated `internal/api/core/tasks_test.go` to verify the runs handler delegates through the new list path rather than adding another full `GetTask` fetch.
