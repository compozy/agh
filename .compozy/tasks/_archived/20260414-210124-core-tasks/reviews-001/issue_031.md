---
status: resolved
file: internal/store/globaldb/global_db_task.go
line: 567
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:1912770d11aa
review_hash: 1912770d11aa
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 031: Consider returning more specific error when taskID is empty.
## Review Comment

The `ensureTaskExists` function returns `taskpkg.ErrTaskNotFound` when the `taskID` is empty (line 570). This conflates "missing input" with "record not found". Consider returning a validation error for empty input to aid debugging.

## Triage

- Decision: `INVALID`
- Notes:
  No reachable public API path in this file relies on `ensureTaskExists` to validate an empty task id. The public callers normalize and validate task ids earlier (`requireTaskValue`, `TaskRun.Validate`, and parent-id guards) before they ever invoke this helper.
  Changing `ensureTaskExists` to return `ErrValidation` for blank input would therefore not fix an observed defect in the current external behavior; it would only alter an internal fallback branch that should already be unreachable through supported entry points.
  Resolution: Closed after analysis. The blank-id branch is not reachable from the supported public entry points in this file, so no user-visible defect was fixed here.
