---
status: resolved
file: internal/store/globaldb/global_db_task_test.go
line: 1148
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:4a791ef847e4
review_hash: 4a791ef847e4
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 027: Consider using existing testutil.EqualStringSlices.
## Review Comment

This local `equalStringSlices` function duplicates `testutil.EqualStringSlices` which is already imported and used elsewhere in this file (line 294, 424, etc.). Consider removing this duplicate and using the shared utility consistently.

## Triage

- Decision: `valid`
- Root cause: `internal/store/globaldb/global_db_task_test.go` defines a local `equalStringSlices` helper even though the file already depends on the shared `testutil.EqualStringSlices` utility.
- Fix approach: remove the duplicate helper and use the shared utility consistently in the remaining call site.

## Resolution

- Removed the duplicate local slice-comparison helper from `internal/store/globaldb/global_db_task_test.go`.
- Switched the remaining call site to the shared `testutil.EqualStringSlices` helper.
- Verification: `go test ./internal/store/globaldb`
