---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/testutil/task_stub.go
line: 390
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:4d14a86e56da
review_hash: 4d14a86e56da
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 017: Return the task sentinel from the default EnqueueRun path.
## Review Comment

`EnqueueRun` creates a run from a task, so falling back to `ErrTaskRunNotFound` sends tests down the wrong not-found branch before a run even exists.

## Triage

- Decision: `VALID`
- Notes:
  `EnqueueRun` creates a run from a task spec, so its default missing-object path should report a missing task, not a missing run. Returning `ErrTaskRunNotFound` misclassifies failures before a run exists.
