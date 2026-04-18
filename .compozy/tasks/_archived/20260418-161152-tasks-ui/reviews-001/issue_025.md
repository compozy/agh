---
status: resolved
file: internal/store/globaldb/global_db_task_aux.go
line: 141
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:1488c7961401
review_hash: 1488c7961401
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 025: Handle the new rows.Close() errors explicitly.
## Review Comment

Both new query paths discard `Close()` errors with `_`, which hides finalization failures and breaks the repo rule for ignored errors.

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification."

Also applies to: 544-546

## Triage

- Decision: `valid`
- Root cause: several new task-store query paths in `global_db_task_aux.go` discard `rows.Close()` errors, violating the repo rule against ignoring errors and making cursor finalization failures invisible.
- Fix approach: replace the ignored-close patterns in the scoped task-store paths with explicit close/error joining so iteration and cleanup failures are both preserved.

## Resolution

- Replaced the ignored `rows.Close()` paths in `internal/store/globaldb/global_db_task_aux.go` with explicit `joinRowsCloseError(...)` handling.
- Updated the affected list/query methods so iteration failures and cursor-finalization failures are both preserved.
- Verification: `go test ./internal/store/globaldb` and `go test -tags integration ./internal/store/globaldb`
