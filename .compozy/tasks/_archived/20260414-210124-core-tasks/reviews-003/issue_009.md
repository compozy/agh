---
status: resolved
file: internal/extension/host_api_tasks.go
line: 187
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107909499,nitpick_hash:d1c438039c88
review_hash: d1c438039c88
source_review_id: "4107909499"
source_review_submitted_at: "2026-04-14T17:26:14Z"
---

# Issue 009: Use ListTaskRuns for the runs endpoint instead of filtering GetTask().Runs.
## Review Comment

There is already a dedicated service query for this path. Pulling the full task view here means limit/status/session filtering happens after the fetch, so behavior and cost can drift from the transport API that already uses `TaskService.ListTaskRuns`.

## Triage

- Decision: `valid`
- Notes:
  The extension `tasks/runs` handler currently fetches the full task view and filters runs locally, even though the task manager already exposes `ListTaskRuns` with canonical filtering and lower fetch cost. I will route this endpoint through `ListTaskRuns` and adjust any required interface/test wiring.
  Resolution: Updated the Host API task-manager interface and `tasks/runs` handler to use `ListTaskRuns(...)` directly instead of fetching a full task view and filtering in-memory.
